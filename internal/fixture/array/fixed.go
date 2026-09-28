// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

//go:generate go tool kanon -type=Fixed

// Fixed has an array of each integer type that the tag option fixed gives
// four or eight bytes, so that every array of a type has one length.
type Fixed struct {
	Int32  [2]int32  `kanon:",fixed"`
	Int64  [2]int64  `kanon:",fixed"`
	Uint32 [2]uint32 `kanon:",fixed"`
	Uint64 [2]uint64 `kanon:",fixed"`
	Level  [2]Level  `kanon:",fixed"`
	Size   [2]Size   `kanon:",fixed"`
}
