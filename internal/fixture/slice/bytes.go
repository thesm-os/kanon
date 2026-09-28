// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has a slice of byte arrays of each length that changes the
// length of their length.
type ByteArrays struct {
	Byte0   [][0]byte
	Byte1   [][1]byte
	Byte32  [][32]byte
	Byte127 [][127]byte
	Byte128 [][128]byte
	Digest  []Digest
}
