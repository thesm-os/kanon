// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// codecMethods returns the methods through which the type name encodes
// itself: AppendBinary and UnmarshalBinary.
func codecMethods(name string) string {
	return "\nfunc (" + name + ") AppendBinary(b []byte) ([]byte, error) { return b, nil }\n" +
		"\nfunc (*" + name + ") UnmarshalBinary([]byte) error { return nil }\n\n"
}

func TestClone(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the clone methods and functions that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				return d.method == cloneKanon || d.method == cloneInner || d.op == opClone
			})
		})
		tests := []struct {
			name  string
			types string
			field string
			want  string
		}{
			{
				name:  "copies a type that encodes itself and refers to memory through its methods",
				types: "type Blob []byte\n" + codecMethods("Blob"),
				field: "B Blob",
				want:  "_a_cloneBlob(&c.B, &m.B)",
			},
			{
				name:  "copies an array type that encodes itself by assignment",
				types: "type Digest [4]byte\n" + codecMethods("Digest"),
				field: "D Digest",
				want:  "c.D = m.D",
			},
			{
				name:  "copies a struct type that encodes itself with one named type twice by assignment",
				types: "type Mark uint16\n\ntype Span struct {\n\tA, B Mark\n}\n" + codecMethods("Span"),
				field: "S Span",
				want:  "c.S = m.S",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				files, err := generate(
					t,
					module(t, map[string]string{source: tt.types + structA(tt.field)}),
					source,
					"A",
				)
				assert.NoError(t, err, "Generate encodes the type that encodes itself")
				assert.Contains(t, files[codeName], tt.want, "CloneKanon copies the field")
			})
		}
	})
}
