// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

//go:generate go tool kanon -type=Declared,Reordered,WithDeclared,WithReordered

// Declared is a struct with a kanon codec whose fields are declared in the
// order of their numbers.
type Declared struct {
	Name  string `kanon:"1"`
	Count uint64 `kanon:"2"`
}

// Reordered is Declared with the declarations of its fields swapped. The
// tags keep the field numbers, so a Reordered encodes as a Declared.
type Reordered struct {
	Count uint64 `kanon:"2"`
	Name  string `kanon:"1"`
}

// WithDeclared contains a Declared, whose fields its samples fill.
type WithDeclared struct {
	In Declared
}

// WithReordered contains a Reordered. Its samples encode as those of
// WithDeclared, since the order of the field declarations does not change
// them.
type WithReordered struct {
	In Reordered
}
