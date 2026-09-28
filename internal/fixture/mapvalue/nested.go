// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Nested

// Graph is a map whose values are of its own type, which nests as deep as
// its value.
type Graph map[string]Graph

// Nested has maps with values that refer to memory, which a decode takes
// from a free list of the previous values: slices, maps, pointers and
// structs that contain them, and arrays of them, and a map type that
// contains itself.
type Nested struct {
	Slice   map[string][]int32
	Map     map[string]map[int64]string
	Sets    map[int32]map[string]bool
	Array   map[string][3]int32
	Lists   map[string][2][]int32
	Int32   map[string]*int32
	Inner   map[string]*Inner
	Label   map[string]*external.Label
	Holders map[string]Holder
	Graph   Graph
}

// Holder is an inline struct that refers to memory, as the value of a map.
type Holder struct {
	Names []string
	Inner *Inner
}
