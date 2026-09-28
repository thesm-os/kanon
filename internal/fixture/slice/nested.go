// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

import "time"

//go:generate go tool kanon -type=Nested

// Tree is a slice of itself, which nests as deep as its value.
type Tree []Tree

// Nested nests slices in slices, arrays and maps, and a slice type that
// contains itself. The elements of Nones are arrays of no pointers, which
// refer to no memory that a decode reuses.
type Nested struct {
	Ints   [][]int32
	Words  [][]string
	Rows   [][2]string
	Tables []map[string]int32
	Stamps [][]time.Time
	Voids  [][0]byte
	Deep   [][][]int64
	Tree   Tree
	Nones  [][0]*int32
}
