// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=Bools

// Flag is a named bool.
type Flag bool

// Bools has a map with keys of each bool type, and a map whose keys and
// values encode to one length.
type Bools struct {
	Bool  map[bool]string
	Flag  map[Flag]string
	Flags map[bool]bool
}
