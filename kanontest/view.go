// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"reflect"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

// viewCheck names the check of the view type of a Spec.
const viewCheck = "View/returns the value of each field as the reference view"

// views checks each method of the view type of T against the reference view
// of the field that the method names, as [resolver.view] returns it, for
// the encoding of every sample that encodes and for every probe of the
// decode checks: the method returns the error of the reference view, and
// without an error its value. A method that names no field of T fails the
// check.
func (s *suite[T, P]) views(tb assert.TB) {
	tb.Helper()
	fields := make(map[string]*field, len(s.l.fields))
	for _, f := range s.l.fields {
		fields[f.Name] = f
	}
	var inputs [][]byte
	for _, x := range s.encodable() {
		inputs = append(inputs, x.enc)
	}
	for _, f := range s.families() {
		for _, p := range f.probes() {
			inputs = append(inputs, p.data)
		}
	}
	for _, data := range inputs {
		for m, method := range reflect.ValueOf(data).Convert(s.view).Methods() {
			f := fields[m.Name]
			if f == nil {
				tb.Fatalf("the view type %v has the method %s, which names no field of %s", s.view, m.Name, s.l.name)
			}
			name := s.view.Name() + "." + m.Name
			out := method.Call(nil)
			err, _ := reflect.TypeAssert[error](out[1])
			want, wantErr := s.r.view(s.l, f, data)
			if !sameError(err, wantErr) {
				tb.Fatalf("%s returns the error of the reference decode\ngot:  %v\nwant: %v", name, err, wantErr)
			}
			if wantErr == nil {
				assert.Equal(tb, s.r.viewPrint(f, out[0]), s.r.viewPrint(f, want),
					name+" returns the value of the reference decode, which their fingerprints compare")
			}
		}
	}
}

// view returns the result of the view method of the field f of a struct of
// l for data, the reference of the generated view methods: the value of the
// last occurrence of f, or of the value that f points at, decoded alone, or
// the zero value when data has no occurrence of f. The value of a string, a
// byte slice and a struct is the bytes after its length, and the value of a
// string or a byte slice of a kanon.Validator is those bytes converted to
// its type, which passes its ValidateKanon at the offset of the value. A
// struct has one occurrence at most, since a decode merges its occurrences,
// and view returns the error of wire.FindOne for it, which wraps
// kanon.ErrRepeatedView for a second one. view returns the error of
// wire.Find for any other value, and the error of the decode of the value.
func (r *resolver) view(l *layout, f *field, data []byte) (reflect.Value, error) {
	s := f.viewShape()
	find := wire.Find
	if s.kind == kindStruct || s.kind == kindInline {
		find = wire.FindOne
	}
	i, err := find(data, f.tag(), l.loc(f))
	if s.kind == kindString || s.kind == kindBytes || s.kind == kindStruct || s.kind == kindInline {
		var b []byte
		if err == nil && i != -1 {
			var start, end int
			start, end, err = length(data, i, 0, l.loc(f), f.Number)
			b = data[start:end]
		}
		if !s.validate {
			return reflect.ValueOf(b), err
		}
		x := reflect.New(s.typ).Elem()
		if err != nil || i == -1 {
			return x, err
		}
		if s.kind == kindString {
			x.SetString(string(b))
		} else {
			x.SetBytes(b)
		}
		return x, validated(s, x, l.loc(f), f.Number, i)
	}
	x := reflect.New(s.typ).Elem()
	if err == nil && i != -1 {
		d := &decoder{r: r}
		_, err = d.read(s, x, data, i, 0, 0, l.loc(f), f.Number)
	}
	return x, err
}

// viewPrint returns the bytes that compare x, a result of the view method of
// the field f: the bytes that the method returns for a string, a byte slice
// and a struct, and the fingerprint of any other value.
func (r *resolver) viewPrint(f *field, x reflect.Value) []byte {
	if x.Kind() == reflect.Slice {
		return x.Bytes()
	}
	e := &encoder{r: r, mode: modeFingerprint}
	return e.appendValue(nil, f.viewShape(), x, "", f.Number)
}

// viewShape returns the shape of the value that the view method of f reads:
// the value of f, or the value that f points at.
func (f *field) viewShape() *shape {
	if f.shape.kind == kindPointer {
		return f.shape.elem
	}
	return f.shape
}
