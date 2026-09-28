// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Unsigned

// Named unsigned integers.
type (
	// Status is a named uint8.
	Status uint8
	// Port is a named uint16.
	Port uint16
	// Count is a named uint32.
	Count uint32
	// Size is a named uint64.
	Size uint64
	// Index is a named uint.
	Index uint
	// Handle is a named uintptr.
	Handle uintptr
)

// Unsigned has a map with values of each unsigned integer type.
type Unsigned struct {
	Uint    map[string]uint
	Uint8   map[string]uint8
	Uint16  map[string]uint16
	Uint32  map[string]uint32
	Uint64  map[string]uint64
	Uintptr map[string]uintptr
	Status  map[string]Status
	Port    map[string]Port
	Count   map[string]Count
	Size    map[string]Size
	Index   map[string]Index
	Handle  map[string]Handle
}
