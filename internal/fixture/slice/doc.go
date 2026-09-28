// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package slice declares fixtures with a slice of every Go type that kanon
// encodes, one element type family per file, and slices of slices, maps
// and pointers. A slice is present when it has elements, and encodes as
// one length-prefixed value that contains its elements back to back.
//
// # Dependency position
//
// slice imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package slice
