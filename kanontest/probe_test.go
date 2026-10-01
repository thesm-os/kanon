// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/canonical"
	"go.thesmos.sh/kanon/internal/fixture/validate"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
	"go.thesmos.sh/kanon/wire"
)

// Names of the checks of the families of probes of a canonical Spec that the
// cases run.
const (
	canonicalPrefixCheck = "DecodeKanon/decodes each prefix of a field as the reference decode"
	longFormCheck        = "DecodeKanon/decodes a field with a varint one byte longer than its shortest form as the " +
		"reference decode"
	fieldOrderCheck   = "DecodeKanon/decodes an encoding with two adjacent fields swapped as the reference decode"
	keyOrderCheck     = "DecodeKanon/decodes an encoding with two adjacent map entries swapped as the reference decode"
	unknownFieldCheck = "DecodeKanon/decodes an encoding with unknown fields at a field boundary as the reference " +
		"decode"
	zeroFieldCheck = "DecodeKanon/decodes an encoding with a field written at its zero value as the reference decode"
	badTagCheck    = "DecodeKanon/decodes an encoding with a malformed tag at a field boundary as the reference decode"
	negZeroCheck   = "DecodeKanon/decodes an encoding with a float component of a map key of -0.0 as the reference " +
		"decode"
)

// Specs of canonical fixtures, whose directives set -canonical.
var (
	// mapsSpec describes canonical.Maps, whose two maps with string keys give
	// the probes a site of every breach but breachNegZero.
	mapsSpec = kanontest.Spec[canonical.Maps]{Fields: fields("Attrs", "Counts"), Canonical: true}
	// scoresSpec describes canonical.Keys, whose map with float64 keys gives
	// the probes the sites of breachNegZero.
	scoresSpec = kanontest.Spec[canonical.Keys]{Fields: fields("Scores", "Refs"), Canonical: true}
)

// lenientOff is a canonical.Maps whose DecodeKanon decodes input that is not
// the canonical encoding of its value without an error.
type lenientOff struct{ canonical.Maps }

// DecodeKanon decodes data, and drops an error that wraps
// kanon.ErrNotCanonical.
func (m *lenientOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Maps.DecodeKanon(data, opts), "Maps", "lenientOff"); !errors.Is(err, kanon.ErrNotCanonical) {
		return err
	}
	return nil
}

// signOff is a canonical.Keys whose DecodeKanon decodes input that is not the
// canonical encoding of its value without an error.
type signOff struct{ canonical.Keys }

// DecodeKanon decodes data, and drops an error that wraps
// kanon.ErrNotCanonical.
func (m *signOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Keys.DecodeKanon(data, opts), "Keys", "signOff"); !errors.Is(err, kanon.ErrNotCanonical) {
		return err
	}
	return nil
}

// Names of the checks of the families of probes that the cases run.
const (
	faultCheck  = "DecodeKanon/decodes a field between unknown fields with one value written wrong as the reference decode"
	rejectCheck = "DecodeKanon/decodes a field between unknown fields with a value that its ValidateKanon rejects as " +
		"the reference decode"
	changeCheck = "DecodeKanon/decodes a field between unknown fields with one changed byte as the reference decode"
	formatCheck = "DecodeKanon/decodes a field between unknown fields with its tag in another wire format as the " +
		"reference decode"
	repeatCheck = "DecodeKanon/decodes a field between unknown fields written twice as the reference decode"
	depthCheck  = "DecodeKanon/decodes a field between unknown fields under each depth limit as the reference decode"
	// errorCheck is the message of a check whose decode returns another
	// error than the reference decode.
	errorCheck = "DecodeKanon returns the error of the reference decode"
)

// nameTag is the tag of the field Name of view.Item, number 1, whose wire
// format is a length.
const nameTag = 1<<3 | wire.Bytes

// valuesSpec describes validate.Values, which has a field of a
// kanon.Validator in every place that a value can be.
var valuesSpec = kanontest.Spec[validate.Values]{
	Fields: []kanontest.Field{
		{Name: "Level", Number: 1},
		{Name: "Amount", Number: 2},
		{Name: "Tick", Number: 3},
		{Name: "Code", Number: 4},
		{Name: "Ratio", Number: 5},
		{Name: "Tags", Number: 6},
		{Name: "Scores", Number: 7},
		{Name: "Hash", Number: 8},
		{Name: "Blob", Number: 9},
		{Name: "Span", Number: 10},
		{Name: "Port", Number: 25},
		{Name: "Weight", Number: 12},
		{Name: "Wave", Number: 13},
		{Name: "Phase", Number: 14},
		{Name: "Grade", Number: 15},
		{Name: "Next", Number: 16},
		{Name: "Codes", Number: 17},
		{Name: "Pair", Number: 18},
		{Name: "Index", Number: 19},
		{Name: "Text", Number: 20, Union: "Kind", Case: validate.KindText},
		{Name: "Count", Number: 21, Union: "Kind", Case: validate.KindCount},
		{Name: "Flag", Number: 11, Union: "Kind", Case: validate.KindFlag},
		{Name: "Any", Number: 22, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[validate.Level](), Number: 1},
			{Type: reflect.TypeFor[validate.Tags](), Number: 2},
		}},
		{Name: "Checked", Number: 23, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[validate.Grade](), Number: 1},
		}},
		{Name: "Fixed", Number: 24, Fixed: true},
		{Name: "Era", Number: 26},
	},
}

// rejectOff is a validate.Values whose DecodeKanon decodes a Level that its
// ValidateKanon rejects without an error.
type rejectOff struct{ validate.Values }

// DecodeKanon decodes data, and drops an error that wraps validate.ErrLevel.
func (m *rejectOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Values.DecodeKanon(data, opts), "Values", "rejectOff"); !errors.Is(err, validate.ErrLevel) {
		return err
	}
	return nil
}

// onceOff is a view.Item whose DecodeKanon fails for a field Name that the
// encoding repeats, as the view of a struct fails for a repeated struct.
type onceOff struct{ view.Item }

// DecodeKanon returns the error of wire.FindOne for a second field Name, and
// decodes data otherwise.
func (m *onceOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if _, err := wire.FindOne(data, nameTag, "onceOff.Name"); errors.Is(err, kanon.ErrRepeatedView) {
		return err
	}
	return renamed(m.Item.DecodeKanon(data, opts), "Item", "onceOff")
}

// eofOff is a view.Item whose DecodeKanon decodes truncated input without
// an error.
type eofOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps
// io.ErrUnexpectedEOF.
func (m *eofOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "eofOff"); !errors.Is(err, io.ErrUnexpectedEOF) {
		return err
	}
	return nil
}

// rangeOff is a view.Item whose DecodeKanon decodes a value outside the
// range of its field without an error.
type rangeOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps kanon.ErrRange.
func (m *rangeOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "rangeOff"); !errors.Is(err, kanon.ErrRange) {
		return err
	}
	return nil
}

// malformedOff is a view.Item whose DecodeKanon decodes malformed input
// without an error.
type malformedOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps
// kanon.ErrMalformed.
func (m *malformedOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "malformedOff"); !errors.Is(err, kanon.ErrMalformed) {
		return err
	}
	return nil
}

// causeOff is a view.Item whose DecodeKanon returns an error that states its
// cause in its detail with kanon.ErrRange as the cause, and the message of
// the error of the decode.
type causeOff struct{ view.Item }

// DecodeKanon decodes data, and sets the cause of an error with a detail to
// kanon.ErrRange.
func (m *causeOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := renamed(m.Item.DecodeKanon(data, opts), "Item", "causeOff")
	if e, ok := errors.AsType[*kanon.DecodeError](err); ok && e.Detail != "" {
		e.Err = kanon.ErrRange
	}
	return err
}

// depthOff is a view.Record whose DecodeKanon ignores the depth limit of its
// options.
type depthOff struct{ view.Record }

// DecodeKanon decodes data with the default depth limit.
func (m *depthOff) DecodeKanon(data []byte, opts kanon.Options) error {
	opts.Depth = 0
	return renamed(m.Record.DecodeKanon(data, opts), "Record", "depthOff")
}

// slabOff is a view.Item whose DecodeKanon reads its slab one byte after the
// offset of its options.
type slabOff struct{ view.Item }

// DecodeKanon decodes data with the offset of a slab one byte too far.
func (m *slabOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if opts.Slab != "" {
		opts.Offset++
	}
	return m.Item.DecodeKanon(data, opts)
}

func TestProbe(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that decodes a cut encoding", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[eofOff]{Fields: itemSpec.Fields}, prefixCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a value outside its range", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[rangeOff]{Fields: itemSpec.Fields}, faultCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a tag of another wire format", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[malformedOff]{Fields: itemSpec.Fields}, changeCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a field whose tag names another wire format", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[malformedOff]{Fields: itemSpec.Fields}, formatCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a value that its ValidateKanon rejects", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[rejectOff]{Fields: valuesSpec.Fields}, rejectCheck, errorCheck)
		})
		t.Run("fails for a codec that returns an error for a field written twice", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[onceOff]{Fields: itemSpec.Fields}, repeatCheck, errorCheck)
		})
		t.Run("fails for a codec that returns an error with the same message and another cause", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[causeOff]{Fields: itemSpec.Fields}, changeCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes deeper than the limit", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[depthOff]{Fields: recordSpec.Fields}, depthCheck, errorCheck)
		})
		t.Run("fails for a codec that reads its slab at another offset", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[slabOff]{Fields: itemSpec.Fields}, slabCheck,
				"DecodeKanon decodes the value of the reference decode")
		})
		t.Run("returns the checks of the pieces of a canonical Spec without unknown fields", func(t *testing.T) {
			t.Parallel()
			holds(t, mapsSpec, canonicalPrefixCheck)
		})
		canonicalCases := []struct {
			name  string
			check string
		}{
			{
				name:  "fails for a canonical codec that decodes a varint longer than its shortest form",
				check: longFormCheck,
			},
			{name: "fails for a canonical codec that decodes two fields out of order", check: fieldOrderCheck},
			{name: "fails for a canonical codec that decodes two map entries out of order", check: keyOrderCheck},
			{name: "fails for a canonical codec that decodes an unknown field", check: unknownFieldCheck},
			{name: "fails for a canonical codec that decodes a field at its zero value", check: zeroFieldCheck},
			{
				name:  "fails for a canonical codec that decodes a tag of field number 0 in a longer form",
				check: badTagCheck,
			},
		}
		for _, tt := range canonicalCases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				rejects(t, kanontest.Spec[lenientOff]{Fields: mapsSpec.Fields, Canonical: true}, tt.check, errorCheck)
			})
		}
		t.Run("fails for a canonical codec that decodes a map key of -0.0", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[signOff]{Fields: scoresSpec.Fields, Canonical: true}, negZeroCheck, errorCheck)
		})
	})
}
