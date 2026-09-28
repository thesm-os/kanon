// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"errors"
	"fmt"
	"math"
	"strconv"
)

// codeBits is the bit size of a Code.
const codeBits = 32

// CodeRetired is the Code whose encoding fails: the largest one.
const CodeRetired Code = math.MaxUint32

// ErrCodeRetired is the error of MarshalBinary for CodeRetired.
var ErrCodeRetired = errors.New("external: code is retired")

// Code encodes itself as its decimal digits, through MarshalBinary and
// UnmarshalBinary. It has no AppendBinary, so that a codec encodes it
// through MarshalBinary.
type Code uint32

// MarshalBinary returns the decimal digits of c. It fails with
// ErrCodeRetired for CodeRetired.
func (c Code) MarshalBinary() ([]byte, error) {
	if c == CodeRetired {
		return nil, ErrCodeRetired
	}
	return strconv.AppendUint(nil, uint64(c), 10), nil
}

// UnmarshalBinary sets c to the decimal number in data. It fails when data
// is not a decimal number of 32 bits.
func (c *Code) UnmarshalBinary(data []byte) error {
	v, err := strconv.ParseUint(string(data), 10, codeBits)
	if err != nil {
		return fmt.Errorf("external: code %q: %w", data, err)
	}
	*c = Code(v)
	return nil
}
