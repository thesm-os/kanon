// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

//go:generate go tool kanon -type=Text

// Named strings and byte slices.
type (
	// Name is a named string.
	Name string
	// Blob is a named byte slice.
	Blob []byte
)

// Text has a pointer field to each string and byte slice type. A pointer
// to an empty string or an empty byte slice is present.
type Text struct {
	String *string
	Bytes  *[]byte
	Name   *Name
	Blob   *Blob
}
