// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// projectionKeys is the source of a code file whose struct A has maps with
// keys of a float, a pointer and L, a struct with a field that the encoding
// leaves out.
const projectionKeys = typeDirective +
	"type L struct {\n\tX int32\n\tN string `kanon:\"-\"`\n}\n\n" +
	"type A struct {\n\tFloats map[float64]string\n\tPtrs map[*int32]string\n\tLs map[L]string\n}\n"

func TestProjection(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			want string
		}{
			{
				name: "writes a float key of -0.0 as +0.0",
				want: "i = wire.PutUint64(buf, i, wire.KeyFloat64(pairs[k].Key))",
			},
			{
				name: "returns ErrInvalidKey from the encode of a float key with a NaN component",
				want: "return 0, wire.InvalidKeyError(loc, num)",
			},
			{
				name: "reports a pointer key as a key that no decode yields",
				want: "func _a_canonPtrInt32(x *int32) bool {\n\treturn false\n}",
			},
			{
				name: "reports a struct key as a key that a decode yields when its fields that the encoding leaves out " +
					"are zero",
				want: "func _a_canonL(x L) bool {\n\treturn x.N == \"\"\n}",
			},
			{
				name: "merges the keys of a decode into a map with keys that a decode does not yield",
				want: "keys, err = wire.MergeKeys(x, keys, _a_canonL, _a_compareL, loc, num, off)",
			},
		}
		files, err := generate(t, module(t, map[string]string{source: projectionKeys}), source, "A")
		assert.NoError(t, err, "Generate orders the keys")
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, files[codeName], tt.want, "the code file writes the projection of the keys")
			})
		}
		t.Run("writes no key check for keys of an array of no pointers", func(t *testing.T) {
			t.Parallel()
			src := typeDirective + structA("M map[[0]*int32]string")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate orders the keys")
			assert.NotContains(t, files[codeName], "wire.OneKey(",
				"the encode of the map does not check its keys, which are all equal")
		})
	})
}
