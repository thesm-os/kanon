// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package array declares fixtures with an array of every Go type that kanon
// encodes, one element type family per file, and arrays of arrays,
// slices, maps and pointers. An array is present when an element of it is
// present, and encodes as one length-prefixed value that contains its
// elements back to back. A decode rejects an array with another number of
// elements. An array whose elements have one value is never present.
//
// # Dependency position
//
// array imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package array
