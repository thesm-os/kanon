// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import "time"

//go:generate go tool kanon -type=Lookups -canonical

// Coord is a map key with float fields, an inline struct, whose projection
// leaves out a field of -0.0 as it leaves out a field of +0.0.
type Coord struct {
	Lat float64
	Lon float32
}

// Lookups has maps with a key of each kind that a canonical decode orders:
// keys with a float component, which a canonical decode checks for -0.0,
// pointer and time keys, whose decode skips the sort of keys of one
// projection, key types of one value, and keys of the types of this package
// that encode themselves.
type Lookups struct {
	Bools      map[bool]bool
	Int8s      map[int8]string
	Uints      map[uint16]string
	Ints       map[int]string
	Floats     map[float32]string
	Ratios     map[Ratio]string
	Complex64  map[complex64]string
	Complex128 map[complex128]string
	Pairs      map[[2]float64]string
	Pointers   map[*float64]string
	Coords     map[Coord]string
	Anys       map[any]string `kanon:",types=float64|string"`
	Units      map[struct{}]string
	Voids      map[[0]int32]string
	Times      map[time.Time]string
	Stamps     map[Stamp]string
	Counts     map[Count]string
	Texts      map[string]string
	Keys       map[[4]byte]string
	Nested     map[string]map[float64]bool
}
