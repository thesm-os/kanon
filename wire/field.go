// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import "strconv"

// Wire formats: the low three bits of a tag, which state how the value
// after the tag is laid out, so that a decoder skips a field it does not
// know. The tag of field n with wire format w is n<<3 | w.
const (
	// Varint is a value of one varint.
	Varint = 0
	// Fixed64 is a value of eight bytes.
	Fixed64 = 1
	// Bytes is a varint length and that many bytes.
	Bytes = 2
	// Fixed32 is a value of four bytes.
	Fixed32 = 5
)

// Skip returns the length of the value of a field that the decode does not
// know, at the start of data, from the field's tag. It fails for a tag with
// field number 0 or with wire format 3, 4, 6 or 7, and for data that ends
// inside the value, with errors that loc and num locate at off, the offset
// of the tag. The decode of a struct passes the struct's type as loc and 0
// as num.
func Skip(data []byte, tag uint64, loc string, num, off int) (int, error) {
	if tag>>3 == 0 {
		return 0, TagError(tag, loc, num, off)
	}
	switch tag & 7 {
	case Varint:
		_, n := Uvarint(data)
		if n <= 0 {
			return 0, ReadError(n, loc, num, off)
		}
		return n, nil
	case Fixed64:
		if len(data) < 8 {
			return 0, ReadError(0, loc, num, off)
		}
		return 8, nil
	case Bytes:
		l, n := Uvarint(data)
		if n <= 0 || uint64(len(data)-n) < l {
			return 0, ReadError(n, loc, num, off)
		}
		return n + int(l), nil
	case Fixed32:
		if len(data) < 4 {
			return 0, ReadError(0, loc, num, off)
		}
		return 4, nil
	default:
		return 0, TagError(tag, loc, num, off)
	}
}

// wireName returns the name of the wire format w in errors.
func wireName(w uint64) string {
	switch w {
	case Varint:
		return "varint"
	case Fixed64:
		return "fixed64"
	case Bytes:
		return "bytes"
	case Fixed32:
		return "fixed32"
	default:
		return "invalid wire format " + strconv.FormatUint(w, 10)
	}
}
