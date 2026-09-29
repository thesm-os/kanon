// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"encoding"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"

	"go.thesmos.sh/kanon"
)

// Errors that mark the encoding of a value of a kanon.Sizer that the
// generated code rejects with the error of wire.SizeError.
var (
	// errSize marks an encoding of another length than SizeKanon.
	errSize = errors.New("kanontest: the encoding differs from SizeKanon")
	// errNegative marks a SizeKanon below 0, which the generated code meets
	// before it calls the encode method. It wraps errSize, since the
	// generated code returns the error of wire.SizeError for both.
	errNegative = fmt.Errorf("%w: SizeKanon is below 0", errSize)
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

// failMode is a way in which a value of a type that encodes itself fails to
// encode, as the generated code meets the failure. The zero failMode is
// none.
type failMode uint8

// Ways in which a value of a type that encodes itself fails to encode, in
// the order in which [resolver.failures] lists their values.
const (
	// failsSize is a SizeKanon below 0 of a kanon.Sizer, which the generated
	// code meets before it calls the encode method.
	failsSize failMode = 1
	// failsEncode is an error of the encode method.
	failsEncode failMode = 2
	// failsLength is an encoding of a kanon.Sizer whose length differs from
	// its SizeKanon.
	failsLength failMode = 3
)

// failModeOf returns the way in which x, a value of a type that encodes
// itself, fails to encode, as the error of [encodeSelf] tells it, and 0 when
// x encodes.
func failModeOf(x reflect.Value) failMode {
	_, err := encodeSelf(x)
	if err == nil {
		return 0
	}
	if errors.Is(err, errNegative) {
		return failsSize
	}
	if errors.Is(err, errSize) {
		return failsLength
	}
	return failsEncode
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

// sizes reports whether the pointer to t, a type that encodes itself, is a
// kanon.Sizer, whose SizeKanon the generated code sizes a value with.
func sizes(t reflect.Type) bool {
	return reflect.PointerTo(t).Implements(reflect.TypeFor[kanon.Sizer]())
}

// sizeKanon returns the SizeKanon of x, a value of a kanon.Sizer.
func sizeKanon(x reflect.Value) int {
	p := reflect.New(x.Type())
	p.Elem().Set(x)
	s, _ := reflect.TypeAssert[kanon.Sizer](p)
	return s.SizeKanon()
}

// encodeSelf returns the encoding of x, a value of a type that encodes
// itself, as the generated code writes it, and its error: the encoding that
// [marshal] returns and the error of the encode method, errNegative for a
// kanon.Sizer whose SizeKanon is below 0, before the encode method runs,
// and errSize for a kanon.Sizer whose SizeKanon differs from the length of
// the encoding.
func encodeSelf(x reflect.Value) ([]byte, error) {
	if !sizes(x.Type()) {
		return marshal(x)
	}
	n := sizeKanon(x)
	if n < 0 {
		return nil, errNegative
	}
	enc, err := marshal(x)
	if err == nil && len(enc) != n {
		return nil, errSize
	}
	return enc, err
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
