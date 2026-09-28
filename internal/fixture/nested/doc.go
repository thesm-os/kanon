// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package nested declares fixtures that nest structs with a kanon codec in
// every shape: by value, behind a pointer, in slices and maps, from
// another source file and from another package, recursively, through
// embedding, without a string, without fields, and with a field whose
// encoding fails. A nested struct encodes as a length and its fields, and
// is present as a field when its encoding has bytes.
//
// # Dependency position
//
// nested imports the fixture packages codec and external, and its
// generated code the kanon runtime.
package nested
