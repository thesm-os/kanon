// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

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

// Signed has a pointer field to each signed integer type.
type Signed struct {
	Int      *int
	Int8     *int8
	Int16    *int16
	Int32    *int32
	Int64    *int64
	Tiny     *Tiny
	Small    *Small
	Level    *Level
	Amount   *Amount
	Offset   *Offset
	Duration *time.Duration
}
