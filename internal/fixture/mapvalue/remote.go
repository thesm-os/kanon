// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Remote

// Remote has a map with values of each named basic type of another
// package.
type Remote struct {
	Rank  map[string]external.Rank
	Ident map[string]external.Ident
}
