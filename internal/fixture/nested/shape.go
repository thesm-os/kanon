// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Shapes

// Shapes nests a struct of this package, which another source file
// generates, and a struct of another package in every shape: by value,
// behind a pointer, as elements, as map values, and as the values of
// pointers in them.
type Shapes struct {
	Value      Inner
	Ptr        *Inner
	Values     []Inner
	Ptrs       []*Inner
	ByName     map[string]Inner
	PtrsByName map[string]*Inner
	ByFlag     map[bool]Inner
	Remote     external.Label
	RemotePtr  *external.Label
	Remotes    []external.Label
	RemotePtrs []*external.Label
	ByRank     map[external.Rank]external.Label
}
