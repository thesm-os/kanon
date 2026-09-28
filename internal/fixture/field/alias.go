// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
	// BytesAlias denotes []byte.
	BytesAlias = []byte
)

// Aliases has a field of an alias of a builtin type, of a named type and
// of a composite type.
type Aliases struct {
	Int32 Int32Alias
	Name  NameAlias
	Bytes BytesAlias
}
