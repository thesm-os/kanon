// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

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

// Unsigned has a field of each unsigned integer type. A decode rejects a
// value outside the range of the type of the field.
type Unsigned struct {
	Uint    uint
	Uint8   uint8
	Uint16  uint16
	Uint32  uint32
	Uint64  uint64
	Uintptr uintptr
	Status  Status
	Port    Port
	Count   Count
	Size    Size
	Index   Index
	Handle  Handle
}
