// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import "errors"

// Lengths of the encoding of a Sum other than the zero one.
const (
	// sumShort is the length of the short encoding.
	sumShort = 4
	// sumLong is the length of the long encoding, and the capacity of a Sum.
	sumLong = 8
)

// First bytes of the Sums whose encode fails, one per way to fail it.
const (
	// sumVoid begins a Sum that has no encoding.
	sumVoid = 0xff
	// sumUnsized begins a Sum whose SizeKanon is below 0.
	sumUnsized = 'a'
	// sumOversized begins a Sum whose SizeKanon counts more bytes than its
	// encoding has.
	sumOversized = 'b'
)

// Errors of the methods of Sum.
var (
	// ErrSumZero is the error of AppendBinary for the zero Sum.
	ErrSumZero = errors.New("codec: the zero sum has no encoding")
	// ErrSumVoid is the error of AppendBinary for a Sum that begins with the
	// byte ff.
	ErrSumVoid = errors.New("codec: a sum that begins with ff has no encoding")
	// ErrSumLength is the error of UnmarshalBinary for data that is neither 4
	// nor 8 bytes long.
	ErrSumLength = errors.New("codec: sum encoding is neither 4 nor 8 bytes")
	// ErrSumLeading is the error of UnmarshalBinary for data that begins with
	// the byte 0, which no Sum begins with.
	ErrSumLeading = errors.New("codec: sum encoding begins with 0")
)

// Sum is a digest of 4 or 8 bytes whose fields are unexported, as the
// digests and identifiers of go.thesmos.sh/core are, so that no reflection
// sets them and its values come from UnmarshalBinary alone, which rejects
// some bytes of an accepted length. It encodes itself as its bytes through
// AppendBinary, and it is a kanon.Sizer with an IsZero method. The zero Sum
// has no encoding, and neither has a Sum that begins with the byte ff.
// SizeKanon returns -1 for a Sum that begins with 'a', and 16 for one that
// begins with 'b', so that the encode of a Sum field fails in every way that
// the generated code handles.
type Sum struct {
	b [sumLong]byte
	n int
}

// AppendBinary appends the bytes of s to b. It fails with ErrSumZero for the
// zero Sum, and with ErrSumVoid for a Sum that begins with the byte ff.
func (s Sum) AppendBinary(b []byte) ([]byte, error) {
	if s.IsZero() {
		return b, ErrSumZero
	}
	if s.b[0] == sumVoid {
		return b, ErrSumVoid
	}
	return append(b, s.b[:s.n]...), nil
}

// UnmarshalBinary sets s to the 4 or 8 bytes in data. It fails with
// ErrSumLength for data of any other length, and with ErrSumLeading for data
// that begins with the byte 0.
func (s *Sum) UnmarshalBinary(data []byte) error {
	if len(data) != sumShort && len(data) != sumLong {
		return ErrSumLength
	}
	if data[0] == 0 {
		return ErrSumLeading
	}
	*s = Sum{n: len(data)}
	copy(s.b[:], data)
	return nil
}

// SizeKanon returns the length of the encoding that AppendBinary appends for
// s: the number of its bytes, and 0 for the zero Sum. It returns -1 for a Sum
// that begins with 'a' and 16 for one that begins with 'b', which it does not
// count.
func (s Sum) SizeKanon() int {
	switch s.b[0] {
	case sumUnsized:
		return -1
	case sumOversized:
		return 2 * sumLong
	default:
		return s.n
	}
}

// IsZero reports whether s is the zero Sum.
func (s Sum) IsZero() bool {
	return s == Sum{}
}
