// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"encoding"
	"encoding/gob"
	"errors"
	"fmt"
	"reflect"
	"sync"

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

// methodSets maps each type to the methods that [methodsOf] finds for it.
// Every resolver and every goroutine of the process reads the same map.
var methodSets sync.Map

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

// methods are the methods through which the pointer to a type encodes and
// decodes the values of the type. The generated code calls the same methods.
// The zero methods have no family. They do not tell a type that encodes
// itself from time.Time or from a struct with a kanon codec. [resolver.shapeOf]
// classifies those types first.
type methods struct {
	// family is the first family whose decode method and append or encode
	// method the pointer has. It is 0 when the pointer has none.
	family family
	// appends reports that the pointer has AppendBinary in the binary family
	// or AppendText in the text family. The generated code calls that method
	// with a stack array. The gob family has no append method.
	appends bool
	// sizer reports that the pointer is a kanon.Sizer. The generated code
	// sizes a value with its SizeKanon.
	sizer bool
	// appender reports that the pointer is a kanon.Appender. The generated
	// code of a kanon.Exact type calls AppendKanon in place of the append
	// method of the family.
	appender bool
}

// methodsOf returns the methods of t. The first call for t tests the pointer
// to t against the interfaces of the families, kanon.Sizer and
// kanon.Appender, and records the methods in methodSets. Every later call
// returns the recorded methods. It is safe for concurrent use.
func methodsOf(t reflect.Type) methods {
	if m, ok := methodSets.Load(t); ok {
		return m.(methods)
	}
	p := reflect.PointerTo(t)
	has := func(i reflect.Type) bool { return p.Implements(i) }
	m := methods{sizer: has(reflect.TypeFor[kanon.Sizer]()), appender: has(reflect.TypeFor[kanon.Appender]())}
	binaryAppends := has(reflect.TypeFor[encoding.BinaryAppender]())
	textAppends := has(reflect.TypeFor[encoding.TextAppender]())
	binaryEncodes := binaryAppends || has(reflect.TypeFor[encoding.BinaryMarshaler]())
	textEncodes := textAppends || has(reflect.TypeFor[encoding.TextMarshaler]())
	if binaryEncodes && has(reflect.TypeFor[encoding.BinaryUnmarshaler]()) {
		m.family, m.appends = familyBinary, binaryAppends
	} else if has(reflect.TypeFor[gob.GobEncoder]()) && has(reflect.TypeFor[gob.GobDecoder]()) {
		m.family = familyGob
	} else if textEncodes && has(reflect.TypeFor[encoding.TextUnmarshaler]()) {
		m.family, m.appends = familyText, textAppends
	}
	methodSets.Store(t, m)
	return m
}

// failMode is a way in which a value of a type that encodes itself fails to
// encode, as the generated code meets the failure. The zero failMode is
// none.
type failMode uint8

// The constants of failMode are the ways in which a value of a type that
// encodes itself fails to encode. [resolver.selfEntriesOf] lists the table
// entries of their values in the same order.
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

// sizeKanon returns the SizeKanon of x, a value of a kanon.Sizer.
func sizeKanon(x reflect.Value) int {
	p := reflect.New(x.Type())
	p.Elem().Set(x)
	s, _ := reflect.TypeAssert[kanon.Sizer](p)
	return s.SizeKanon()
}

// appendKanon appends the encoding of x, a value of a kanon.Appender, to b
// through AppendKanon, and returns the extended slice.
func appendKanon(x reflect.Value, b []byte) []byte {
	p := reflect.New(x.Type())
	p.Elem().Set(x)
	a, _ := reflect.TypeAssert[kanon.Appender](p)
	return a.AppendKanon(b)
}

// encodeSelf returns the encoding of x, a value of a type that encodes
// itself, as the generated code writes it, and its error: the encoding that
// [marshal] returns and the error of the encode method, errNegative for a
// kanon.Sizer whose SizeKanon is below 0, before the encode method runs,
// and errSize for a kanon.Sizer whose SizeKanon differs from the length of
// the encoding.
func encodeSelf(x reflect.Value) ([]byte, error) {
	if !methodsOf(x.Type()).sizer {
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

// marshal returns the encoding of v and the error of its encode. The type
// of v encodes itself. marshal calls the append method of the family when
// [methodsOf] reports one, and the encode method otherwise. The generated
// code calls the same method.
func marshal(v reflect.Value) ([]byte, error) {
	m := methodsOf(v.Type())
	if m.appends {
		return appendTo(v, nil)
	}
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	switch x := p.Interface(); m.family {
	case familyBinary:
		return x.(encoding.BinaryMarshaler).MarshalBinary()
	case familyGob:
		return x.(gob.GobEncoder).GobEncode()
	default:
		return x.(encoding.TextMarshaler).MarshalText()
	}
}

// appendTo appends the encoding of v to b through the append method of its
// family, and returns the extended slice and the error of the method. The
// type of v has that method when [methodsOf] reports appends.
func appendTo(v reflect.Value, b []byte) ([]byte, error) {
	p := reflect.New(v.Type())
	p.Elem().Set(v)
	if methodsOf(v.Type()).family == familyBinary {
		return p.Interface().(encoding.BinaryAppender).AppendBinary(b)
	}
	return p.Interface().(encoding.TextAppender).AppendText(b)
}

// unmarshal sets x, an addressable value of a type that encodes itself, to
// its zero value and decodes data into it through the decode method of its
// family, as the generated code decodes it, and returns the error of the
// method.
func unmarshal(x reflect.Value, data []byte) error {
	x.SetZero()
	p := x.Addr().Interface()
	switch methodsOf(x.Type()).family {
	case familyBinary:
		return p.(encoding.BinaryUnmarshaler).UnmarshalBinary(data)
	case familyGob:
		return p.(gob.GobDecoder).GobDecode(data)
	default:
		return p.(encoding.TextUnmarshaler).UnmarshalText(data)
	}
}
