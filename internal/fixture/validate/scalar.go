// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"errors"
	"math"
	"math/cmplx"
)

//go:generate go tool kanon -type=Flag,Weight,Wave,Phase -validate=valid

// Errors of the valid methods of Flag, Weight, Wave and Phase.
var (
	// ErrFlagSet is the error of Flag.valid for true.
	ErrFlagSet = errors.New("validate: flag is set")
	// ErrWeightNaN is the error of Weight.valid for NaN.
	ErrWeightNaN = errors.New("validate: weight is NaN")
	// ErrWaveNaN is the error of Wave.valid for a value with a NaN part.
	ErrWaveNaN = errors.New("validate: wave has a NaN part")
	// ErrPhaseNaN is the error of Phase.valid for a value with a NaN part.
	ErrPhaseNaN = errors.New("validate: phase has a NaN part")
)

// Flag is a bool that is valid when false.
type Flag bool

// valid returns ErrFlagSet for true.
func (f Flag) valid() error {
	if f {
		return ErrFlagSet
	}
	return nil
}

// Weight is a float32 whose valid method rejects NaN.
type Weight float32

// valid returns ErrWeightNaN for NaN.
func (w Weight) valid() error {
	if math.IsNaN(float64(w)) {
		return ErrWeightNaN
	}
	return nil
}

// Wave is a complex64 whose valid method rejects a value with a NaN part.
type Wave complex64

// valid returns ErrWaveNaN for a value with a NaN part.
func (w Wave) valid() error {
	if cmplx.IsNaN(complex128(w)) {
		return ErrWaveNaN
	}
	return nil
}

// Phase is a complex128 whose valid method rejects a value with a NaN part.
type Phase complex128

// valid returns ErrPhaseNaN for a value with a NaN part.
func (p Phase) valid() error {
	if cmplx.IsNaN(complex128(p)) {
		return ErrPhaseNaN
	}
	return nil
}
