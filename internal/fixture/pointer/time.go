// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

import "time"

//go:generate go tool kanon -type=Times

// Times has a pointer field to a time. A pointer to the zero time is
// present.
type Times struct {
	Time *time.Time
}
