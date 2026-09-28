// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"encoding/binary"
	"errors"
	"math"
)

// markLength is the length of the encoding of a Mark.
const markLength = 2

// MarkVoid is the Mark whose encoding fails: the largest one.
const MarkVoid Mark = math.MaxUint16

// Errors of the methods of Mark.
var (
	// ErrMarkVoid is the error of AppendBinary for MarkVoid.
	ErrMarkVoid = errors.New("external: mark is void")
	// ErrMarkLength is the error of UnmarshalBinary for data that is not two
	// bytes long.
	ErrMarkLength = errors.New("external: mark encoding is not two bytes")
)

// Mark encodes itself as two big-endian bytes, through AppendBinary and
// UnmarshalBinary, so that a codec appends its encoding to a stack array
// and its encode allocates nothing.
type Mark uint16

// AppendBinary appends the two big-endian bytes of m to b. It fails with
// ErrMarkVoid for MarkVoid.
func (m Mark) AppendBinary(b []byte) ([]byte, error) {
	if m == MarkVoid {
		return b, ErrMarkVoid
	}
	return binary.BigEndian.AppendUint16(b, uint16(m)), nil
}

// UnmarshalBinary sets m to the two big-endian bytes in data. It fails with
// ErrMarkLength when data is not two bytes long.
func (m *Mark) UnmarshalBinary(data []byte) error {
	if len(data) != markLength {
		return ErrMarkLength
	}
	*m = Mark(binary.BigEndian.Uint16(data))
	return nil
}
