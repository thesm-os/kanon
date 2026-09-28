// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

//go:generate go tool kanon -type=Containers

// Containers has a pointer field to each container type: a slice, a map,
// an array and an array of no elements. A pointer to an empty container is
// present.
type Containers struct {
	Slice   *[]int32
	Strings *[]string
	Map     *map[string]int32
	Array   *[3]int32
	Empty   *[0]int32
}
