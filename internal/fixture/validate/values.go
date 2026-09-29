// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

//go:generate go tool kanon -type=Values -views

// Kind selects the member of the union of Values.
type Kind uint8

// Kinds of the union of Values.
const (
	// KindText selects the member Text.
	KindText Kind = 1
	// KindCount selects the member Count.
	KindCount Kind = 2
)

// Valid is an interface whose method set has ValidateKanon. kanon encodes a
// value of Valid by its concrete type, and calls ValidateKanon on the value
// of the concrete type alone.
type Valid interface {
	ValidateKanon() error
}

// Values has fields of the types of the package in every position that a
// codec encodes, so that its generated tests call ValidateKanon on a value
// of each position that the encode writes and the decode reads.
type Values struct {
	Level   Level
	Amount  Amount
	Tick    Tick
	Code    Code
	Ratio   Ratio
	Tags    Tags
	Scores  Scores
	Hash    Hash
	Blob    Blob
	Span    Span
	Flag    Flag
	Weight  Weight
	Wave    Wave
	Phase   Phase
	Grade   Grade
	Next    *Level
	Codes   []Code
	Pair    [2]Amount
	Index   map[Code]Level
	Kind    Kind
	Text    Code  `kanon:",union=Kind"`
	Count   Level `kanon:",union=Kind"`
	Any     any   `kanon:",types=Level|Tags"`
	Checked Valid `kanon:",types=Grade"`
	Fixed   Tick  `kanon:",fixed"`
}
