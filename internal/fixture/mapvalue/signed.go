// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

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

// Signed has a map with values of each signed integer type.
type Signed struct {
	Int      map[string]int
	Int8     map[string]int8
	Int16    map[string]int16
	Int32    map[string]int32
	Int64    map[string]int64
	Tiny     map[string]Tiny
	Small    map[string]Small
	Level    map[string]Level
	Amount   map[string]Amount
	Offset   map[string]Offset
	Duration map[string]time.Duration
}
