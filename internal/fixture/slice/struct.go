// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

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

// Structs has a slice of each struct type: a struct with a kanon codec of
// this package and of another, an inline struct of this package and of
// another, an instantiation of a generic struct, an anonymous struct, and
// an empty struct. An element encodes a struct with an empty encoding as
// the length 0.
type Structs struct {
	Inner   []Inner
	Label   []external.Label
	Loose   []Loose
	Remote  []external.Remote
	Pair    []external.Pair[string, int32]
	Anonym  []struct{ A, B int32 }
	Nothing []struct{}
}
