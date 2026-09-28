// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has a map with byte array keys of each length that changes the
// length of their length. The array of no bytes has one value, so that its
// map has one entry at most.
type ByteArrays struct {
	Byte0   map[[0]byte]string
	Byte1   map[[1]byte]string
	Byte32  map[[32]byte]string
	Byte127 map[[127]byte]string
	Byte128 map[[128]byte]string
	Digest  map[Digest]string
}
