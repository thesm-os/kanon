// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

//go:generate go tool kanon -type=Pointers

// PointerKind selects the member of the union of Pointers.
type PointerKind uint8

// The members of the union of Pointers.
const (
	PointerKindInt32  PointerKind = 1
	PointerKindString PointerKind = 2
	PointerKindTwice  PointerKind = 3
	PointerKindSlice  PointerKind = 4
	PointerKindEmpty  PointerKind = 5
)

// Pointers has a union with pointer members: to a number, to a string, to a
// pointer, to a slice and to an array of no elements. A selected nil member
// encodes the zero value that it would point at, and a nil pointer to a
// pointer encodes as the presence byte 0.
type Pointers struct {
	Kind   PointerKind
	Int32  *int32    `kanon:",union=Kind"`
	String *string   `kanon:",union=Kind"`
	Twice  **int32   `kanon:",union=Kind"`
	Slice  *[]int32  `kanon:",union=Kind"`
	Empty  *[0]int32 `kanon:",union=Kind"`
}
