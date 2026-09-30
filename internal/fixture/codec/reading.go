// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/binary"
	"errors"
	"math"
)

// readingLength is the length of the encoding of a Reading other than the
// zero one.
const readingLength = 8

// Errors of the methods of Reading.
var (
	// ErrReadingNaN is the error of MarshalBinary for a Reading whose Value
	// is NaN.
	ErrReadingNaN = errors.New("codec: reading is NaN")
	// ErrReadingLength is the error of UnmarshalBinary for data that is
	// neither empty nor eight bytes long.
	ErrReadingLength = errors.New("codec: reading encoding is not eight bytes")
)

// Reading is a struct that encodes itself as the eight big-endian bytes of
// the bits of its Value, through MarshalBinary and UnmarshalBinary, and the
// zero Reading as no bytes. == reports a Value of -0.0 equal to +0.0, so a
// codec decides the presence of a Reading field by the length of its
// encoding, and a Reading of -0.0 is present. SizeKanon returns -1 for a
// Value of +Inf and 16 for -Inf, so that the encode of an infinite Reading
// fails before MarshalBinary or with kanon.ErrSize.
type Reading struct {
	Value float64
}

// MarshalBinary returns the eight big-endian bytes of the bits of r.Value,
// and no bytes when every bit is zero. It fails with ErrReadingNaN for a
// NaN.
func (r Reading) MarshalBinary() ([]byte, error) {
	if math.IsNaN(r.Value) {
		return nil, ErrReadingNaN
	}
	bits := math.Float64bits(r.Value)
	if bits == 0 {
		return nil, nil
	}
	return binary.BigEndian.AppendUint64(nil, bits), nil
}

// SizeKanon returns the length of the encoding that MarshalBinary returns
// for r: 0 when every bit of r.Value is zero, and 8 otherwise. It returns -1
// for +Inf and 16 for -Inf, which it does not count. It makes Reading a
// kanon.Sizer, whose field is present when SizeKanon is not 0.
func (r Reading) SizeKanon() int {
	switch {
	case math.IsInf(r.Value, 1):
		return -1
	case math.IsInf(r.Value, -1):
		return 2 * readingLength
	case math.Float64bits(r.Value) == 0:
		return 0
	default:
		return readingLength
	}
}

// UnmarshalBinary sets r.Value to the float64 whose bits are the eight
// big-endian bytes in data, and to +0.0 for empty data. It fails with
// ErrReadingLength for data of any other length.
func (r *Reading) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		r.Value = 0
		return nil
	}
	if len(data) != readingLength {
		return ErrReadingLength
	}
	r.Value = math.Float64frombits(binary.BigEndian.Uint64(data))
	return nil
}
