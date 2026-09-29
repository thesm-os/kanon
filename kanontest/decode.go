// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"math"
	"reflect"
	"slices"
	"strings"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// decoder is the reference decode: it reads encodings into values by
// reflection, as the wire format and the generated code define the decode,
// and fails for malformed input with the error of the generated code, its
// location and its offset.
//
// A struct, a slice, a map, a pointer and an interface are each one level
// below the value that contains them, and the elements of an array are at
// the level of the array. The value of a field is one level below its
// struct. A value at a level below zero fails with kanon.ErrDepth: a struct
// names itself, and a container names its field. A pointer field whose
// target is no level fails at its own level, one below its struct.
type decoder struct {
	r *resolver
	// slab is the slab of the decode, from which a struct with a kanon codec
	// decodes its strings.
	slab string
}

// fields merges data, the fields of a struct of l that start at offset off,
// into v at level lv: each occurrence of a field of l merges into the value
// of the field, and an unknown field is skipped, and appended to the field
// that keeps unknown fields when l has one. The tag of a field of l must
// have the wire format of the field. A field takes a byte at least, so that
// the loop runs len(data) times at most.
func (d *decoder) fields(l *layout, v reflect.Value, data []byte, off, lv int) error {
	i := 0
	for range len(data) {
		if i == len(data) {
			break
		}
		at := i
		tag, n := wire.Uvarint(data[i:])
		if n <= 0 {
			return wire.ReadError(n, l.name, 0, off+i)
		}
		i += n
		f := l.byNumber(tag >> 3)
		var err error
		if f == nil {
			i, err = skip(l, v, tag, data, at, i, off)
		} else if tag != f.tag() {
			err = wire.FormatError(tag, f.wire(), l.loc(f), off+at)
		} else {
			i, err = d.field(l, v, f, data, i, off, lv)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// field decodes the value at data[i] of one occurrence of the field f of v,
// a struct of l at level lv, and returns the offset after it. A struct,
// and the struct that a pointer field points at, merge the occurrence into
// their value, a slice appends its elements, a map adds its entries, an
// interface merges a value of the concrete type that it stores, and any
// other value takes the value of the occurrence. An occurrence of a union
// member replaces the union: it selects the member, whose value it replaces.
func (d *decoder) field(l *layout, v reflect.Value, f *field, data []byte, i, off, lv int) (int, error) {
	s, x, loc, num := f.shape, f.of(v), l.loc(f), f.Number
	if f.Union != "" {
		l.choose(v, f)
	}
	switch s.kind {
	case kindStruct, kindInline:
		start, end, err := length(data, i, off, loc, num)
		if err != nil {
			return 0, err
		}
		return end, d.readStruct(s, x, data[start:end], off+start, lv-1, true)
	case kindSlice, kindMap:
		start, end, err := length(data, i, off, loc, num)
		if err != nil {
			return 0, err
		}
		if s.kind == kindSlice {
			err = d.readSlice(s, x, data[start:end], off+start, lv-1, loc, num)
		} else {
			err = d.readMap(s, x, data[start:end], off+start, lv-1, loc, num)
		}
		if err == nil {
			err = validated(s, x, loc, num, off+i)
		}
		return end, err
	case kindInterface:
		return d.mergeInterface(s, x, data, i, off, lv-1, loc, num)
	case kindPointer:
		return d.pointerField(s, x, data, i, off, lv, loc, num)
	default:
		return d.read(s, x, data, i, off, lv-1, loc, num)
	}
}

// pointerField decodes the value at data[i] of the pointer field x of s,
// whose struct is at level lv, into the value that x points at, or into a
// new one, and returns the offset after it. A struct merges into the value,
// a pointer and an interface take their value from a length of their own,
// and any other value is read as it is read inside a container.
func (d *decoder) pointerField(
	s *shape,
	x reflect.Value,
	data []byte,
	i, off, lv int,
	loc string,
	num int,
) (int, error) {
	e := s.elem
	if !e.level() && lv < 1 {
		return 0, wire.DepthError(loc, num, off+i)
	}
	if x.IsNil() {
		x.Set(reflect.New(e.typ))
	}
	if e.kind == kindStruct || e.kind == kindInline {
		start, end, err := length(data, i, off, loc, num)
		if err != nil {
			return 0, err
		}
		return end, d.readStruct(e, x.Elem(), data[start:end], off+start, lv-2, true)
	}
	if e.unframed() {
		return d.framed(e, x.Elem(), data, i, off, lv-2, loc, num)
	}
	return d.read(e, x.Elem(), data, i, off, lv-2, loc, num)
}

// framed decodes the pointer or the interface x of s at level lv, which a
// field puts in a length of its own at data[i], and returns the offset after
// the length and the value. The value must take the length exactly.
func (d *decoder) framed(s *shape, x reflect.Value, data []byte, i, off, lv int, loc string, num int) (int, error) {
	start, end, err := length(data, i, off, loc, num)
	if err != nil {
		return 0, err
	}
	used, err := d.unframed(s, x, data[start:end], off+start, lv, loc, num)
	if err != nil {
		return 0, err
	}
	if start+used != end {
		return 0, wire.TrailingError(end-start-used, loc, num, off+start+used)
	}
	return end, nil
}

// mergeInterface decodes the interface field x of s at level lv, whose
// length and value start at data[i], and returns the offset after them. The
// occurrence decodes as framed decodes it, and fails as it fails. An
// occurrence of the concrete type that x stores then merges into the value
// when the type merges, as [shape.merges] reports, and any other occurrence
// replaces the value. The decode of an occurrence fails for its bytes and
// its levels alone, whatever x stores, so that the merge fails for none,
// except the ValidateKanon of a kanon.Validator: as the generated code
// validates the merged value alone, the decode of an occurrence that merges
// skips the ValidateKanon of its concrete value, and the merged value
// passes it at the offset of the occurrence's value.
func (d *decoder) mergeInterface(
	s *shape,
	x reflect.Value,
	data []byte,
	i, off, lv int,
	loc string,
	num int,
) (int, error) {
	start, _, err := length(data, i, off, loc, num)
	if err != nil {
		return 0, err
	}
	t, n := wire.Uvarint(data[start:])
	w := s.numbered(t)
	merges := !x.IsNil() && w != nil && x.Elem().Type() == w.shape.typ && w.shape.merges()
	decoded := s
	if merges && w.shape.validate {
		decoded = unvalidated(s, w)
	}
	fresh := reflect.New(s.typ).Elem()
	end, err := d.framed(decoded, fresh, data, i, off, lv, loc, num)
	if err != nil {
		return 0, err
	}
	if !merges {
		x.Set(fresh)
		return end, nil
	}
	c := reflect.New(w.shape.typ).Elem()
	c.Set(x.Elem())
	d.merge(w.shape, c, data, start+n, off, loc, num)
	if err := validated(w.shape, c, loc, num, off+start+n); err != nil {
		return 0, err
	}
	x.Set(c)
	return end, nil
}

// unvalidated returns a copy of the interface shape s in which the variant w
// skips the ValidateKanon of its values, which the values that they contain
// still pass.
func unvalidated(s *shape, w *variant) *shape {
	c := *s
	c.variants = slices.Clone(s.variants)
	for k := range c.variants {
		if c.variants[k].num == w.num {
			plain := *w.shape
			plain.validate = false
			c.variants[k].shape = &plain
		}
	}
	return &c
}

// merge decodes the value of s at data[i] into x as a repeated field of its
// type merges: a slice appends its elements, a map adds its entries, a
// struct merges, and a pointer to a struct merges into the struct that it
// points at, or into a new one. The decode of the value succeeded before and
// checked its levels, so that its decode into x fails for nothing and has
// no level limit.
func (d *decoder) merge(s *shape, x reflect.Value, data []byte, i, off int, loc string, num int) {
	if s.kind == kindPointer {
		// The decode before read the presence byte, which is 0 or 1.
		if data[i] == 0 {
			x.SetZero()
			return
		}
		if x.IsNil() {
			x.Set(reflect.New(s.elem.typ))
		}
		d.merge(s.elem, x.Elem(), data, i+1, off, loc, num)
		return
	}
	start, end, _ := length(data, i, off, loc, num)
	switch s.kind {
	case kindSlice:
		_ = d.readSlice(s, x, data[start:end], off+start, math.MaxInt, loc, num)
	case kindMap:
		_ = d.readMap(s, x, data[start:end], off+start, math.MaxInt, loc, num)
	default:
		_ = d.readStruct(s, x, data[start:end], off+start, math.MaxInt, true)
	}
}

// read decodes the value of s at data[i], at level lv, into x, which takes
// the decoded value whole, and returns the offset after the value. loc and
// num locate the value in errors. A kanon.Validator then passes its
// ValidateKanon, whose error names the offset of the value.
func (d *decoder) read(s *shape, x reflect.Value, data []byte, i, off, lv int, loc string, num int) (int, error) {
	end, err := d.readKind(s, x, data, i, off, lv, loc, num)
	if err == nil {
		err = validated(s, x, loc, num, off+i)
	}
	if err != nil {
		return 0, err
	}
	return end, nil
}

// readKind decodes the value of s at data[i] by the kind of s, as
// [decoder.read] states.
func (d *decoder) readKind(s *shape, x reflect.Value, data []byte, i, off, lv int, loc string, num int) (int, error) {
	switch s.kind {
	case kindBool, kindInt, kindUint:
		u, n := wire.Uvarint(data[i:])
		if n <= 0 {
			return 0, wire.ReadError(n, loc, num, off+i)
		}
		return i + n, setVarint(s, x, u, loc, num, off+i)
	case kindFixed, kindFloat, kindComplex64:
		u, n := fixed(s, data[i:])
		if n <= 0 {
			return 0, wire.ReadError(n, loc, num, off+i)
		}
		setFixed(s, x, u)
		return i + n, nil
	case kindPointer, kindInterface:
		used, err := d.unframed(s, x, data[i:], off+i, lv, loc, num)
		return i + used, err
	default:
		if want, ok := exact(s); ok {
			if l, n := wire.Uvarint(data[i:]); n > 0 && l != uint64(want) {
				return 0, wire.LengthError(l, want, loc, num, off+i)
			}
		}
		start, end, err := length(data, i, off, loc, num)
		if err != nil {
			return 0, err
		}
		return end, d.body(s, x, data[start:end], off+start, lv, loc, num)
	}
}

// body decodes data, the value of s after its length, at level lv and at
// offset off, into x, which takes the decoded value whole.
func (d *decoder) body(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) error {
	switch s.kind {
	case kindComplex128:
		re, _ := wire.Uint64(data)
		im, _ := wire.Uint64(data[fixed64Width:])
		x.SetComplex(complex(math.Float64frombits(re), math.Float64frombits(im)))
	case kindString:
		x.SetString(string(data))
	case kindBytes:
		x.SetBytes(append(x.Bytes()[:0], data...))
	case kindByteArray:
		reflect.Copy(x, reflect.ValueOf(data))
	case kindTime:
		t, err := wire.Time(data, loc, num, off)
		if err != nil {
			return err
		}
		x.Set(reflect.ValueOf(t))
	case kindStruct, kindInline:
		return d.readStruct(s, x, data, off, lv, false)
	case kindBinary:
		if err := unmarshal(x, data); err != nil {
			return wire.UnmarshalError(err, loc, num, off)
		}
	case kindArray:
		return d.readArray(s, x, data, off, lv, loc, num)
	case kindSlice:
		x.SetZero()
		return d.readSlice(s, x, data, off, lv, loc, num)
	default:
		x.SetZero()
		return d.readMap(s, x, data, off, lv, loc, num)
	}
	return nil
}

// readStruct decodes data, the encoding of the struct x of s at level lv
// and at offset off: merged into x with merge set, and into the zero value
// otherwise. A struct with a kanon codec decodes through its own MergeKanon.
func (d *decoder) readStruct(s *shape, x reflect.Value, data []byte, off, lv int, merge bool) error {
	if lv < 0 {
		return wire.DepthError(d.r.name(s), 0, off)
	}
	if !merge {
		x.SetZero()
	}
	if s.kind == kindInline {
		return d.fields(s.layout, x, data, off, lv)
	}
	m, _ := reflect.TypeAssert[kanon.Message](x.Addr())
	return m.MergeKanon(data, wire.Nested(d.slab, off, lv))
}

// readSlice appends the elements in data, a slice of s at level lv and at
// offset off without its length, to the slice x. An element takes a byte
// at least, so that the loop runs len(data) times at most.
func (d *decoder) readSlice(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) error {
	if lv < 0 {
		return wire.DepthError(loc, num, off)
	}
	i := 0
	for range len(data) {
		if i == len(data) {
			break
		}
		e := reflect.New(s.elem.typ).Elem()
		var err error
		if i, err = d.read(s.elem, e, data, i, off, lv-1, loc, num); err != nil {
			return err
		}
		x.Set(reflect.Append(x, e))
	}
	return nil
}

// readArray decodes the elements in data, an array of s at level lv and at
// offset off without its length, into the array x. data must contain
// exactly the elements of x.
func (d *decoder) readArray(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) error {
	i := 0
	for k := range x.Len() {
		var err error
		if i, err = d.read(s.elem, x.Index(k), data, i, off, lv, loc, num); err != nil {
			return err
		}
	}
	if i != len(data) {
		return wire.TrailingError(len(data)-i, loc, num, off+i)
	}
	return nil
}

// readMap adds the entries in data, a map of s at level lv and at offset off
// without its length, to the map x, which it makes when x is nil. Keys of
// one projection are one key: an entry replaces the entry of x whose key
// has its projection, as [resolver.compareKeys] orders them equal, so that
// the later value takes effect. It fails with kanon.ErrInvalidKey at a key
// with a NaN component, and with kanon.ErrAmbiguousKey, before it changes
// x, when two keys of x have one projection. An entry takes a byte at
// least, so that the loop runs len(data) times at most.
func (d *decoder) readMap(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) error {
	if lv < 0 {
		return wire.DepthError(loc, num, off)
	}
	keys := x.MapKeys()
	for a := range keys {
		for b := range a {
			if d.r.compareKeys(s.key, keys[a], keys[b]) == 0 {
				return ambiguousError(loc, num, off)
			}
		}
	}
	if x.IsNil() {
		x.Set(reflect.MakeMap(s.typ))
	}
	i := 0
	for range len(data) {
		if i == len(data) {
			break
		}
		at := i
		k, v := reflect.New(s.key.typ).Elem(), reflect.New(s.elem.typ).Elem()
		var err error
		if i, err = d.read(s.key, k, data, i, off, lv-1, loc, num); err != nil {
			return err
		}
		if d.r.hasNaN(s.key, k) {
			return wire.KeyError(loc, num, off+at)
		}
		if i, err = d.read(s.elem, v, data, i, off, lv-1, loc, num); err != nil {
			return err
		}
		for it := x.MapRange(); it.Next(); {
			if d.r.compareKeys(s.key, it.Key(), k) == 0 {
				x.SetMapIndex(it.Key(), reflect.Value{})
				break
			}
		}
		x.SetMapIndex(k, v)
	}
	return nil
}

// ambiguousError returns the error of a merge into a map whose keys have two
// of one projection, as wire.MergeKeys returns it: a *kanon.DecodeError that
// wraps kanon.ErrAmbiguousKey at offset off, the offset of the entries of
// the map, of the field that loc, "Type.Field", and num locate.
func ambiguousError(loc string, num, off int) error {
	k := strings.LastIndexByte(loc, '.')
	return &kanon.DecodeError{Type: loc[:k], Field: loc[k+1:], Number: num, Offset: off, Err: kanon.ErrAmbiguousKey}
}

// unframed decodes the pointer or the interface x of s at the start of
// data, at level lv and at offset off, and returns the length of its
// encoding.
func (d *decoder) unframed(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) (int, error) {
	if lv < 0 {
		return 0, wire.DepthError(loc, num, off)
	}
	if s.kind == kindPointer {
		return d.readPointer(s, x, data, off, lv, loc, num)
	}
	return d.readInterface(s, x, data, off, lv, loc, num)
}

// readPointer decodes the pointer x of s at the start of data: nil for the
// presence byte 0, and for the presence byte 1 the value that follows, into
// the value that x points at or into a new one.
func (d *decoder) readPointer(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) (int, error) {
	present, err := wire.Presence(data, loc, num, off)
	if err != nil {
		return 0, err
	}
	if !present {
		x.SetZero()
		return 1, nil
	}
	if x.IsNil() {
		x.Set(reflect.New(s.elem.typ))
	}
	return d.read(s.elem, x.Elem(), data, 1, off, lv-1, loc, num)
}

// readInterface decodes the interface x of s at the start of data: nil for
// the type number 0, and for the number of a variant the value of its
// concrete type that follows. It fails for a type number that the Types of
// the field do not list.
func (d *decoder) readInterface(s *shape, x reflect.Value, data []byte, off, lv int, loc string, num int) (int, error) {
	t, i := wire.Uvarint(data)
	if i <= 0 {
		return 0, wire.ReadError(i, loc, num, off)
	}
	if t == 0 {
		x.SetZero()
		return i, nil
	}
	w := s.numbered(t)
	if w == nil {
		return 0, wire.TypeError(t, loc, num, off)
	}
	c := reflect.New(w.shape.typ).Elem()
	i, err := d.read(w.shape, c, data, i, off, lv-1, loc, num)
	if err != nil {
		return 0, err
	}
	x.Set(c)
	return i, nil
}

// skip skips the unknown field of v, a struct of l, whose tag is tag and
// starts at data[at], and whose value starts at data[i], and returns the
// offset after it. The field, tag included, joins the unknown fields of v
// when l keeps them.
func skip(l *layout, v reflect.Value, tag uint64, data []byte, at, i, off int) (int, error) {
	n, err := wire.Skip(data[i:], tag, l.name, 0, off+at)
	if err != nil {
		return 0, err
	}
	i += n
	if l.unknown != nil {
		u := v.FieldByIndex(l.unknown)
		u.SetBytes(append(u.Bytes(), data[at:i]...))
	}
	return i, nil
}

// length reads the length at data[i] of a value that starts at offset off+i,
// and returns the offsets in data of the first byte of the value and of the
// byte after it. It fails when the length does not read or the value runs
// past data.
func length(data []byte, i, off int, loc string, num int) (int, int, error) {
	l, n := wire.Uvarint(data[i:])
	if n <= 0 || uint64(len(data)-i-n) < l {
		return 0, 0, wire.ReadError(n, loc, num, off+i)
	}
	return i + n, i + n + int(l), nil
}

// exact returns the length that a value of s must have after its length,
// and reports whether s has one: 16 for a complex128, the length of a byte
// array, and 0 for an array of no elements.
func exact(s *shape) (int, bool) {
	switch s.kind {
	case kindComplex128:
		return complex128Width, true
	case kindByteArray:
		return s.typ.Len(), true
	case kindArray:
		return 0, s.typ.Len() == 0
	default:
		return 0, false
	}
}

// setVarint sets x, a bool or an integer of s, to the value of the varint u
// at offset off. It fails for a value outside the range of the integer type
// of x.
func setVarint(s *shape, x reflect.Value, u uint64, loc string, num, off int) error {
	switch s.kind {
	case kindBool:
		x.SetBool(u != 0)
	case kindInt:
		v := wire.Unzigzag(u)
		if x.OverflowInt(v) {
			return wire.RangeError(v, x.Kind().String(), loc, num, off)
		}
		x.SetInt(v)
	default:
		if x.OverflowUint(u) {
			return wire.RangeError(u, x.Kind().String(), loc, num, off)
		}
		x.SetUint(u)
	}
	return nil
}

// fixed returns the four or eight little-endian bytes of a value of the
// fixed-size shape s at the start of data, and their length, which is 0 when
// data is shorter.
func fixed(s *shape, data []byte) (uint64, int) {
	if s.wire() == wire.Fixed32 {
		u, n := wire.Uint32(data)
		return uint64(u), n
	}
	return wire.Uint64(data)
}

// setFixed sets x, a value of the fixed-size shape s, to the value whose
// little-endian bits are u: the bits of a float, the bits of the real part
// and then of the imaginary part of a complex64, and the two's complement of
// an integer.
func setFixed(s *shape, x reflect.Value, u uint64) {
	narrow := s.wire() == wire.Fixed32
	switch s.kind {
	case kindComplex64:
		x.SetComplex(complex(float64(math.Float32frombits(uint32(u))), float64(math.Float32frombits(uint32(u>>32)))))
	case kindFloat:
		f := math.Float64frombits(u)
		if narrow {
			f = float64(math.Float32frombits(uint32(u)))
		}
		x.SetFloat(f)
	default:
		if x.CanUint() {
			x.SetUint(u)
		} else if narrow {
			x.SetInt(int64(int32(u)))
		} else {
			x.SetInt(int64(u))
		}
	}
}

// decode merges data, the encoding of a struct of l that starts at offset
// off of slab, into v, as MergeKanon with the depth limit depth merges it,
// and returns the error of the generated code. The merge into a zero v is
// the decode of DecodeKanon.
func (r *resolver) decode(l *layout, v reflect.Value, data []byte, slab string, off, depth int) error {
	d := &decoder{r: r, slab: slab}
	return d.fields(l, v, data, off, depth)
}

// name returns the name of the struct of s in the errors of the generated
// code of T: the name of an inline struct in its Spec, and the name of a
// struct with a kanon codec, qualified by the name of its package outside
// the package of T.
func (r *resolver) name(s *shape) string {
	if s.layout != nil {
		return s.layout.name
	}
	if s.typ.PkgPath() == r.pkg {
		return s.typ.Name()
	}
	return s.typ.String()
}

// byNumber returns the field of l with the field number num, and nil when l
// has none.
func (l *layout) byNumber(num uint64) *field {
	for _, f := range l.fields {
		if uint64(f.Number) == num {
			return f
		}
	}
	return nil
}

// wire returns the wire format of the tag of f: the wire format of the
// value of f, or of the value that f points at for a pointer.
func (f *field) wire() int {
	if f.shape.kind == kindPointer {
		return f.shape.elem.wire()
	}
	return f.shape.wire()
}

// tag returns the tag of f: its field number in the high bits, and its wire
// format in the low three.
func (f *field) tag() uint64 {
	return uint64(f.Number)<<3 | uint64(f.wire())
}

// choose selects the union member f of the struct v of l: it zeroes the
// member that the discriminator of f selects, sets the discriminator to
// the Case of f, and zeroes f, so that the occurrence of f that follows
// replaces its value.
func (l *layout) choose(v reflect.Value, f *field) {
	for _, m := range l.fields {
		if m.Union == f.Union && l.selected(v, m) {
			m.of(v).SetZero()
		}
	}
	v.FieldByIndex(f.disc).Set(reflect.ValueOf(f.Case))
	f.of(v).SetZero()
}

// merges reports whether a repeated occurrence of a value of s merges into
// the value instead of replacing it: a slice, a map, a struct, and a pointer
// to a struct.
func (s *shape) merges() bool {
	switch s.kind {
	case kindSlice, kindMap, kindStruct, kindInline:
		return true
	case kindPointer:
		return s.elem.kind == kindStruct || s.elem.kind == kindInline
	default:
		return false
	}
}

// numbered returns the variant of the interface shape s whose type number
// is t, and nil when the Types of its field list none.
func (s *shape) numbered(t uint64) *variant {
	for k := range s.variants {
		if uint64(s.variants[k].num) == t {
			return &s.variants[k]
		}
	}
	return nil
}
