// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"errors"
	"math"
	"slices"
)

// blobLimit is the length of the longest Blob.
const blobLimit = math.MaxInt8

// ErrBlobLength is the error of MarshalBinary and UnmarshalBinary for a
// Blob longer than 127 bytes.
var ErrBlobLength = errors.New("codec: blob is longer than 127 bytes")

// Blob is a byte slice of at most 127 bytes that encodes itself as its
// bytes, through MarshalBinary and UnmarshalBinary. A slice has no ==, so a
// codec decides the presence of a Blob field by the length of its encoding,
// and an empty Blob is absent, nil or not.
type Blob []byte

// MarshalBinary returns a copy of the bytes of b. It fails with
// ErrBlobLength when b is longer than 127 bytes.
func (b Blob) MarshalBinary() ([]byte, error) {
	if len(b) > blobLimit {
		return nil, ErrBlobLength
	}
	return slices.Clone(b), nil
}

// UnmarshalBinary sets b to a copy of data. It fails with ErrBlobLength
// when data is longer than 127 bytes.
func (b *Blob) UnmarshalBinary(data []byte) error {
	if len(data) > blobLimit {
		return ErrBlobLength
	}
	*b = slices.Clone(data)
	return nil
}
