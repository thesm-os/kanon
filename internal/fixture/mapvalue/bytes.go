// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=ByteArrays

// Digest is a named byte array.
type Digest [32]byte

// ByteArrays has a map with byte array values of each length that changes
// the length of their length.
type ByteArrays struct {
	Byte0   map[string][0]byte
	Byte1   map[string][1]byte
	Byte32  map[string][32]byte
	Byte127 map[string][127]byte
	Byte128 map[string][128]byte
	Digest  map[string]Digest
}
