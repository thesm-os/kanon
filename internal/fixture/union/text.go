// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

import "time"

//go:generate go tool kanon -type=Texts

// TextKind selects the member of the union of Texts.
type TextKind uint8

// The members of the union of Texts.
const (
	TextKindString TextKind = 1
	TextKindBytes  TextKind = 2
	TextKindName   TextKind = 3
	TextKindDigest TextKind = 4
	TextKindEmpty  TextKind = 5
	TextKindTime   TextKind = 6
)

// Name is a named string.
type Name string

// Texts has a union with a member of each string and byte type, and a time.
// A selected member of one value, the array of no bytes, encodes its
// length 0.
type Texts struct {
	Kind   TextKind
	String string    `kanon:",union=Kind"`
	Bytes  []byte    `kanon:",union=Kind"`
	Name   Name      `kanon:",union=Kind"`
	Digest [32]byte  `kanon:",union=Kind"`
	Empty  [0]byte   `kanon:",union=Kind"`
	Time   time.Time `kanon:",union=Kind"`
}
