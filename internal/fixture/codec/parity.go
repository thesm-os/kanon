// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"errors"
	"math"
)

// ParityVoid is the Parity whose encoding fails: the smallest one.
const ParityVoid Parity = math.MinInt32

// Errors of the methods of Parity.
var (
	// ErrParityVoid is the error of MarshalBinary for ParityVoid.
	ErrParityVoid = errors.New("codec: parity is void")
	// ErrParityBit is the error of UnmarshalBinary for data that is not one
	// byte of 0 or 1.
	ErrParityBit = errors.New("codec: parity encoding is not one byte of 0 or 1")
)

// Parity is an integer that encodes itself as its lowest bit, through
// MarshalBinary and UnmarshalBinary, so that two Parity values with one
// lowest bit encode alike, and two map keys of them have one projection.
type Parity int32

// MarshalBinary returns the lowest bit of p as one byte. It fails with
// ErrParityVoid for ParityVoid.
func (p Parity) MarshalBinary() ([]byte, error) {
	if p == ParityVoid {
		return nil, ErrParityVoid
	}
	return []byte{byte(p & 1)}, nil
}

// UnmarshalBinary sets p to the bit in data. It fails with ErrParityBit
// when data is not one byte of 0 or 1.
func (p *Parity) UnmarshalBinary(data []byte) error {
	if len(data) != 1 || data[0] > 1 {
		return ErrParityBit
	}
	*p = Parity(data[0])
	return nil
}
