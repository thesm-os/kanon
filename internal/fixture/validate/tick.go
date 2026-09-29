// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/binary"
	"errors"
)

//go:generate go tool kanon -type=Tick

// tickSize is the length of the binary form of a Tick.
const tickSize = 8

// ErrTickLength is the error of UnmarshalBinary for data that is not eight
// bytes long.
var ErrTickLength = errors.New("validate: tick encoding is not eight bytes")

// Tick is a counter of every uint64 value, with a binary form of eight
// big-endian bytes, as epoch.Epoch of go.thesmos.sh/core has. Its
// ValidateKanon, which accepts every value, makes kanon encode it as a
// varint instead.
type Tick uint64

// AppendBinary appends the eight big-endian bytes of t to b. It does not
// fail.
func (t Tick) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, uint64(t)), nil
}

// UnmarshalBinary sets t to the eight big-endian bytes in data. It fails
// with ErrTickLength for data that is not eight bytes long.
func (t *Tick) UnmarshalBinary(data []byte) error {
	if len(data) != tickSize {
		return ErrTickLength
	}
	*t = Tick(binary.BigEndian.Uint64(data))
	return nil
}
