// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

func TestTag(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		numbers := []struct {
			name  string
			files map[string]string
			want  []string
		}{
			{
				name:  "leaves out a field tagged -",
				files: map[string]string{source: structA("X int32 `kanon:\"-\"`", "Y int32")},
				want:  []string{"//kanon:numbers A Y=1"},
			},
			{
				name:  "skips empty words",
				files: map[string]string{source: structA("X int32 `kanon:\",fixed,\"`")},
				want:  []string{"//kanon:numbers A X=1"},
			},
			{
				name: "reads a comma inside the brackets of a type as part of the type",
				files: map[string]string{source: "type P[K, V any] struct {\n\tKey K\n\tValue V\n}\n\n" +
					structA("S any `kanon:\",types=P[int32, string]\"`")},
				want: []string{
					"//kanon:numbers A S=1",
					"//kanon:numbers P Key=1 Value=2",
					"//kanon:numbers A.S \"P[int32, string]\"=1",
				},
			},
			{
				name: "reads a comma after the brackets of a type as a separator",
				files: map[string]string{source: "type P[K, V any] struct {\n\tKey K\n\tValue V\n}\n\n" +
					structA("S any `kanon:\",types=P[int32, string],3\"`")},
				want: []string{
					"//kanon:numbers A S=3",
					"//kanon:numbers P Key=1 Value=2",
					"//kanon:numbers A.S \"P[int32, string]\"=1",
				},
			},
		}
		for _, tt := range numbers {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, tt.files), source, "A")
				assert.NoError(t, err, "Generate reads the tags")
				assert.Equal(t, numbersLines(files[codeName]), tt.want, "the code file records the numbers")
			})
		}
		failures := []struct {
			name  string
			field string
			want  string
		}{
			{
				name:  "returns an error for the word unknown beside another word",
				field: "U []byte `kanon:\"unknown,1\"`",
				want:  "kanon: a.go:4:2: A.U: tag \"unknown,1\": the word unknown takes no other word",
			},
			{
				name:  "returns an error for a union that names no discriminator field",
				field: "X int32 `kanon:\"union=\"`",
				want:  "kanon: a.go:4:2: A.X: tag \"union=\": union names no discriminator field",
			},
			{
				name:  "returns an error for a second types option",
				field: "S any `kanon:\",types=int32,types=string\"`",
				want:  "kanon: a.go:4:2: A.S: tag \",types=int32,types=string\": second types option",
			},
			{
				name:  "returns an error for a types option with an empty type",
				field: "S any `kanon:\",types=int32|\"`",
				want:  "kanon: a.go:4:2: A.S: tag \",types=int32|\": types lists an empty type",
			},
			{
				name:  "returns an error for an unknown word",
				field: "X int32 `kanon:\"bogus\"`",
				want:  "kanon: a.go:4:2: A.X: tag \"bogus\": unknown word \"bogus\"",
			},
			{
				name:  "returns an error for a second field number",
				field: "X int32 `kanon:\"1,2\"`",
				want:  "kanon: a.go:4:2: A.X: tag \"1,2\": second field number 2",
			},
			{
				name:  "returns an error for field number 0",
				field: "X int32 `kanon:\"0\"`",
				want:  "kanon: a.go:4:2: A.X: tag \"0\": field number 0 outside 1 to 2147483647",
			},
			{
				name:  "returns an error for a field number above the largest",
				field: "X int32 `kanon:\"2147483648\"`",
				want:  "kanon: a.go:4:2: A.X: tag \"2147483648\": field number 2147483648 outside 1 to 2147483647",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, map[string]string{source: structA(tt.field)}), tt.want,
					"Generate states why the tag fails")
			})
		}
	})
}
