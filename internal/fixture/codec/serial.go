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

// ErrSerialLength is the error of UnmarshalBinary for data that is neither
// empty nor 4 or 8 bytes long.
var ErrSerialLength = errors.New("codec: serial encoding is neither empty nor 4 or 8 bytes")

// Serial is an identifier of 4 or 8 bytes whose fields are unexported, as
// the identifiers of go.thesmos.sh/core are, so that the conformance suite
// builds its values through UnmarshalBinary. It declares kanon.Exact and
// keeps both guarantees: AppendBinary never fails and appends SizeKanon
// bytes, and UnmarshalBinary accepts the empty input, which decodes to the
// zero Serial, and 4 or 8 bytes, which AppendBinary appends back.
type Serial struct {
	b [serialLong]byte
	n int
}

// AppendBinary appends the bytes of s to b, none for the zero Serial.
func (s Serial) AppendBinary(b []byte) ([]byte, error) {
	return append(b, s.b[:s.n]...), nil
}

// UnmarshalBinary sets s to the bytes in data: the zero Serial for empty
// data. It fails with ErrSerialLength for data of another length than 0, 4
// and 8.
func (s *Serial) UnmarshalBinary(data []byte) error {
	if len(data) != 0 && len(data) != serialShort && len(data) != serialLong {
		return ErrSerialLength
	}
	*s = Serial{n: len(data)}
	copy(s.b[:], data)
	return nil
}

// SizeKanon returns the length of the encoding that AppendBinary appends for
// s.
func (s Serial) SizeKanon() int {
	return s.n
}

// ExactKanon marks Serial as a kanon.Exact type.
func (Serial) ExactKanon() {}
