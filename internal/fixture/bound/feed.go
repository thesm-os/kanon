// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound

//go:generate go tool kanon -type=Entry,Feed

// Entry is the element type of the streamed slice of Feed.
type Entry struct {
	Name string
}

// Feed is a type with a streamed slice with the bound 2. It accepts every
// encoding of a value, so that the slice can occur more than once, and the
// bound counts the elements of every occurrence.
type Feed struct {
	Title   string
	Entries []Entry `kanon:",stream,max=2"`
}
