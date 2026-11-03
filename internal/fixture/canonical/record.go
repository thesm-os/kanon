// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

//go:generate go tool kanon -type=Line,Record -canonical

// Line is one line of a Record: a name and a value.
type Line struct {
	Name  string `kanon:"1"`
	Value []byte `kanon:"2"`
}

// Record is a count with optional lines, whose names are strings. A decode
// with the zero Options into a Record that decoded the same lines before
// keeps their names and does not copy its input.
type Record struct {
	Lines []Line `kanon:"2"`
	Count uint64 `kanon:"1"`
}
