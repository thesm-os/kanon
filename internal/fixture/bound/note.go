// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound

//go:generate go tool kanon -type=Note -views

// Note is the type of the test vectors of the tag option max on a string and
// a byte slice: a label with the bound 2, a streamed payload with the bound 3,
// and a sequence number without a bound. It accepts every encoding of a
// value, so that a field can occur more than once, and the bound checks each
// occurrence.
type Note struct {
	Label string `kanon:"1,max=2"`
	Body  []byte `kanon:"2,stream,max=3"`
	Seq   uint64 `kanon:"3"`
}
