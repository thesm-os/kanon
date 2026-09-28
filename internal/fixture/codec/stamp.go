// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/binary"
	"errors"
	"math"
)

// stampLength is the length of the encoding of a Stamp.
const stampLength = 2

// stampSealed is the sequence number of the Stamp whose encoding fails:
// the largest one.
const stampSealed = math.MaxUint16

// Errors of the methods of Stamp.
var (
	// ErrStampSealed is the error of AppendBinary for the Stamp with the
	// largest sequence number.
	ErrStampSealed = errors.New("codec: stamp is sealed")
	// ErrStampLength is the error of UnmarshalBinary for data that is not
	// two bytes long.
	ErrStampLength = errors.New("codec: stamp encoding is not two bytes")
)

// Stamp is a struct that encodes itself as the two big-endian bytes of its
// sequence number. Its methods take a pointer receiver, so that a codec
// calls them on addressable values only. Every Stamp, the zero one
// included, encodes to two bytes.
type Stamp struct {
	Seq uint16
}

// AppendBinary appends the two big-endian bytes of s.Seq to b. It fails
// with ErrStampSealed for the largest sequence number.
func (s *Stamp) AppendBinary(b []byte) ([]byte, error) {
	if s.Seq == stampSealed {
		return b, ErrStampSealed
	}
	return binary.BigEndian.AppendUint16(b, s.Seq), nil
}

// UnmarshalBinary sets s.Seq to the two big-endian bytes in data. It fails
// with ErrStampLength when data is not two bytes long.
func (s *Stamp) UnmarshalBinary(data []byte) error {
	if len(data) != stampLength {
		return ErrStampLength
	}
	s.Seq = binary.BigEndian.Uint16(data)
	return nil
}
