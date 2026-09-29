// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/mapvalue"
	"go.thesmos.sh/kanon/internal/fixture/nested"
	"go.thesmos.sh/kanon/internal/fixture/number"
	"go.thesmos.sh/kanon/internal/fixture/pointer"
	"go.thesmos.sh/kanon/internal/fixture/slice"
	"go.thesmos.sh/kanon/internal/fixture/union"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// The field Children of view.Record, number 18, and the field Bool of
// mapvalue.Bools, number 1, each as a present field of length 0: the tag,
// whose wire format is a length, and the length.
const (
	emptyChildrenField = "\x92\x01\x00"
	emptyBoolField     = "\x0a\x00"
)

// emptyChildren is a view.Record whose MarshalBinary writes an empty slice
// of Children that is not nil as a present field.
type emptyChildren struct{ view.Record }

// MarshalBinary returns the encoding, and the field Children of length 0
// after it when Children is empty and not nil.
func (m *emptyChildren) MarshalBinary() ([]byte, error) {
	b, err := m.Record.MarshalBinary()
	if m.Children != nil && len(m.Children) == 0 {
		b = append(b, emptyChildrenField...)
	}
	return b, err
}

// emptyBool is a mapvalue.Bools whose MarshalBinary writes an empty map of
// Bool that is not nil as a present field.
type emptyBool struct{ mapvalue.Bools }

// MarshalBinary returns the encoding, and the field Bool of length 0 after
// it when Bool is empty and not nil.
func (m *emptyBool) MarshalBinary() ([]byte, error) {
	b, err := m.Bools.MarshalBinary()
	if m.Bool != nil && len(m.Bool) == 0 {
		b = append(b, emptyBoolField...)
	}
	return b, err
}

// pairEntries is the number of entries of a map that its put function
// sorts in a stack array. The put function sorts the keys of a map with
// more entries in a slice, in a loop of its own.
const pairEntries = 16

// wideOff is a mapvalue.Nested whose MarshalBinary drops the error of a
// value whose map Label has more than pairEntries entries.
type wideOff struct{ mapvalue.Nested }

// MarshalBinary returns the encoding, and no error when Label has more than
// pairEntries entries.
func (m *wideOff) MarshalBinary() ([]byte, error) {
	b, err := m.Nested.MarshalBinary()
	if len(m.Label) > pairEntries {
		return b, nil
	}
	return b, renamed(err, "Nested", "wideOff")
}

// nestedSpec describes mapvalue.Nested, whose maps have maps, slices,
// arrays, pointers and structs as values, and one of which contains
// itself.
var nestedSpec = kanontest.Spec[mapvalue.Nested]{
	Fields: fields("Slice", "Map", "Sets", "Array", "Lists", "Int32", "Inner", "Label", "Holders", "Graph"),
	Structs: []kanontest.Struct{
		{Type: reflect.TypeFor[mapvalue.Holder](), Name: "Holder", Fields: fields("Names", "Inner")},
	},
}

// member returns the field of a union member named name, with the number
// num, of the union whose discriminator is disc and whose value c selects
// the member.
func member(name string, num int, disc string, c any) kanontest.Field {
	return kanontest.Field{Name: name, Number: num, Union: disc, Case: c}
}

// signedSpec describes union.Signed, whose discriminator is a signed
// integer.
var signedSpec = kanontest.Spec[union.Signed]{Fields: []kanontest.Field{
	member("Low", 1, "Kind", union.SignedKindLow),
	member("High", 2, "Kind", union.SignedKindHigh),
}}

// layoutSpec describes union.Layout, whose two unions interleave with other
// fields and whose discriminators are unsigned integers.
var layoutSpec = kanontest.Spec[union.Layout]{Fields: []kanontest.Field{
	{Name: "Name", Number: 1},
	member("Circle", 2, "Shape", union.ShapeKindCircle),
	{Name: "Note", Number: 3},
	member("Square", 4, "Shape", union.ShapeKindSquare),
	member("Light", 5, "Tone", union.ToneKindLight),
	member("Dark", 6, "Tone", union.ToneKindDark),
	{Name: "Last", Number: 7},
}}

func TestSample(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a union of every kind of number", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[union.Numbers]{Fields: []kanontest.Field{
				member("Bool", 1, "Kind", union.NumberKindBool),
				member("Int", 2, "Kind", union.NumberKindInt),
				member("Int8", 3, "Kind", union.NumberKindInt8),
				member("Uint16", 4, "Kind", union.NumberKindUint16),
				member("Uint64", 5, "Kind", union.NumberKindUint64),
				{Name: "Fixed32", Number: 6, Fixed: true, Union: "Kind", Case: union.NumberKindFixed32},
				{Name: "Fixed64", Number: 7, Fixed: true, Union: "Kind", Case: union.NumberKindFixed64},
				member("Float32", 8, "Kind", union.NumberKindFloat32),
				member("Float64", 9, "Kind", union.NumberKindFloat64),
				member("Complex64", 10, "Kind", union.NumberKindComplex64),
				member("Complex128", 11, "Kind", union.NumberKindComplex128),
				member("Duration", 12, "Kind", union.NumberKindDuration),
			}})
		})
		t.Run("passes a union of pointers", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[union.Pointers]{Fields: []kanontest.Field{
				member("Int32", 1, "Kind", union.PointerKindInt32),
				member("String", 2, "Kind", union.PointerKindString),
				member("Twice", 3, "Kind", union.PointerKindTwice),
				member("Slice", 4, "Kind", union.PointerKindSlice),
				member("Empty", 5, "Kind", union.PointerKindEmpty),
			}})
		})
		t.Run("passes a union of structs and pointers to structs", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[union.Structs]{
				Fields: []kanontest.Field{
					member("Inner", 1, "Kind", union.StructKindInner),
					member("Label", 2, "Kind", union.StructKindLabel),
					member("Loose", 3, "Kind", union.StructKindLoose),
					member("Nothing", 4, "Kind", union.StructKindNothing),
					member("InnerPtr", 5, "Kind", union.StructKindInnerPtr),
					member("LabelPtr", 6, "Kind", union.StructKindLabelPtr),
					member("LoosePtr", 7, "Kind", union.StructKindLoosePtr),
				},
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[union.Loose](), Name: "Loose", Fields: fields("Note")},
					{Type: reflect.TypeFor[struct{}](), Name: "Structs.Nothing", Fields: []kanontest.Field{}},
				},
			})
		})
		t.Run("passes a union whose discriminator is a signed integer", func(t *testing.T) {
			t.Parallel()
			passes(t, signedSpec)
		})
		t.Run("passes a nested struct with a kanon codec and an inline struct of its own", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[nested.Compounds]{Fields: fields("One", "Many")})
		})
		t.Run("passes a nested struct with fields that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[number.Skips]{Fields: fields("Value", "Pointer")})
		})
		t.Run("passes two unions whose members interleave with other fields", func(t *testing.T) {
			t.Parallel()
			passes(t, layoutSpec)
		})
		t.Run("passes maps of maps and a map that contains itself", func(t *testing.T) {
			t.Parallel()
			passes(t, nestedSpec)
		})
		t.Run("passes chains of pointers and a pointer that points at itself", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[pointer.Chains]{
				Fields: fields("Twice", "TwiceInner", "TwiceSlice", "Thrice", "Loop"),
			}
			passes(t, spec)
		})
		t.Run("passes slices of slices and a slice that contains itself", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[slice.Nested]{Fields: fields(
				"Ints", "Words", "Rows", "Tables", "Stamps", "Voids", "Deep", "Tree", "Nones",
			)})
		})
		t.Run("fails for a codec that writes an empty slice that is not nil", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[emptyChildren]{Fields: recordSpec.Fields},
				"MarshalBinary/returns the reference encoding", "MarshalBinary returns the reference encoding")
		})
		t.Run("fails for a codec that writes an empty map that is not nil", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[emptyBool]{Fields: fields("Bool", "Flag")},
				"MarshalBinary/returns the reference encoding", "MarshalBinary returns the reference encoding")
		})
		t.Run("fails for a codec that drops the error of a map with more entries than a stack array sorts",
			func(t *testing.T) {
				t.Parallel()
				rejects(t, kanontest.Spec[wideOff]{Fields: nestedSpec.Fields, Structs: nestedSpec.Structs},
					marshalErrorCheck, "MarshalBinary returns the error of the value that fails to encode")
			})
	})
}
