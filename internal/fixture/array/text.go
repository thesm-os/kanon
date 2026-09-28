// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=Text

// Named strings and byte slices.
type (
	// Name is a named string.
	Name string
	// Blob is a named byte slice.
	Blob []byte
)

// Text has an array of each string and byte slice type. An array of byte
// slices has no ==, and is present when an element has bytes.
type Text struct {
	String [2]string
	Bytes  [2][]byte
	Name   [2]Name
	Blob   [2]Blob
}
