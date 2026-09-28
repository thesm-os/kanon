// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"encoding/binary"
	"math/bits"
)

// Uvarint returns the value of the varint at the start of data and its
// length. The length is 0 when data ends inside the varint, and -1 when the
// varint exceeds 64 bits: a 10th byte above 1, which a 10th byte with the
// continuation bit is, since it announces an 11th. A varint with leading
// zero groups, such as 80 00 for 0, reads as its value.
func Uvarint(data []byte) (uint64, int) {
	if len(data) > 0 && data[0] < 0x80 {
		return uint64(data[0]), 1
	}
	var x uint64
	for i, b := range data {
		if i == binary.MaxVarintLen64-1 && b > 1 {
			return 0, -1
		}
		if b < 0x80 {
			return x | uint64(b)<<(7*i), i + 1
		}
		x |= uint64(b&0x7f) << (7 * i)
	}
	return 0, 0
}

// SizeUvarint returns the length of the shortest varint of v, 1 to 10.
func SizeUvarint(v uint64) int {
	return (bits.Len64(v|1) + 6) / 7
}

// PutUvarint writes the shortest varint of v into the bytes of buf that
// end at i, and returns the offset of its first byte.
func PutUvarint(buf []byte, i int, v uint64) int {
	if v < 0x80 {
		i--
		buf[i] = byte(v)
		return i
	}
	i -= SizeUvarint(v)
	binary.PutUvarint(buf[i:], v)
	return i
}

// PutTag writes the tag of a field into the bytes of buf that end at i, and
// returns the offset of its first byte. The tag is below 1<<14, the tag of
// a field number below 2048, which takes one or two bytes. A larger tag
// goes through [PutUvarint]. With a constant tag, the compiled code is one
// or two stores.
func PutTag(buf []byte, i int, tag uint64) int {
	if tag < 0x80 {
		i--
		buf[i] = byte(tag)
		return i
	}
	i -= 2
	buf[i], buf[i+1] = byte(tag)|0x80, byte(tag>>7)
	return i
}

// PutBool writes the varint of v, 1 for true and 0 for false, into the byte
// of buf before i, and returns i-1. It writes the presence byte of a
// pointer the same way.
func PutBool(buf []byte, i int, v bool) int {
	var b byte
	if v {
		b = 1
	}
	i--
	buf[i] = b
	return i
}

// Presence reads the presence byte at the start of data and reports
// whether the value of a pointer follows it: false for 0, a nil pointer,
// and true for 1. The byte opens the encoding of every pointer that is not
// a field, such as an element of a slice or the value of a map.
//
// Presence returns an error that wraps io.ErrUnexpectedEOF for empty data,
// and one that wraps kanon.ErrMalformed for any byte other than 0 and 1.
// Both errors name loc and num, at offset off.
func Presence(data []byte, loc string, num, off int) (bool, error) {
	if len(data) == 0 {
		return false, ReadError(0, loc, num, off)
	}
	switch data[0] {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, PresenceError(data[0], loc, num, off)
	}
}

// Zigzag returns the zigzag encoding of v, which maps 0, -1, 1, -2 and 2
// to 0, 1, 2, 3 and 4, so that a small negative number is a short varint.
func Zigzag(v int64) uint64 {
	return uint64(v<<1) ^ uint64(v>>63)
}

// Unzigzag returns the int64 whose zigzag encoding is u.
func Unzigzag(u uint64) int64 {
	return int64(u>>1) ^ -int64(u&1)
}

// CountVarints returns the number of varints that end in data: the number
// of its bytes without the continuation bit. A decode sizes a slice of
// varints with it before it appends.
func CountVarints(data []byte) int {
	n := 0
	for _, b := range data {
		if b < 0x80 {
			n++
		}
	}
	return n
}
