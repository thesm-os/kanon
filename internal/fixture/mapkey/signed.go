// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

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

// Signed has a map with keys of each signed integer type.
type Signed struct {
	Int      map[int]string
	Int8     map[int8]string
	Int16    map[int16]string
	Int32    map[int32]string
	Int64    map[int64]string
	Tiny     map[Tiny]string
	Small    map[Small]string
	Level    map[Level]string
	Amount   map[Amount]string
	Offset   map[Offset]string
	Duration map[time.Duration]string
}
