// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package pointer declares fixtures with a pointer field to every Go type
// that kanon encodes, one type family per file, and chains of pointers. A
// pointer field is present whenever it is not nil, a pointer to a zero
// value included, and encodes the value that it points at. A pointer to a
// pointer puts the length of the encoding of the inner pointer before it.
//
// # Dependency position
//
// pointer imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package pointer
