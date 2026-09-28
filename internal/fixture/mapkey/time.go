// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

import "time"

//go:generate go tool kanon -type=Times

// Moment is a struct of this package without a kanon codec with a time
// field, which the code file of Times encodes as an inline struct. A decode
// yields a Moment whose time is in UTC, the local zone or a zone that
// time.FixedZone shares.
type Moment struct {
	At  time.Time
	Seq int32
}

// Times has a map with time keys and a map with keys of a struct with a
// time field. Two times of one instant in two zones are two keys, which sort
// by the offsets of their zones.
type Times struct {
	Time   map[time.Time]string
	Moment map[Moment]string
}
