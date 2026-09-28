// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

//go:generate go tool kanon -type=Aliases

// Aliases, which kanon resolves to the types they denote.
type (
	// Int32Alias denotes int32.
	Int32Alias = int32
	// NameAlias denotes Name.
	NameAlias = Name
	// Ref denotes *int64.
	Ref = *int64
)

// Aliases has a pointer field to an alias of a builtin type and of a named
// type, and a field of an alias of a pointer type.
type Aliases struct {
	Int32   *Int32Alias
	Name    *NameAlias
	Pointer Ref
}
