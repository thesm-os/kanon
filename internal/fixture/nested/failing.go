// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

import "go.thesmos.sh/kanon/internal/fixture/codec"

//go:generate go tool kanon -type=Failing,Chain

// Failing is a struct with a kanon codec whose encoding fails for a Word
// that contains a NUL byte.
type Failing struct {
	Word codec.Word
}

// Chain nests Failing in every shape, so that its encoding returns the
// error of the nested struct.
type Chain struct {
	Value  Failing
	Ptr    *Failing
	Values []Failing
	ByName map[string]Failing
}
