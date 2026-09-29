// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"os"
	"path/filepath"
	"testing"

	"go.dokimi.dev/assert"
)

// keysDirective names the struct type A of the key tests and the types of
// its map keys, so that the keys have a kanon codec before their code file
// exists. It is on line 3 of a.go.
const keysDirective = "//go:generate go tool kanon -type=A,K,L\n\n"

func TestKey(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("records the field numbers of a key type that the code file does not generate", func(t *testing.T) {
			t.Parallel()
			src := keysDirective + "type K struct {\n\tY int32\n\tX int32\n}\n\n" + structA("M map[K]string")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate orders the keys")
			assert.Contains(t, files["a.kanon_test.go"], "{Name: \"Y\", Number: 1},\n\t\t\t\t{Name: \"X\", Number: 2},",
				"the Spec numbers the fields of the key type in declaration order")
		})
		t.Run("records the field numbers of a key type with a kanon codec that contains itself", func(t *testing.T) {
			t.Parallel()
			src := keysDirective + "type K struct {\n\tNext *K\n\tX    int32\n}\n\n" + structA("M map[K]string")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate orders the keys")
			assert.Contains(t, files["a.kanon_test.go"],
				"{Name: \"Next\", Number: 1},\n\t\t\t\t{Name: \"X\", Number: 2},",
				"the Spec numbers the fields of the key type in declaration order")
		})
		t.Run("returns an error for a key type with a kanon codec that is not a struct", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{
				"options.go": "// Options takes the place of the options of the runtime.\ntype Options struct{}\n",
				"sub/a.go": "import kanon \"go.thesmos.sh/kanon\"\n\ntype K int32\n" + kanonMethods("K") +
					structA("M map[K]string"),
			})
			assert.NoError(t, os.WriteFile(filepath.Join(dir, modName), []byte(runtimeMod), fileMode),
				"the go.mod of the runtime path writes")
			_, err := generate(t, filepath.Join(dir, "sub"), source, "A")
			assert.HasError(t, err, "Generate fails for the key type")
			assert.Equal(t, err.Error(), "kanon: a.go:20:2: A.M: map key type go.thesmos.sh/kanon/sub.K has kanon "+
				"methods and is not a struct: kanon cannot order it", "Generate states why the key fails")
		})
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name: "returns an error for a key type with a field that can contain an interface",
				files: map[string]string{source: keysDirective +
					"type K struct {\n\tS any `kanon:\",types=int32\"`\n}\n\n" + structA("M map[K]string")},
				want: "kanon: a.go:10:2: A.M: map key type example.com/m.K has field S, which can contain an " +
					"interface: kanon cannot order it",
			},
			{
				name: "returns an error for a key type with a union member",
				files: map[string]string{source: keysDirective + kindType + "const KX K = 1\n\n" +
					"type L struct {\n\tKind K\n\tX int32 `kanon:\",union=Kind\"`\n}\n\n" + structA("M map[L]string")},
				want: "kanon: a.go:15:2: A.M: map key type example.com/m.L has the union member X, whose encoding " +
					"depends on its discriminator: kanon cannot order it",
			},
			{
				name: "returns an error for a key type with a field that kanon does not encode",
				files: map[string]string{source: keysDirective +
					"type K struct {\n\tC [1]chan int\n}\n\n" + structA("M map[K]string")},
				want: "kanon: a.go:10:2: A.M: type chan int is not supported",
			},
			{
				name: "returns an error for a key that stores such a key type",
				files: map[string]string{source: keysDirective +
					"type K struct {\n\tC [1]chan int\n}\n\n" + structA("M map[any]string `kanon:\",types=K\"`")},
				want: "kanon: a.go:10:2: A.M: type chan int is not supported",
			},
			{
				name: "returns an error for a key type with a field of such a key type",
				files: map[string]string{source: keysDirective +
					"type K struct {\n\tIn L\n}\n\ntype L struct {\n\tC [1]chan int\n}\n\n" + structA("M map[K]string")},
				want: "kanon: a.go:14:2: A.M: type chan int is not supported",
			},
			{
				name: "returns an error for a key type whose fields take one number",
				files: map[string]string{source: keysDirective +
					"type K struct {\n\tX int32 `kanon:\"1\"`\n\tY int32 `kanon:\"1\"`\n}\n\n" +
					structA("M []map[K]string")},
				want: "kanon: a.go:7:2: K.Y: field number 1 is also field X",
			},
			{
				name: "returns an error for a key type of another package with an unexported field that a tag opts in",
				files: map[string]string{
					"dep/dep.go": "//go:generate go tool kanon -type=K\n\n// K is a key type.\n" +
						"type K struct {\n\tx int32 `kanon:\"\"`\n}\n",
					source: "import \"example.com/m/dep\"\n\n" + structA("M map[dep.K]string"),
				},
				want: "kanon: a.go:6:2: A.M: map key type example.com/m/dep.K has the unexported field x, which code " +
					"outside package dep cannot read: kanon cannot order it",
			},
			{
				name: "returns an error for a code file of the package of a key type that does not parse",
				files: map[string]string{
					"dep/dep.go": "//go:generate go tool kanon -type=K\n\n// K is a key type.\n" +
						"type K struct {\n\tX int32\n}\n",
					"dep/dep.kanon.go": "//kanon:numbers\n",
					source:             "import \"example.com/m/dep\"\n\n" + structA("M map[dep.K]string"),
				},
				want: "kanon: a numbers line names no struct",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the key fails")
			})
		}
	})
}
