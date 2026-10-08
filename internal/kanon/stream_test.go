// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// streamA is the source of a struct type A with the streamed byte slice
// Data, which ends on line 5 of a.go.
const streamA = "type A struct {\n\tData []byte `kanon:\",stream\"`\n}\n"

// validatedBytes is the source of a named byte slice N that the kanon
// directive of its file names with -validate.
const validatedBytes = "//go:generate go tool kanon -type=N -validate=valid\n\n// N is bytes.\ntype N []byte\n\n" +
	"// valid returns nil.\nfunc (n N) valid() error { return nil }\n"

// unionK is the source of the discriminator type K with the constant KX,
// which selects the union member X, on lines 3 to 5 of a.go.
const unionK = "type K uint8\n\nconst KX K = 1\n\n"

func TestStream(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the stream decoder of a struct type with a streamed field", func(t *testing.T) {
			t.Parallel()
			got, err := generate(t, module(t, map[string]string{source: streamA}), source, "A")
			assert.NoError(t, err, "Generate generates the struct type")
			assert.Contains(t, got[codeName],
				"func NewAStream(r io.Reader, size int64, m *A, opts kanon.StreamOptions)",
				"the code file declares the constructor of the stream decoder")
			assert.Contains(t, got[codeName], "func (d *AStream) WriteTo(w io.Writer) (int64, error)",
				"the code file declares WriteTo for the streamed byte slice")
			assert.Contains(t, got[testName], "Stream: func(r io.Reader, size int64, m *A, opts kanon.StreamOptions)",
				"the test file names the constructor of the stream decoder in the Spec")
		})
		t.Run("writes no stream decoder for a struct type without a streamed field", func(t *testing.T) {
			t.Parallel()
			got, err := generate(t, module(t, map[string]string{source: structA("Data []byte")}), source, "A")
			assert.NoError(t, err, "Generate generates the struct type")
			assert.NotContains(t, got[codeName], "AStream", "the code file declares no stream decoder")
		})
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name: "returns an error for the tag option stream on a field of an inline struct",
				files: map[string]string{
					source: "type B struct {\n\tData []byte `kanon:\",stream\"`\n}\n\n" + structA("X B"),
				},
				want: "kanon: a.go:8:2: A.X: a.go:4:2: B.Data: the tag option stream applies to the fields of a struct " +
					"type that a -type flag names, and an inline struct has no stream decoder",
			},
			{
				name:  "returns an error for the tag option stream on an unexported field",
				files: map[string]string{source: structA("data []byte `kanon:\",stream\"`")},
				want:  "kanon: a.go:4:2: A.data: the tag option stream applies to an exported field",
			},
			{
				name:  "returns an error for the tag option stream on a union member",
				files: map[string]string{source: unionK + structA("Kind K", "X []byte `kanon:\",union=Kind,stream\"`")},
				want:  "kanon: a.go:9:2: A.X: a union member decodes whole: remove the tag option stream",
			},
			{
				name:  "returns an error for the tag option stream on an integer",
				files: map[string]string{source: structA("X int64 `kanon:\",stream\"`")},
				want: "kanon: a.go:4:2: A.X: the tag option stream applies to a byte slice, a string and a slice of a " +
					"struct type with a kanon codec, each of a type whose ValidateKanon the generated code does not call",
			},
			{
				name: "returns an error for the tag option stream on a slice of an inline struct",
				files: map[string]string{
					source: "type B struct {\n\tY int32\n}\n\n" + structA("X []B `kanon:\",stream\"`"),
				},
				want: "kanon: a.go:8:2: A.X: the tag option stream applies to a byte slice, a string and a slice of a " +
					"struct type with a kanon codec, each of a type whose ValidateKanon the generated code does not call",
			},
			{
				name:  "returns an error for the tag option stream on a byte slice whose ValidateKanon the code calls",
				files: map[string]string{"b.go": validatedBytes, source: structA("X N `kanon:\",stream\"`")},
				want: "kanon: a.go:4:2: A.X: the tag option stream applies to a byte slice, a string and a slice of a " +
					"struct type with a kanon codec, each of a type whose ValidateKanon the generated code does not call",
			},
			{
				name:  "returns an error for the word stream beside the word unknown",
				files: map[string]string{source: structA("U []byte `kanon:\"unknown,stream\"`")},
				want:  "kanon: a.go:4:2: A.U: tag \"unknown,stream\": the word unknown takes no other word",
			},
			{
				name:  "returns an error for a declaration named like the stream decoder",
				files: map[string]string{source: streamA + "\ntype AStream struct{}\n"},
				want:  "kanon: a.go:7:6: AStream: AStream is the name of a declaration of the generated code: rename it",
			},
			{
				name:  "returns an error for a declaration named like the type of the streamed fields",
				files: map[string]string{source: streamA + "\ntype AField int\n"},
				want:  "kanon: a.go:7:6: AField: AField is the name of a declaration of the generated code: rename it",
			},
			{
				name:  "returns an error for a declaration named like the constant of a streamed field",
				files: map[string]string{source: streamA + "\nconst AFieldData = 2\n"},
				want: "kanon: a.go:7:7: AFieldData: AFieldData is the name of a declaration of the generated code: " +
					"rename it",
			},
			{
				name:  "returns an error for a declaration named like the schema of the stream decoder",
				files: map[string]string{source: streamA + "\nvar _a_streamA int\n"},
				want:  "kanon: a.go:7:5: _a_streamA: _a_streamA is the name of a declaration of the generated code: rename it",
			},
			{
				name:  "returns an error for a declaration named like the constructor of the stream decoder",
				files: map[string]string{source: streamA + "\nfunc NewAStream() {}\n"},
				want: "kanon: a.go:7:6: NewAStream: NewAStream is the name of a declaration of the generated code: " +
					"rename it",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the stream fails")
			})
		}
	})
}
