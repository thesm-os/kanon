// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

import "time"

//go:generate go tool kanon -type=Times

// Times has a time field. A time encodes its Unix seconds, its nanoseconds
// and the offset of its zone, and the zero time is absent.
type Times struct {
	Time time.Time
}
