// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package mapkey declares fixtures with a map whose keys have every
// comparable Go type that kanon encodes, one key type family per file. A
// map writes its keys in ascending order: bools false first, numbers and
// strings as cmp.Compare orders them, floats with NaN first, complex
// numbers by the real and then the imaginary part, byte arrays bytewise,
// times by instant and then zone, arrays element by element, pointers nil
// first, structs field by field in ascending field number, and types that
// encode themselves by their encoding. A key type of one value allows one
// entry.
//
// # Dependency position
//
// mapkey imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package mapkey
