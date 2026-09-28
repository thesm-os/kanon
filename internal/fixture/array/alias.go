// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
	// Triple denotes [3]int64.
	Triple = [3]int64
)

// Aliases has an array of an alias of a builtin type and of a named type,
// and a field of an alias of an array type.
type Aliases struct {
	Int32 [2]Int32Alias
	Name  [2]NameAlias
	Array Triple
}
