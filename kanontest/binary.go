// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"encoding"
	"encoding/gob"
	"reflect"
)

// family is a family of methods through which a type encodes itself. The
// zero family is none.
type family uint8

// Families, in the order in which the generator tries them.
const (
	// familyBinary is AppendBinary or MarshalBinary, with UnmarshalBinary.
	familyBinary family = 1
	// familyGob is GobEncode with GobDecode.
	familyGob family = 2
	// familyText is AppendText or MarshalText, with UnmarshalText.
	familyText family = 3
)

// familyOf returns the first family whose decode method, and whose append
// or encode method, the pointer to t has, and 0 when there is none. It does
// not tell a type that encodes itself from time.Time and from a struct with
// a kanon codec, which [resolver.shapeOf] classifies first.
func familyOf(t reflect.Type) family {
	p := reflect.PointerTo(t)
	has := func(i reflect.Type) bool { return p.Implements(i) }
	binaryEncode := has(reflect.TypeFor[encoding.BinaryAppender]()) || has(reflect.TypeFor[encoding.BinaryMarshaler]())
	textEncode := has(reflect.TypeFor[encoding.TextAppender]()) || has(reflect.TypeFor[encoding.TextMarshaler]())
	if binaryEncode && has(reflect.TypeFor[encoding.BinaryUnmarshaler]()) {
		return familyBinary
	}
	if has(reflect.TypeFor[gob.GobEncoder]()) && has(reflect.TypeFor[gob.GobDecoder]()) {
		return familyGob
	}
	if textEncode && has(reflect.TypeFor[encoding.TextUnmarshaler]()) {
		return familyText
	}
	return 0
}

// appends reports whether the pointer to t, a type that encodes itself, has
// the append method of its family, which the generated code calls with a
// stack array: AppendBinary or AppendText. GobEncode has no append method.
func appends(t reflect.Type) bool {
	p := reflect.PointerTo(t)
	switch familyOf(t) {
	case familyBinary:
		return p.Implements(reflect.TypeFor[encoding.BinaryAppender]())
	case familyText:
		return p.Implements(reflect.TypeFor[encoding.TextAppender]())
	default:
		return false
	}
}

// marshal returns the encoding of v, a value of a type that encodes itself,
// through the method that the generated code calls: the append method of
// its family when the type has one, and the encode method otherwise.
func marshal(v reflect.Value) ([]byte, error) {
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	x := p.Interface()
	switch familyOf(v.Type()) {
	case familyBinary:
		if a, ok := x.(encoding.BinaryAppender); ok {
			return a.AppendBinary(nil)
		}
		return x.(encoding.BinaryMarshaler).MarshalBinary()
	case familyGob:
		return x.(gob.GobEncoder).GobEncode()
	default:
		if a, ok := x.(encoding.TextAppender); ok {
			return a.AppendText(nil)
		}
		return x.(encoding.TextMarshaler).MarshalText()
	}
}

// unmarshal sets x, an addressable value of a type that encodes itself, to
// its zero value and decodes data into it through the decode method of its
// family, as the generated code decodes it, and returns the error of the
// method.
func unmarshal(x reflect.Value, data []byte) error {
	x.SetZero()
	p := x.Addr().Interface()
	switch familyOf(x.Type()) {
	case familyBinary:
		return p.(encoding.BinaryUnmarshaler).UnmarshalBinary(data)
	case familyGob:
		return p.(gob.GobDecoder).GobDecode(data)
	default:
		return p.(encoding.TextUnmarshaler).UnmarshalText(data)
	}
}
