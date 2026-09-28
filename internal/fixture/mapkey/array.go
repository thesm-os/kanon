// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

import "time"

//go:generate go tool kanon -type=Arrays

// Arrays has a map with keys of arrays: of numbers, of structs, of floats,
// which a lookup does not always find, of times and of bools, an array of
// one element, and arrays of one value, whose maps have one entry at most.
// Points and NoPoint map arrays of structs to structs. The key of Points
// checks the depth level of its value, and the key of NoPoint, which has no
// element, checks none.
type Arrays struct {
	Int32   map[[2]int32]string
	Point   map[[2]Point]string
	Float   map[[2]float64]string
	Time    map[[2]time.Time]string
	One     map[[1]int32]string
	Empty   map[[0]int32]string
	Void    map[[2][0]int32]string
	Bool    map[[2]bool]string
	Points  map[[2]Point]Point
	NoPoint map[[0]Point]Point
}
