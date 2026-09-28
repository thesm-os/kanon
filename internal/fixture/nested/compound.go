// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

//go:generate go tool kanon -type=Compound,Compounds

// Area is a struct of this package without a kanon codec, which the code
// file of Compound encodes as an inline struct.
type Area struct {
	Width  int32
	Height int32
}

// Compound is a struct with a kanon codec that has an inline struct.
type Compound struct {
	Area Area
	Name string
}

// Compounds nests Compound by value and in a slice, so that its encoding
// contains the inline struct that the code of Compound writes.
type Compounds struct {
	One  Compound
	Many []Compound
}
