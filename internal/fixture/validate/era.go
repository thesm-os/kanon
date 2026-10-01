// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/binary"
	"errors"
)

//go:generate go tool kanon -type=Era

// eraSize is the length of the binary form of an Era.
const eraSize = 8

// ErrEraLength is the error of UnmarshalBinary for data that is not eight
// bytes long.
var ErrEraLength = errors.New("validate: era encoding is not eight bytes")

// Era is a counter with a binary form of eight big-endian bytes, as
// epoch.Epoch of go.thesmos.sh/core has. Its kanon directive has no
// -validate, so the ValidateKanon that kanon generates for it returns nil:
// kanon encodes an Era as a varint, and the generated code of a struct with
// an Era field writes and reads it without the call.
type Era uint64

// AppendBinary appends the eight big-endian bytes of e to b.
func (e Era) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, uint64(e)), nil
}

// UnmarshalBinary sets e to the eight big-endian bytes in data. It fails
// with ErrEraLength for data that is not eight bytes long.
func (e *Era) UnmarshalBinary(data []byte) error {
	if len(data) != eraSize {
		return ErrEraLength
	}
	*e = Era(binary.BigEndian.Uint64(data))
	return nil
}
