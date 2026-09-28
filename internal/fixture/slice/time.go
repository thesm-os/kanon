// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package slice

import "time"

//go:generate go tool kanon -type=Times

// Times has a slice of times. An element encodes the zero time as it
// encodes any other time.
type Times struct {
	Time []time.Time
}
