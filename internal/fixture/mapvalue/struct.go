// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

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

// Structs has a map with values of each struct type: a struct with a kanon
// codec of this package and of another, an inline struct of this package
// and of another, an instantiation of a generic struct, an anonymous
// struct, and an empty struct, so that Nothing is a set.
type Structs struct {
	Inner   map[string]Inner
	Label   map[string]external.Label
	Loose   map[string]Loose
	Remote  map[string]external.Remote
	Pair    map[string]external.Pair[string, int32]
	Anonym  map[string]struct{ A, B int32 }
	Nothing map[string]struct{}
}
