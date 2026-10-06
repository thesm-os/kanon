// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
)

// runtimeMod is the go.mod of a module whose path is the import path of the
// kanon runtime, so that its root package declares an Options type that the
// method sets of its structs name as the runtime's.
const runtimeMod = "module go.thesmos.sh/kanon\n\ngo 1.27\n"

// kanonMethods returns the methods of kanon.Message and CloneKanon on the
// struct type name, declared by hand, with the runtime imported as kanon.
func kanonMethods(name string) string {
	return "\nfunc (*" + name + ") SizeKanon() int { return 0 }\n" +
		"\nfunc (*" + name + ") EncodeKanon([]byte) (int, error) { return 0, nil }\n" +
		"\nfunc (*" + name + ") DecodeKanon([]byte, kanon.Options) error { return nil }\n" +
		"\nfunc (*" + name + ") MergeKanon([]byte, kanon.Options) error { return nil }\n" +
		"\nfunc (*" + name + ") Reset() {}\n" +
		"\nfunc (m *" + name + ") CloneKanon() *" + name + " { return m }\n\n"
}

func TestMethods(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("encodes a struct whose DecodeKanon has another signature as an inline struct", func(t *testing.T) {
			t.Parallel()
			src := "type T struct {\n\tX int32\n}\n\n" +
				"func (*T) SizeKanon() int { return 0 }\n\n" +
				"func (*T) EncodeKanon([]byte) (int, error) { return 0, nil }\n\n" +
				"func (*T) DecodeKanon([]byte) error { return nil }\n\n" +
				structA("T T")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate encodes the struct")
			assert.Contains(t, numbersLines(files[codeName]), "//kanon:numbers T X=1",
				"the code file numbers the fields of the inline struct")
		})
		t.Run("encodes a struct that declares the kanon methods through its methods", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{
				modName:      runtimeMod,
				"options.go": "// Options takes the place of the options of the runtime.\ntype Options struct{}\n",
				"sub/a.go": "import kanon \"go.thesmos.sh/kanon\"\n\ntype T struct {\n\tX int32\n}\n" +
					kanonMethods("T") + structA("T T"),
			})
			files, err := generate(t, filepath.Join(dir, "sub"), source, "A")
			assert.NoError(t, err, "Generate encodes the struct")
			assert.Equal(t, numbersLines(files[codeName]), []string{"//kanon:numbers A T=1"},
				"the code file numbers no field of the struct")
		})
		failures := []struct {
			name  string
			types string
			field string
			want  string
		}{
			{
				name:  "returns an error for a field type whose ValidateKanon has another signature",
				types: "type N int32\n\nfunc (N) ValidateKanon() bool { return true }\n\n",
				field: "F N",
				want: "kanon: a.go:8:2: A.F: example.com/m.N.ValidateKanon has the signature func() bool: declare " +
					"it as func() error",
			},
			{
				name:  "returns an error for a field type with ValidateKanon on a pointer receiver",
				types: "type N int32\n\nfunc (*N) ValidateKanon() error { return nil }\n\n",
				field: "F N",
				want: "kanon: a.go:8:2: A.F: example.com/m.N.ValidateKanon has a pointer receiver: declare it on a " +
					"value receiver",
			},
			{
				name:  "returns an error for a field of a struct type with ValidateKanon",
				types: "type S struct {\n\tX int32\n}\n\nfunc (S) ValidateKanon() error { return nil }\n\n",
				field: "S S",
				want: "kanon: a.go:10:2: A.S: struct type example.com/m.S declares ValidateKanon, which applies to " +
					"types that are not structs",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, map[string]string{source: tt.types + structA(tt.field)}), tt.want,
					"Generate states why the field fails")
			})
		}
	})
}
