// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

//go:generate go tool kanon -type=Containers

// ContainerKind selects the member of the union of Containers.
type ContainerKind uint8

// The members of the union of Containers.
const (
	ContainerKindSlice  ContainerKind = 1
	ContainerKindMap    ContainerKind = 2
	ContainerKindArray  ContainerKind = 3
	ContainerKindFloats ContainerKind = 4
	ContainerKindEmpty  ContainerKind = 5
)

// Containers has a union with container members: a slice, a map, arrays and
// an array of no elements. A selected empty container encodes its length
// 0. A slice member that the discriminator selects anew drops the elements
// of its previous value, and a map member its entries.
type Containers struct {
	Kind   ContainerKind
	Slice  []int32          `kanon:",union=Kind"`
	Map    map[string]int32 `kanon:",union=Kind"`
	Array  [3]int32         `kanon:",union=Kind"`
	Floats [2]float64       `kanon:",union=Kind"`
	Empty  [0]int32         `kanon:",union=Kind"`
}
