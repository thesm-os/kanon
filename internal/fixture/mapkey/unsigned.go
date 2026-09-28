// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

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

// Unsigned has a map with keys of each unsigned integer type.
type Unsigned struct {
	Uint    map[uint]string
	Uint8   map[uint8]string
	Uint16  map[uint16]string
	Uint32  map[uint32]string
	Uint64  map[uint64]string
	Uintptr map[uintptr]string
	Status  map[Status]string
	Port    map[Port]string
	Count   map[Count]string
	Size    map[Size]string
	Index   map[Index]string
	Handle  map[Handle]string
}
