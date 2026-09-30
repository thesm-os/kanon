// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"math"
	"testing"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/canonical"
	"go.thesmos.sh/kanon/internal/fixture/union"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the decode checks that the cases run.
const (
	unmarshalCheck = "UnmarshalBinary/decodes the reference encoding of each sample as the reference decode"
	slabCheck      = "DecodeKanon/decodes an encoding in a slab at its offset as the reference decode"
	prefixCheck    = "DecodeKanon/decodes each prefix of a field between unknown fields as the reference decode"
)

// scalarsSpec describes canonical.Scalars, a canonical struct with a field of
// each kind of number, bools, text, arrays, a time, and types that encode
// themselves, of which Count, Digest and Stamp are absent at their zero value
// and Gauge is present when its encoding has bytes.
var scalarsSpec = kanontest.Spec[canonical.Scalars]{
	Fields: []kanontest.Field{
		{Name: "Bool", Number: 1},
		{Name: "Flag", Number: 2},
		{Name: "Int", Number: 3},
		{Name: "Int8", Number: 4},
		{Name: "Int16", Number: 5},
		{Name: "Int32", Number: 6},
		{Name: "Int64", Number: 7},
		{Name: "Uint", Number: 8},
		{Name: "Uint8", Number: 9},
		{Name: "Uint16", Number: 10},
		{Name: "Uint32", Number: 11},
		{Name: "Uint64", Number: 12},
		{Name: "Uintptr", Number: 13},
		{Name: "Fixed32", Number: 14, Fixed: true},
		{Name: "Fixed64", Number: 15, Fixed: true},
		{Name: "Float32", Number: 16},
		{Name: "Float64", Number: 17},
		{Name: "Ratio", Number: 18},
		{Name: "Complex64", Number: 19},
		{Name: "Complex128", Number: 20},
		{Name: "Phase", Number: 21},
		{Name: "Duration", Number: 22},
		{Name: "Text", Number: 23},
		{Name: "Blob", Number: 24},
		{Name: "Key", Number: 25},
		{Name: "Empty", Number: 26},
		{Name: "Floats", Number: 27},
		{Name: "Bools", Number: 28},
		{Name: "At", Number: 29},
		{Name: "Count", Number: 30},
		{Name: "Digest", Number: 31},
		{Name: "Stamp", Number: 32},
		{Name: "Level", Number: 33},
		{Name: "Digests", Number: 34},
		{Name: "Gauge", Number: 35},
	},
	Canonical: true,
}

// choicesSpec describes canonical.Choices, a canonical struct with two unions
// and a field after them.
var choicesSpec = kanontest.Spec[canonical.Choices]{
	Fields: []kanontest.Field{
		member("Ref", 1, "Shape", canonical.ShapeKindRef),
		member("Inner", 2, "Shape", canonical.ShapeKindInner),
		member("Items", 3, "Shape", canonical.ShapeKindItems),
		member("At", 4, "Tone", canonical.ToneKindAt),
		member("Loud", 5, "Tone", canonical.ToneKindLoud),
		{Name: "Note", Number: 6},
	},
	Canonical: true,
}

// countOff is a view.Item whose DecodeKanon decodes Count one too high.
type countOff struct{ view.Item }

// DecodeKanon decodes data, and adds 1 to Count when the decode succeeds.
func (m *countOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := m.Item.DecodeKanon(data, opts)
	if err == nil {
		m.Count++
	}
	return err
}

// offsetOff is a view.Item whose DecodeKanon reports the offset of an error
// one byte too far.
type offsetOff struct{ view.Item }

// DecodeKanon decodes data, and adds 1 to the offset of its error.
func (m *offsetOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := renamed(m.Item.DecodeKanon(data, opts), "Item", "offsetOff")
	if e, ok := errors.AsType[*kanon.DecodeError](err); ok {
		e.Offset++
	}
	return err
}

// unmarshalOff is a view.Item whose UnmarshalBinary decodes Count one too
// high.
type unmarshalOff struct{ view.Item }

// UnmarshalBinary decodes data, and adds 1 to Count when the decode
// succeeds.
func (m *unmarshalOff) UnmarshalBinary(data []byte) error {
	err := m.Item.UnmarshalBinary(data)
	if err == nil {
		m.Count++
	}
	return err
}

// shapeOff is a union.Layout whose DecodeKanon selects Circle when the
// encoding selects no shape.
type shapeOff struct{ union.Layout }

// DecodeKanon decodes data, and sets Shape to ShapeKindCircle when the
// decode leaves it 0.
func (m *shapeOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := m.Layout.DecodeKanon(data, opts)
	if m.Shape == 0 {
		m.Shape = union.ShapeKindCircle
	}
	return err
}

// kindOff is a union.Signed whose DecodeKanon selects Low when the encoding
// selects no member.
type kindOff struct{ union.Signed }

// DecodeKanon decodes data, and sets Kind to SignedKindLow when the decode
// leaves it 0.
func (m *kindOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := m.Signed.DecodeKanon(data, opts)
	if m.Kind == 0 {
		m.Kind = union.SignedKindLow
	}
	return err
}

// nanOff is a view.Record whose DecodeKanon decodes a NaN of Float64 as a
// NaN of another payload.
type nanOff struct{ view.Record }

// DecodeKanon decodes data, and flips the lowest bit of the payload of a
// NaN in Float64.
func (m *nanOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := m.Record.DecodeKanon(data, opts)
	if math.IsNaN(m.Float64) {
		m.Float64 = math.Float64frombits(math.Float64bits(m.Float64) ^ 1)
	}
	return err
}

func TestDecode(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that decodes a field to another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[countOff]{Fields: itemSpec.Fields}, slabCheck,
				"DecodeKanon decodes the value of the reference decode")
		})
		t.Run("fails for a codec that reports another offset for an error", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[offsetOff]{Fields: itemSpec.Fields}, prefixCheck,
				"DecodeKanon returns the error of the reference decode")
		})
		t.Run("fails for a codec whose UnmarshalBinary decodes a field to another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[unmarshalOff]{Fields: itemSpec.Fields}, unmarshalCheck,
				"UnmarshalBinary decodes the value of the reference decode")
		})
		t.Run("fails for a codec that decodes an unsigned discriminator to another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[shapeOff]{Fields: layoutSpec.Fields}, slabCheck,
				"DecodeKanon decodes the value of the reference decode")
		})
		t.Run("fails for a codec that decodes a signed discriminator to another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[kindOff]{Fields: signedSpec.Fields}, slabCheck,
				"DecodeKanon decodes the value of the reference decode")
		})
		t.Run("passes a codec that decodes a NaN to a NaN of another payload", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[nanOff]{Fields: recordSpec.Fields}, slabCheck)
		})
		t.Run("passes a canonical struct of every scalar type and of types that encode themselves", func(t *testing.T) {
			t.Parallel()
			passes(t, scalarsSpec)
		})
		t.Run("passes a canonical struct of two unions", func(t *testing.T) {
			t.Parallel()
			passes(t, choicesSpec)
		})
		t.Run("passes a kanon.Validator in every place of a value", func(t *testing.T) {
			t.Parallel()
			passes(t, valuesSpec)
		})
	})
}
