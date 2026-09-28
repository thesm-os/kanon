// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

// Wire formats: the low three bits of a tag, which state how the value
// after the tag is laid out, so that a decoder skips a field it does not
// know. The generated code spells them as the constants of package wire
// that [wireName] names.
const (
	// wireVarint is one varint.
	wireVarint = 0
	// wireFixed64 is eight bytes.
	wireFixed64 = 1
	// wireBytes is a varint length and that many bytes.
	wireBytes = 2
	// wireFixed32 is four bytes.
	wireFixed32 = 5
)

// Names of the constants of package wire for the wire formats.
const (
	varintName  = "Varint"
	fixed64Name = "Fixed64"
	bytesName   = "Bytes"
	fixed32Name = "Fixed32"
)

// Widths of the fixed-size values, in bytes.
const (
	fixed32Width = 4
	fixed64Width = 8
	// complex128Width is the length of the encoding of a complex128 after
	// its varint length.
	complex128Width = 16
)

// kind is the encoding of a Go value. A value of a kind encodes alike
// wherever it occurs: as a field, an element, a map key or a map value, and
// as the value that a pointer or an interface holds. The zero kind is
// invalid, and every value that [classifier.classify] returns has one of
// the kinds below.
type kind uint8

// Kinds.
const (
	// kindBool is a varint of 0 or 1. The decode reads any other varint as
	// true.
	kindBool kind = 1
	// kindInt is the zigzag varint of a signed integer. The decode rejects
	// a value outside the range of the Go type.
	kindInt kind = 2
	// kindUint is the varint of an unsigned integer. The decode rejects a
	// value outside the range of the Go type.
	kindUint kind = 3
	// kindFixed32 is the four little-endian bytes of an int32 or a uint32
	// under the tag option fixed.
	kindFixed32 kind = 4
	// kindFixed64 is the eight little-endian bytes of an int64 or a uint64
	// under the tag option fixed.
	kindFixed64 kind = 5
	// kindFloat32 is the four little-endian IEEE 754 bytes of a float32.
	kindFloat32 kind = 6
	// kindFloat64 is the eight little-endian IEEE 754 bytes of a float64.
	kindFloat64 kind = 7
	// kindComplex64 is the four little-endian IEEE 754 bytes of the real
	// part of a complex64, then those of its imaginary part.
	kindComplex64 kind = 8
	// kindComplex128 is a varint length of 16, then the eight little-endian
	// IEEE 754 bytes of the real part of a complex128 and those of its
	// imaginary part.
	kindComplex128 kind = 9
	// kindString is a varint length and the bytes of a string.
	kindString kind = 10
	// kindBytes is a varint length and the bytes of a byte slice.
	kindBytes kind = 11
	// kindByteArray is a varint length and the bytes of a [N]byte. The
	// decode accepts exactly N bytes.
	kindByteArray kind = 12
	// kindStruct is a varint length and the numbered fields of a struct:
	// the encoding of the struct's kanon codec, or, for an inline struct,
	// the one that the functions of the code file write, which is the
	// same.
	kindStruct kind = 13
	// kindTime is a varint length and the encoding of a time.Time: its Unix
	// seconds as field 1, its nanoseconds as field 2, and the offset of its
	// zone in seconds east of UTC as field 3 when its location is not UTC.
	kindTime kind = 14
	// kindBinary is a varint length and the encoding of a type that encodes
	// itself, through the first family of methods of [selfCodecs] that it
	// has.
	kindBinary kind = 15
	// kindSlice is a varint length and the elements of a slice other than
	// a byte slice, back to back.
	kindSlice kind = 16
	// kindArray is a varint length and the elements of an array other than
	// a byte array, back to back. The decode accepts exactly N elements.
	kindArray kind = 17
	// kindMap is a varint length and the keys and values of a map,
	// alternating, in ascending key order.
	kindMap kind = 18
	// kindPointer is a presence byte, 0 for nil or 1, and after a 1 the
	// encoding of the value the pointer points at. A pointer field leaves
	// the presence byte out: its tag is present exactly when the pointer is
	// not nil. When that field points at a pointer or an interface in turn,
	// a varint length precedes the encoding of that value.
	kindPointer kind = 19
	// kindInterface is the varint number of the dynamic type of an
	// interface, 0 for nil, and the encoding of the value of that type. An
	// interface field is present exactly when it is not nil, and a varint
	// length precedes its encoding.
	kindInterface kind = 20
)

// wire returns the wire format of a value of k: a varint for a bool and an
// integer, four or eight bytes for a fixed-size value, and a length and
// bytes otherwise.
func (k kind) wire() int {
	switch k {
	case kindBool, kindInt, kindUint:
		return wireVarint
	case kindFixed64, kindFloat64, kindComplex64:
		return wireFixed64
	case kindFixed32, kindFloat32:
		return wireFixed32
	default:
		return wireBytes
	}
}

// width returns the length in bytes of a value of the fixed-size kind k:
// four for a kind of the wire format Fixed32, and eight for one of Fixed64.
func (k kind) width() int {
	if k.wire() == wireFixed32 {
		return fixed32Width
	}
	return fixed64Width
}

// wireName returns the name of the constant of package wire for the wire
// format w, which the generated code spells.
func wireName(w int) string {
	switch w {
	case wireVarint:
		return varintName
	case wireFixed64:
		return fixed64Name
	case wireBytes:
		return bytesName
	default:
		return fixed32Name
	}
}
