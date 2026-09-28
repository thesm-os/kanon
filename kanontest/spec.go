// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"reflect"

	"go.thesmos.sh/kanon"
)

// Codec constrains P to the pointer to the struct type T, which implements
// the methods of [kanon.Message] that kanon generates.
type Codec[T any] interface {
	*T
	kanon.Message
}

// Spec describes a struct type T to the checks: what the Go type of T does
// not state about its encoding. kanon generates one Spec per -type struct
// into the test file of its code file, and the fields of T that a Spec
// leaves out are not encoded.
type Spec[T any] struct {
	// Fields describes every encoded field of T, in declaration order.
	Fields []Field
	// Structs describes every inline struct in the types of the fields of
	// T: a struct type without a kanon codec, which the code file encodes
	// with functions of its own.
	Structs []Struct
	// Keys describes the struct types with a kanon codec in the map keys of
	// the code file of T, whose field numbers order the keys. Their Name
	// and Unknown are empty.
	Keys []Struct
	// Unknown names the []byte field of T that keeps the unknown fields of
	// a decode, and is empty when T keeps none.
	Unknown string
	// View is a nil value of the view type of T, which kanon declares with
	// the -views flag, and nil when T has none. The checks compare each
	// method of the view type with the field that it reads.
	View any
}

// Field describes one encoded field of a struct.
type Field struct {
	// Name is the name of the Go struct field. An unexported field, which a
	// kanon tag opts into the encoding, is named as any other field.
	Name string
	// Number is the field number.
	Number int
	// Fixed reports the tag option fixed: the 32- and 64-bit integers in
	// the type of the field take the fixed-size encoding, except in map
	// keys.
	Fixed bool
	// Union names the discriminator of a union member, and is empty for
	// any other field.
	Union string
	// Case is the value of the discriminator that selects the member.
	Case any
	// Types lists the concrete types that the interfaces in the type of
	// the field store, as the tag option types lists them.
	Types []ConcreteType
}

// ConcreteType is a concrete type that the interfaces of a field store.
type ConcreteType struct {
	// Type is the concrete type, which is not an interface type.
	Type reflect.Type
	// Number is the type number that precedes a value of Type on the wire.
	// Number 0 is nil.
	Number int
}

// Struct describes an inline struct.
type Struct struct {
	// Type is the struct type.
	Type reflect.Type
	// Name names the struct in the errors of the codec.
	Name string
	// Fields describes every encoded field of Type, in declaration order.
	Fields []Field
	// Unknown names the []byte field of Type that keeps the unknown fields
	// of a decode, and is empty when Type keeps none.
	Unknown string
}
