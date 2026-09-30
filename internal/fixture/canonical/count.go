// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"errors"
	"math"
	"strconv"
)

// CountVoid is the Count without an encoding, the largest.
const CountVoid Count = math.MaxUint32

// Errors of the methods of Count.
var (
	// ErrCountVoid is the error of AppendText for CountVoid.
	ErrCountVoid = errors.New("canonical: the void count has no encoding")
	// ErrCountText is the error of UnmarshalText for text that is not a
	// decimal count of 32 bits.
	ErrCountText = errors.New("canonical: count text is not a decimal count of 32 bits")
)

// Count is a count that encodes itself as its decimal text, through
// AppendText and UnmarshalText. Its decode method also decodes a text with
// leading zeros, such as 007, which its encode method writes as 7, so that
// such a text is not the canonical encoding of its value. The zero Count
// encodes as 0, and CountVoid has no encoding.
type Count uint32

// AppendText appends the decimal text of c to b. It fails with ErrCountVoid
// for CountVoid.
func (c Count) AppendText(b []byte) ([]byte, error) {
	if c == CountVoid {
		return b, ErrCountVoid
	}
	return strconv.AppendUint(b, uint64(c), 10), nil
}

// UnmarshalText sets c to the decimal count in text. It fails with
// ErrCountText for text that is not a decimal count of 32 bits.
func (c *Count) UnmarshalText(text []byte) error {
	n, err := strconv.ParseUint(string(text), 10, 32)
	if err != nil {
		return ErrCountText
	}
	*c = Count(n)
	return nil
}
