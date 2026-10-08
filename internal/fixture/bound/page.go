// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound

//go:generate go tool kanon -type=Page -canonical

// Page is the type of the test vectors of the tag option max: a slice of
// byte slices with the bound 2, and a map with the bound 1.
type Page struct {
	Items [][]byte          `kanon:"1,max=2"`
	Meta  map[string]uint64 `kanon:"2,max=1"`
}
