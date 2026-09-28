// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

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

// Floats has a pointer field to each float type.
type Floats struct {
	Float32 *float32
	Float64 *float64
	Ratio   *Ratio
	Score   *Score
}

// Complexes has a pointer field to each complex type.
type Complexes struct {
	Complex64  *complex64
	Complex128 *complex128
	Phase      *Phase
	Wave       *Wave
}
