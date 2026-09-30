// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import "time"

//go:generate go tool kanon -type=Choices -canonical

type (
	// ShapeKind selects the member of the first union of Choices.
	ShapeKind uint8
	// ToneKind selects the member of the second union of Choices.
	ToneKind uint8
)

// The members of the unions of Choices.
const (
	ShapeKindRef   ShapeKind = 1
	ShapeKindInner ShapeKind = 2
	ShapeKindItems ShapeKind = 3
	ToneKindAt     ToneKind  = 1
	ToneKindLoud   ToneKind  = 2
)

// Choices has two unions, whose members are a pointer, a struct with a kanon
// codec, a slice, a time and a bool, and a field after them: a canonical
// decode records the member of each union in a bit of its own.
type Choices struct {
	Shape ShapeKind
	Ref   *int32  `kanon:",union=Shape"`
	Inner Inner   `kanon:",union=Shape"`
	Items []int32 `kanon:",union=Shape"`
	Tone  ToneKind
	At    time.Time `kanon:",union=Tone"`
	Loud  bool      `kanon:",union=Tone"`
	Note  string
}
