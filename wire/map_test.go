// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"cmp"
	"slices"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

func BenchmarkMap(b *testing.B) {
	keys, values := make([]string, 16), make([]int, 16)
	for i := range keys {
		keys[i] = string(rune('a' + i))
	}
	b.Run("Take", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.Take(keys, values, len(keys), "p")
		}
	})
}

func TestMap(t *testing.T) {
	t.Parallel()
	t.Run("Pair", func(t *testing.T) {
		t.Parallel()
		t.Run("sorts by key with slices.SortFunc", func(t *testing.T) {
			t.Parallel()
			type pair = wire.Pair[float64, string]
			pairs := []pair{{Key: 2, Value: "b"}, {Key: 1, Value: "a"}}
			slices.SortFunc(pairs, func(a, b pair) int { return cmp.Compare(a.Key, b.Key) })
			assert.Equal(t, pairs, []pair{{Key: 1, Value: "a"}, {Key: 2, Value: "b"}},
				"a pair keeps its value next to its key")
		})
	})
	t.Run("Take", func(t *testing.T) {
		t.Parallel()
		t.Run("moves the value of the key to the end of the list", func(t *testing.T) {
			t.Parallel()
			keys, values := []string{"a", "b", "c"}, []int{1, 2, 3}
			n := wire.Take(keys, values, 3, "a")
			assert.Equal(t, n, 2, "Take shortens the list by one")
			assert.Equal(t, values[n], 1, "Take takes the value of the key")
			assert.Equal(t, keys, []string{"c", "b", "a"}, "Take swaps the key with the last key of the list")
			assert.Equal(t, values, []int{3, 2, 1}, "Take swaps the values with the keys")
		})
		t.Run("takes the last value when no key matches", func(t *testing.T) {
			t.Parallel()
			keys, values := []string{"a", "b", "c", "d"}, []int{1, 2, 3, 4}
			n := wire.Take(keys, values, 3, "d")
			assert.Equal(t, n, 2, "Take shortens the list by one")
			assert.Equal(t, values[n], 3, "Take takes the last value of the list")
			assert.Equal(t, keys, []string{"a", "b", "c", "d"}, "Take moves nothing")
		})
		t.Run("takes the one value of a list of one", func(t *testing.T) {
			t.Parallel()
			n := wire.Take([]string{"a"}, []int{1}, 1, "z")
			assert.Equal(t, n, 0, "Take empties the list")
		})
	})
}
