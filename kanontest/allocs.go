// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"hash/fnv"
	"reflect"
	"slices"
	"strconv"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// scratchLength is the length of the stack array that the generated code
// appends the encoding of a type that encodes itself into, so that a longer
// encoding allocates.
const scratchLength = 128

// names returns the number of the samples and the 64-bit FNV-1a hash of
// their names, which pin the samples that an allocation check measures in
// the golden file.
func names[T any](samples []sample[T]) string {
	h := fnv.New64a()
	for _, x := range samples {
		// The Write method of a hash.Hash never returns an error.
		_, _ = h.Write([]byte(x.name + "\n"))
	}
	return strconv.Itoa(len(samples)) + " samples, digest " + strconv.FormatUint(h.Sum64(), 16)
}

// encodeAllocates reports whether the encode of x, a value of s, can
// allocate: x has a type that encodes itself, and its family has no append
// method, or its encoding is longer than scratchLength bytes and the type
// is not a kanon.Sizer, whose encoding the generated code appends in place.
func encodeAllocates(s *shape, x reflect.Value) bool {
	if s.kind != kindBinary {
		return false
	}
	enc, _ := marshal(x)
	return !methodsOf(s.typ).appends || !s.sizer && len(enc) > scratchLength
}

// sizeAllocs checks that SizeKanon allocates nothing for every sample that
// measured returns with encodeAllocates.
func (s *suite[T, P]) sizeAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(encodeAllocates) {
		v := x.value
		msg := x.name + ": SizeKanon allocates nothing"
		assert.MaxAllocs(tb, func() { P(&v).SizeKanon() }, 0, msg)
	}
}

// encodeAllocs checks that EncodeKanon into a buffer of the length of the
// encoding allocates nothing for every sample that measured returns with
// encodeAllocates.
func (s *suite[T, P]) encodeAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(encodeAllocates) {
		v := x.value
		buf := make([]byte, len(x.enc))
		msg := x.name + ": EncodeKanon allocates nothing into a buffer of the length of the encoding"
		assert.MaxAllocs(tb, func() { _, _ = P(&v).EncodeKanon(buf) }, 0, msg)
	}
}

// appendAllocs checks that AppendBinary into a buffer with room for the
// encoding allocates nothing for every sample that measured returns with
// encodeAllocates.
func (s *suite[T, P]) appendAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(encodeAllocates) {
		v := x.value
		buf := make([]byte, 0, len(x.enc))
		msg := x.name + ": AppendBinary allocates nothing into a buffer with room for the encoding"
		assert.MaxAllocs(tb, func() { _, _ = P(&v).AppendBinary(buf) }, 0, msg)
	}
}

// decodeAllocs checks that DecodeKanon with a slab, into a receiver that
// decoded the same encoding before, allocates nothing for every sample that
// measured returns with decodeAllocates.
func (s *suite[T, P]) decodeAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(s.r.decodeAllocates) {
		var v T
		p := P(&v)
		opts := kanon.Options{Slab: string(x.enc)}
		_ = p.DecodeKanon(x.enc, opts)
		msg := x.name + ": DecodeKanon allocates nothing into a receiver that decoded the encoding before"
		assert.MaxAllocs(tb, func() { _ = p.DecodeKanon(x.enc, opts) }, 0, msg)
	}
}

// keepAllocs checks that DecodeKanon with the zero Options, into a receiver
// that decoded the same encoding before, allocates nothing for every sample
// that measured returns with keepAllocates. The decode keeps every string of
// the receiver, which equals the decoded string at its place.
func (s *suite[T, P]) keepAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(s.r.keepAllocates) {
		var v T
		p := P(&v)
		_ = p.DecodeKanon(x.enc, kanon.Options{})
		msg := x.name + ": DecodeKanon with the zero Options allocates nothing into a receiver that decoded the " +
			"encoding before"
		assert.MaxAllocs(tb, func() { _ = p.DecodeKanon(x.enc, kanon.Options{}) }, 0, msg)
	}
}

// copyOnce checks, for every sample that measured returns with
// keepAllocates, that DecodeKanon with the zero Options allocates one copy of
// the input at most into a receiver that decoded the same encoding before
// and whose strings then changed, as [resolver.changeFields] changes them. It
// then checks that the strings of a decode with the zero Options do not
// change with the input.
func (s *suite[T, P]) copyOnce(tb assert.TB) {
	tb.Helper()
	pkg := s.l.typ.PkgPath()
	for _, x := range s.measured(s.r.keepAllocates) {
		var v T
		_ = P(&v).DecodeKanon(x.enc, kanon.Options{})
		rv := reflect.ValueOf(&v).Elem()
		changed := func() *T {
			s.r.changeFields(s.l, rv, pkg)
			return &v
		}
		msg := x.name + ": DecodeKanon with the zero Options copies the input once for the strings that change"
		assert.MaxAllocsWithSetup(tb, changed, func(p *T) { _ = P(p).DecodeKanon(x.enc, kanon.Options{}) }, 1, msg)
		in := slices.Clone(x.enc)
		_ = P(changed()).DecodeKanon(in, kanon.Options{})
		assert.Pure(tb, func() []byte { return s.r.fingerprint(s.l, rv) }, func() { clear(in) },
			x.name+": the strings that DecodeKanon decodes with the zero Options do not change with the input")
	}
}

// measured returns the samples whose encode or decode the allocation checks
// measure: the samples that encode and decode, except for the bulk samples,
// whose maps are larger than the stack arrays of the generated code, and
// except for the samples whose reference decode contains a value for which
// allocates reports true, as kanon.Message lists the values that allocate.
func (s *suite[T, P]) measured(allocates func(*shape, reflect.Value) bool) []sample[T] {
	var out []sample[T]
	for _, x := range encodes(s.samples) {
		want, err := s.reference(probe{data: x.enc})
		if err == nil && !s.r.fieldsContain(s.l, reflect.ValueOf(&want).Elem(), allocates) {
			out = append(out, x)
		}
	}
	return out
}

// fieldsContain reports whether a field of v, a struct of l, contains a
// value at any depth for which is reports true. A union member that the
// discriminator does not select counts as none, since the codec neither
// encodes nor decodes it.
func (r *resolver) fieldsContain(l *layout, v reflect.Value, is func(*shape, reflect.Value) bool) bool {
	for _, f := range l.fields {
		if f.Union != "" && !l.selected(v, f) {
			continue
		}
		if r.contains(f.shape, f.of(v), is) {
			return true
		}
	}
	return false
}

// contains reports whether x, a value of s, or a value that it contains at
// any depth, is a value for which is reports true. An interface whose value
// has a type that its Types do not list, as in a struct that no Spec
// describes, counts as containing one.
func (r *resolver) contains(s *shape, x reflect.Value, is func(*shape, reflect.Value) bool) bool {
	if is(s, x) {
		return true
	}
	switch s.kind {
	case kindStruct:
		return r.fieldsContain(r.looseLayout(s.typ), x, is)
	case kindInline:
		return r.fieldsContain(s.layout, x, is)
	case kindSlice, kindArray:
		for i := range x.Len() {
			if r.contains(s.elem, x.Index(i), is) {
				return true
			}
		}
	case kindMap:
		for it := x.MapRange(); it.Next(); {
			if r.contains(s.key, it.Key(), is) || r.contains(s.elem, it.Value(), is) {
				return true
			}
		}
	case kindPointer:
		return !x.IsNil() && r.contains(s.elem, x.Elem(), is)
	case kindInterface:
		if x.IsNil() {
			return false
		}
		w := s.variant(x.Elem().Type())
		return w == nil || r.contains(w.shape, x.Elem(), is)
	default:
	}
	return false
}

// decodeAllocates reports whether the decode of x, a value of s, into a
// receiver that decoded it before can allocate: x has a type that decodes
// itself, is a time in a zone other than UTC whose offset the decode does
// not share, as wire.SharesZone reports, is a map with entries whose keys
// refer to memory, or is an interface that stores a value other than a
// pointer. The offset alone decides a time, so that a time in the local zone
// at an offset that the decode does not share counts as allocating on every
// machine.
func (r *resolver) decodeAllocates(s *shape, x reflect.Value) bool {
	switch s.kind {
	case kindBinary:
		return true
	case kindTime:
		t, _ := reflect.TypeAssert[time.Time](x)
		_, off := t.Zone()
		return t.Location() != time.UTC && !wire.SharesZone(off)
	case kindMap:
		return x.Len() > 0 && r.refers(s.key)
	case kindInterface:
		return !x.IsNil() && x.Elem().Kind() != reflect.Pointer
	default:
		return false
	}
}

// keepAllocates reports whether the decode of x, a value of s, with the zero
// Options into a receiver that decoded it before can allocate: x is a value
// for which decodeAllocates reports true, or a map with a string that is not
// empty in a key, or in a value that does not refer to memory, as
// [resolver.memory] reports. The decode clears a map before it reads its
// entries and takes only the values that refer to memory from the map, so it
// copies the input for such a string.
func (r *resolver) keepAllocates(s *shape, x reflect.Value) bool {
	if r.decodeAllocates(s, x) {
		return true
	}
	if s.kind != kindMap {
		return false
	}
	text := func(t *shape, y reflect.Value) bool { return t.kind == kindString && y.Len() > 0 }
	for it := x.MapRange(); it.Next(); {
		if r.contains(s.key, it.Key(), text) || !r.memory(s.elem) && r.contains(s.elem, it.Value(), text) {
			return true
		}
	}
	return false
}

// memory reports whether a value of s refers to memory that a decode reuses:
// a pointer, a byte slice, a slice, a map, an interface, or a struct or an
// array that contains one. The decode of a map takes such values from the
// values of the map. A struct contains itself only through a pointer, a
// slice, a map or an interface, so the walk ends.
func (r *resolver) memory(s *shape) bool {
	switch s.kind {
	case kindPointer, kindBytes, kindSlice, kindMap, kindInterface:
		return true
	case kindInline:
		return slices.ContainsFunc(s.layout.fields, func(f *field) bool { return r.memory(f.shape) })
	case kindStruct:
		return slices.ContainsFunc(r.looseLayout(s.typ).fields, func(f *field) bool { return r.memory(f.shape) })
	case kindArray:
		return r.memory(s.elem)
	default:
		return false
	}
}

// changeFields changes every string that is not empty in v, a struct of l,
// as [resolver.changeIn] changes it. A union member that the discriminator
// does not select is left alone, since the codec does not decode it.
func (r *resolver) changeFields(l *layout, v reflect.Value, pkg string) {
	for _, f := range l.fields {
		if f.Union != "" && !l.selected(v, f) {
			continue
		}
		r.changeIn(f.shape, f.of(v), pkg)
	}
}

// changeIn changes every string that is not empty in x, an addressable value
// of s, which the decode of a struct of the package pkg reads itself: the
// string takes a value of its length whose first byte differs. The walk
// enters structs, the elements of slices and arrays, the values that
// pointers point at, the values of maps and the pointers that interfaces
// store. It leaves out a map key, which the decode reads into a new key, and
// a struct with a kanon codec of another package, which copies its own
// encoding.
func (r *resolver) changeIn(s *shape, x reflect.Value, pkg string) {
	switch s.kind {
	case kindString:
		if x.Len() > 0 {
			b := []byte(x.String())
			b[0] ^= 1
			x.SetString(string(b))
		}
	case kindInline:
		r.changeFields(s.layout, x, pkg)
	case kindStruct:
		if s.typ.PkgPath() == pkg {
			r.changeFields(r.looseLayout(s.typ), x, pkg)
		}
	case kindPointer:
		if !x.IsNil() {
			r.changeIn(s.elem, x.Elem(), pkg)
		}
	case kindSlice, kindArray:
		for i := range x.Len() {
			r.changeIn(s.elem, x.Index(i), pkg)
		}
	case kindMap:
		for it := x.MapRange(); it.Next(); {
			value := reflect.New(s.elem.typ).Elem()
			value.Set(it.Value())
			r.changeIn(s.elem, value, pkg)
			x.SetMapIndex(it.Key(), value)
		}
	case kindInterface:
		if x.IsNil() || x.Elem().Kind() != reflect.Pointer {
			return
		}
		if w := s.variant(x.Elem().Type()); w != nil {
			r.changeIn(w.shape, x.Elem(), pkg)
		}
	default:
		// No other value contains a string that the decode reads.
	}
}

// refers reports whether a map key of s refers to memory, which the decode
// of the key allocates: a pointer, an interface, or a struct or an array
// that contains one. A struct contains itself only through a pointer or an
// interface, so the walk ends.
func (r *resolver) refers(s *shape) bool {
	switch s.kind {
	case kindPointer, kindInterface:
		return true
	case kindStruct, kindInline:
		for _, f := range r.keyLayout(s.typ).fields {
			if r.refers(f.shape) {
				return true
			}
		}
	case kindArray:
		return r.refers(s.elem)
	default:
	}
	return false
}
