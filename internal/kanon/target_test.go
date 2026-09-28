// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// kindType declares the discriminator type K of the union tests, which
// starts on line 3 of a.go, before the struct type A.
const kindType = "type K uint8\n\n"

func TestTarget(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		numbers := []struct {
			name  string
			files map[string]string
			want  []string
		}{
			{
				name: "leaves the discriminator of a union out of the encoding",
				files: map[string]string{source: kindType + "const (\n\tKX K = 1\n\tKY K = 2\n)\n\n" +
					structA("Kind K", "X int32 `kanon:\",union=Kind\"`", "Y string `kanon:\",union=Kind\"`")},
				want: []string{"//kanon:numbers A X=1 Y=2"},
			},
			{
				name:  "leaves out an unexported field, a function and a channel",
				files: map[string]string{source: structA("x int32", "F func()", "C chan int", "Y int32")},
				want:  []string{"//kanon:numbers A Y=1"},
			},
			{
				name:  "leaves out a pointer to a function and a pointer to a pointer to a channel",
				files: map[string]string{source: structA("F *func()", "C **chan int", "Y int32")},
				want:  []string{"//kanon:numbers A Y=1"},
			},
			{
				name:  "encodes a field of a named pointer type that points at itself",
				files: map[string]string{source: "type L *L\n\n" + structA("P L", "Y int32")},
				want:  []string{"//kanon:numbers A P=1 Y=2"},
			},
			{
				name:  "encodes an unexported field with a kanon tag",
				files: map[string]string{source: structA("x int32 `kanon:\"\"`", "Y int32")},
				want:  []string{"//kanon:numbers A x=1 Y=2"},
			},
			{
				name:  "leaves out an unexported field tagged -",
				files: map[string]string{source: structA("x int32 `kanon:\"-\"`", "Y int32")},
				want:  []string{"//kanon:numbers A Y=1"},
			},
			{
				name:  "leaves out a blank field",
				files: map[string]string{source: structA("_ int32", "Y int32")},
				want:  []string{"//kanon:numbers A Y=1"},
			},
			{
				name:  "gives the field that keeps unknown fields no number",
				files: map[string]string{source: structA("X int32", "U []byte `kanon:\"unknown\"`")},
				want:  []string{"//kanon:numbers A X=1"},
			},
		}
		for _, tt := range numbers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, tt.files), source, "A")
				assert.NoError(t, err, "Generate encodes the struct type")
				assert.Equal(t, numbersLines(files[codeName]), tt.want, "the code file records the numbers")
			})
		}
		failures := []struct {
			name   string
			source string
			want   string
		}{
			{
				name:   "returns an error for a field tagged unknown that is not a byte slice",
				source: structA("U string `kanon:\"unknown\"`"),
				want:   "kanon: a.go:4:2: A.U: the field tagged unknown must be an exported []byte",
			},
			{
				name:   "returns an error for an unexported field tagged unknown",
				source: structA("u []byte `kanon:\"unknown\"`"),
				want:   "kanon: a.go:4:2: A.u: the field tagged unknown must be an exported []byte",
			},
			{
				name:   "returns an error for a second field tagged unknown",
				source: structA("U []byte `kanon:\"unknown\"`", "V []byte `kanon:\"unknown\"`"),
				want:   "kanon: a.go:5:2: A.V: field U keeps the unknown fields already",
			},
			{
				name:   "returns an error for a kanon tag on a function",
				source: structA("F func() `kanon:\"1\"`"),
				want: "kanon: a.go:4:2: A.F: the field has a kanon tag, and kanon does not encode blank fields, " +
					"functions, channels or pointers to them",
			},
			{
				name:   "returns an error for a kanon tag on a pointer to a channel",
				source: structA("C *chan int `kanon:\"1\"`"),
				want: "kanon: a.go:4:2: A.C: the field has a kanon tag, and kanon does not encode blank fields, " +
					"functions, channels or pointers to them",
			},
			{
				name:   "returns an error for a kanon tag on a blank field",
				source: structA("_ int32 `kanon:\"\"`"),
				want: "kanon: a.go:4:2: A._: the field has a kanon tag, and kanon does not encode blank fields, " +
					"functions, channels or pointers to them",
			},
			{
				name:   "returns an error for a union that names no field",
				source: structA("X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:4:2: A.X: union discriminator Kind is not an exported field of A",
			},
			{
				name:   "returns an error for a union that names an unexported field",
				source: kindType + structA("kind K", "X int32 `kanon:\",union=kind\"`"),
				want:   "kanon: a.go:7:2: A.X: union discriminator kind is not an exported field of A",
			},
			{
				name:   "returns an error for a discriminator of an unnamed type",
				source: structA("Kind int32", "X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:4:2: A.Kind: union discriminator type int32 is not a named integer type",
			},
			{
				name:   "returns an error for a discriminator of a named type that is no integer",
				source: "type K string\n\n" + structA("Kind K", "X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:6:2: A.Kind: union discriminator type example.com/m.K is not a named integer type",
			},
			{
				name:   "returns an error for a member without its constant",
				source: kindType + structA("Kind K", "X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:7:2: A.X: the union member needs the constant KX of type K",
			},
			{
				name:   "returns an error for an unexported member without the constant of its name in upper case",
				source: kindType + structA("Kind K", "text int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:7:2: A.text: the union member needs the constant KText of type K",
			},
			{
				name:   "returns an error for a member whose constant has another type",
				source: kindType + "const KX = 1\n\n" + structA("Kind K", "X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:9:2: A.X: the union member needs the constant KX of type K",
			},
			{
				name:   "returns an error for a member whose constant is zero",
				source: kindType + "const KX K = 0\n\n" + structA("Kind K", "X int32 `kanon:\",union=Kind\"`"),
				want:   "kanon: a.go:9:2: A.X: union constant KX is zero, which selects no member",
			},
			{
				name: "returns an error for two members with one constant",
				source: kindType + "const (\n\tKX K = 1\n\tKY K = 1\n)\n\n" +
					structA("Kind K", "X int32 `kanon:\",union=Kind\"`", "Y int32 `kanon:\",union=Kind\"`"),
				want: "kanon: a.go:13:2: A.Y: union constant KY equals the constant of member X",
			},
			{
				name: "returns an error for a discriminator with a kanon tag",
				source: kindType + "const KX K = 1\n\n" +
					structA("Kind K `kanon:\"1\"`", "X int32 `kanon:\",union=Kind\"`"),
				want: "kanon: a.go:8:2: A.Kind: a union discriminator is not encoded: remove its kanon tag",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, map[string]string{source: tt.source}), tt.want,
					"Generate states why the struct type fails")
			})
		}
		t.Run("returns an error for an unexported field that a tag opts in within a struct of another package",
			func(t *testing.T) {
				t.Parallel()
				files := map[string]string{
					"dep/dep.go": "// B has an unexported field that a tag opts in.\n" +
						"type B struct {\n\tx int32 `kanon:\"\"`\n}\n",
					source: "import \"example.com/m/dep\"\n\n" + structA("B dep.B"),
				}
				assert.Equal(t, generateError(t, files), "kanon: a.go:6:2: A.B: dep.go:5:1: dep.B.x: the field is "+
					"unexported, and code outside package dep cannot reach it: generate a kanon codec for dep.B in its "+
					"package",
					"Generate states why the struct type fails")
			})
	})
}
