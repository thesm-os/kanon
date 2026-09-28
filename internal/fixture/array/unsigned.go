// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

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

// Unsigned has an array of each unsigned integer type. An array of uint8
// elements is a byte array, and an array of Status elements is an array
// of numbers.
type Unsigned struct {
	Uint    [2]uint
	Uint16  [2]uint16
	Uint32  [2]uint32
	Uint64  [2]uint64
	Uintptr [2]uintptr
	Status  [2]Status
	Port    [2]Port
	Count   [2]Count
	Size    [2]Size
	Index   [2]Index
	Handle  [2]Handle
}
