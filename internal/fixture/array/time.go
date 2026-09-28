// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

import "time"

//go:generate go tool kanon -type=Times

// Times has an array of times. An array is present when an element is not
// the zero time, which == on times in other locations cannot decide.
type Times struct {
	Time [2]time.Time
}
