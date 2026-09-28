// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Point,Chain,Structs

// Point is a struct with a kanon codec of this package. Its tags number Y
// before X, so that its keys sort by Y first.
type Point struct {
	X int32 `kanon:"2"`
	Y int32 `kanon:"1"`
}

// Chain is a struct with a kanon codec of this package that contains
// itself behind a pointer and an array, neither of which has a float, so
// that a map key of it writes itself through its methods, and its order
// compares one link after another.
type Chain struct {
	Next *Chain
	Ends [2]int32
}

// Spot is a struct of this package without a kanon codec, which the code
// file of Structs encodes as an inline struct. Its fields that the encoding
// leaves out tell apart two keys of one projection.
type Spot struct {
	Row  int8
	Col  int8
	Note string  `kanon:"-"`
	Tags [2]int8 `kanon:"-"`
}

// Link is an inline struct that contains itself behind a pointer, so that
// its order compares one link after another.
type Link struct {
	N    int32
	Next *Link
}

// Padded is an inline struct with a field of one value, which its order
// leaves out.
type Padded struct {
	N   int32
	Pad [0]int32
}

// Mute is an inline struct whose only field the encoding leaves out, so
// that its keys have one projection and a map of two of them fails the
// encode.
type Mute struct {
	Note string `kanon:"-"`
}

// Structs has a map with keys of each struct type: a struct with a kanon
// codec of this package and of another, an inline struct of this package,
// an anonymous struct, an instantiation of a generic struct of another
// package, a struct that contains itself, a struct with a field of one
// value, an empty struct, which has one value, with a value of one value
// too, a struct of one projection with a field that the encoding leaves
// out, and a struct with a kanon codec that contains itself.
type Structs struct {
	Point   map[Point]string
	Label   map[external.Label]string
	Spot    map[Spot]string
	Cell    map[struct{ Row, Col int8 }]string
	Pair    map[external.Pair[int32, string]]string
	Link    map[Link]string
	Padded  map[Padded]string
	Nothing map[struct{}]string
	Unit    map[struct{}]struct{}
	Mute    map[Mute]string
	Chain   map[Chain]string
}
