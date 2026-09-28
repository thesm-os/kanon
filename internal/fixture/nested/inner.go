// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

//go:generate go tool kanon -type=Inner

// Inner is a struct with a kanon codec, which the structs of the other
// source files of the package nest through its unexported methods.
type Inner struct {
	Label string
	Count int64
}
