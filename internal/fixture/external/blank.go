// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

//go:generate go tool kanon -type=Blank

// Blank is a struct of another package with a kanon codec whose field has
// one value, so that its encoding is empty and a field of it is never
// present.
type Blank struct {
	None [0]int32
}
