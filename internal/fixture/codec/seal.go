// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import "errors"

// sealLength is the length of a Seal and of its encoding.
const sealLength = 4

// SealVoid is the Seal ff 00 00 00, which has no encoding.
var SealVoid = Seal{0xff}

// Errors of the methods of Seal.
var (
	// ErrSealZero is the error of AppendBinary for the zero Seal.
	ErrSealZero = errors.New("codec: the zero seal has no encoding")
	// ErrSealVoid is the error of AppendBinary for SealVoid.
	ErrSealVoid = errors.New("codec: the void seal has no encoding")
	// ErrSealLength is the error of UnmarshalBinary for data that is not
	// four bytes long.
	ErrSealLength = errors.New("codec: seal encoding is not four bytes")
)

// Seal is an array of four bytes that encodes itself as its bytes, through
// AppendBinary and UnmarshalBinary. The zero Seal has no encoding, as the
// zero digest of a hash has none. == compares every bit of a Seal, so a
// codec leaves out a field of the zero Seal without calling AppendBinary.
// SealVoid has no encoding either, so that the encode of a Seal field fails
// for it.
type Seal [sealLength]byte

// AppendBinary appends the bytes of s to b. It fails with ErrSealZero for
// the zero Seal, and with ErrSealVoid for SealVoid.
func (s Seal) AppendBinary(b []byte) ([]byte, error) {
	if s == (Seal{}) {
		return b, ErrSealZero
	}
	if s == SealVoid {
		return b, ErrSealVoid
	}
	return append(b, s[:]...), nil
}

// UnmarshalBinary sets s to the four bytes in data. It fails with
// ErrSealLength when data is not four bytes long.
func (s *Seal) UnmarshalBinary(data []byte) error {
	if len(data) != sealLength {
		return ErrSealLength
	}
	*s = Seal(data)
	return nil
}
