// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

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

// Floats has an array of each float type. An array is present when a bit
// of an element is set, so that an array of negative zeros is present,
// which == on the array cannot decide.
type Floats struct {
	Float32 [2]float32
	Float64 [2]float64
	Ratio   [2]Ratio
	Score   [2]Score
}

// Complexes has an array of each complex type.
type Complexes struct {
	Complex64  [2]complex64
	Complex128 [2]complex128
	Phase      [2]Phase
	Wave       [2]Wave
}
