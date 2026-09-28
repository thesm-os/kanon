// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

//go:generate go tool kanon -type=Fixed

// Fixed has a slice of each integer type that the tag option fixed gives
// four or eight bytes, so that its length is a multiple of the width.
type Fixed struct {
	Int32  []int32  `kanon:",fixed"`
	Int64  []int64  `kanon:",fixed"`
	Uint32 []uint32 `kanon:",fixed"`
	Uint64 []uint64 `kanon:",fixed"`
	Level  []Level  `kanon:",fixed"`
	Size   []Size   `kanon:",fixed"`
}
