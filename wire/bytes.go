// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

// SizeBytes returns the length of the encoding of a value of n bytes with
// its length prefix: the varint of n, then the n bytes.
func SizeBytes(n int) int {
	return SizeUvarint(uint64(n)) + n
}

// PutRaw writes the bytes of s, without a length, into the bytes of buf
// that end at i, and returns the offset of the first.
func PutRaw[S ~string | ~[]byte](buf []byte, i int, s S) int {
	i -= len(s)
	copy(buf[i:], s)
	return i
}

// CountValues returns the number of values in data that each take a varint
// length and that many bytes. It stops at a length that does not read or
// that runs past data, which the decode then reports. A decode sizes a
// slice of such values with it before it appends. A value takes a byte at
// least, so that the loop runs len(data) times at most.
func CountValues(data []byte) int {
	n := 0
	for range len(data) {
		l, k := Uvarint(data)
		if k <= 0 || uint64(len(data)-k) < l {
			break
		}
		data = data[k+int(l):]
		n++
	}
	return n
}
