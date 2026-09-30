// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/binary"
	"errors"
	"math"
)

//go:generate go tool kanon -type=Tick -validate=valid

// tickSize is the length of the binary form of a Tick.
const tickSize = 8

// Errors of the methods of Tick.
var (
	// ErrTickLength is the error of UnmarshalBinary for data that is not
	// eight bytes long.
	ErrTickLength = errors.New("validate: tick encoding is not eight bytes")
	// ErrTickOverflow is the error of Tick.valid for the largest uint64,
	// which marks a counter that overflowed.
	ErrTickOverflow = errors.New("validate: tick overflowed")
)

// Tick is a counter with a binary form of eight big-endian bytes, as
// epoch.Epoch of go.thesmos.sh/core has. Its ValidateKanon makes kanon
// encode it as a varint instead. Both reject the largest uint64, which marks
// a counter that overflowed.
type Tick uint64

// AppendBinary appends the eight big-endian bytes of t to b. It fails with
// ErrTickOverflow for the largest uint64.
func (t Tick) AppendBinary(b []byte) ([]byte, error) {
	if err := t.valid(); err != nil {
		return b, err
	}
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

// valid returns ErrTickOverflow for the largest uint64.
func (t Tick) valid() error {
	if t == math.MaxUint64 {
		return ErrTickOverflow
	}
	return nil
}
