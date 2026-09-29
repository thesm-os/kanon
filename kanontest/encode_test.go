// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"reflect"
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/external"
	"go.thesmos.sh/kanon/internal/fixture/inline"
	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/internal/fixture/nested"
	"go.thesmos.sh/kanon/internal/fixture/unknown"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// pointKey describes mapkey.Point as a map key, whose tags number Y before
// X, so that its keys order by Y first.
var pointKey = kanontest.Struct{
	Type:   reflect.TypeFor[mapkey.Point](),
	Fields: []kanontest.Field{{Name: "Y", Number: 1}, {Name: "X", Number: 2}},
}

// labelKey describes external.Label as a map key.
var labelKey = kanontest.Struct{
	Type:   reflect.TypeFor[external.Label](),
	Fields: fields("Key", "Value", "Rank", "Mark"),
}

// recordSpec describes view.Record, a struct of every scalar type, a time,
// nested structs, a type that encodes itself and pointers.
var recordSpec = kanontest.Spec[view.Record]{Fields: []kanontest.Field{
	{Name: "Flag", Number: 1},
	{Name: "Int8", Number: 2},
	{Name: "Level", Number: 3},
	{Name: "Uint16", Number: 4},
	{Name: "Fixed", Number: 5, Fixed: true},
	{Name: "Float32", Number: 6},
	{Name: "Float64", Number: 7},
	{Name: "Complex", Number: 8},
	{Name: "Wave", Number: 9},
	{Name: "Text", Number: 10},
	{Name: "Blob", Number: 11},
	{Name: "Digest", Number: 12},
	{Name: "At", Number: 13},
	{Name: "Item", Number: 14},
	{Name: "Label", Number: 15},
	{Name: "Ref", Number: 16},
	{Name: "ItemPtr", Number: 17},
	{Name: "Children", Number: 18},
	{Name: "Token", Number: 19},
	{Name: "TokenPtr", Number: 20},
}}

// fields returns the fields of a Spec named names, numbered from 1 in that
// order.
func fields(names ...string) []kanontest.Field {
	out := make([]kanontest.Field, 0, len(names))
	for k, name := range names {
		out = append(out, kanontest.Field{Name: name, Number: k + 1})
	}
	return out
}

func TestEncode(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a struct of every scalar type, a time, nested structs and pointers", func(t *testing.T) {
			t.Parallel()
			passes(t, recordSpec)
		})
		t.Run("passes a struct that keeps unknown fields", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[unknown.Record]{Fields: fields("ID", "Name", "Inner"), Unknown: "Rest"})
		})
		t.Run("passes a struct without fields", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[nested.Empty]{Fields: []kanontest.Field{}})
		})
		t.Run("passes a struct whose fields have one value", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[nested.Hollow]{
				Fields: fields("None", "Void"),
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[struct{}](), Name: "Hollow.Void", Fields: []kanontest.Field{}},
				},
			})
		})
		t.Run("passes instantiations of generic inline structs", func(t *testing.T) {
			t.Parallel()
			box := func(t reflect.Type, name string) kanontest.Struct {
				return kanontest.Struct{Type: t, Name: name, Fields: fields("Value", "Note")}
			}
			pair := func(t reflect.Type, name string) kanontest.Struct {
				return kanontest.Struct{Type: t, Name: name, Fields: fields("Key", "Val")}
			}
			passes(t, kanontest.Spec[inline.Generic]{
				Fields: fields("Box", "Boxes", "Pair", "Pairs", "Nested"),
				Structs: []kanontest.Struct{
					box(reflect.TypeFor[inline.Box[int32]](), "Box[int32]"),
					box(reflect.TypeFor[inline.Box[[]string]](), "Box[[]string]"),
					pair(reflect.TypeFor[external.Pair[string, int32]](), "external.Pair[string, int32]"),
					pair(reflect.TypeFor[external.Pair[external.Rank, inline.Box[string]]](),
						"external.Pair[external.Rank, Box[string]]"),
					box(reflect.TypeFor[inline.Box[string]](), "Box[string]"),
					box(reflect.TypeFor[inline.Box[inline.Box[bool]]](), "Box[Box[bool]]"),
					box(reflect.TypeFor[inline.Box[bool]](), "Box[bool]"),
				},
			})
		})
		t.Run("passes anonymous inline structs", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[inline.Anonymous]{
				Fields: fields("Meta", "Twin", "Rows", "Cells", "Tags"),
				Structs: []kanontest.Struct{
					{
						Type: reflect.TypeFor[struct {
							Host string
							Port uint16 "kanon:\"4\""
						}](),
						Name:   "Anonymous.Meta",
						Fields: []kanontest.Field{{Name: "Host", Number: 1}, {Name: "Port", Number: 4}},
					},
					{Type: reflect.TypeFor[struct{ A, B int32 }](), Name: "Anonymous.Rows[]", Fields: fields("A", "B")},
					{
						Type:   reflect.TypeFor[struct{ Row, Col int8 }](),
						Name:   "Anonymous.Cells[key]",
						Fields: fields("Row", "Col"),
					},
					{
						Type:   reflect.TypeFor[struct{ Value string }](),
						Name:   "Anonymous.Cells[value]",
						Fields: fields("Value"),
					},
					{Type: reflect.TypeFor[struct{}](), Name: "Anonymous.Tags[value]", Fields: []kanontest.Field{}},
				},
			})
		})
		t.Run("passes map keys of bools", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Bools]{Fields: fields("Bool", "Flag", "Flags")})
		})
		t.Run("passes map keys of signed integers", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Signed]{Fields: fields(
				"Int", "Int8", "Int16", "Int32", "Int64", "Tiny", "Small", "Level", "Amount", "Offset", "Duration",
			)})
		})
		t.Run("passes map keys of unsigned integers", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Unsigned]{Fields: fields(
				"Uint", "Uint8", "Uint16", "Uint32", "Uint64", "Uintptr", "Status", "Port", "Count", "Size", "Index",
				"Handle",
			)})
		})
		t.Run("passes map keys of floats", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Floats]{
				Fields: fields("Float32", "Float64", "Ratio", "Score", "Coord", "Coords", "Gauge", "Pointer"),
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[mapkey.Coord](), Name: "Coord", Fields: fields("Lat", "Lon", "Span")},
				},
				Keys: []kanontest.Struct{{Type: reflect.TypeFor[mapkey.Gauge](), Fields: fields("Phase", "Wave")}},
			})
		})
		t.Run("passes map keys of complex numbers", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Complexes]{Fields: fields("Complex64", "Complex128", "Phase", "Wave")})
		})
		t.Run("passes map keys of strings", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Text]{Fields: fields("String", "Name")})
		})
		t.Run("passes map keys of byte arrays", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.ByteArrays]{Fields: fields(
				"Byte0", "Byte1", "Byte32", "Byte127", "Byte128", "Digest",
			)})
		})
		t.Run("passes map keys of times", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Times]{
				Fields: fields("Time", "Moment"),
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[mapkey.Moment](), Name: "Moment", Fields: fields("At", "Seq")},
				},
			})
		})
		t.Run("passes map keys of arrays", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Arrays]{
				Fields: fields("Int32", "Point", "Float", "Time", "One", "Empty", "Void", "Bool", "Points", "NoPoint"),
				Keys:   []kanontest.Struct{pointKey},
			})
		})
		t.Run("passes map keys of pointers", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Pointers]{
				Fields: fields("Int32", "Point", "Empty", "Array", "Loop"),
				Keys:   []kanontest.Struct{pointKey},
			})
		})
		t.Run("passes map keys of structs", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[mapkey.Structs]{
				Fields: fields(
					"Point", "Label", "Spot", "Cell", "Pair", "Link", "Padded", "Nothing", "Unit", "Mute", "Chain",
				),
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[mapkey.Spot](), Name: "Spot", Fields: fields("Row", "Col")},
					{
						Type:   reflect.TypeFor[struct{ Row, Col int8 }](),
						Name:   "Structs.Cell[key]",
						Fields: fields("Row", "Col"),
					},
					{
						Type:   reflect.TypeFor[external.Pair[int32, string]](),
						Name:   "external.Pair[int32, string]",
						Fields: fields("Key", "Val"),
					},
					{Type: reflect.TypeFor[mapkey.Link](), Name: "Link", Fields: fields("N", "Next")},
					{Type: reflect.TypeFor[mapkey.Padded](), Name: "Padded", Fields: fields("N", "Pad")},
					{Type: reflect.TypeFor[struct{}](), Name: "Structs.Nothing[key]", Fields: []kanontest.Field{}},
					{Type: reflect.TypeFor[mapkey.Mute](), Name: "Mute", Fields: []kanontest.Field{}},
				},
				Keys: []kanontest.Struct{
					labelKey,
					{Type: reflect.TypeFor[mapkey.Chain](), Fields: fields("Next", "Ends")},
					pointKey,
				},
			})
		})
	})
}
