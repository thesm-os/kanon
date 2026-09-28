// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package number

//go:generate go tool kanon -type=Skipped,Skips

// Skipped has fields that the encoding leaves out: fields tagged "-", an
// unexported field, a function, a channel, pointers to a function and to a
// channel, and a blank field. The fields of the encoding take the numbers of
// their order among them, so that the fields left out take no number. Reset
// and a decode clear every field left out but the blank one, which no code
// reads or writes.
type Skipped struct {
	First    int32
	Note     string `kanon:"-"`
	hidden   string
	Hook     func()
	Events   chan int
	Callback *func()
	Feed     **chan int
	Handlers map[string]func() `kanon:"-"`
	Extra    any               `kanon:"-"`
	_        int32
	Second   string
}

// Skips nests Skipped by value and behind a pointer, so that its encoding
// leaves out the fields that the encoding of Skipped leaves out.
type Skips struct {
	Value   Skipped
	Pointer *Skipped
}
