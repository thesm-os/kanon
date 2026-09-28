// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package field declares fixtures with a field of every Go type that kanon
// encodes, one type family per file: bools, signed and unsigned integers,
// the fixed-size encoding, floats and complex numbers, strings and byte
// slices, byte arrays, times, aliases, named types of another package,
// structs, and types that encode themselves. Each family has the builtin
// types and named types of this package. A field encodes unless it has
// the zero value.
//
// # Dependency position
//
// field imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package field
