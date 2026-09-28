// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

//go:generate go tool kanon -type=Point,Grid

// Point is a struct with a kanon codec without a string.
type Point struct {
	X int32
	Y int32
}

// Grid nests structs without a string in every shape, so that no decoded
// string refers to the input and a decode copies no input.
type Grid struct {
	Origin  Point
	Corner  *Point
	Points  []Point
	ByIndex map[int32]Point
}
