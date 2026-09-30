// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"encoding/binary"
	"errors"
	"math"
)

// stampLength is the length of the encoding of a Stamp.
const stampLength = 4

// StampVoid is the Stamp without an encoding, the largest.
const StampVoid Stamp = math.MaxUint32

// Errors of the methods of Stamp.
var (
	// ErrStampVoid is the error of MarshalBinary for StampVoid.
	ErrStampVoid = errors.New("canonical: the void stamp has no encoding")
	// ErrStampLength is the error of UnmarshalBinary for data that is empty or
	// longer than four bytes.
	ErrStampLength = errors.New("canonical: stamp encoding is not one to four bytes")
)

// Stamp is a count that encodes itself as four big-endian bytes, its zero
// value included, through MarshalBinary alone and UnmarshalBinary. Its decode
// method also decodes a shorter big-endian count, which its encode method
// does not write, so that a canonical decode meets bytes that the encode
// method does not write. StampVoid has no encoding. == compares every bit of
// a Stamp, so a codec leaves out a field of the zero Stamp.
type Stamp uint32

// MarshalBinary returns the four big-endian bytes of s. It fails with
// ErrStampVoid for StampVoid.
func (s Stamp) MarshalBinary() ([]byte, error) {
	if s == StampVoid {
		return nil, ErrStampVoid
	}
	return binary.BigEndian.AppendUint32(nil, uint32(s)), nil
}

// UnmarshalBinary sets s to the big-endian count in data, of one to four
// bytes. It fails with ErrStampLength for any other length.
func (s *Stamp) UnmarshalBinary(data []byte) error {
	if len(data) == 0 || len(data) > stampLength {
		return ErrStampLength
	}
	var v uint32
	for _, b := range data {
		v = v<<8 | uint32(b)
	}
	*s = Stamp(v)
	return nil
}
