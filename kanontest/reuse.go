// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"errors"
	"reflect"
	"slices"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// reuse checks that DecodeKanon decodes the reference encoding of every
// sample that encodes into a receiver that decoded before as into a zero
// one: a receiver that decoded a bulk sample, whose maps have more entries
// than a decode reuses, a receiver that decoded the first wide sample and
// was reset, and a receiver that decoded the sample itself. Each receiver
// has a value in every field that the encoding leaves out, as
// [resolver.taint] sets them, and the decode clears them.
func (s *suite[T, P]) reuse(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		want, wantErr := s.reference(probe{data: x.enc})
		for _, seed := range slices.Concat(s.bulk, []sample[T]{x}) {
			v := s.tainted(seed)
			err := P(&v).DecodeKanon(x.enc, kanon.Options{})
			s.same(tb, x.name+": DecodeKanon into a receiver that decoded "+seed.name, v, err, want, wantErr)
			s.clears(tb, x.name+": DecodeKanon into a receiver that decoded "+seed.name, &v)
		}
		v := s.tainted(s.bulk[0])
		P(&v).Reset()
		err := P(&v).DecodeKanon(x.enc, kanon.Options{})
		s.same(tb, x.name+": DecodeKanon into a reset receiver", v, err, want, wantErr)
	}
}

// merge checks that MergeKanon merges the reference encoding of every table
// sample, in a slab between two slabGuard, into the decoded value of every
// table sample that decodes, as the reference decode merges it. Both
// receivers have a value in every field that the encoding leaves out, map
// keys and values included, which the merge keeps where it keeps the struct
// of the field. A map key with such a value is a key that no decode yields,
// which the merge replaces with the decoded key of its projection.
func (s *suite[T, P]) merge(tb assert.TB) {
	tb.Helper()
	for _, a := range s.tables {
		if _, err := s.reference(probe{data: a.enc}); err != nil {
			continue
		}
		for _, b := range s.tables {
			slab := slabGuard + string(b.enc) + slabGuard
			got := s.tainted(a)
			gotErr := P(&got).MergeKanon(b.enc, kanon.Options{Slab: slab, Offset: len(slabGuard)})
			want, _ := s.reference(probe{data: a.enc})
			s.r.taint(s.l, reflect.ValueOf(&want).Elem())
			wantErr := s.r.decode(s.l, reflect.ValueOf(&want).Elem(), b.enc, slab, len(slabGuard), kanon.DefaultDepth)
			name := b.name + " merged into " + a.name + ": MergeKanon"
			s.same(tb, name, got, gotErr, want, wantErr)
			assert.Equal(tb, s.r.cleared(s.l, reflect.ValueOf(&got).Elem()),
				s.r.cleared(s.l, reflect.ValueOf(&want).Elem()),
				name+" keeps the fields that the encoding leaves out as the reference decode keeps them")
		}
	}
}

// mergeFailing checks that MergeKanon merges the reference encoding of each
// table sample into each sample of it that fails to encode, built anew, as
// the reference decode merges it. The merge fails with kanon.ErrAmbiguousKey
// at a map of the receiver with two keys of one projection when the
// encoding has the field of the map, before the map changes. Any other
// failing value merges as a value that encodes.
func (s *suite[T, P]) mergeFailing(tb assert.TB) {
	tb.Helper()
	for i, a := range s.tables {
		for j := range s.fails[i] {
			got, _ := reflect.TypeAssert[T](s.failed(i, j))
			gotErr := P(&got).MergeKanon(a.enc, kanon.Options{})
			want := s.failed(i, j)
			wantErr := s.r.decode(s.l, want, a.enc, string(a.enc), 0, kanon.DefaultDepth)
			w, _ := reflect.TypeAssert[T](want)
			s.same(tb, a.name+" merged into "+failingName(i, j)+": MergeKanon", got, gotErr, w, wantErr)
		}
	}
}

// reset checks that Reset of a nil receiver returns, and that Reset sets a
// receiver that decoded a sample, with a value in every field that the
// encoding leaves out, to the zero value.
func (s *suite[T, P]) reset(tb assert.TB) {
	tb.Helper()
	var p P
	p.Reset()
	zero := s.r.fingerprint(s.l, reflect.New(s.l.typ).Elem())
	for _, x := range s.encodable() {
		v := s.tainted(x)
		P(&v).Reset()
		assert.Equal(tb, s.r.fingerprint(s.l, reflect.ValueOf(&v).Elem()), zero,
			x.name+": Reset sets the receiver to the zero value")
		s.clears(tb, x.name+": Reset", &v)
	}
}

// clone checks that CloneKanon returns nil for a nil receiver, and a copy of
// every sample and of the value that each sample that encodes decodes to,
// with the fingerprint of the receiver. A sample with two map keys of one
// projection is left out: its copy can have one key for them, since the copy
// clears the fields that the encoding leaves out of a key and copies a key
// of a type that encodes itself through its encoding. The copy of a decoded
// value, whose fields that the encoding leaves out have values, has the zero
// value in them. The copy keeps its fingerprint when the receiver decodes the
// first wide sample into the memory that it keeps: the values that its
// pointers point at, the arrays of its slices and its maps, so that the copy
// shares none of them with the receiver.
func (s *suite[T, P]) clone(tb assert.TB) {
	tb.Helper()
	var p P
	c, ok := any(p).(kanon.Cloner[T])
	assert.True(tb, ok, "the codec implements kanon.Cloner")
	assert.Nil(tb, c.CloneKanon(), "CloneKanon returns nil for a nil receiver")
	for _, x := range s.all() {
		if errors.Is(x.err, kanon.ErrAmbiguousKey) {
			// The copy of two keys of one projection can be one key.
			continue
		}
		v := x.value
		s.copies(tb, x.name, &v)
	}
	for _, x := range s.encodable() {
		d := s.tainted(x)
		cp := s.copies(tb, x.name+", decoded,", &d)
		s.clears(tb, x.name+": CloneKanon", cp)
		before := s.r.fingerprint(s.l, reflect.ValueOf(cp).Elem())
		_ = P(&d).DecodeKanon(s.bulk[0].enc, kanon.Options{})
		assert.Equal(tb, s.r.fingerprint(s.l, reflect.ValueOf(cp).Elem()), before,
			x.name+": the copy that CloneKanon returns shares no memory with the receiver")
	}
}

// copies checks that CloneKanon returns a copy of the value that p points
// at, with its fingerprint, and returns the copy.
func (s *suite[T, P]) copies(tb assert.TB, name string, p *T) *T {
	tb.Helper()
	cp := any(P(p)).(kanon.Cloner[T]).CloneKanon()
	assert.Equal(tb, s.r.fingerprint(s.l, reflect.ValueOf(cp).Elem()), s.r.fingerprint(s.l, reflect.ValueOf(p).Elem()),
		name+": CloneKanon returns a copy of the receiver")
	return cp
}

// decoded returns the value of T that DecodeKanon decodes from the encoding
// of x into a zero receiver. The decode checks cover its error.
func (*suite[T, P]) decoded(x sample[T]) T {
	var v T
	_ = P(&v).DecodeKanon(x.enc, kanon.Options{})
	return v
}

// tainted returns the value of T that DecodeKanon decodes from the encoding
// of x, with a value in every field that the encoding leaves out, as
// [resolver.taint] sets them. The value does not share memory with x.
func (s *suite[T, P]) tainted(x sample[T]) T {
	v := s.decoded(x)
	s.r.taint(s.l, reflect.ValueOf(&v).Elem())
	return v
}

// clears checks that every field that the encoding leaves out of the value
// that p points at has the zero value, after the method that op names.
func (s *suite[T, P]) clears(tb assert.TB, op string, p *T) {
	tb.Helper()
	assert.False(tb, slices.Contains(s.r.cleared(s.l, reflect.ValueOf(p).Elem()), false),
		op+" clears the fields that the encoding leaves out")
}

// taint sets every field that the encoding leaves out of v, a struct of l,
// as [resolver.leftOutFields] walks them, as [resolver.taintField] sets it.
func (r *resolver) taint(l *layout, v reflect.Value) {
	r.leftOutFields(l, v, r.taintField)
}

// taintField sets x, an addressable field that the encoding leaves out, to
// the value that [resolver.prepare] found for its type, when
// [resolver.taintOf] returns one. It sets the field through its address,
// since reflection does not set an unexported field, so that the checks
// cover the unexported fields that the encoding leaves out, those of a
// struct of another package included. The value is not zero, and every
// tainted field of its type shares it, which no code changes.
func (r *resolver) taintField(x reflect.Value) {
	if t := r.taints[x.Type()]; t.IsValid() {
		reflect.NewAt(x.Type(), x.Addr().UnsafePointer()).Elem().Set(t)
	}
}

// prepare fills the caches of r that the checks read while they run in
// parallel, from the shapes that the fields of l reach: the loose layout of
// every struct with a kanon codec among them, and the value of
// [resolver.taintOf] for the type of every field that the encoding leaves
// out of a struct among them.
func (r *resolver) prepare(l *layout) {
	shapes := make(map[*shape]bool)
	layouts := make(map[*layout]bool)
	var walkLayout func(l *layout)
	var walkShape func(s *shape)
	walkLayout = func(l *layout) {
		if layouts[l] {
			return
		}
		layouts[l] = true
		for sf := range l.typ.Fields() {
			if _, ok := r.taints[sf.Type]; leftOut(sf) && !ok {
				r.taints[sf.Type], _ = r.taintOf(sf.Type)
			}
		}
		for _, f := range l.fields {
			walkShape(f.shape)
		}
	}
	walkShape = func(s *shape) {
		if s == nil || shapes[s] {
			return
		}
		shapes[s] = true
		switch s.kind {
		case kindInline:
			walkLayout(s.layout)
		case kindStruct:
			walkLayout(r.looseLayout(s.typ))
		default:
			walkShape(s.elem)
			walkShape(s.key)
			for _, w := range s.variants {
				walkShape(w.shape)
			}
		}
	}
	walkLayout(l)
}

// cleared returns, for every field that the encoding leaves out of v, a
// struct of l, in the order of [resolver.leftOutFields], whether the field
// has the zero value.
func (r *resolver) cleared(l *layout, v reflect.Value) []bool {
	var out []bool
	r.leftOutFields(l, v, func(x reflect.Value) { out = append(out, x.IsZero()) })
	return out
}

// taintOf returns a value of type t that is not zero, and false for a type
// of which it returns none: a function that the checks never call, a
// channel, and for any other type that kanon encodes, the value of table
// entry 1, which is present for every type with a value of its own. An
// interface without concrete types, a struct without exported fields that
// kanon encodes, and a type that kanon does not encode have none.
func (r *resolver) taintOf(t reflect.Type) (reflect.Value, bool) {
	switch t.Kind() {
	case reflect.Func:
		return reflect.MakeFunc(t, nil), true
	case reflect.Chan:
		return reflect.MakeChan(reflect.ChanOf(reflect.BothDir, t.Elem()), 0).Convert(t), true
	default:
		s, err := r.shapeOf(t, opts{seen: make(map[seenKey]*shape), loose: true})
		if err != nil {
			return reflect.Value{}, false
		}
		x := r.build(table(fieldEntry), s, nesting)
		return x, !x.IsZero()
	}
}

// leftOutFields calls visit with every field that the encoding leaves out,
// as [leftOut] reports, of v, a struct of l, and of the structs that v
// contains by value, points at, lists in a slice or an array, or has as a
// key or a value of a map, in the order of the fields of their types. The
// walk enters the fields that l lists, and a struct with a kanon codec
// through its loose layout. It does not enter an interface, whose value
// reflection cannot set in place.
func (r *resolver) leftOutFields(l *layout, v reflect.Value, visit func(reflect.Value)) {
	for sf := range l.typ.Fields() {
		if leftOut(sf) {
			visit(v.FieldByIndex(sf.Index))
		}
	}
	for _, f := range l.fields {
		r.leftOutIn(f.shape, f.of(v), visit)
	}
}

// leftOutIn calls visit with every field that the encoding leaves out of the
// structs that x, a value of s, contains by value, points at or lists, as
// [resolver.leftOutFields] walks them.
func (r *resolver) leftOutIn(s *shape, x reflect.Value, visit func(reflect.Value)) {
	switch s.kind {
	case kindInline:
		r.leftOutFields(s.layout, x, visit)
	case kindStruct:
		r.leftOutFields(r.looseLayout(s.typ), x, visit)
	case kindPointer:
		if !x.IsNil() {
			r.leftOutIn(s.elem, x.Elem(), visit)
		}
	case kindSlice, kindArray:
		for i := range x.Len() {
			r.leftOutIn(s.elem, x.Index(i), visit)
		}
	case kindMap:
		r.leftOutInMap(s, x, visit)
	default:
		// Reflection cannot set the value of an interface in place, and no
		// other value contains a struct.
	}
}

// leftOutInMap calls visit with every field that the encoding leaves out of
// the keys and the values of x, a map of s, entry by entry in the order of
// the keys, as [resolver.compareKeys] orders them. Reflection cannot set a
// key or a value of a map in place, so the walk visits a copy of each entry
// and stores the copy in place of the entry. The keys of x have distinct
// projections, as the keys of a decode have, so the copies are distinct
// keys after visit sets them.
func (r *resolver) leftOutInMap(s *shape, x reflect.Value, visit func(reflect.Value)) {
	keys := x.MapKeys()
	slices.SortFunc(keys, func(a, b reflect.Value) int { return r.compareKeys(s.key, a, b) })
	for _, k := range keys {
		key := reflect.New(s.key.typ).Elem()
		key.Set(k)
		value := reflect.New(s.elem.typ).Elem()
		value.Set(x.MapIndex(k))
		r.leftOutIn(s.key, key, visit)
		r.leftOutIn(s.elem, value, visit)
		x.SetMapIndex(k, reflect.Value{})
		x.SetMapIndex(key, value)
	}
}
