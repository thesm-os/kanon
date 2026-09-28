// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=Floats,Complexes,Gauge

// Named floats and complex numbers.
type (
	// Ratio is a named float32.
	Ratio float32
	// Score is a named float64.
	Score float64
	// Phase is a named complex64.
	Phase complex64
	// Wave is a named complex128.
	Wave complex128
)

// Coord is a struct of this package without a kanon codec whose fields are
// floats, which the code file of Floats encodes as an inline struct. As a
// map key, it writes a field of -0.0 as absent, and an array field of
// elements of -0.0 as absent too.
type Coord struct {
	Lat  float64
	Lon  float32
	Span [2]float64
}

// Gauge is a struct with a kanon codec of this package whose fields are
// complex numbers. As a map key, it writes its fields itself, since its
// methods write a field of -0.0 as present.
type Gauge struct {
	Phase complex64
	Wave  complex128
}

// Floats has a map with keys of each float type, and with keys that contain
// floats: an inline struct, an array of them, a struct with a kanon codec
// and a pointer. A lookup does not find a NaN key, so that the keys sort
// with their values, and a key with a NaN component fails the encode.
type Floats struct {
	Float32 map[float32]string
	Float64 map[float64]string
	Ratio   map[Ratio]string
	Score   map[Score]string
	Coord   map[Coord]string
	Coords  map[[2]Coord]string
	Gauge   map[Gauge]string
	Pointer map[*float64]string
}

// Complexes has a map with keys of each complex type.
type Complexes struct {
	Complex64  map[complex64]string
	Complex128 map[complex128]string
	Phase      map[Phase]string
	Wave       map[Wave]string
}
