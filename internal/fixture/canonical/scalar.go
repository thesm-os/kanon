// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import (
	"time"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/validate"
)

//go:generate go tool kanon -type=Scalars -canonical

type (
	// Flag is a named bool.
	Flag bool
	// Ratio is a named float32.
	Ratio float32
	// Phase is a named complex64.
	Phase complex64
)

// Scalars has a field of each kind of number, bools, text, a byte array,
// arrays, a time, the types of this package that encode themselves, a
// kanon.Validator of another package and a kanon.Exact type of another
// package: a field of each kind whose presence and whose varints a canonical
// decode checks. Digests encodes the zero Digest, which a field leaves out,
// so that its encode fails. A Gauge field is present when its encoding has
// bytes, since == does not compare every bit of a Gauge. A Serial field
// decodes without a second encode, and Serials encodes the zero Serial, so
// that the put function of an element of a kanon.Exact type fails.
type Scalars struct {
	Bool       bool
	Flag       Flag
	Int        int
	Int8       int8
	Int16      int16
	Int32      int32
	Int64      int64
	Uint       uint
	Uint8      uint8
	Uint16     uint16
	Uint32     uint32
	Uint64     uint64
	Uintptr    uintptr
	Fixed32    int32  `kanon:",fixed"`
	Fixed64    uint64 `kanon:",fixed"`
	Float32    float32
	Float64    float64
	Ratio      Ratio
	Complex64  complex64
	Complex128 complex128
	Phase      Phase
	Duration   time.Duration
	Text       string
	Blob       []byte
	Key        [4]byte
	Empty      [0]int32
	Floats     [2]float32
	Bools      [2]bool
	At         time.Time
	Count      Count
	Digest     Digest
	Stamp      Stamp
	Level      validate.Level
	Digests    []Digest
	Gauge      Gauge
	Serial     codec.Serial
	Serials    []codec.Serial
}
