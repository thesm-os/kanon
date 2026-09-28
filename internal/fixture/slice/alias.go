// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
	// List denotes []int64.
	List = []int64
)

// Aliases has a slice of an alias of a builtin type and of a named type,
// and a field of an alias of a slice type.
type Aliases struct {
	Int32 []Int32Alias
	Name  []NameAlias
	Slice List
}
