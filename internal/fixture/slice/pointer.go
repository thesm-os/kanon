// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Pointers

// Pointers has a slice of pointers to a number, a string, a struct with a
// kanon codec of this package and of another, an inline struct and a
// slice, and a slice of pointers to pointers. A nil element encodes as its
// presence byte 0.
type Pointers struct {
	Int32  []*int32
	String []*string
	Inner  []*Inner
	Label  []*external.Label
	Loose  []*Loose
	Slice  []*[]int32
	Twice  []**int32
}
