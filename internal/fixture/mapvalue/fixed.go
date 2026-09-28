// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

//go:generate go tool kanon -type=Fixed

// Fixed has a map with values of each integer type that the tag option
// fixed gives four or eight bytes. The option does not apply to the keys,
// so that a map with integer keys keeps varint keys.
type Fixed struct {
	Int32  map[string]int32  `kanon:",fixed"`
	Int64  map[string]int64  `kanon:",fixed"`
	Uint32 map[string]uint32 `kanon:",fixed"`
	Uint64 map[string]uint64 `kanon:",fixed"`
	Level  map[string]Level  `kanon:",fixed"`
	Size   map[int64]Size    `kanon:",fixed"`
}
