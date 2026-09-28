// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Inner,Structs

// Inner is a struct with a kanon codec of this package.
type Inner struct {
	Label string
	Count int64
}

// Loose is a struct of this package without a kanon codec, which the code
// file of Structs encodes as an inline struct.
type Loose struct {
	Note string
	Size uint16
}

// Structs has an array of each struct type: a struct with a kanon codec of
// this package and of another, an inline struct of this package and of
// another, an instantiation of a generic struct, an anonymous struct, and
// an empty struct. An array of structs is present when the encoding of an
// element has bytes, and an array of empty structs has one value.
type Structs struct {
	Inner   [2]Inner
	Label   [2]external.Label
	Loose   [2]Loose
	Remote  [2]external.Remote
	Pair    [2]external.Pair[string, int32]
	Anonym  [2]struct{ A, B int32 }
	Nothing [2]struct{}
}
