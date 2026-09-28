// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

//go:generate go tool kanon -type=Keeps

// Kept is an inline struct that keeps the fields that its decode does not
// know, and writes them after its own fields.
type Kept struct {
	Name string
	Rest []byte `kanon:",unknown"`
}

// Keeps has an inline struct that keeps unknown fields.
type Keeps struct {
	Kept Kept
}
