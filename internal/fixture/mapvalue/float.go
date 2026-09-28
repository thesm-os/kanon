// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Floats,Complexes

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

// Floats has a map with values of each float type.
type Floats struct {
	Float32 map[string]float32
	Float64 map[string]float64
	Ratio   map[string]Ratio
	Score   map[string]Score
}

// Complexes has a map with values of each complex type.
type Complexes struct {
	Complex64  map[string]complex64
	Complex128 map[string]complex128
	Phase      map[string]Phase
	Wave       map[string]Wave
}
