// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Remote

// Remote has a slice of each named basic type of another package.
type Remote struct {
	Rank  []external.Rank
	Ident []external.Ident
}
