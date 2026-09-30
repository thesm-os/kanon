// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"encoding/binary"
	"errors"
	"math"
)

// gaugeLength is the length of the encoding of a Gauge whose Level has a bit
// set.
const gaugeLength = 8

// Errors of the methods of Gauge.
var (
	// ErrGaugeNaN is the error of AppendBinary for a Gauge whose Level is NaN.
	ErrGaugeNaN = errors.New("canonical: gauge is NaN")
	// ErrGaugeLength is the error of UnmarshalBinary for data that is neither
	// empty nor eight bytes long.
	ErrGaugeLength = errors.New("canonical: gauge encoding is not eight bytes")
)

// Gauge is a struct that encodes itself as the eight big-endian bytes of the
// bits of its Level, through AppendBinary and UnmarshalBinary, and a Gauge
// whose Level has no bit set as no bytes. == reports a Level of -0.0 equal
// to +0.0, so a codec decides the presence of a Gauge field by the length of
// its encoding. Its decode method also decodes eight zero bytes to the zero
// Gauge, which its encode method writes as no bytes.
type Gauge struct {
	Level float64
}

// AppendBinary appends the eight big-endian bytes of the bits of g.Level to
// b, and nothing when no bit is set. It fails with ErrGaugeNaN for a NaN.
func (g Gauge) AppendBinary(b []byte) ([]byte, error) {
	if math.IsNaN(g.Level) {
		return b, ErrGaugeNaN
	}
	bits := math.Float64bits(g.Level)
	if bits == 0 {
		return b, nil
	}
	return binary.BigEndian.AppendUint64(b, bits), nil
}

// UnmarshalBinary sets g.Level to the float64 whose bits are the eight
// big-endian bytes in data, and to +0.0 for empty data. It fails with
// ErrGaugeLength for data of any other length.
func (g *Gauge) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		g.Level = 0
		return nil
	}
	if len(data) != gaugeLength {
		return ErrGaugeLength
	}
	g.Level = math.Float64frombits(binary.BigEndian.Uint64(data))
	return nil
}
