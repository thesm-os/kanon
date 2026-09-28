// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has a pointer field to a byte array of each length that
// changes the length of its length. A pointer to an array of no elements
// is present when it is not nil.
type ByteArrays struct {
	Byte0   *[0]byte
	Byte1   *[1]byte
	Byte32  *[32]byte
	Byte127 *[127]byte
	Byte128 *[128]byte
	Digest  *Digest
}
