// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

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

// Signed has a field of each signed integer type. A decode rejects a
// value outside the range of the type of the field.
type Signed struct {
	Int      int
	Int8     int8
	Int16    int16
	Int32    int32
	Int64    int64
	Tiny     Tiny
	Small    Small
	Level    Level
	Amount   Amount
	Offset   Offset
	Duration time.Duration
}
