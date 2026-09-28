// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

import "time"

//go:generate go tool kanon -type=Signed

// Named signed integers.
type (
	// Tiny is a named int8.
	Tiny int8
	// Small is a named int16.
	Small int16
	// Level is a named int32.
	Level int32
	// Amount is a named int64.
	Amount int64
	// Offset is a named int.
	Offset int
)

// Signed has an array of each signed integer type.
type Signed struct {
	Int      [2]int
	Int8     [2]int8
	Int16    [2]int16
	Int32    [2]int32
	Int64    [2]int64
	Tiny     [2]Tiny
	Small    [2]Small
	Level    [2]Level
	Amount   [2]Amount
	Offset   [2]Offset
	Duration [2]time.Duration
}
