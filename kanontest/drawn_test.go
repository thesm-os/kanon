// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/field"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// marshalCheck is the message of a check whose MarshalBinary returns
// another encoding than the reference encoding.
const marshalCheck = "MarshalBinary returns the reference encoding"

// Specs of fixtures that no value fails to encode, whose wrappers below
// break one encoding each.
var (
	// pageSpec describes view.Page, a struct of a slice of integers and a
	// slice of byte slices.
	pageSpec = kanontest.Spec[view.Page]{Fields: fields("Items", "Path")}
	// floatsSpec describes field.Floats, a struct of a field of each float
	// type.
	floatsSpec = kanontest.Spec[field.Floats]{Fields: fields("Float32", "Float64", "Ratio", "Score")}
)

// nilItems is a view.Page whose MarshalBinary appends a byte to the encoding
// of a nil Items, which encodes as an empty one does.
type nilItems struct{ view.Page }

// MarshalBinary returns the encoding, and a zero byte after it when Items
// is nil.
func (m *nilItems) MarshalBinary() ([]byte, error) {
	b, err := m.Page.MarshalBinary()
	if m.Items == nil {
		b = append(b, 0)
	}
	return b, err
}

// DecodeKanon decodes data as a view.Page, and names nilItems in its error,
// as the reference decode names it.
func (m *nilItems) DecodeKanon(data []byte, opts kanon.Options) error {
	return renamed(m.Page.DecodeKanon(data, opts), "Page", "nilItems")
}

// negativeFloat is a field.Floats whose MarshalBinary appends a byte to the
// encoding of a negative Float64.
type negativeFloat struct{ field.Floats }

// MarshalBinary returns the encoding, and a zero byte after it when Float64
// is negative.
func (m *negativeFloat) MarshalBinary() ([]byte, error) {
	b, err := m.Floats.MarshalBinary()
	if m.Float64 < 0 {
		b = append(b, 0)
	}
	return b, err
}

// DecodeKanon decodes data as a field.Floats, and names negativeFloat in
// its error, as the reference decode names it.
func (m *negativeFloat) DecodeKanon(data []byte, opts kanon.Options) error {
	return renamed(m.Floats.DecodeKanon(data, opts), "Floats", "negativeFloat")
}

func TestDrawn(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a struct of a slice of integers and a slice of byte slices", func(t *testing.T) {
			t.Parallel()
			holds(t, pageSpec, propertyCheck)
		})
		t.Run("fails for a codec that encodes a nil slice wrong", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[nilItems]{Fields: pageSpec.Fields}, propertyCheck, marshalCheck)
		})
		t.Run("fails for a codec that encodes a negative float wrong", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[negativeFloat]{Fields: floatsSpec.Fields}, propertyCheck, marshalCheck)
		})
	})
}
