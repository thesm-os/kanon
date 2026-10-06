// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// numbersMark begins the line of a code file that records the numbers of a
// struct.
const numbersMark = "//kanon:numbers "

// Sources of the struct type A in the numbering tests.
const (
	// structXY declares the fields X and Y.
	structXY = "type A struct {\n\tX int32\n\tY int32\n}\n"
	// structXZ declares the fields X and Z.
	structXZ = "type A struct {\n\tX int32\n\tZ int32\n}\n"
)

// numbersLines returns the lines of the code file code that record numbers,
// in their order.
func numbersLines(code string) []string {
	var out []string
	for line := range strings.Lines(code) {
		if strings.HasPrefix(line, numbersMark) {
			out = append(out, strings.TrimSuffix(line, "\n"))
		}
	}
	return out
}

func TestNumbering(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		numbers := []struct {
			name  string
			files map[string]string
			types string
			want  []string
		}{
			{
				name:  "numbers the fields in declaration order",
				files: map[string]string{source: structXY},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2"},
			},
			{
				name:  "keeps the numbers that the code file records",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A Y=1 X=2\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=2 Y=1"},
			},
			{
				name: "keeps the numbers that the code file records under a later site of an anonymous struct type",
				files: map[string]string{
					source:   "type A struct {\n\tX struct{ P, Q int32 }\n\tY struct{ P, Q int32 }\n}\n",
					codeName: "//kanon:numbers A X=1 Y=2\n//kanon:numbers A.Y Q=1 P=2\n",
				},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2", "//kanon:numbers A.X P=2 Q=1"},
			},
			{
				name:  "reserves the number of a removed field",
				files: map[string]string{source: structXZ, codeName: "//kanon:numbers A X=1 W=2 Z=3\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Z=3 ~2"},
			},
			{
				name:  "gives a new field the smallest number that is neither taken nor reserved",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=1 ~2\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=3 ~2"},
			},
			{
				name: "gives a tag a reserved number",
				files: map[string]string{
					source:   "type A struct {\n\tX int32\n\tY int32 `kanon:\"2\"`\n}\n",
					codeName: "//kanon:numbers A X=1 ~2\n",
				},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2"},
			},
			{
				name:  "takes the numbers that another code file records for a struct",
				files: map[string]string{source: structXY, "b.kanon.go": "//kanon:numbers A X=2 Y=1\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=2 Y=1"},
			},
			{
				name: "keeps the numbers line of a struct that the file no longer generates",
				files: map[string]string{
					source:   structXY + "\ntype B struct {\n\tZ int32\n}\n",
					codeName: "//kanon:numbers A X=1 Y=2\n//kanon:numbers B Z=1\n",
				},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2", "//kanon:numbers B Z=1"},
			},
			{
				name: "drops the numbers line of a struct that the package no longer declares",
				files: map[string]string{
					source:   structXY,
					codeName: "//kanon:numbers A X=1 Y=2\n//kanon:numbers B Z=1\n",
				},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2"},
			},
			{
				name: "drops the numbers line of a struct that another code file records",
				files: map[string]string{
					source:       structXY + "\ntype B struct {\n\tZ int32\n}\n",
					codeName:     "//kanon:numbers A X=1 Y=2\n//kanon:numbers B Z=1\n",
					"b.kanon.go": "//kanon:numbers B Z=1\n",
				},
				types: "A",
				want:  []string{"//kanon:numbers A X=1 Y=2"},
			},
			{
				name:  "reads a quoted struct name",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers \"A\" X=2 Y=1\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=2 Y=1"},
			},
			{
				name:  "reads a quoted field name",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A \"X\"=2 Y=1\n"},
				types: "A",
				want:  []string{"//kanon:numbers A X=2 Y=1"},
			},
		}
		for _, tt := range numbers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := generate(t, module(t, tt.files), source, tt.types)
				assert.NoError(t, err, "Generate numbers the fields")
				assert.Equal(t, numbersLines(got[codeName]), tt.want, "the code file records the numbers")
			})
		}
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name: "returns an error for a tag that renumbers a recorded field",
				files: map[string]string{
					source:   "type A struct {\n\tX int32 `kanon:\"5\"`\n}\n",
					codeName: "//kanon:numbers A X=1\n",
				},
				want: "kanon: a.go:4:2: A.X: the tag numbers the field 5 but it is recorded as 1: " +
					"renumbering breaks encoded data",
			},
			{
				name: "returns an error for a tag that renumbers a recorded field of a generic struct type",
				files: map[string]string{
					source:   "type P[T any] struct {\n\tX T `kanon:\"5\"`\n}\n\n" + structA("S P[int32]"),
					codeName: "//kanon:numbers A S=1\n//kanon:numbers P X=1\n",
				},
				want: "kanon: a.go:4:2: P.X: the tag numbers the field 5 but it is recorded as 1: " +
					"renumbering breaks encoded data",
			},
			{
				name: "returns an error for two fields that take one number",
				files: map[string]string{
					source: "type A struct {\n\tX int32 `kanon:\"1\"`\n\tY int32 `kanon:\"1\"`\n}\n",
				},
				want: "kanon: a.go:5:2: A.Y: field number 1 is also field X",
			},
			{
				name: "returns an error for two code files that record different numbers",
				files: map[string]string{
					source:       structXY,
					"b.kanon.go": "//kanon:numbers A X=1 Y=2\n",
					"c.kanon.go": "//kanon:numbers A X=2 Y=1\n",
				},
				want: "kanon: b.kanon.go and c.kanon.go record different numbers for A: " +
					"regenerate the one that does not generate A",
			},
			{
				name:  "returns an error for a numbers line without a struct",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers\n"},
				want:  "kanon: a numbers line names no struct",
			},
			{
				name:  "returns an error for a numbers line with a quote that does not end",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers \"A X=1\n"},
				want: "kanon: the numbers line " + strconv.Quote(
					"//kanon:numbers \"A X=1",
				) + " has a quote that does not end",
			},
			{
				name:  "returns an error for a numbers line with a key that continues after its quote",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers \"A\"B X=1\n"},
				want:  "kanon: a numbers line has the malformed key " + strconv.Quote("\"A\"B"),
			},
			{
				name:  "returns an error for a struct that two numbers lines record",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=1\n//kanon:numbers A X=1\n"},
				want:  "kanon: the numbers line for A repeats",
			},
			{
				name:  "returns an error for a word without a number",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X\n"},
				want:  "kanon: the numbers line for A has the malformed word \"X\"",
			},
			{
				name:  "returns an error for a word without a name",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A =1\n"},
				want:  "kanon: the numbers line for A has the malformed word \"=1\"",
			},
			{
				name:  "returns an error for a quoted name without a number",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A \"X\"1\n"},
				want:  "kanon: the numbers line for A has the malformed word " + strconv.Quote("\"X\"1"),
			},
			{
				name:  "returns an error for a field that a numbers line repeats",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=1 X=2\n"},
				want:  "kanon: the numbers line for A repeats field X",
			},
			{
				name:  "returns an error for field number 0",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=0\n"},
				want:  "kanon: the numbers line for A has the malformed field number \"0\"",
			},
			{
				name:  "returns an error for a field number above the largest",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=2147483648\n"},
				want:  "kanon: the numbers line for A has the malformed field number \"2147483648\"",
			},
			{
				name:  "returns an error for a malformed reserved number",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=1 ~x\n"},
				want:  "kanon: the numbers line for A has the malformed field number \"x\"",
			},
			{
				name:  "returns an error for a record that gives two fields one number",
				files: map[string]string{source: structXY, codeName: "//kanon:numbers A X=1 Y=1\n"},
				want:  "kanon: a.go:5:2: A.Y: field number 1 is also field X",
			},
			{
				name: "returns an error for an inline struct whose fields take one number",
				files: map[string]string{
					source: "type B struct {\n\tX int32 `kanon:\"1\"`\n\tY int32 `kanon:\"1\"`\n}\n\n" +
						structA("B B"),
				},
				want: "kanon: a.go:5:2: B.Y: field number 1 is also field X",
			},
			{
				name: "returns an error for two code files that record different numbers for an inline struct",
				files: map[string]string{
					source:       "type B struct {\n\tX int32\n}\n\n" + structA("B B"),
					"b.kanon.go": "//kanon:numbers B X=1\n",
					"c.kanon.go": "//kanon:numbers B X=2\n",
				},
				want: "kanon: b.kanon.go and c.kanon.go record different numbers for B: " +
					"regenerate the one that does not generate B",
			},
			{
				name:  "returns an error for another code file that does not parse",
				files: map[string]string{source: structXY, "b.kanon.go": "//kanon:numbers\n"},
				want:  "kanon: a numbers line names no struct",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := generate(t, module(t, tt.files), source, "A")
				assert.HasError(t, err, "Generate fails for the numbering")
				assert.Equal(t, err.Error(), tt.want, "Generate states why the numbering fails")
			})
		}
		t.Run("returns an error for a code file that does not read", func(t *testing.T) {
			t.Parallel()
			dir := files.Workspace(t, files.Tree{
				modName:  files.Text(goMod),
				source:   files.Text(pkgClause + structXY),
				codeName: files.Dir(),
			})
			_, err := generate(t, dir, source, "A")
			assert.HasError(t, err, "Generate fails for a code file that does not read")
			assert.HasPrefix(t, err.Error(), "kanon: read the numbers of ", "Generate states why the numbering fails")
		})
	})
}
