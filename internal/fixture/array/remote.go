// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Remote

// Remote has an array of each named basic type of another package.
type Remote struct {
	Rank  [2]external.Rank
	Ident [2]external.Ident
}
