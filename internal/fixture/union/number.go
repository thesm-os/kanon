// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

//go:generate go tool kanon -type=Numbers

// NumberKind selects the member of the union of Numbers.
type NumberKind uint8

// The members of the union of Numbers.
const (
	NumberKindBool       NumberKind = 1
	NumberKindInt        NumberKind = 2
	NumberKindInt8       NumberKind = 3
	NumberKindUint16     NumberKind = 4
	NumberKindUint64     NumberKind = 5
	NumberKindFixed32    NumberKind = 6
	NumberKindFixed64    NumberKind = 7
	NumberKindFloat32    NumberKind = 8
	NumberKindFloat64    NumberKind = 9
	NumberKindComplex64  NumberKind = 10
	NumberKindComplex128 NumberKind = 11
	NumberKindDuration   NumberKind = 12
)

// Numbers has a union with a member of each kind of number: a bool,
// signed and unsigned integers, the fixed-size encoding, floats and
// complex numbers.
type Numbers struct {
	Kind       NumberKind
	Bool       bool       `kanon:",union=Kind"`
	Int        int        `kanon:",union=Kind"`
	Int8       int8       `kanon:",union=Kind"`
	Uint16     uint16     `kanon:",union=Kind"`
	Uint64     uint64     `kanon:",union=Kind"`
	Fixed32    int32      `kanon:",union=Kind,fixed"`
	Fixed64    uint64     `kanon:",union=Kind,fixed"`
	Float32    float32    `kanon:",union=Kind"`
	Float64    float64    `kanon:",union=Kind"`
	Complex64  complex64  `kanon:",union=Kind"`
	Complex128 complex128 `kanon:",union=Kind"`
	Duration   Duration   `kanon:",union=Kind"`
}

// Duration is a named int64 of this package.
type Duration int64
