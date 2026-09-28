// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

//go:generate go tool kanon -type=Recursive

// Hop is an inline struct that contains itself behind a pointer, in a
// slice and in a map, so that its functions call themselves.
type Hop struct {
	Depth int32
	Next  *Hop
	Forks []Hop
	Named map[string]*Hop
}

// Recursive has an inline struct that contains itself.
type Recursive struct {
	Hop Hop
}
