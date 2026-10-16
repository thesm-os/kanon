// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound

//go:generate go tool kanon -type=List,Shelf

// ListKind selects the member of the union of List.
type ListKind uint8

// The members of the union of List.
const (
	ListKindTags ListKind = 1
	ListKindCode ListKind = 2
	ListKindName ListKind = 3
)

// List is a type that accepts every encoding of a value, with a bounded
// field in each place. Counts is a slice whose read function the unbounded
// Totals and the elements of Grid share. Scores and Refs are maps, and two of
// the pointer keys of Refs can have one projection. Tables is a slice of maps
// of the type of Scores. Tags and Name are bounded union
// members, a slice and a string. Window is an inline struct with a bounded
// slice and a bounded byte slice.
type List struct {
	Counts []int64 `kanon:",max=3"`
	Totals []int64
	Grid   [][]int64
	Scores map[string]int32   `kanon:",max=3"`
	Refs   map[*int32]string  `kanon:",max=4"`
	Tables []map[string]int32 `kanon:",max=2"`
	Kind   ListKind
	Tags   []string `kanon:",union=Kind,max=2"`
	Code   int64    `kanon:",union=Kind"`
	Name   string   `kanon:",union=Kind,max=2"`
	Window window
}

// window is an inline struct with a bounded slice and a bounded byte slice.
type window struct {
	Marks []uint32 `kanon:",max=2"`
	Stamp []byte   `kanon:",max=2"`
}

// Shelf is a type with a slice of List, whose encode fails when the encode
// of an element fails at the bound of one of its fields.
type Shelf struct {
	Name  string
	Lists []List
}
