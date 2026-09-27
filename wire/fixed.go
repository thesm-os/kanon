// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import "encoding/binary"

// Uint32 returns the four little-endian bytes at the start of data as a
// uint32, the value of the wire format Fixed32, and their length 4, or a
// length of 0 when data is shorter.
func Uint32(data []byte) (uint32, int) {
	if len(data) < 4 {
		return 0, 0
	}
	return binary.LittleEndian.Uint32(data), 4
}

// Uint64 returns the eight little-endian bytes at the start of data as a
// uint64, the value of the wire format Fixed64, and their length 8, or a
// length of 0 when data is shorter.
func Uint64(data []byte) (uint64, int) {
	if len(data) < 8 {
		return 0, 0
	}
	return binary.LittleEndian.Uint64(data), 8
}

// PutUint32 writes the four little-endian bytes of v into the bytes of buf
// that end at i, and returns the offset of the first.
func PutUint32(buf []byte, i int, v uint32) int {
	i -= 4
	binary.LittleEndian.PutUint32(buf[i:], v)
	return i
}

// PutUint64 writes the eight little-endian bytes of v into the bytes of buf
// that end at i, and returns the offset of the first.
func PutUint64(buf []byte, i int, v uint64) int {
	i -= 8
	binary.LittleEndian.PutUint64(buf[i:], v)
	return i
}
