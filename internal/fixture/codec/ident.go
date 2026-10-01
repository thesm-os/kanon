// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/binary"
	"errors"
)

// identLength is the length of the encoding of an Ident other than the zero
// one.
const identLength = 8

// ErrIdentLength is the error of UnmarshalBinary for data that is neither
// empty nor the 8 bytes of an Ident other than the zero one.
var ErrIdentLength = errors.New("codec: ident encoding is neither empty nor 8 bytes of a nonzero ident")

// Ident is an identifier whose zero value encodes as no bytes, as the
// identifiers of go.thesmos.sh/core encode. It is a kanon.Appender: its
// append method fails for no value, and AppendKanon appends the same bytes
// without an error result, so that kanon writes it in every position
// without an error path.
type Ident uint64

// AppendBinary appends the encoding of x to b, as AppendKanon appends it.
func (x Ident) AppendBinary(b []byte) ([]byte, error) {
	return x.AppendKanon(b), nil
}

// AppendKanon appends the 8 big-endian bytes of x to b, and nothing for the
// zero Ident.
func (x Ident) AppendKanon(b []byte) []byte {
	if x == 0 {
		return b
	}
	return binary.BigEndian.AppendUint64(b, uint64(x))
}

// UnmarshalBinary sets x to the zero Ident for empty data, and to the 8
// big-endian bytes in data otherwise. It fails with ErrIdentLength for data
// of any other length, and for 8 zero bytes, which AppendKanon does not
// write.
func (x *Ident) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		*x = 0
		return nil
	}
	if len(data) != identLength || binary.BigEndian.Uint64(data) == 0 {
		return ErrIdentLength
	}
	*x = Ident(binary.BigEndian.Uint64(data))
	return nil
}

// SizeKanon returns the length of the encoding of x: 8, and 0 for the zero
// Ident.
func (x Ident) SizeKanon() int {
	if x == 0 {
		return 0
	}
	return identLength
}

// ExactKanon marks Ident as a kanon.Exact type.
func (Ident) ExactKanon() {}
