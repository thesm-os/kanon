// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has an array of byte arrays of each length that changes the
// length of their length. An array of arrays of no bytes has one value.
type ByteArrays struct {
	Byte0   [2][0]byte
	Byte1   [2][1]byte
	Byte32  [2][32]byte
	Byte127 [2][127]byte
	Byte128 [2][128]byte
	Digest  [2]Digest
}
