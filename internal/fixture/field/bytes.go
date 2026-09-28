// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has a byte array field of each length that changes the length
// of its length: 0, which is never present, 1 and 32, whose lengths take
// one byte, 127, the longest one of them, and 128, whose length takes two.
type ByteArrays struct {
	Byte0   [0]byte
	Byte1   [1]byte
	Byte32  [32]byte
	Byte127 [127]byte
	Byte128 [128]byte
	Digest  Digest
}
