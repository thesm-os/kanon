// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Bools

// Flag is a named bool.
type Flag bool

// Bools has a map with values of each bool type.
type Bools struct {
	Bool map[string]bool
	Flag map[string]Flag
}
