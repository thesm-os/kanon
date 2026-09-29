// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import "cmp"

// Pair is a key of a map with its value. The encode of a map sorts pairs
// instead of keys when a lookup cannot always find a key, because the key is
// or contains a float that can be NaN, and when a map of at most 16 integer or
// string keys has values small enough that a copy into a pair costs less than
// the lookup of each value after the sort.
type Pair[K comparable, V any] struct {
	Key   K
	Value V
}

// SortPairs sorts p by key in the order of cmp.Less, the order of
// slices.Sort, with an insertion sort. It does not allocate or call a
// comparison function. Its number of comparisons grows with the square of
// len(p), so the encode of a map calls it for the at most 16 pairs of a stack
// array.
func SortPairs[K cmp.Ordered, V any](p []Pair[K, V]) {
	for i := range len(p) {
		for j := i; j > 0 && cmp.Less(p[j].Key, p[j-1].Key); j-- {
			p[j], p[j-1] = p[j-1], p[j]
		}
	}
}

// Take takes a value from the free list of a map decode: the first n
// values of values, whose keys are the first n keys of keys. It moves the
// value whose key is key, or else the last value, to index n-1, and returns
// n-1, the length of the list without it. n must be positive. A decode
// that reuses the previous value of a key reuses memory of the size the
// key needed before.
func Take[K comparable, V any](keys []K, values []V, n int, key K) int {
	n--
	for j := range n {
		if keys[j] == key {
			keys[j], keys[n] = keys[n], keys[j]
			values[j], values[n] = values[n], values[j]
			break
		}
	}
	return n
}
