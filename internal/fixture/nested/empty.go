// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Empty,Hollow,Holder

// Empty is a struct with a kanon codec and no fields, whose encoding is
// empty.
type Empty struct{}

// Hollow is a struct with a kanon codec whose encoded fields have one value
// each, so that its encoding is empty too. Its tag leaves Note out of the
// encoding, so that a Hollow can have a value that Reset clears.
type Hollow struct {
	None [0]int32
	Void struct{}
	Note string `kanon:"-"`
}

// Padded is a struct without a kanon codec whose encoded field has one
// value and whose other field is blank, which no code can set, so that a
// Padded is always its zero value.
type Padded struct {
	None [0]int32
	_    int64
}

// Holder nests the structs whose encodings are empty, of this package and of
// another. A field of them is never present, and an element encodes as the
// length 0.
type Holder struct {
	Empty   Empty
	Hollow  Hollow
	Empties []Empty
	Ptr     *Empty
	Blank   external.Blank
	Padded  Padded
}
