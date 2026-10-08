// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"slices"
	"time"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// Widths of the fixed-size values, in bytes.
const (
	fixed32Width = 4
	fixed64Width = 8
	// complex128Width is the length of the encoding of a complex128 after
	// its length.
	complex128Width = 16
)

// mode is the variant of the encoding that an encoder writes.
type mode uint8

// Modes of an encoder. The zero mode writes the reference encoding.
const (
	// modeReference writes the reference encoding: the encoding that the
	// wire format gives a value.
	modeReference mode = 0
	// modeFingerprint writes the fingerprint of a value, which differs for
	// two values that a decode tells apart: every union member by the rules
	// of presence, whatever the discriminators, the discriminators after the
	// fields of their struct, every NaN as one bit pattern, and the entries
	// of a map ordered by key and then by encoding.
	modeFingerprint mode = 1
	// modeRedundant writes every field of every inline struct twice,
	// whatever its presence and its union, except a nil pointer, and
	// unknownFields after the fields: an encoding that no encoder writes,
	// and that a decode merges.
	modeRedundant mode = 2
)

// wideVarint is the value that a fault writes for the varint of an
// integer: a value of 41 bits, outside every integer type of 32 bits or
// less.
const wideVarint = 1 << 40

// breach is a rule of the canonical encoding that a fault breaks at one site,
// the target of the fault, for the probes of a canonical Spec. Each breach
// counts its own sites. The zero breach breaks no rule: its fault writes a
// value wrong.
type breach uint8

// Breaches of the rules of the canonical encoding.
const (
	// breachLong writes a varint one byte longer than its shortest form. Its
	// sites are the varints of a struct encoding: tags, lengths, integers,
	// bools, type numbers and the length of a complex128.
	breachLong breach = 1
	// breachFields swaps two adjacent fields. Its sites are the pairs of
	// adjacent fields that a struct encoding writes.
	breachFields breach = 2
	// breachKeys swaps two adjacent entries of a map. Its sites are the pairs
	// of adjacent entries that a map writes.
	breachKeys breach = 3
	// breachUnknown writes unknownFields at a boundary of the fields of a
	// struct encoding. Its sites are the boundaries: before each field that
	// the encoding writes, and after the last.
	breachUnknown breach = 4
	// breachZero writes a field at the zero value of its type, as a selected
	// union member is written: in place of the value that the encoding
	// writes, or where the encoding leaves the field out, and with a pointer
	// to the zero value of its target for a pointer. Its sites are the fields
	// of a struct encoding.
	breachZero breach = 5
	// breachTag writes a malformed tag at a boundary of the fields of a struct
	// encoding: a tag cut short, as cutTag, which ends the encoding, the tag of
	// field number 0, as zeroTag, and that tag one byte longer than its
	// shortest form, as longZeroTag. Its sites are the three tags at each
	// boundary.
	breachTag breach = 6
	// breachNegZero writes a float component of a map key that is zero as
	// -0.0: a float, a part of a complex number, an element of an array, and
	// a field of a struct, which the projection leaves out at +0.0. Its sites
	// are those components.
	breachNegZero breach = 7
	// breachZeroBytes writes the bytes of a value with a length as zero bytes,
	// and keeps its length, so that a field of a type that encodes itself
	// decodes to its zero value, whose encoding it leaves out. Its sites are
	// the values with a length.
	breachZeroBytes breach = 8
)

// Malformed tags that a fault of breachTag writes.
var (
	// cutTag is the first byte of a tag of two bytes, which ends an encoding.
	cutTag = []byte{0x80}
	// zeroTag is the tag of field number 0 with the wire format varint, and a
	// value of 0.
	zeroTag = []byte{0x00, 0x00}
	// longZeroTag is the tag of zeroTag one byte longer than its shortest
	// form, which a canonical decode rejects before it checks the number.
	longZeroTag = []byte{0x80, 0x00}
)

// encoder writes the reference encoding of values: the encoding that the
// wire format gives them, derived by reflection, or a variant of it that
// its mode selects. A struct with a kanon codec encodes through its own
// methods, a type that encodes itself through the methods of its family,
// and a time through package wire.
type encoder struct {
	r    *resolver
	mode mode
	// fault writes one value of the encoding wrong when it is not nil.
	fault *fault
	// reject writes one value of a kanon.Validator as a value that its
	// ValidateKanon rejects when it is not nil.
	reject *rejection
	// from leaves out the first from fields of every inline struct, and the
	// first from elements of every slice and entries of every map, so that
	// the next one comes first in the order of the decode.
	from int
	// err is the error of the last value in the order of the encoding that
	// fails to encode. The generated code writes backward and returns the
	// first failure that it meets, which is this one.
	err error
	// key reports that the encoder writes a map key, whose projection
	// writes every float component of -0.0 as +0.0: a float field of -0.0
	// is absent, and a struct with a kanon codec that contains a float
	// writes its fields in place of its methods.
	key bool
	// negate makes the encoder write the next float component as -0.0: the
	// zero field of a struct key that a fault of breachNegZero targets,
	// which [encoder.present] reports present.
	negate bool
}

// fault makes an encoder write one value wrong: the value that the encoder
// completes as its target-th, counted from 1 in the order in which it
// completes the values with a length and the varints of integers, inner
// values first. A value with a length keeps its first at bytes when at is
// below its length, and takes zero bytes after it up to at otherwise, and
// the lengths around it follow it. A varint takes the value wideVarint.
//
// A fault whose breach is not zero breaks the rule of its breach at its
// target-th site instead, counted from 1 in the order in which the encoder
// meets the sites of the breach, and leaves the bytes of every other site as
// the encoding writes them.
type fault struct {
	target, at int
	breach     breach
	// lengths records the length of every value that the fault counts, and
	// -1 for a varint and for a site of a breach.
	lengths []int
}

// rejection makes an encoder write one value of a kanon.Validator as the
// value that [resolver.rejected] returns for its shape: the value that the
// encoder meets as its target-th, counted from 1 in the order in which it
// meets the values of the Validators that reject a value, outer values
// first.
type rejection struct {
	target int
	// met counts the values that the rejection counts.
	met int
}

// encode returns the reference encoding of v, a struct of l, and the error
// of the codec for it.
func (r *resolver) encode(l *layout, v reflect.Value) ([]byte, error) {
	e := &encoder{r: r, mode: modeReference}
	b := e.encodeStruct(l, v)
	return b, e.err
}

// fingerprint returns the fingerprint of v, a struct of l, as
// modeFingerprint writes it: two decoded values of l are equal when their
// fingerprints are.
func (r *resolver) fingerprint(l *layout, v reflect.Value) []byte {
	e := &encoder{r: r, mode: modeFingerprint}
	return e.encodeStruct(l, v)
}

// redundant returns the redundant encoding of v, a struct of l, as
// modeRedundant writes it.
func (r *resolver) redundant(l *layout, v reflect.Value) []byte {
	e := &encoder{r: r, mode: modeRedundant}
	return e.encodeStruct(l, v)
}

// breached returns the reference encoding of v, a struct of l, with the rule
// of the breach of ft broken at the target of ft, as [fault] states.
func (r *resolver) breached(l *layout, v reflect.Value, ft *fault) []byte {
	e := &encoder{r: r, fault: ft}
	return e.encodeStruct(l, v)
}

// encodeStruct returns the encoding of v, a struct of l, in the mode of e:
// its present fields in ascending field number, the member that each
// discriminator selects whatever its value, and the unknown fields that v
// keeps. A fault of a breach of the fields changes them at its target, as
// [encoder.encodeField] and [encoder.breakFields] state.
func (e *encoder) encodeStruct(l *layout, v reflect.Value) []byte {
	var parts [][]byte
	for _, f := range l.fields[min(e.from, len(l.fields)):] {
		if part := e.encodeField(l, v, f); len(part) > 0 {
			parts = append(parts, part)
		}
	}
	b := slices.Concat(e.breakFields(parts)...)
	if l.unknown != nil {
		b = append(b, v.FieldByIndex(l.unknown).Bytes()...)
	}
	if e.mode == modeRedundant {
		return append(b, unknownFields...)
	}
	if e.mode != modeFingerprint {
		return b
	}
	for _, u := range l.unions() {
		d := v.FieldByIndex(u[0].disc)
		if d.CanInt() {
			b = binary.AppendVarint(b, d.Int())
		} else {
			b = binary.AppendUvarint(b, d.Uint())
		}
	}
	return b
}

// encodeField returns the encoding of the field f of v, a struct of l, in the
// mode of e: twice whatever its presence in the redundant encoding, every
// union member by the rules of presence in a fingerprint, and otherwise the
// field when it is present or the member that its discriminator selects.
// Every field is a site of breachZero, whose fault writes the field at the
// zero value of its type, as [encoder.appendSelected] writes a member, at its
// target.
func (e *encoder) encodeField(l *layout, v reflect.Value, f *field) []byte {
	if e.site(breachZero) {
		return e.appendSelected(nil, l, f, reflect.Zero(f.shape.typ))
	}
	x := f.of(v)
	if e.mode == modeRedundant {
		return e.appendField(e.appendField(nil, l.loc(f), f, x, true), l.loc(f), f, x, true)
	}
	if f.Union == "" || e.mode == modeFingerprint {
		return e.appendField(nil, l.loc(f), f, x, false)
	}
	if l.selected(v, f) {
		return e.appendSelected(nil, l, f, x)
	}
	return nil
}

// breakFields returns parts, the encodings of the fields of a struct encoding
// in their order, with the rule of a fault of breachFields, breachUnknown or
// breachTag broken at its target: two adjacent fields swapped, unknownFields
// at a boundary, or a malformed tag at a boundary. cutTag ends the encoding,
// so that it takes the place of the fields after the boundary.
func (e *encoder) breakFields(parts [][]byte) [][]byte {
	swap(e, breachFields, parts)
	for k := range len(parts) + 1 {
		if e.site(breachUnknown) {
			return slices.Insert(parts, k, unknownFields)
		}
	}
	for k := range len(parts) + 1 {
		if e.site(breachTag) {
			return append(parts[:k], cutTag)
		}
		if e.site(breachTag) {
			return slices.Insert(parts, k, zeroTag)
		}
		if e.site(breachTag) {
			return slices.Insert(parts, k, longZeroTag)
		}
	}
	return parts
}

// swap swaps parts[k-1] and parts[k] for the site of the breach b that is
// the target of the fault of e, counting one site for each k from 1: the
// pairs of adjacent fields of a struct encoding, or of adjacent entries of a
// map.
func swap[E any](e *encoder, b breach, parts []E) {
	for k := 1; k < len(parts); k++ {
		if e.site(b) {
			parts[k-1], parts[k] = parts[k], parts[k-1]
		}
	}
}

// appendSelected appends the field f of l, whose value is x, as a union
// member that its discriminator selects: whatever its presence, and with a
// pointer to the zero value of its target for a nil pointer.
func (e *encoder) appendSelected(b []byte, l *layout, f *field, x reflect.Value) []byte {
	if f.shape.kind == kindPointer && x.IsNil() {
		x = reflect.New(f.shape.elem.typ)
	}
	return e.appendField(b, l.loc(f), f, x, true)
}

// appendField appends the field f, whose value is x and whose errors loc
// names, when it is present or a selected union member: its tag and the
// value that the tag introduces, with a length before a pointer and an
// interface there. A type that encodes itself and whose == compares every
// bit is absent at its zero value, which does not encode. The value of a
// struct and of any other type that encodes itself encodes before its
// presence is known, as the generated code encodes it, so its failure
// counts when the value has no bytes. A field with the tag option max whose
// value has more elements than its bound fails with kanon.ErrMax, after the
// failures of its value, since the generated code checks the bound before it
// writes the value. The encoding still has the value.
func (e *encoder) appendField(b []byte, loc string, f *field, x reflect.Value, member bool) []byte {
	s := f.shape
	if s.kind == kindPointer {
		if x.IsNil() {
			return b
		}
		s, x, member = s.elem, x.Elem(), true
	}
	var value []byte
	if member {
		value = e.appendValue(nil, s, x, loc, f.Number)
	} else if s.zeroAbsent {
		if x.IsZero() {
			return b
		}
		value = e.appendValue(nil, s, x, loc, f.Number)
	} else if s.kind == kindStruct || s.kind == kindInline || s.kind == kindBinary {
		if value = e.appendValue(nil, s, x, loc, f.Number); len(value) == 1 {
			return b
		}
	} else if e.present(s, x, loc, f.Number) {
		value = e.appendValue(nil, s, x, loc, f.Number)
	} else {
		return b
	}
	if f.Max > 0 && x.Len() > f.Max {
		e.fail(wire.MarshalError(kanon.ErrMax, loc, f.Number))
	}
	b = e.appendUvarint(b, uint64(f.Number)<<3|uint64(s.wire()))
	if s.unframed() {
		return e.appendBytes(b, value)
	}
	return append(b, value...)
}

// present reports whether a field whose value is x, a value of s, is
// present: a bool when it is true, a number when a bit of it is set, a
// string, a slice and a map with an element, a time that is not at the zero
// instant or not in UTC, an array with a present element, a pointer and an
// interface that are not nil, a type that encodes itself and whose ==
// compares every bit when it is not its zero value, and any other value
// whose encoding has bytes after its length. In a map key, a float and a
// complex number are present when they are not zero, since the projection
// writes -0.0 as +0.0, and a zero one when a fault of breachNegZero targets
// it, as [encoder.negates] states. A value that fails to encode itself has no
// bytes, and its failure is not recorded: the generated code sizes it
// without an error. loc and num locate the field of x.
func (e *encoder) present(s *shape, x reflect.Value, loc string, num int) bool {
	switch s.kind {
	case kindFloat:
		if e.key {
			return x.Float() != 0 || e.negates()
		}
		return math.Float64bits(x.Float()) != 0
	case kindComplex64, kindComplex128:
		c := x.Complex()
		if e.key {
			return c != 0 || e.negates()
		}
		return math.Float64bits(real(c))|math.Float64bits(imag(c)) != 0
	case kindString, kindBytes, kindSlice, kindMap:
		return x.Len() > 0
	case kindArray:
		for i := range x.Len() {
			if e.present(s.elem, x.Index(i), loc, num) {
				return true
			}
		}
		return false
	case kindTime:
		t, _ := reflect.TypeAssert[time.Time](x)
		return !t.IsZero() || t.Location() != time.UTC
	case kindStruct, kindInline, kindBinary:
		if s.zeroAbsent {
			return !x.IsZero()
		}
		scratch := &encoder{r: e.r, mode: e.mode, key: e.key}
		return len(scratch.appendValue(nil, s, x, loc, num)) > 1
	case kindPointer, kindInterface:
		return !x.IsNil()
	default:
		return !x.IsZero()
	}
}

// appendValue appends the encoding of x, a value of s, as it occurs inside
// a container, to b, or the value that the rejection of e writes in its
// place. loc and num locate the field of the value in the error of a value
// that fails to encode. A kanon.Validator records the error of its
// ValidateKanon after the errors of the values that it contains, as the
// generated code, which writes backward, meets it before theirs.
func (e *encoder) appendValue(b []byte, s *shape, x reflect.Value, loc string, num int) []byte {
	x = e.rejects(s, x)
	b = e.appendKind(b, s, x, loc, num)
	if s.validate {
		if err := validate(x); err != nil {
			e.fail(wire.MarshalError(err, loc, num))
		}
	}
	return b
}

// rejects returns the value that e writes for x, a value of s: the value
// that [resolver.rejected] returns for s when s is a kanon.Validator that
// rejects a value and x is the target of the rejection of e, and x
// otherwise. It counts the values of such Validators.
func (e *encoder) rejects(s *shape, x reflect.Value) reflect.Value {
	if e.reject == nil || !s.validate {
		return x
	}
	if bad, ok := e.r.rejected(s); ok {
		e.reject.met++
		if e.reject.met == e.reject.target {
			return bad
		}
	}
	return x
}

// appendKind appends the encoding of x, a value of the kind of s, to b, as
// [encoder.appendValue] states.
func (e *encoder) appendKind(b []byte, s *shape, x reflect.Value, loc string, num int) []byte {
	switch s.kind {
	case kindBool:
		if x.Bool() {
			return e.appendUvarint(b, 1)
		}
		return e.appendUvarint(b, 0)
	case kindInt:
		return e.appendVarint(b, wire.Zigzag(x.Int()))
	case kindUint:
		return e.appendVarint(b, x.Uint())
	case kindFixed:
		if x.CanUint() {
			return appendFixed(b, s, x.Uint())
		}
		return appendFixed(b, s, uint64(x.Int()))
	case kindFloat:
		if s.typ.Size() == fixed32Width {
			return appendFixed(b, s, uint64(math.Float32bits(written(e, float32Of(x)))))
		}
		return appendFixed(b, s, math.Float64bits(written(e, x.Float())))
	case kindComplex64:
		c := complex64Of(x)
		b = binary.LittleEndian.AppendUint32(b, math.Float32bits(written(e, real(c))))
		return binary.LittleEndian.AppendUint32(b, math.Float32bits(written(e, imag(c))))
	case kindComplex128:
		c := x.Complex()
		b = e.appendUvarint(b, complex128Width)
		b = binary.LittleEndian.AppendUint64(b, math.Float64bits(written(e, real(c))))
		return binary.LittleEndian.AppendUint64(b, math.Float64bits(written(e, imag(c))))
	case kindString:
		return e.appendBytes(b, []byte(x.String()))
	case kindBytes:
		return e.appendBytes(b, x.Bytes())
	case kindByteArray:
		return e.appendBytes(b, byteArray(x))
	case kindTime:
		t, _ := reflect.TypeAssert[time.Time](x)
		body := make([]byte, wire.SizeTime(t))
		wire.PutTime(body, len(body), t)
		return e.appendBytes(b, body)
	case kindStruct:
		if e.key && e.r.floats(s) {
			return e.appendBytes(b, e.encodeStruct(e.r.keyLayout(s.typ), x))
		}
		body, err := encodeCodec(x)
		if err != nil {
			e.fail(err)
			body = failed()
		}
		return e.appendBytes(b, body)
	case kindInline:
		return e.appendBytes(b, e.encodeStruct(s.layout, x))
	case kindBinary:
		body, err := encodeSelf(x)
		if errors.Is(err, errSize) {
			e.fail(wire.SizeError(loc, num))
			body = failed()
		} else if err != nil {
			e.fail(wire.MarshalError(err, loc, num))
			body = failed()
		}
		return e.appendBytes(b, body)
	case kindSlice, kindArray:
		first := 0
		if s.kind == kindSlice {
			first = min(e.from, x.Len())
		}
		var body []byte
		for i := first; i < x.Len(); i++ {
			body = e.appendValue(body, s.elem, x.Index(i), loc, num)
		}
		return e.appendBytes(b, body)
	case kindMap:
		pairs := e.pairs(s, x)
		entries := slices.Clone(pairs[min(e.from, len(pairs)):])
		swap(e, breachKeys, entries)
		var body []byte
		for _, p := range entries {
			body = e.appendKey(body, s.key, p[0], loc, num)
			body = e.appendValue(body, s.elem, p[1], loc, num)
		}
		e.checkKeys(s, pairs, loc, num)
		return e.appendBytes(b, body)
	case kindPointer:
		if x.IsNil() {
			return append(b, 0)
		}
		return e.appendValue(append(b, 1), s.elem, x.Elem(), loc, num)
	default:
		return e.appendInterface(b, s, x, loc, num)
	}
}

// appendInterface appends the encoding of the interface x of s: 0 for nil,
// and the number of the concrete type and its value otherwise. A concrete
// type that the Types of the field do not list fails to encode.
func (e *encoder) appendInterface(b []byte, s *shape, x reflect.Value, loc string, num int) []byte {
	if x.IsNil() {
		return e.appendUvarint(b, 0)
	}
	w := s.variant(x.Elem().Type())
	if w == nil {
		e.fail(wire.UnlistedError(x.Elem().Interface(), loc, num))
		return b
	}
	b = e.appendUvarint(b, uint64(w.num))
	return e.appendValue(b, w.shape, x.Elem(), loc, num)
}

// pairs returns the keys and the values of the map x of s in the order of
// the encoding, ascending as [resolver.compareKeys] orders the keys. Two
// keys of one projection order by the encodings of their entries, so that
// a fingerprint of a map with such keys does not depend on the order of
// the map. The encoding of such a map fails, and the generated code checks
// the keys before it writes an entry.
func (e *encoder) pairs(s *shape, x reflect.Value) [][2]reflect.Value {
	out := make([][2]reflect.Value, 0, x.Len())
	for it := x.MapRange(); it.Next(); {
		out = append(out, [2]reflect.Value{it.Key(), it.Value()})
	}
	slices.SortFunc(out, func(a, b [2]reflect.Value) int {
		if c := e.r.compareKeys(s.key, a[0], b[0]); c != 0 {
			return c
		}
		return bytes.Compare(e.entry(s, a), e.entry(s, b))
	})
	return out
}

// entry returns the encoding of the entry p of a map of s, its key and then
// its value, as an encoder of the mode of e writes it.
func (e *encoder) entry(s *shape, p [2]reflect.Value) []byte {
	scratch := &encoder{r: e.r, mode: e.mode}
	return scratch.appendValue(scratch.appendKey(nil, s.key, p[0], keyLoc, 0), s.elem, p[1], keyLoc, 0)
}

// appendKey appends the projection of x, a map key of s, to b: its encoding
// with every float component of -0.0 written as +0.0.
func (e *encoder) appendKey(b []byte, s *shape, x reflect.Value, loc string, num int) []byte {
	key := e.key
	e.key = true
	b = e.appendValue(b, s, x, loc, num)
	e.key = key
	return b
}

// checkKeys records the error of the encode of a map of s whose sorted
// entries are pairs, which the generated code checks before it writes the
// entries, so that the error follows the errors of the entries: a key with
// a NaN component, and else two keys of one projection, which sort next to
// each other. loc and num locate the field of the map.
func (e *encoder) checkKeys(s *shape, pairs [][2]reflect.Value, loc string, num int) {
	for _, p := range pairs {
		if e.r.hasNaN(s.key, p[0]) {
			e.fail(wire.InvalidKeyError(loc, num))
			return
		}
	}
	for i := 1; i < len(pairs); i++ {
		if e.r.compareKeys(s.key, pairs[i-1][0], pairs[i][0]) == 0 {
			e.fail(wire.MarshalError(kanon.ErrAmbiguousKey, loc, num))
			return
		}
	}
}

// hit counts a value of length n, or -1 for a varint of an integer, which a
// fault that writes a value wrong counts, and a fault of breachLong among its
// sites, and reports whether it is the target of the fault. It reports false
// for an encoder without such a fault.
func (e *encoder) hit(n int) bool {
	f := e.fault
	if f == nil || f.breach != 0 && f.breach != breachLong {
		return false
	}
	return f.count(n)
}

// site counts a site of the breach b, and reports whether it is the target of
// the fault of e. It reports false for an encoder without a fault of b.
func (e *encoder) site(b breach) bool {
	f := e.fault
	if f == nil || f.breach != b {
		return false
	}
	return f.count(-1)
}

// count records a value or a site of length n, and reports whether it is the
// target of f.
func (f *fault) count(n int) bool {
	f.lengths = append(f.lengths, n)
	return len(f.lengths) == f.target
}

// appendVarint appends v, the varint of an integer, to b: wideVarint when
// a fault that writes a value wrong targets it, v one byte longer than its
// shortest form when a fault of breachLong does, and v in its shortest form
// otherwise.
func (e *encoder) appendVarint(b []byte, v uint64) []byte {
	if !e.hit(-1) {
		return binary.AppendUvarint(b, v)
	}
	if e.fault.breach == breachLong {
		return appendLong(b, v)
	}
	return binary.AppendUvarint(b, wideVarint)
}

// appendUvarint appends v to b, a varint that only a fault of breachLong
// counts: a tag, a bool, a type number or the length of a complex128. It
// writes v one byte longer than its shortest form when the fault targets it.
func (e *encoder) appendUvarint(b []byte, v uint64) []byte {
	if e.site(breachLong) {
		return appendLong(b, v)
	}
	return binary.AppendUvarint(b, v)
}

// written returns f, a float or a part of a complex number, as e writes it:
// the NaN of math.NaN for any NaN in a fingerprint, +0.0 for -0.0 in a map
// key, and f bit for bit otherwise. A zero component of a map key is a site
// of breachNegZero, which its fault writes as -0.0 at its target, as it
// writes the component after [encoder.negates] reports true. It takes a
// float32 as a float32, since a conversion to float64 sets the quiet bit of
// a signaling NaN.
func written[F float32 | float64](e *encoder, f F) F {
	if e.mode == modeFingerprint && math.IsNaN(float64(f)) {
		return F(math.NaN())
	}
	if e.key && f == 0 && (e.negate || e.site(breachNegZero)) {
		e.negate = false
		return F(math.Copysign(0, -1))
	}
	if e.key && f == 0 {
		return 0
	}
	return f
}

// negates counts the zero float component of a map key that the encoding
// leaves out, a field of a struct key, as a site of breachNegZero, and
// reports whether it is the target of the fault of e. The encoder then writes
// the component, whose next float [written] writes as -0.0.
func (e *encoder) negates() bool {
	e.negate = e.site(breachNegZero)
	return e.negate
}

// appendLong appends v as a varint one byte longer than its shortest form:
// its shortest form with the continuation bit set on its last byte, and a
// byte of 0.
func appendLong(b []byte, v uint64) []byte {
	b = binary.AppendUvarint(b, v)
	b[len(b)-1] |= 0x80
	return append(b, 0)
}

// fail records err, when it is not nil, as the error of the encoding.
func (e *encoder) fail(err error) {
	if err != nil {
		e.err = err
	}
}

// compareKeys returns -1, 0 or +1 as the map key a of shape s sorts
// before, with or after b, in the order of the wire format: false before
// true; numbers and strings as cmp.Compare orders them, NaN first; complex
// numbers by their real, then their imaginary parts; byte arrays bytewise;
// arrays element by element; times as package wire orders them; pointers
// nil first, then by their values; interfaces by type number, nil first,
// then by value; structs field by field in ascending field number; and a
// type that encodes itself by its encoding.
func (r *resolver) compareKeys(s *shape, a, b reflect.Value) int {
	switch s.kind {
	case kindBool:
		return wire.CompareBool(a.Bool(), b.Bool())
	case kindInt:
		return cmp.Compare(a.Int(), b.Int())
	case kindUint:
		return cmp.Compare(a.Uint(), b.Uint())
	case kindFloat:
		return cmp.Compare(a.Float(), b.Float())
	case kindComplex64, kindComplex128:
		return wire.CompareComplex(a.Complex(), b.Complex())
	case kindString:
		return cmp.Compare(a.String(), b.String())
	case kindByteArray:
		return bytes.Compare(byteArray(a), byteArray(b))
	case kindTime:
		at, _ := reflect.TypeAssert[time.Time](a)
		bt, _ := reflect.TypeAssert[time.Time](b)
		return wire.CompareTime(at, bt)
	case kindArray:
		for i := range a.Len() {
			if c := r.compareKeys(s.elem, a.Index(i), b.Index(i)); c != 0 {
				return c
			}
		}
		return 0
	case kindPointer:
		if a.IsNil() || b.IsNil() {
			return wire.CompareBool(!a.IsNil(), !b.IsNil())
		}
		return r.compareKeys(s.elem, a.Elem(), b.Elem())
	case kindInterface:
		na, nb := s.number(a), s.number(b)
		if na != nb || na == 0 {
			return cmp.Compare(na, nb)
		}
		return r.compareKeys(s.variant(a.Elem().Type()).shape, a.Elem(), b.Elem())
	case kindStruct, kindInline:
		for _, f := range r.keyLayout(s.typ).fields {
			if c := r.compareKeys(f.shape, f.of(a), f.of(b)); c != 0 {
				return c
			}
		}
		return 0
	default:
		ae, _ := marshal(a)
		be, _ := marshal(b)
		return bytes.Compare(ae, be)
	}
}

// floats reports whether the projection of a map key of s, a struct with a
// kanon codec, has a float component: s is a float or a complex number, or
// contains one in an array element, the value that a pointer points at, or
// a field of a struct, as [resolver.keyLayout] lists them. The fields of a
// struct in a map key contain no interface, which the generator rejects.
func (r *resolver) floats(s *shape) bool {
	seen := make(map[*shape]bool)
	var walk func(s *shape) bool
	walk = func(s *shape) bool {
		if seen[s] {
			return false
		}
		seen[s] = true
		switch s.kind {
		case kindFloat, kindComplex64, kindComplex128:
			return true
		case kindArray, kindPointer:
			return walk(s.elem)
		case kindStruct, kindInline:
			return slices.ContainsFunc(r.keyLayout(s.typ).fields, func(f *field) bool { return walk(f.shape) })
		default:
			return false
		}
	}
	return walk(s)
}

// hasNaN reports whether x, a map key of s, has a NaN component in the parts
// of it that its projection contains, as x != x reports it for a float and
// a complex number, whose part of NaN makes it unequal to itself.
func (r *resolver) hasNaN(s *shape, x reflect.Value) bool {
	switch s.kind {
	case kindFloat:
		return math.IsNaN(x.Float())
	case kindComplex64, kindComplex128:
		c := x.Complex()
		return math.IsNaN(real(c)) || math.IsNaN(imag(c))
	case kindArray:
		for i := range x.Len() {
			if r.hasNaN(s.elem, x.Index(i)) {
				return true
			}
		}
		return false
	case kindPointer:
		return !x.IsNil() && r.hasNaN(s.elem, x.Elem())
	case kindInterface:
		if x.IsNil() {
			return false
		}
		w := s.variant(x.Elem().Type())
		return w != nil && r.hasNaN(w.shape, x.Elem())
	case kindStruct, kindInline:
		return slices.ContainsFunc(r.keyLayout(s.typ).fields, func(f *field) bool { return r.hasNaN(f.shape, f.of(x)) })
	default:
		return false
	}
}

// hasNegativeZero reports whether x, a map key of s, has a float component
// of -0.0 in the parts of it that its projection contains, which the
// projection writes as +0.0: a float that is zero with its sign bit set, and
// a complex number with such a part.
func (r *resolver) hasNegativeZero(s *shape, x reflect.Value) bool {
	switch s.kind {
	case kindFloat:
		return negativeZero(x.Float())
	case kindComplex64, kindComplex128:
		c := x.Complex()
		return negativeZero(real(c)) || negativeZero(imag(c))
	case kindArray:
		for i := range x.Len() {
			if r.hasNegativeZero(s.elem, x.Index(i)) {
				return true
			}
		}
		return false
	case kindPointer:
		return !x.IsNil() && r.hasNegativeZero(s.elem, x.Elem())
	case kindInterface:
		if x.IsNil() {
			return false
		}
		w := s.variant(x.Elem().Type())
		return w != nil && r.hasNegativeZero(w.shape, x.Elem())
	case kindStruct, kindInline:
		return slices.ContainsFunc(r.keyLayout(s.typ).fields, func(f *field) bool {
			return r.hasNegativeZero(f.shape, f.of(x))
		})
	default:
		return false
	}
}

// negativeZero reports whether f is zero with its sign bit set. A NaN with
// its sign bit set is not zero.
func negativeZero(f float64) bool {
	return f == 0 && math.Signbit(f)
}

// number returns the type number of the interface value x of s: 0 for nil
// and for a concrete type that the Types of its field do not list.
func (s *shape) number(x reflect.Value) int {
	if x.IsNil() {
		return 0
	}
	if w := s.variant(x.Elem().Type()); w != nil {
		return w.num
	}
	return 0
}

// keyLayout returns the layout that orders the map keys of the struct type
// t: its layout in the Spec, which numbers the fields of an inline struct
// and of a struct with a kanon codec in a map key, or a loose layout for a
// value that a sample fills, whose order only tells equal keys apart.
func (r *resolver) keyLayout(t reflect.Type) *layout {
	if l := r.layouts[t]; l != nil {
		return l
	}
	return r.looseLayout(t)
}

// encodeCodec returns the encoding of x, a struct with a kanon codec,
// through its own SizeKanon and EncodeKanon, as the generated code of an
// enclosing struct writes it, and the error of EncodeKanon.
func encodeCodec(x reflect.Value) ([]byte, error) {
	p := reflect.New(x.Type())
	p.Elem().Set(x)
	m, _ := reflect.TypeAssert[kanon.Message](p)
	buf := make([]byte, m.SizeKanon())
	n, err := m.EncodeKanon(buf)
	return buf[len(buf)-n:], err
}

// failed returns the bytes that a value that fails to encode takes in the
// reference encoding: one byte, as the generated size of a failing value of
// a type that encodes itself counts it, so that the field or the array
// around the value is present and its encode reports the failure.
func failed() []byte {
	return []byte{0}
}

// appendFixed appends the four or eight little-endian bytes of u, a value
// of the fixed-size shape s, to b.
func appendFixed(b []byte, s *shape, u uint64) []byte {
	if s.typ.Size() == fixed32Width {
		return binary.LittleEndian.AppendUint32(b, uint32(u))
	}
	return binary.LittleEndian.AppendUint64(b, u)
}

// appendBytes appends body to b after its length, or the length and the
// bytes that the fault of e gives body when it targets it: body after a
// length one byte longer than its shortest form for a fault of breachLong,
// and zero bytes of the length of body for a fault of breachZeroBytes.
func (e *encoder) appendBytes(b, body []byte) []byte {
	if e.site(breachZeroBytes) {
		body = make([]byte, len(body))
	}
	if e.hit(len(body)) {
		if e.fault.breach == breachLong {
			return append(appendLong(b, uint64(len(body))), body...)
		}
		faulty := make([]byte, e.fault.at)
		copy(faulty, body)
		body = faulty
	}
	return append(binary.AppendUvarint(b, uint64(len(body))), body...)
}

// byteArray returns a copy of the bytes of the byte array x.
func byteArray(x reflect.Value) []byte {
	out := make([]byte, x.Len())
	reflect.Copy(reflect.ValueOf(out), x)
	return out
}

// float32Of returns x, a value of a float32 type, bit for bit. x.Float
// converts x to a float64, which sets the quiet bit of a signaling NaN, and a
// conversion between two float32 types keeps every bit.
func float32Of(x reflect.Value) float32 {
	f, _ := reflect.TypeAssert[float32](x.Convert(reflect.TypeFor[float32]()))
	return f
}

// complex64Of returns x, a value of a complex64 type, bit for bit, read
// through a complex64 view of a copy of x. x.Complex and a conversion to
// complex64 both convert the parts to float64, which sets the quiet bit of a
// signaling NaN.
func complex64Of(x reflect.Value) complex64 {
	p := reflect.New(x.Type())
	p.Elem().Set(x)
	c, _ := reflect.TypeAssert[complex64](complex64View(p))
	return c
}

// complex64View returns the value that p, a pointer to a value of a
// complex64 type, points at, as a complex64, which shares the memory of the
// value.
func complex64View(p reflect.Value) reflect.Value {
	return reflect.NewAt(reflect.TypeFor[complex64](), p.UnsafePointer()).Elem()
}
