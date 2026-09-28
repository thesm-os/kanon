// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package number

//go:generate go tool kanon -type=Order

// Order declares its fields in the reverse order of their numbers, so that
// its encoding writes them in ascending field number, not in declaration
// order.
type Order struct {
	Third  string `kanon:"3"`
	Second int64  `kanon:"2"`
	First  bool   `kanon:"1"`
}
