// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=Bools

// Flag is a named bool.
type Flag bool

// Bools has an array of each bool type.
type Bools struct {
	Bool [2]bool
	Flag [2]Flag
}
