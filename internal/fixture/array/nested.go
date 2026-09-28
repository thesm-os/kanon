// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=Nested

// Nested nests arrays in arrays and holds slices, maps and pointers in
// arrays: an array of no elements, which is never present, arrays of
// arrays, an array whose elements have one value, and arrays of values
// that refer to memory, which a decode reuses.
type Nested struct {
	Empty   [0]int32
	Grid    [2][3]int32
	Voids   [2][0]int32
	Lists   [2][]int32
	Maps    [2]map[string]int32
	Corners [2]*Inner
	Numbers [2]*int32
	Deep    [2][2][2]float64
}
