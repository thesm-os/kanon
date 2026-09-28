// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// listA is the source of a struct type A whose field S stores an int32 or a
// string.
var listA = structA("S any `kanon:\",types=int32|string\"`")

func TestList(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		numbers := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "numbers the listed types in tag order",
				files: map[string]string{source: listA},
				want:  "//kanon:numbers A.S int32=1 string=2",
			},
			{
				name:  "keeps the type numbers that the code file records",
				files: map[string]string{source: listA, codeName: "//kanon:numbers A.S string=1 int32=2\n"},
				want:  "//kanon:numbers A.S int32=2 string=1",
			},
			{
				name:  "reserves the number of a type that the list no longer names",
				files: map[string]string{source: listA, codeName: "//kanon:numbers A.S int32=1 bool=2 string=3\n"},
				want:  "//kanon:numbers A.S int32=1 string=3 ~2",
			},
			{
				name:  "keeps the reserved type numbers",
				files: map[string]string{source: listA, codeName: "//kanon:numbers A.S int32=1 ~2\n"},
				want:  "//kanon:numbers A.S int32=1 string=3 ~2",
			},
			{
				name: "shares the list of a field among the instantiations of a generic struct",
				files: map[string]string{
					source: "type P[T any] struct {\n\tS any `kanon:\",types=int32\"`\n\tV T\n}\n\n" +
						structA("P1 P[int32]", "P2 P[string]"),
				},
				want: "//kanon:numbers P.S int32=1",
			},
			{
				name: "evaluates the listed types of a struct of another package in its package",
				files: map[string]string{
					"dep/dep.go": "// Local is a named type of the package.\ntype Local int32\n\n" +
						"// T lists a type of its package.\ntype T struct {\n\tS any `kanon:\",types=Local\"`\n}\n",
					source: "import \"example.com/m/dep\"\n\n" + structA("T dep.T"),
				},
				want: "//kanon:numbers example.com/m/dep.T.S example.com/m/dep.Local=1",
			},
		}
		for _, tt := range numbers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, tt.files), source, "A")
				assert.NoError(t, err, "Generate numbers the listed types")
				assert.Contains(t, numbersLines(files[codeName]), tt.want, "the code file records the type numbers")
			})
		}
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "returns an error for a numbers line that gives two types one number",
				files: map[string]string{source: listA, codeName: "//kanon:numbers A.S int32=1 string=1\n"},
				want:  "kanon: the numbers line for A.S gives int32 and string type number 1",
			},
			{
				name: "returns an error for two code files that record different type numbers",
				files: map[string]string{
					source:       listA,
					"b.kanon.go": "//kanon:numbers A.S int32=1 string=2\n",
					"c.kanon.go": "//kanon:numbers A.S int32=2 string=1\n",
				},
				want: "kanon: b.kanon.go and c.kanon.go record different numbers for A.S: " +
					"regenerate the one that does not generate A.S",
			},
			{
				name:  "returns an error for a listed name that the package does not declare",
				files: map[string]string{source: structA("S any `kanon:\",types=Missing\"`")},
				want:  "kanon: a.go:4:2: A.S: the tag option types lists Missing: undefined: Missing",
			},
			{
				name:  "returns an error for a listed expression that does not parse",
				files: map[string]string{source: structA("S any `kanon:\",types=[\"`")},
				want:  "kanon: a.go:4:2: A.S: the tag option types lists [, which does not parse as a type",
			},
			{
				name:  "returns an error for a listed expression that is not a type",
				files: map[string]string{source: structA("S any `kanon:\",types=len\"`")},
				want:  "kanon: a.go:4:2: A.S: the tag option types lists len, which is not a type",
			},
			{
				name:  "returns an error for a listed interface type",
				files: map[string]string{source: structA("S any `kanon:\",types=any\"`")},
				want:  "kanon: a.go:4:2: A.S: the tag option types lists interface type any, which no value has",
			},
			{
				name:  "returns an error for a type that the list names twice",
				files: map[string]string{source: structA("S any `kanon:\",types=int32|int32\"`")},
				want:  "kanon: a.go:4:2: A.S: the tag option types lists int32 twice",
			},
			{
				name:  "returns an error for a listed type that kanon does not encode",
				files: map[string]string{source: structA("S any `kanon:\",types=func()\"`")},
				want:  "kanon: a.go:4:2: A.S: type func() is not supported",
			},
			{
				name:  "returns an error for an interface that no listed type implements",
				files: map[string]string{source: ifaceSource + structA("S I `kanon:\",types=int32\"`")},
				want:  "kanon: a.go:10:2: A.S: the tag option types lists no concrete type of interface type example.com/m.I",
			},
			{
				name:  "returns an error for an interface key that no listed comparable type implements",
				files: map[string]string{source: structA("M map[any]string `kanon:\",types=[]int32\"`")},
				want: "kanon: a.go:4:2: A.M: the tag option types lists no comparable concrete type of interface " +
					"type any inside a map key",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the list fails")
			})
		}
	})
}
