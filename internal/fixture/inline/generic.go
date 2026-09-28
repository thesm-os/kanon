// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Generic

// Box is a generic struct of this package without a kanon codec.
type Box[T any] struct {
	Value T
	Note  string
}

// Generic has instantiations of generic structs of this package and of
// another, which share the field numbers of their generic type: by value,
// behind a pointer, as the value of a map, and one instantiation inside
// another.
type Generic struct {
	Box    Box[int32]
	Boxes  *Box[[]string]
	Pair   external.Pair[string, int32]
	Pairs  map[int32]external.Pair[external.Rank, Box[string]]
	Nested Box[Box[bool]]
}
