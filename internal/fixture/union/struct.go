// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Inner,Structs

// Inner is a struct with a kanon codec of this package.
type Inner struct {
	Label string
	Count int64
}

// Loose is a struct of this package without a kanon codec.
type Loose struct {
	Note string
}

// StructKind selects the member of the union of Structs.
type StructKind uint8

// The members of the union of Structs.
const (
	StructKindInner    StructKind = 1
	StructKindLabel    StructKind = 2
	StructKindLoose    StructKind = 3
	StructKindNothing  StructKind = 4
	StructKindInnerPtr StructKind = 5
	StructKindLabelPtr StructKind = 6
	StructKindLoosePtr StructKind = 7
)

// Structs has a union with a member of each struct type and of a pointer to
// one. A selected struct member encodes even when its encoding is empty,
// and a member that another selected member follows in the input decodes
// as the first occurrence.
type Structs struct {
	Kind     StructKind
	Inner    Inner           `kanon:",union=Kind"`
	Label    external.Label  `kanon:",union=Kind"`
	Loose    Loose           `kanon:",union=Kind"`
	Nothing  struct{}        `kanon:",union=Kind"`
	InnerPtr *Inner          `kanon:",union=Kind"`
	LabelPtr *external.Label `kanon:",union=Kind"`
	LoosePtr *Loose          `kanon:",union=Kind"`
}
