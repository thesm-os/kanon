// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
)

// Aliases has a map with keys of an alias of a builtin type and of a named
// type.
type Aliases struct {
	Int32 map[Int32Alias]string
	Name  map[NameAlias]string
}
