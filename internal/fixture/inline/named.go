// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

import (
	"time"

	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Named

// Loose is a struct of this package without a kanon codec. Its own tags
// apply to its fields in every code file that encodes it.
type Loose struct {
	Label string
	Total int64 `kanon:",fixed"`
	At    time.Time
}

// Named has named inline structs of this package and of another, as
// fields, elements and map values. A remote struct records its numbers
// under its import path, and its anonymous field under the path of the
// field. The remote struct Sealed has an unexported field.
type Named struct {
	Loose   Loose
	Looses  []Loose
	ByName  map[string]Loose
	Remote  external.Remote
	Remotes []*external.Remote
	Sealed  external.Sealed
}
