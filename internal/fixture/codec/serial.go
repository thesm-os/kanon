// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import "errors"

// Lengths of the encoding of a Serial other than the zero one.
const (
	// serialShort is the length of the short encoding.
	serialShort = 4
	// serialLong is the length of the long encoding, and the capacity of a
	// Serial.
	serialLong = 8
)

// Errors of the methods of Serial.
var (
	// ErrSerialZero is the error of AppendBinary for the zero Serial.
	ErrSerialZero = errors.New("codec: the zero serial has no encoding")
	// ErrSerialLength is the error of UnmarshalBinary for data that is
	// neither 4 nor 8 bytes long.
	ErrSerialLength = errors.New("codec: serial encoding is neither 4 nor 8 bytes")
)

// Serial is an identifier of 4 or 8 bytes whose fields are unexported, as
// the digests of go.thesmos.sh/core are, so that the conformance suite
// builds its values through UnmarshalBinary. It declares kanon.Exact and
// keeps both guarantees: AppendBinary fails for the zero Serial alone, which
// has no encoding, and appends SizeKanon bytes for any other, and
// UnmarshalBinary accepts the 4 or 8 bytes that AppendBinary appends.
type Serial struct {
	b [serialLong]byte
	n int
}

// AppendBinary appends the bytes of s to b. It fails with ErrSerialZero for
// the zero Serial.
func (s Serial) AppendBinary(b []byte) ([]byte, error) {
	if s == (Serial{}) {
		return b, ErrSerialZero
	}
	return append(b, s.b[:s.n]...), nil
}

// UnmarshalBinary sets s to the 4 or 8 bytes in data. It fails with
// ErrSerialLength for data of any other length.
func (s *Serial) UnmarshalBinary(data []byte) error {
	if len(data) != serialShort && len(data) != serialLong {
		return ErrSerialLength
	}
	*s = Serial{n: len(data)}
	copy(s.b[:], data)
	return nil
}

// SizeKanon returns the length of the encoding that AppendBinary appends for
// s, and 0 for the zero Serial.
func (s Serial) SizeKanon() int {
	return s.n
}

// ExactKanon marks Serial as a kanon.Exact type.
func (Serial) ExactKanon() {}
