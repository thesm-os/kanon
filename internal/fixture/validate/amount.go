// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/binary"
	"errors"
	"math"
)

// Errors of the methods of Amount.
var (
	// ErrAmountRange is the error of valid, AppendBinary and
	// UnmarshalBinary for math.MinInt64, which the domain of Amount
	// excludes.
	ErrAmountRange = errors.New("validate: amount is math.MinInt64")
	// ErrAmountLength is the error of UnmarshalBinary for data that is not
	// eight bytes long.
	ErrAmountLength = errors.New("validate: amount encoding is not eight bytes")
)

// Amount is an int64 whose domain excludes math.MinInt64, with a binary form
// of eight big-endian bytes, as fixed.Fixed64 of go.thesmos.sh/core has.
// Its ValidateKanon makes kanon encode it as a zigzag varint instead.
type Amount int64

// valid returns ErrAmountRange for math.MinInt64.
func (a Amount) valid() error {
	if a == math.MinInt64 {
		return ErrAmountRange
	}
	return nil
}

// AppendBinary appends the eight big-endian bytes of the two's complement
// of a to b. It fails with ErrAmountRange for math.MinInt64.
func (a Amount) AppendBinary(b []byte) ([]byte, error) {
	if err := a.valid(); err != nil {
		return b, err
	}
	return binary.Append(b, binary.BigEndian, int64(a))
}

// UnmarshalBinary sets a to the int64 whose two's complement is the eight
// big-endian bytes in data. It fails with ErrAmountLength for data that is
// not eight bytes long, and with ErrAmountRange for math.MinInt64, and
// leaves a unchanged then.
func (a *Amount) UnmarshalBinary(data []byte) error {
	var v int64
	if n, err := binary.Decode(data, binary.BigEndian, &v); err != nil || n != len(data) {
		return ErrAmountLength
	}
	if err := Amount(v).valid(); err != nil {
		return err
	}
	*a = Amount(v)
	return nil
}
