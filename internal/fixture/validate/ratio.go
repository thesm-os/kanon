// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"encoding/binary"
	"errors"
	"math"
)

// ratioSize is the length of the gob form of a Ratio.
const ratioSize = 8

// Errors of the methods of Ratio.
var (
	// ErrRatioNaN is the error of valid, GobEncode and GobDecode for a NaN,
	// which has no order.
	ErrRatioNaN = errors.New("validate: ratio is NaN")
	// ErrRatioLength is the error of GobDecode for data that is not eight
	// bytes long.
	ErrRatioLength = errors.New("validate: ratio encoding is not eight bytes")
)

// Ratio is a float64 other than NaN, with a gob form of the eight
// little-endian bytes of its bits. Its ValidateKanon makes kanon encode it
// as a float64 instead.
type Ratio float64

// valid returns ErrRatioNaN for a NaN.
func (r Ratio) valid() error {
	if math.IsNaN(float64(r)) {
		return ErrRatioNaN
	}
	return nil
}

// GobEncode returns the eight little-endian bytes of the bits of r. It
// fails with ErrRatioNaN for a NaN.
func (r Ratio) GobEncode() ([]byte, error) {
	if err := r.valid(); err != nil {
		return nil, err
	}
	return binary.LittleEndian.AppendUint64(nil, math.Float64bits(float64(r))), nil
}

// GobDecode sets r to the float64 whose bits are the eight little-endian
// bytes in data. It fails with ErrRatioLength for data that is not eight
// bytes long, and with ErrRatioNaN for a NaN, and leaves r unchanged then.
func (r *Ratio) GobDecode(data []byte) error {
	if len(data) != ratioSize {
		return ErrRatioLength
	}
	v := Ratio(math.Float64frombits(binary.LittleEndian.Uint64(data)))
	if err := v.valid(); err != nil {
		return err
	}
	*r = v
	return nil
}
