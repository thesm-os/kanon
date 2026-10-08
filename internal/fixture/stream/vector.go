// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package stream

//go:generate go tool kanon -type=Inner,Bundle -canonical

// Inner is the element type of the test vectors of the stream decoder: a
// string and an integer.
type Inner struct {
	Label string
	Count int64
}

// Bundle is the type of the test vectors of the stream decoder: a name, a
// streamed byte slice and a streamed slice of Inner.
type Bundle struct {
	Name    string
	Data    []byte  `kanon:",stream"`
	Members []Inner `kanon:",stream"`
}
