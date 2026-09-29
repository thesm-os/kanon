// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import "errors"

// hashCapacity is the number of bytes that a Hash can contain.
const hashCapacity = 8

// ErrHashSize is the error of AppendBinary for a Hash whose Size is outside
// 1 to 8, and of UnmarshalBinary for data of such a length. A Hash of Size 0
// has no encoding.
var ErrHashSize = errors.New("codec: hash size outside 1 to 8 bytes")

// Hash is a digest of 1 to 8 bytes and its size, as crypto.Digest of
// go.thesmos.sh/core has. It encodes itself as its first Size bytes, through
// AppendBinary and UnmarshalBinary. It is a kanon.Sizer, and it declares
// IsZero, which a codec tests the presence of a Hash field with in place of
// ==.
type Hash struct {
	Sum  [hashCapacity]byte
	Size int
}

// AppendBinary appends the first h.Size bytes of h.Sum to b. It fails with
// ErrHashSize for a Size outside 1 to 8.
func (h Hash) AppendBinary(b []byte) ([]byte, error) {
	if h.Size < 1 || h.Size > hashCapacity {
		return b, ErrHashSize
	}
	return append(b, h.Sum[:h.Size]...), nil
}

// UnmarshalBinary sets h to the bytes in data. It fails with ErrHashSize
// for empty data and for data longer than 8 bytes.
func (h *Hash) UnmarshalBinary(data []byte) error {
	if len(data) == 0 || len(data) > hashCapacity {
		return ErrHashSize
	}
	*h = Hash{Size: len(data)}
	copy(h.Sum[:], data)
	return nil
}

// SizeKanon returns h.Size, the length of the encoding that AppendBinary
// appends for a Size from 1 to 8.
func (h Hash) SizeKanon() int {
	return h.Size
}

// IsZero reports whether h is the zero Hash.
func (h Hash) IsZero() bool {
	return h == Hash{}
}
