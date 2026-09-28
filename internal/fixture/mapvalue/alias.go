// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
	// MapAlias denotes map[string]int64.
	MapAlias = map[string]int64
)

// Aliases has a map with values of an alias of a builtin type and of a
// named type, and a field of an alias of a map type.
type Aliases struct {
	Int32 map[string]Int32Alias
	Name  map[string]NameAlias
	Map   MapAlias
}
