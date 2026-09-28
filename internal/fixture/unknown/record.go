// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package unknown

//go:generate go tool kanon -type=Record

// Record keeps the fields that its decode does not know in Rest.
type Record struct {
	ID    int64
	Name  string
	Inner *Record
	Rest  []byte `kanon:",unknown"`
}
