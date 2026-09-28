// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

//go:generate go tool kanon -type=Bools

// Flag is a named bool.
type Flag bool

// Bools has a field of each bool type.
type Bools struct {
	Bool bool
	Flag Flag
}
