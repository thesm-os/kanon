// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// Sources of the classify tests: a package sub, whose declarations the
// generated code of package m cannot name, an interface I, which C
// implements, and the import of sub.
const (
	subFile   = "sub/sub.go"
	subSource = "type hidden int32\n\n" +
		"// Alias denotes a type that sub does not export.\ntype Alias = hidden\n\n" +
		"// Open is a struct type with a field that sub does not export.\ntype Open = struct{ x int32 }\n\n" +
		"// Closed is an interface type with a method that sub does not export.\ntype Closed = interface{ m() }\n\n" +
		"// Box is a generic struct type.\ntype Box[T any] struct{ V T }\n"
	ifaceSource = "type I interface{ m() }\n\ntype C struct{}\n\nfunc (C) m() {}\n\n"
	subImport   = "import \"example.com/m/sub\"\n\n"
)

func TestClassify(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "returns an error for the option fixed on a type without a fixed-size integer",
				files: map[string]string{source: structA("X string `kanon:\",fixed\"`")},
				want:  "kanon: a.go:4:2: A.X: tag option fixed does not apply to type string",
			},
			{
				name:  "returns an error for the option types on a type without an interface",
				files: map[string]string{source: structA("X int32 `kanon:\",types=int32\"`")},
				want:  "kanon: a.go:4:2: A.X: tag option types does not apply to type int32",
			},
			{
				name:  "returns an error for a listed type that fits no interface of the field",
				files: map[string]string{source: ifaceSource + structA("S I `kanon:\",types=C|int32\"`")},
				want:  "kanon: a.go:10:2: A.S: the tag option types lists int32, which fits no interface of type example.com/m.I",
			},
			{
				name:  "returns an error for an interface without the option types",
				files: map[string]string{source: structA("S any")},
				want:  "kanon: a.go:4:2: A.S: interface type any needs the tag option types, which lists its concrete types",
			},
			{
				name:  "returns an error for a type that does not type-check",
				files: map[string]string{source: structA("X Missing")},
				want:  "kanon: a.go:4:2: A.X: the type does not type-check: go build reports why",
			},
			{
				name:  "returns an error for an unsafe pointer",
				files: map[string]string{source: "import \"unsafe\"\n\n" + structA("X unsafe.Pointer")},
				want:  "kanon: a.go:6:2: A.X: type unsafe.Pointer is not supported",
			},
			{
				name:  "returns an error for a function in a slice",
				files: map[string]string{source: structA("X []func()")},
				want:  "kanon: a.go:4:2: A.X: type func() is not supported",
			},
			{
				name:  "returns an error for a channel in a map",
				files: map[string]string{source: structA("X map[string]chan int")},
				want:  "kanon: a.go:4:2: A.X: type chan int is not supported",
			},
			{
				name: "returns an error for a kanon.Validator of a byte array with no bytes",
				files: map[string]string{
					source: "type E [0]byte\n\nfunc (E) ValidateKanon() error { return nil }\n\n" +
						structA("E []E"),
				},
				want: "kanon: a.go:8:2: A.E: ValidateKanon of type example.com/m.E has nothing to check: the zero " +
					"value is the only value of the type",
			},
			{
				name: "returns an error for a kanon.Validator of an array of arrays with no elements",
				files: map[string]string{
					source: "type E [2][0]int32\n\nfunc (E) ValidateKanon() error { return nil }\n\n" +
						structA("E []E"),
				},
				want: "kanon: a.go:8:2: A.E: ValidateKanon of type example.com/m.E has nothing to check: the zero " +
					"value is the only value of the type",
			},
		}
		// hiddenPart is the reason of the errors for the types that contain
		// sub.hidden, which sub.Alias denotes.
		const hiddenPart = "package sub does not export type example.com/m/sub.hidden"
		unnameable := []struct {
			name  string
			field string
			typ   string
			part  string
		}{
			{
				name: "returns an error for a type that another package does not export", field: "X sub.Alias",
				typ: "example.com/m/sub.hidden", part: hiddenPart,
			},
			{
				name: "returns an error for a slice of such a type", field: "X []sub.Alias",
				typ: "[]example.com/m/sub.Alias", part: hiddenPart,
			},
			{
				name: "returns an error for an array of pointers to such a type", field: "X [2]*sub.Alias",
				typ: "[2]*example.com/m/sub.Alias", part: hiddenPart,
			},
			{
				name: "returns an error for a map of such a type", field: "X map[string]sub.Alias",
				typ: "map[string]example.com/m/sub.Alias", part: hiddenPart,
			},
			{
				name: "returns an error for a function of such a type in a map", field: "X map[string]func(sub.Alias)",
				typ: "map[string]func(example.com/m/sub.Alias)", part: hiddenPart,
			},
			{
				name:  "returns an error for a function that returns such a type",
				field: "X map[string]func() sub.Alias",
				typ:   "map[string]func() example.com/m/sub.Alias", part: hiddenPart,
			},
			{
				name: "returns an error for a channel of such a type in a slice", field: "X []chan sub.Alias",
				typ: "[]chan example.com/m/sub.Alias", part: hiddenPart,
			},
			{
				name:  "returns an error for a struct type with a field that another package does not export",
				field: "X sub.Open", typ: "struct{x int32}",
				part: "package sub does not export field x of struct{x int32}",
			},
			{
				name:  "returns an error for a struct type with a field of such a type",
				field: "X struct{ Y sub.Alias }", typ: "struct{Y example.com/m/sub.Alias}", part: hiddenPart,
			},
			{
				name:  "returns an error for an interface type with a method that takes such a type",
				field: "X interface{ M(sub.Alias) } `kanon:\",types=int32\"`",
				typ:   "interface{M(example.com/m/sub.Alias)}", part: hiddenPart,
			},
			{
				name:  "returns an error for an interface type with a method that another package does not export",
				field: "X sub.Closed `kanon:\",types=int32\"`", typ: "interface{m()}",
				part: "package sub does not export method m of interface{m()}",
			},
			{
				name:  "returns an error for an interface type that embeds such an interface",
				field: "X interface{ sub.Closed } `kanon:\",types=int32\"`",
				typ:   "interface{example.com/m/sub.Closed}",
				part:  "package sub does not export method m of interface{m()}",
			},
			{
				name: "returns an error for an instantiation with such a type argument", field: "X sub.Box[sub.Alias]",
				typ: "example.com/m/sub.Box[example.com/m/sub.Alias]", part: hiddenPart,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the type fails")
			})
		}
		for _, tt := range unnameable {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files := map[string]string{subFile: subSource, source: subImport + structA(tt.field)}
				assert.Equal(t, generateError(t, files),
					"kanon: a.go:6:2: A.X: the generated code cannot name type "+tt.typ+": "+tt.part,
					"Generate states why the type fails")
			})
		}
	})
}
