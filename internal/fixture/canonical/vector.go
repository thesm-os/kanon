// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import "time"

//go:generate go tool kanon -type=Inner,Numbers,Lists,Maps,Container,Tree,Times,Keys,Bytes,Opaque,Fixture,Patch,Holder,Repeats,Nest -canonical

// Inner is the vector type of a string and an integer.
type Inner struct {
	Label string
	Count int64
}

// Numbers is the vector type of integers, a bool and pointers, with gaps
// between its field numbers.
type Numbers struct {
	A uint32
	D int64  `kanon:"4,fixed"`
	E bool   `kanon:"5"`
	F int32  `kanon:"6"`
	H *int32 `kanon:"8"`
	I *bool  `kanon:"9"`
	J *int64 `kanon:"10,fixed"`
}

// Lists is the vector type of a slice.
type Lists struct {
	Values32 []int32
}

// Maps is the vector type of maps with string keys.
type Maps struct {
	Attrs  map[string]string
	Counts map[string]int64
}

// Container is the vector type of a struct with a kanon codec behind a
// pointer and in a slice of pointers.
type Container struct {
	Name     string
	Inner    *Inner
	Children []*Inner
}

// Tree is the vector type that contains itself.
type Tree struct {
	Label    string
	Children []*Tree
}

// Times is the vector type of times and a duration.
type Times struct {
	CreatedAt time.Time
	Timeout   time.Duration
	Stamps    []time.Time
}

// Keys is the vector type of map keys of a float and of a pointer.
type Keys struct {
	Scores map[float64]string
	Refs   map[*int32]string
}

// Bytes is the vector type of a byte slice.
type Bytes struct {
	Payload []byte
}

// Opaque is the vector type of the types that encode themselves.
type Opaque struct {
	Digest Digest
	Stamp  Stamp
}

// Fixture is the vector type of a string, an integer, a bool, a byte array
// and a slice of strings, with gaps between its field numbers.
type Fixture struct {
	ID      string
	Score   int64    `kanon:"4"`
	Enabled bool     `kanon:"6"`
	Ref     [32]byte `kanon:"8"`
	Tags    []string `kanon:"9"`
}

// Patch is the vector type of a fixed-size integer and a byte array.
type Patch struct {
	Fixed64Val int64    `kanon:"7,fixed"`
	BlobRef    [32]byte `kanon:"8"`
}

// Shape is the interface of the shapes of Holder.
type Shape interface {
	shape()
}

// Circle is a Shape with a value receiver, which the code file encodes as an
// inline struct.
type Circle struct {
	Radius float64
}

// shape marks Circle as a Shape.
func (Circle) shape() {}

// Square is a Shape with a pointer receiver, which the code file encodes as
// an inline struct.
type Square struct {
	Side int32
}

// shape marks *Square as a Shape.
func (*Square) shape() {}

// Point is the map key and the pointer target of Holder, an inline struct.
type Point struct {
	X int32
	Y int32
}

// HolderKind selects the member of the union of Holder.
type HolderKind uint8

// The members of the union of Holder.
const (
	HolderKindText HolderKind = 1
	HolderKindNum  HolderKind = 2
)

// Holder is the vector type of an interface, a union, a pointer to a
// pointer, an array, maps with struct and time keys, a pointer to an inline
// struct, and an interface that contains itself.
type Holder struct {
	Shape  Shape `kanon:"1,types=Circle|*Square"`
	Kind   HolderKind
	Text   string              `kanon:"2,union=Kind"`
	Num    int64               `kanon:"3,union=Kind"`
	PP     **int32             `kanon:"4"`
	Arr    [2]int32            `kanon:"5"`
	Points map[Point]string    `kanon:"6"`
	Times  map[time.Time]int32 `kanon:"7"`
	Opt    *Point              `kanon:"8"`
	Any    any                 `kanon:"9,types=string|int64|[]any"`
}

// RepeatKind selects the member of the union of Repeats.
type RepeatKind uint8

// RepeatKindSlice selects Slice, the member of the union of Repeats.
const RepeatKindSlice RepeatKind = 1

// Repeats is the vector type of a union member and an interface, which the
// decode vectors of the wire format repeat.
type Repeats struct {
	Kind   RepeatKind
	Slice  []int32 `kanon:"1,union=Kind"`
	Nested any     `kanon:"8,types=[]int32"`
}

// Nest is the vector type of a struct field with a kanon codec that is not a
// pointer.
type Nest struct {
	Inner Inner
}
