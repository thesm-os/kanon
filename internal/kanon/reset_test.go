// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

func TestReset(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the reset methods and functions that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				return d.method == resetMethod || d.op == opReset
			})
		})
		clears := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "sets an unexported unsafe pointer that the encoding leaves out to nil",
				files: map[string]string{source: "import \"unsafe\"\n\n" + structA("X int32", "p unsafe.Pointer")},
				want:  "\tm.p = nil\n",
			},
			{
				name: "zeroes a struct of another package whose field left out has a type that the code cannot name",
				files: map[string]string{
					"dep/dep.go": "// B has a field left out of an unexported type.\n" +
						"type B struct {\n\tX []int32\n\tY inner `kanon:\"-\"`\n}\n\n" +
						"// inner is unexported.\ntype inner struct{ n int32 }\n",
					source: "import \"example.com/m/dep\"\n\n" + structA("B dep.B"),
				},
				want: "\t*m = dep.B{X: m.X}\n",
			},
		}
		for _, tt := range clears {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, tt.files), source, "A")
				assert.NoError(t, err, "Generate encodes the struct type")
				assert.Contains(t, files[codeName], tt.want, "the reset clears the fields that the encoding leaves out")
			})
		}
	})
}
