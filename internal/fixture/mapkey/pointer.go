// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=Pointers

// Loop is a named pointer type that points at itself.
type Loop *Loop

// Pointers has a map with keys of pointers to a number, a struct and an
// array of no elements, of arrays of pointers, and of a Loop. A nil key
// sorts first, and any other key by the value that it points at.
type Pointers struct {
	Int32 map[*int32]string
	Point map[*Point]string
	Empty map[*[0]int32]string
	Array map[[2]*int32]string
	Loop  map[Loop]string
}
