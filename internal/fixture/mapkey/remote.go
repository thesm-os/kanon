// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Remote

// Remote has a map with keys of each named basic type of another package,
// and a map with keys of a struct of another package with an unexported
// field, which the code file cannot read, so that no key of it counts as a
// key that a decode yields.
type Remote struct {
	Rank  map[external.Rank]string
	Ident map[external.Ident]string
	Badge map[external.Badge]string
}
