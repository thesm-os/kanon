// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package stream

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Choice,Pick -canonical

type (
	// ChoiceKind selects the member of the union of Choice.
	ChoiceKind uint8
	// PickKind selects the member of the union of Pick.
	PickKind uint8
)

// The members of the unions of Choice and Pick.
const (
	ChoiceKindSmall ChoiceKind = 1
	ChoiceKindLarge ChoiceKind = 2
	PickKindNumber  PickKind   = 1
	PickKindWord    PickKind   = 2
)

// Choice is a canonical type whose union has a member on each side of a
// streamed slice, and whose member Large the decode tracks, so that its
// stream decoder records the members of the union and the tracked fields
// across the runs of the encoding.
type Choice struct {
	Kind  ChoiceKind
	Small int64   `kanon:"1,union=Kind"`
	Items []Inner `kanon:"2,stream"`
	Large *Inner  `kanon:"3,union=Kind"`
	Tail  string  `kanon:"4"`
}

// Pick is a canonical type whose union has a member on each side of a
// streamed byte slice, without a field that the decode tracks, and whose
// streamed slice has elements of a canonical struct type of another
// package.
type Pick struct {
	Kind   PickKind
	Number int64            `kanon:"1,union=Kind"`
	Data   []byte           `kanon:"2,stream"`
	Word   string           `kanon:"3,union=Kind"`
	Proofs []external.Proof `kanon:"4,stream"`
}
