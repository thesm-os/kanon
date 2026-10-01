// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"os"
	"reflect"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
	"go.thesmos.sh/kanon/internal/fixture/iface"
	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/internal/fixture/nested"
	"go.thesmos.sh/kanon/internal/fixture/number"
	"go.thesmos.sh/kanon/internal/fixture/union"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// specCheck names the check of a Spec that does not describe its struct
// type.
const specCheck = "Spec/describes the fields of the struct type"

// Golden files of the specs of nested.WithDeclared and nested.WithReordered,
// which the check that goldenCheck names compares and the -update flag
// writes.
const (
	declaredGolden  = "testdata/golden/nested.WithDeclared.kanon.golden"
	reorderedGolden = "testdata/golden/nested.WithReordered.kanon.golden"
	// goldenCheck names the check of the golden file of a Spec.
	goldenCheck = "MarshalBinary/matches the golden file of the samples and the probes"
)

// placesSpec describes iface.Places, which has an interface in every place
// that a value can be.
var placesSpec = kanontest.Spec[iface.Places]{
	Fields: []kanontest.Field{
		{Name: "Field", Number: 1, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Square](), Number: 2},
			{Type: reflect.TypeFor[iface.Patch](), Number: 3},
		}},
		{Name: "Slice", Number: 2, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Square](), Number: 2},
		}},
		{Name: "Array", Number: 3, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Patch](), Number: 2},
		}},
		{Name: "Map", Number: 4, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Square](), Number: 2},
		}},
		{Name: "Ptr", Number: 5, Types: []kanontest.ConcreteType{{Type: reflect.TypeFor[*iface.Square](), Number: 1}}},
		{Name: "Twice", Number: 6, Types: []kanontest.ConcreteType{{Type: reflect.TypeFor[iface.Circle](), Number: 1}}},
		{Name: "Nested", Number: 7},
	},
	Structs: []kanontest.Struct{
		{Type: reflect.TypeFor[iface.Patch](), Name: "Patch", Fields: []kanontest.Field{{Name: "Cells", Number: 1}}},
		{
			Type: reflect.TypeFor[struct {
				Shape iface.Shape "kanon:\",types=Circle\""
			}](),
			Name: "Places.Nested",
			Fields: []kanontest.Field{
				{
					Name:   "Shape",
					Number: 1,
					Types:  []kanontest.ConcreteType{{Type: reflect.TypeFor[iface.Circle](), Number: 1}},
				},
			},
		},
	},
}

// keysSpec describes iface.Keys, which has maps with interface keys.
var keysSpec = kanontest.Spec[iface.Keys]{
	Fields: []kanontest.Field{
		{Name: "Keys", Number: 1, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[string](), Number: 1},
			{Type: reflect.TypeFor[int32](), Number: 2},
			{Type: reflect.TypeFor[[2]any](), Number: 3},
			{Type: reflect.TypeFor[iface.Circle](), Number: 4},
			{Type: reflect.TypeFor[[]int32](), Number: 5},
		}},
		{Name: "Shapes", Number: 2, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Square](), Number: 2},
		}},
		{Name: "Units", Number: 3, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[struct{}](), Number: 1},
			{Type: reflect.TypeFor[[0]int32](), Number: 2},
		}},
		{Name: "Ints", Number: 4, Types: []kanontest.ConcreteType{{Type: reflect.TypeFor[int32](), Number: 1}}},
		{Name: "Moments", Number: 5, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[time.Time](), Number: 1},
			{Type: reflect.TypeFor[int32](), Number: 2},
		}},
		{Name: "Epoch", Number: 6},
	},
	Structs: []kanontest.Struct{
		{Type: reflect.TypeFor[struct{}](), Name: "Keys.Units[key].(struct{})", Fields: []kanontest.Field{}},
	},
	Keys: []kanontest.Struct{
		{Type: reflect.TypeFor[iface.Circle](), Fields: []kanontest.Field{{Name: "Radius", Number: 1}}},
		{
			Type:   reflect.TypeFor[iface.Square](),
			Fields: []kanontest.Field{{Name: "Side", Number: 1}, {Name: "Label", Number: 2}},
		},
	},
}

// privateSpec describes number.Private, whose unexported fields a kanon tag
// opts into the encoding in every role of a field, among them a map whose key
// struct has an unexported field of its own.
var privateSpec = kanontest.Spec[number.Private]{
	Fields: []kanontest.Field{
		{Name: "Name", Number: 1},
		{Name: "count", Number: 2},
		{Name: "total", Number: 3, Fixed: true},
		{Name: "tags", Number: 4},
		{Name: "byKey", Number: 5},
		{Name: "next", Number: 6},
		{Name: "detail", Number: 7},
		member("text", 8, "Kind", number.PrivateKindText),
		member("number", 9, "Kind", number.PrivateKindNumber),
	},
	Structs: []kanontest.Struct{
		{Type: fieldType[number.Private]("detail"), Name: "privateDetail", Fields: fields("label", "Rank")},
	},
	Keys: []kanontest.Struct{{Type: reflect.TypeFor[number.PrivateKey](), Fields: fields("id", "Name")}},
}

// fieldType returns the type of the field of the struct type T named name,
// for a type that a test outside the package of T cannot name, such as an
// unexported struct type, and nil when T has no such field.
func fieldType[T any](name string) reflect.Type {
	f, _ := reflect.TypeFor[T]().FieldByName(name)
	return f.Type
}

// treesSpec describes iface.Trees, whose interface stores slices and maps
// of itself.
var treesSpec = kanontest.Spec[iface.Trees]{
	Fields: []kanontest.Field{
		{Name: "Tree", Number: 1, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[string](), Number: 1},
			{Type: reflect.TypeFor[int64](), Number: 2},
			{Type: reflect.TypeFor[float64](), Number: 3},
			{Type: reflect.TypeFor[bool](), Number: 4},
			{Type: reflect.TypeFor[[]any](), Number: 5},
			{Type: reflect.TypeFor[map[string]any](), Number: 6},
		}},
	},
}

// variantsSpec describes iface.Variants, whose interfaces store a type of
// every family.
var variantsSpec = kanontest.Spec[iface.Variants]{
	Fields: []kanontest.Field{
		{Name: "Numbers", Number: 1, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[bool](), Number: 1},
			{Type: reflect.TypeFor[int8](), Number: 2},
			{Type: reflect.TypeFor[int](), Number: 3},
			{Type: reflect.TypeFor[uint16](), Number: 4},
			{Type: reflect.TypeFor[uint64](), Number: 5},
			{Type: reflect.TypeFor[uintptr](), Number: 6},
			{Type: reflect.TypeFor[float32](), Number: 7},
			{Type: reflect.TypeFor[float64](), Number: 8},
			{Type: reflect.TypeFor[complex64](), Number: 9},
			{Type: reflect.TypeFor[complex128](), Number: 10},
			{Type: reflect.TypeFor[iface.Level](), Number: 11},
		}},
		{Name: "Fixed", Number: 2, Fixed: true, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[int32](), Number: 1},
			{Type: reflect.TypeFor[uint64](), Number: 2},
		}},
		{Name: "Text", Number: 3, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[string](), Number: 1},
			{Type: reflect.TypeFor[[]byte](), Number: 2},
			{Type: reflect.TypeFor[iface.Name](), Number: 3},
			{Type: reflect.TypeFor[[4]byte](), Number: 4},
			{Type: reflect.TypeFor[[0]byte](), Number: 5},
		}},
		{Name: "Times", Number: 4, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[time.Time](), Number: 1},
			{Type: reflect.TypeFor[time.Duration](), Number: 2},
		}},
		{Name: "Remote", Number: 5, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[external.Rank](), Number: 1},
			{Type: reflect.TypeFor[external.Ident](), Number: 2},
			{Type: reflect.TypeFor[external.Label](), Number: 3},
		}},
		{Name: "Structs", Number: 6, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[iface.Circle](), Number: 1},
			{Type: reflect.TypeFor[*iface.Square](), Number: 2},
			{Type: reflect.TypeFor[iface.Patch](), Number: 3},
			{Type: reflect.TypeFor[struct{}](), Number: 4},
			{Type: reflect.TypeFor[external.Pair[string, int8]](), Number: 5},
		}},
		{Name: "Codecs", Number: 7, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[codec.Token](), Number: 1},
			{Type: reflect.TypeFor[codec.Word](), Number: 2},
			{Type: reflect.TypeFor[*codec.Stamp](), Number: 3},
			{Type: reflect.TypeFor[external.Code](), Number: 4},
		}},
		{Name: "Nested", Number: 8, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[[]int32](), Number: 1},
			{Type: reflect.TypeFor[map[string]int32](), Number: 2},
			{Type: reflect.TypeFor[iface.Grid](), Number: 3},
			{Type: reflect.TypeFor[*int32](), Number: 4},
			{Type: reflect.TypeFor[**int32](), Number: 5},
			{Type: reflect.TypeFor[[]*iface.Circle](), Number: 6},
			{Type: reflect.TypeFor[[]*iface.Square](), Number: 7},
			{Type: reflect.TypeFor[map[string]*iface.Square](), Number: 8},
			{Type: reflect.TypeFor[[]iface.Square](), Number: 9},
			{Type: reflect.TypeFor[map[string]iface.Square](), Number: 10},
		}},
		{Name: "Deadline", Number: 9},
		{Name: "Token", Number: 10},
		{Name: "Rank", Number: 11},
		{Name: "Pointers", Number: 12, Types: []kanontest.ConcreteType{
			{Type: reflect.TypeFor[*int32](), Number: 1},
			{Type: reflect.TypeFor[**int32](), Number: 2},
		}},
	},
	Structs: []kanontest.Struct{
		{Type: reflect.TypeFor[iface.Patch](), Name: "Patch", Fields: []kanontest.Field{{Name: "Cells", Number: 1}}},
		{Type: reflect.TypeFor[struct{}](), Name: "Variants.Structs.(struct{})", Fields: []kanontest.Field{}},
		{
			Type:   reflect.TypeFor[external.Pair[string, int8]](),
			Name:   "external.Pair[string, int8]",
			Fields: []kanontest.Field{{Name: "Key", Number: 1}, {Name: "Val", Number: 2}},
		},
	},
}

func TestShape(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes interfaces in every place of a value", func(t *testing.T) {
			t.Parallel()
			passes(t, placesSpec)
		})
		t.Run("passes interfaces as map keys", func(t *testing.T) {
			t.Parallel()
			passes(t, keysSpec)
		})
		t.Run("passes an interface whose concrete types contain it", func(t *testing.T) {
			t.Parallel()
			passes(t, treesSpec)
		})
		t.Run("passes interfaces that store a type of every family", func(t *testing.T) {
			t.Parallel()
			passes(t, variantsSpec)
		})
		t.Run("passes unexported fields that a kanon tag opts into the encoding", func(t *testing.T) {
			t.Parallel()
			passes(t, privateSpec)
		})
		t.Run("fills a struct with a kanon codec alike in either order of its field declarations", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[nested.WithDeclared]{Fields: fields("In")}, goldenCheck)
			holds(t, kanontest.Spec[nested.WithReordered]{Fields: fields("In")}, goldenCheck)
			declared, err := os.ReadFile(declaredGolden)
			assert.NoError(t, err, "the golden file of WithDeclared reads")
			reordered, err := os.ReadFile(reorderedGolden)
			assert.NoError(t, err, "the golden file of WithReordered reads")
			assert.Equal(t, string(reordered), string(declared),
				"the samples of WithReordered encode as the samples of WithDeclared")
		})
		t.Run("fails for a field that the struct type does not declare", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: []kanontest.Field{{Name: "Missing", Number: 1}}}, specCheck,
				"the Spec names field Missing, which view.Item does not declare")
		})
		t.Run("fails for a discriminator that the struct type does not declare", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[union.Numbers]{Fields: []kanontest.Field{
				{Name: "Bool", Number: 1, Union: "Missing", Case: union.NumberKindBool},
			}}
			rejects(t, spec, specCheck, "the Spec names discriminator Missing, which union.Numbers does not declare")
		})
		t.Run("fails for an unknown field that the struct type does not declare", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, Unknown: "Missing"}, specCheck,
				"the Spec names unknown field Missing, which view.Item does not declare")
		})
		t.Run("fails for a discriminator that the struct type does not export", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[number.Private]{Fields: []kanontest.Field{
				{Name: "Name", Number: 1, Union: "secret", Case: number.PrivateKindText},
			}}
			rejects(t, spec, specCheck, "the Spec names discriminator secret, which number.Private does not export")
		})
		t.Run("fails for an unknown field that the struct type does not export", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[number.Private]{
				Fields:  []kanontest.Field{{Name: "Name", Number: 1}},
				Unknown: "secret",
			}
			rejects(t, spec, specCheck, "the Spec names unknown field secret, which number.Private does not export")
		})
		t.Run("fails for a Struct without a type", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, Structs: []kanontest.Struct{{}}}, specCheck,
				"the Spec describes <nil>, which is not a struct type")
		})
		t.Run("fails for a Struct of a type that is not a struct", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[view.Item]{
				Fields:  itemSpec.Fields,
				Structs: []kanontest.Struct{{Type: reflect.TypeFor[int]()}},
			}
			rejects(t, spec, specCheck, "the Spec describes int, which is not a struct type")
		})
		t.Run("fails for a field of a type that kanon does not encode", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[number.Skipped]{Fields: []kanontest.Field{{Name: "Hook", Number: 1}}}, specCheck,
				"field Hook of number.Skipped: type func() is not encoded")
		})
		t.Run("fails for an inline struct that no Struct describes", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[mapkey.Structs]{Fields: []kanontest.Field{{Name: "Spot", Number: 3}}}
			rejects(t, spec, specCheck, "struct type mapkey.Spot has no kanon codec and no Struct in the Spec")
		})
		t.Run("fails for a field of a Struct that its type does not declare", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[mapkey.Structs]{
				Fields: []kanontest.Field{{Name: "Spot", Number: 3}},
				Structs: []kanontest.Struct{
					{Type: reflect.TypeFor[mapkey.Spot](), Fields: []kanontest.Field{{Name: "Missing"}}},
				},
			}
			rejects(t, spec, specCheck, "the Spec names field Missing, which mapkey.Spot does not declare")
		})
		t.Run("fails for an interface that no concrete type of its field fits", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[iface.Places]{Fields: []kanontest.Field{{Name: "Field", Number: 1}}}, specCheck,
				"the Types of the field list no concrete type of interface type iface.Shape")
		})
		t.Run("fails for the value of a map that no concrete type of its field fits", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[iface.Places]{Fields: []kanontest.Field{{Name: "Map", Number: 4}}}, specCheck,
				"the Types of the field list no concrete type of interface type iface.Shape")
		})
		t.Run("fails for a concrete type of an interface that kanon does not encode", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[iface.Trees]{Fields: []kanontest.Field{
				{
					Name:   "Tree",
					Number: 1,
					Types:  []kanontest.ConcreteType{{Type: reflect.TypeFor[func()](), Number: 1}},
				},
			}}
			rejects(t, spec, specCheck, "field Tree of iface.Trees: type func() is not encoded")
		})
	})
}
