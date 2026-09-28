// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"os"
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
				"options.go": "// Options stands in for the options of the runtime.\ntype Options struct{}\n",
				"sub/a.go": "import kanon \"go.thesmos.sh/kanon\"\n\ntype T struct {\n\tX int32\n}\n" +
					kanonMethods("T") + structA("T T"),
			})
			assert.NoError(t, os.WriteFile(filepath.Join(dir, modName), []byte(runtimeMod), fileMode),
				"the go.mod of the runtime path writes")
			files, err := generate(t, filepath.Join(dir, "sub"), source, "A")
			assert.NoError(t, err, "Generate encodes the struct")
			assert.Equal(t, numbersLines(files[codeName]), []string{"//kanon:numbers A T=1"},
				"the code file numbers no field of the struct")
		})
	})
}
