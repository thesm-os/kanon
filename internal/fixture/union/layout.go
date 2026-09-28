// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

//go:generate go tool kanon -type=Layout

// ShapeKind selects the member of the union of Layout that is a shape.
type ShapeKind uint8

// ToneKind selects the member of the union of Layout that is a tone.
type ToneKind uint8

// The members of the unions of Layout.
const (
	ShapeKindCircle ShapeKind = 1
	ShapeKindSquare ShapeKind = 2
	ToneKindLight   ToneKind  = 1
	ToneKindDark    ToneKind  = 2
)

// Layout has two unions whose members take field numbers between the
// numbers of the other fields, so that its encoding writes the members of
// one union in two runs, each in ascending field number.
type Layout struct {
	Name   string `kanon:"1"`
	Shape  ShapeKind
	Circle int32  `kanon:"2,union=Shape"`
	Note   string `kanon:"3"`
	Square int32  `kanon:"4,union=Shape"`
	Tone   ToneKind
	Light  bool   `kanon:"5,union=Tone"`
	Dark   bool   `kanon:"6,union=Tone"`
	Last   uint32 `kanon:"7"`
}
