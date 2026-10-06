// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"errors"
	"reflect"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// Names of the checks of the view type of a Spec and of its index type.
const (
	viewCheck  = "View/returns the value of each field as the reference view"
	indexCheck = "IndexKanon/returns the value of each field as the reference view"
)

// indexName names the method of a view type that returns its index.
const indexName = "IndexKanon"

// indexError returns the error of the reference index of data, an encoding
// of a struct of l, for the fields that a view type reads: the first error,
// by offset in data, of the scans of the reference views of the fields, as
// [field.find] returns them, which one scan of data meets in that order,
// and nil when none of them fails.
func indexError(l *layout, fields []*field, data []byte) error {
	var first *kanon.DecodeError
	for _, f := range fields {
		_, err := f.find()(data, f.tag(), l.loc(f))
		if d, ok := errors.AsType[*kanon.DecodeError](err); ok && (first == nil || d.Offset < first.Offset) {
			first = d
		}
	}
	if first == nil {
		return nil
	}
	return first
}

// views checks each method of the view type of T against the reference view
// of the field that the method names, as [suite.viewResult] checks it, for
// every input of inputs. The table checks pass the inputs that
// [suite.viewInputs] returns. IndexKanon names no field, and
// [suite.indexes] checks it.
func (s *suite[T, P]) views(tb assert.TB, inputs [][]byte) {
	tb.Helper()
	fields := s.l.byName()
	for _, data := range inputs {
		for m, method := range reflect.ValueOf(data).Convert(s.view).Methods() {
			if m.Name != indexName {
				s.viewResult(tb, fields, s.view, m.Name, data, method.Call(nil))
			}
		}
	}
}

// indexes checks IndexKanon of the view type of T for every input of
// inputs: it returns an error of the [errorIdentity] of the reference index,
// as [indexError] returns it, and without an error each method of the index
// returns what the reference view of the field that it names returns, as
// [suite.viewResult] checks it. The table checks pass the inputs that
// [suite.viewInputs] returns.
func (s *suite[T, P]) indexes(tb assert.TB, inputs [][]byte) {
	tb.Helper()
	fields := s.l.byName()
	var read []*field
	for m := range s.view.Methods() {
		if f := fields[m.Name]; f != nil {
			read = append(read, f)
		}
	}
	for _, data := range inputs {
		out := reflect.ValueOf(data).Convert(s.view).MethodByName(indexName).Call(nil)
		err, _ := reflect.TypeAssert[error](out[1])
		if gotID, wantID := identityOf(err), identityOf(indexError(s.l, read, data)); gotID != wantID {
			assert.Equal(tb, gotID, wantID, s.view.Name()+"."+indexName+" returns the error of the reference index")
		}
		if err != nil {
			continue
		}
		for m, method := range out[0].Methods() {
			s.viewResult(tb, fields, out[0].Type(), m.Name, data, method.Call(nil))
		}
	}
}

// viewInputs returns the inputs of the checks of the view type: the
// encoding of every sample that encodes, and every probe of the decode
// checks.
func (s *suite[T, P]) viewInputs() [][]byte {
	var inputs [][]byte
	for _, x := range s.encodable() {
		inputs = append(inputs, x.enc)
	}
	for _, f := range s.families() {
		for _, p := range f.probes() {
			inputs = append(inputs, p.data)
		}
	}
	return inputs
}

// viewResult checks out, the results of the method named method of the
// view or index type owner for data, against the reference view of the
// field that the method names, which fields maps by name, as
// [resolver.view] returns it: the method returns an error of the
// [errorIdentity] of the reference view, and without an error its value, as
// their [resolver.viewPrint] compares them. A method that names no field of
// T fails the check.
func (s *suite[T, P]) viewResult(tb assert.TB, fields map[string]*field, owner reflect.Type, method string,
	data []byte, out []reflect.Value,
) {
	tb.Helper()
	f := fields[method]
	name := owner.Name() + "." + method
	assert.NotNil(tb, f, name+" names a field of "+s.l.name)
	err, _ := reflect.TypeAssert[error](out[1])
	want, wantErr := s.r.view(s.l, f, data)
	if gotID, wantID := identityOf(err), identityOf(wantErr); gotID != wantID {
		assert.Equal(tb, gotID, wantID, name+" returns the error of the reference decode")
	}
	if wantErr == nil {
		assert.Equal(tb, s.r.viewPrint(s.l, f, out[0]), s.r.viewPrint(s.l, f, want),
			name+" returns the value of the reference decode, which their fingerprints compare")
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
	d := &decoder{r: r}
	i, err := f.find()(data, f.tag(), l.loc(f))
	if s.kind == kindString || s.kind == kindBytes || s.kind == kindStruct || s.kind == kindInline {
		var b []byte
		if err == nil && i != -1 {
			var start, end int
			start, end, err = d.length(data, i, 0, l.loc(f), f.Number)
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
		_, err = d.read(s, x, data, i, 0, 0, l.loc(f), f.Number)
	}
	return x, err
}

// viewPrint returns the bytes that compare x, a result of the view method of
// the field f of a struct of l: the bytes that the method returns for a
// string, a byte slice and a struct, no bytes for a value that the encoding
// leaves out, as [encoder.present] reports it, and the fingerprint of any
// other value. A value that the encoding leaves out is the result for a
// field that the encoding does not contain, and neither a method of its
// type nor its ValidateKanon has to accept it.
func (r *resolver) viewPrint(l *layout, f *field, x reflect.Value) []byte {
	if x.Kind() == reflect.Slice {
		return x.Bytes()
	}
	s, loc := f.viewShape(), l.loc(f)
	e := &encoder{r: r, mode: modeFingerprint}
	if !e.present(s, x, loc, f.Number) {
		return nil
	}
	return e.appendValue(nil, s, x, loc, f.Number)
}

// viewShape returns the shape of the value that the view method of f reads:
// the value of f, or the value that f points at.
func (f *field) viewShape() *shape {
	if f.shape.kind == kindPointer {
		return f.shape.elem
	}
	return f.shape
}

// find returns the scan that the view method of f finds its field with:
// wire.FindOne for a struct, which one byte slice cannot merge, and
// wire.Find for any other value.
func (f *field) find() func(data []byte, tag uint64, loc string) (int, error) {
	if s := f.viewShape(); s.kind == kindStruct || s.kind == kindInline {
		return wire.FindOne
	}
	return wire.Find
}

// byName returns the fields of l by name.
func (l *layout) byName() map[string]*field {
	fields := make(map[string]*field, len(l.fields))
	for _, f := range l.fields {
		fields[f.Name] = f
	}
	return fields
}
