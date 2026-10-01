// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

//go:generate go tool kanon -type=Idents

// IdentKind selects the member of the union of Idents.
type IdentKind uint8

// Kinds of the union of Idents.
const (
	// IdentKindLeft selects the member Left.
	IdentKindLeft IdentKind = 1
	// IdentKindRight selects the member Right.
	IdentKindRight IdentKind = 2
)

// Idents has an Ident in a field, the elements of a slice and an array, the
// values of a map, the target of a pointer and the members of a union. An
// Ident is a kanon.Appender, so the encode of Idents has no error path.
type Idents struct {
	One   Ident
	Many  []Ident
	Pair  [2]Ident
	ByKey map[string]Ident
	Ptr   *Ident
	Kind  IdentKind
	Left  Ident `kanon:",union=Kind"`
	Right Ident `kanon:",union=Kind"`
}
