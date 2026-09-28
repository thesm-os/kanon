// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

// sliceA is the source of a struct type A with a field of a slice type,
// whose code file declares the helper _a_sizeSliceInt32.
const sliceA = "type A struct {\n\tX []int32\n}\n\n"

func TestCode(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the version checks and the interface assertions that go generate wrote for every fixture",
			func(t *testing.T) {
				t.Parallel()
				goldenDeclarations(t, codeSuffix, func(d declaration) bool {
					return strings.HasPrefix(d.key, "const ") || strings.HasPrefix(d.key, "var ")
				})
			})
		failures := []struct {
			name   string
			source string
			want   string
		}{
			{
				name:   "returns an error for a declaration named like a helper",
				source: sliceA + "func _a_sizeSliceInt32() {}\n",
				want: "kanon: a.go:7:6: _a_sizeSliceInt32: _a_sizeSliceInt32 is the name of a declaration of the " +
					"generated code: rename it",
			},
			{
				name:   "returns an error for a declaration named like a predeclared identifier",
				source: sliceA + "func len() {}\n",
				want:   "kanon: a.go:7:6: len: len is a predeclared identifier that the generated code uses: rename it",
			},
			{
				name:   "returns an error for a type that the code names like a local variable",
				source: "type data int32\n\ntype A struct {\n\tX data\n}\n",
				want:   "kanon: a.go:3:6: data: data is the name of a local variable of the generated code: rename it",
			},
			{
				name:   "returns an error for a method that kanon generates",
				source: sliceA + "func (A) Reset() {}\n",
				want:   "kanon: a.go:7:10: A.Reset: A declares Reset, which kanon generates",
			},
			{
				name:   "returns an error for a field named like a method that kanon generates",
				source: structA("X int32", "SizeKanon int32 `kanon:\"-\"`"),
				want:   "kanon: a.go:5:2: A.SizeKanon: A declares SizeKanon, which kanon generates",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, map[string]string{source: tt.source}), tt.want,
					"Generate states the name that the code file cannot use")
			})
		}
		t.Run("returns an error for a declaration named like a view type", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{source: structA("X int32") + "\ntype AView []byte\n"})
			_, err := kanon.Generate(dir, source, kanon.Options{Types: []string{"A"}, Views: true})
			assert.HasError(t, err, "Generate fails for the view type")
			assert.Equal(t, err.Error(),
				"kanon: a.go:7:6: AView: AView is the name of a declaration of the generated code: rename it",
				"Generate states the name that the code file cannot use")
		})
	})
}
