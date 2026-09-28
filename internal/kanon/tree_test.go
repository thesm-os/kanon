// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// typeDirective names the struct type A of a code file, which the tree
// tests generate.
const typeDirective = "//go:generate go tool kanon -type=A\n\n"

// leftOutArrayKey returns the source of a code file whose struct A has a map
// with keys of K, a struct with an array of n elements of L, a struct with a
// field that the encoding leaves out.
func leftOutArrayKey(n string) string {
	return typeDirective +
		"type L struct {\n\tX int32\n\tN string `kanon:\"-\"`\n}\n\n" +
		"type K struct {\n\tA [" + n + "]L\n}\n\n" + structA("M map[K]string")
}

func TestTree(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run(
			"encodes a nested struct with an unexported field as a struct of more than one value",
			func(t *testing.T) {
				t.Parallel()
				src := "//go:generate go tool kanon -type=A,B\n\n" +
					"type B struct {\n\tx int32\n\tY int32\n}\n\n" + structA("B B")
				files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
				assert.NoError(t, err, "Generate encodes the nested struct")
				assert.Contains(
					t,
					files[codeName],
					"if s := m.B.SizeKanon(); s > 0 {",
					"SizeKanon sizes the nested struct",
				)
			},
		)
		t.Run(
			"writes the key check of a struct key with an array of structs with a field that the encoding leaves out",
			func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, map[string]string{source: leftOutArrayKey("1")}), source, "A")
				assert.NoError(t, err, "Generate orders the keys")
				assert.Contains(t, files[codeName], "err := wire.PairTies(pairs, _a_compareK, loc, num)",
					"the encode of the map checks for two keys of one projection")
			},
		)
		t.Run(
			"writes no key check for a struct key with an array of no structs with a field that the encoding leaves out",
			func(t *testing.T) {
				t.Parallel()
				files, err := generate(t, module(t, map[string]string{source: leftOutArrayKey("0")}), source, "A")
				assert.NoError(t, err, "Generate orders the keys")
				assert.NotContains(t, files[codeName], "wire.OneKey(",
					"the encode of the map does not check its keys, which are all equal")
			},
		)
		t.Run("encodes a map with keys of an array of no floats without a failure", func(t *testing.T) {
			t.Parallel()
			src := typeDirective + structA("M map[[0]float64]string")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate orders the keys")
			assert.Contains(t, files[codeName], "i -= _a_putMapArray0Float64String(buf[:i], m.M)",
				"the encode of the map returns no error")
		})
		t.Run("encodes a map with keys of an array of no pointers without a failure", func(t *testing.T) {
			t.Parallel()
			src := typeDirective + structA("M map[[0]*int32]string")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate orders the keys")
			assert.Contains(t, files[codeName], "i -= _a_putMapArray0PtrInt32String(buf[:i], m.M)",
				"the encode of the map returns no error")
		})
	})
}
