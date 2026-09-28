// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"hash/fnv"
	"reflect"
	"strconv"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
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
// method, or its encoding is longer than scratchLength bytes.
func encodeAllocates(s *shape, x reflect.Value) bool {
	if s.kind != kindBinary {
		return false
	}
	enc, _ := marshal(x)
	return !appends(s.typ) || len(enc) > scratchLength
}

// sharedZone reports whether time.FixedZone returns one location for every
// call with the unnamed zone at offset off, as it does for the whole hours
// from UTC-12 to UTC+14, so that the decode of a time at that offset
// allocates no location.
func sharedZone(off int) bool {
	zone := time.FixedZone("", off)
	return zone == time.FixedZone("", off)
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
// itself, is a time in a zone other than UTC whose offset time.FixedZone
// does not share, is a map with entries whose keys refer to memory, or is
// an interface that stores a value other than a pointer. The offset alone
// decides a time, so that a time in the local zone at an offset that
// FixedZone does not share counts as allocating on every machine.
func (r *resolver) decodeAllocates(s *shape, x reflect.Value) bool {
	switch s.kind {
	case kindBinary:
		return true
	case kindTime:
		t, _ := reflect.TypeAssert[time.Time](x)
		_, off := t.Zone()
		return t.Location() != time.UTC && !sharedZone(off)
	case kindMap:
		return x.Len() > 0 && r.refers(s.key)
	case kindInterface:
		return !x.IsNil() && x.Elem().Kind() != reflect.Pointer
	default:
		return false
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
