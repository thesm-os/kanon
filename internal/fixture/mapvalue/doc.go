// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package mapvalue declares fixtures with a map whose values have every Go
// type that kanon encodes, one value type family per file, and maps of
// slices, maps and pointers. A map is present when it has entries, and
// encodes as one length-prefixed value that contains its keys and values,
// alternating, in ascending key order.
//
// # Dependency position
//
// mapvalue imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package mapvalue
