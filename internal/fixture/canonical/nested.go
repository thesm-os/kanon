// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"go.thesmos.sh/kanon/internal/fixture/external"
	"go.thesmos.sh/kanon/internal/fixture/validate"
)

//go:generate go tool kanon -type=Deep -canonical

// Cell is an inline struct, which the code file of Deep decodes canonically
// with functions of its own.
type Cell struct {
	Row   int8
	Col   int8
	Value string
}

// Deep has inline structs as a field, behind a pointer, in a slice and as map
// values, nested slices and maps, a struct with a kanon codec of another
// package whose directive sets -canonical, and a kanon.Validator slice.
type Deep struct {
	Cell    Cell
	CellPtr *Cell
	Cells   []Cell
	ByName  map[string]Cell
	Grid    [][]int32
	Groups  map[string][]int32
	Pairs   [2][]int32
	Proof   external.Proof
	Proofs  []*external.Proof
	Tags    validate.Tags
	Anonym  struct{ A, B int32 }
}
