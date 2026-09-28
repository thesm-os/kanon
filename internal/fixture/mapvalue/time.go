// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

import "time"

//go:generate go tool kanon -type=Times

// Times has a map with time values.
type Times struct {
	Time map[string]time.Time
}
