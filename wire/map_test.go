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

// descending returns 16 pairs whose keys descend from "p" to "a", each with
// the index of its key in the alphabet as the value.
func descending() []wire.Pair[string, int] {
	pairs := make([]wire.Pair[string, int], 16)
	for i := range pairs {
		pairs[i] = wire.Pair[string, int]{Key: string(rune('p' - i)), Value: 15 - i}
	}
	return pairs
}

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
	b.Run("SortPairs", func(b *testing.B) {
		source, pairs := descending(), descending()
		b.ReportAllocs()
		for b.Loop() {
			copy(pairs, source)
			wire.SortPairs(pairs)
		}
	})
}

// TestMapAllocs checks that Take and SortPairs allocate nothing. It runs
// serially: testing.AllocsPerRun panics while a parallel test runs.
func TestMapAllocs(t *testing.T) {
	keys, values := []string{"a", "b", "c"}, []int{1, 2, 3}
	t.Run("Take/allocates nothing", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkInt = wire.Take(keys, values, len(keys), "b") }, 0,
			"Take allocates nothing")
	})
	t.Run("SortPairs/allocates nothing", func(t *testing.T) {
		source, pairs := descending(), descending()
		sort := func() {
			copy(pairs, source)
			wire.SortPairs(pairs)
		}
		assert.MaxAllocs(t, sort, 0, "SortPairs allocates nothing")
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
	t.Run("SortPairs", func(t *testing.T) {
		t.Parallel()
		type pair = wire.Pair[int64, string]
		tests := []struct {
			name string
			give []pair
			want []pair
		}{
			{name: "sorts no pairs", give: []pair{}, want: []pair{}},
			{name: "sorts one pair", give: []pair{{Key: 1, Value: "a"}}, want: []pair{{Key: 1, Value: "a"}}},
			{
				name: "keeps sorted pairs in their order",
				give: []pair{{Key: -1, Value: "a"}, {Key: 0, Value: "b"}, {Key: 1, Value: "c"}},
				want: []pair{{Key: -1, Value: "a"}, {Key: 0, Value: "b"}, {Key: 1, Value: "c"}},
			},
			{
				name: "sorts descending pairs by key",
				give: []pair{{Key: 3, Value: "c"}, {Key: 2, Value: "b"}, {Key: 1, Value: "a"}},
				want: []pair{{Key: 1, Value: "a"}, {Key: 2, Value: "b"}, {Key: 3, Value: "c"}},
			},
			{
				name: "sorts shuffled pairs by key",
				give: []pair{{Key: 2, Value: "b"}, {Key: -5, Value: "z"}, {Key: 9, Value: "i"}, {Key: 0, Value: "o"}},
				want: []pair{{Key: -5, Value: "z"}, {Key: 0, Value: "o"}, {Key: 2, Value: "b"}, {Key: 9, Value: "i"}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				wire.SortPairs(tt.give)
				assert.Equal(t, tt.give, tt.want, "the pairs ascend by key with their values")
			})
		}
		t.Run("sorts string keys in the order of slices.Sort", func(t *testing.T) {
			t.Parallel()
			pairs := descending()
			wire.SortPairs(pairs)
			keys := make([]string, len(pairs))
			for i, p := range pairs {
				keys[i] = p.Key
				assert.Equal(t, p.Value, int(p.Key[0]-'a'), "each key keeps its value")
			}
			assert.True(t, slices.IsSorted(keys), "the keys ascend as slices.Sort orders them")
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
