// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

//go:generate go tool kanon -type=Tree

// Tree nests itself: behind a pointer, in slices of values and of pointers,
// and in maps of values and of pointers, so that a value is as deep as its
// data.
type Tree struct {
	Label     string
	Parent    *Tree
	Children  []*Tree
	Siblings  []Tree
	ByName    map[string]*Tree
	ByOrdinal map[int32]Tree
}
