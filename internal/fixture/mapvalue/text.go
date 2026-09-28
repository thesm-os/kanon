// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Text

// Named strings and byte slices.
type (
	// Name is a named string.
	Name string
	// Blob is a named byte slice.
	Blob []byte
)

// Text has a map with values of each string and byte slice type.
type Text struct {
	String map[string]string
	Bytes  map[string][]byte
	Name   map[string]Name
	Blob   map[string]Blob
}
