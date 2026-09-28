// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/binary"
	"errors"
	"math"
)

// tokenLength is the length of the encoding of a Token.
const tokenLength = 4

// TokenRevoked is the Token whose encoding fails: the largest one.
const TokenRevoked Token = math.MaxUint32

// Errors of the methods of Token.
var (
	// ErrTokenRevoked is the error of AppendBinary for TokenRevoked.
	ErrTokenRevoked = errors.New("codec: token is revoked")
	// ErrTokenLength is the error of UnmarshalBinary for data that is not
	// four bytes long.
	ErrTokenLength = errors.New("codec: token encoding is not four bytes")
)

// Token encodes itself as four big-endian bytes, through AppendBinary and
// UnmarshalBinary. It has no MarshalBinary, so a codec appends its
// encoding to a stack array.
type Token uint32

// AppendBinary appends the four big-endian bytes of t to b. It fails with
// ErrTokenRevoked for TokenRevoked.
func (t Token) AppendBinary(b []byte) ([]byte, error) {
	if t == TokenRevoked {
		return b, ErrTokenRevoked
	}
	return binary.BigEndian.AppendUint32(b, uint32(t)), nil
}

// UnmarshalBinary sets t to the four big-endian bytes in data. It fails
// with ErrTokenLength when data is not four bytes long.
func (t *Token) UnmarshalBinary(data []byte) error {
	if len(data) != tokenLength {
		return ErrTokenLength
	}
	*t = Token(binary.BigEndian.Uint32(data))
	return nil
}
