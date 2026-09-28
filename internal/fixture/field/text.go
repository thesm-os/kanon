// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

//go:generate go tool kanon -type=Text

// Named strings and byte slices.
type (
	// Name is a named string.
	Name string
	// Blob is a named byte slice.
	Blob []byte
)

// Text has a field of each string and byte slice type. A string need not
// be valid UTF-8, and a nil and an empty byte slice encode alike.
type Text struct {
	String string
	Bytes  []byte
	Name   Name
	Blob   Blob
}
