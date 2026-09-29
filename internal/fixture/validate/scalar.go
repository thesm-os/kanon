// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

//go:generate go tool kanon -type=Flag,Weight,Wave,Phase

// Flag is a bool whose ValidateKanon accepts both values.
type Flag bool

// Weight is a float32 whose ValidateKanon accepts every value, a NaN
// included.
type Weight float32

// Wave is a complex64 whose ValidateKanon accepts every value.
type Wave complex64

// Phase is a complex128 whose ValidateKanon accepts every value.
type Phase complex128
