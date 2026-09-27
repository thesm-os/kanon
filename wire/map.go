// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

// Pair is a key of a map with its value. The encode of a map whose keys a
// lookup cannot always find, because a key is or contains a float that can
// be NaN, sorts the pairs instead of the keys.
type Pair[K comparable, V any] struct {
	Key   K
	Value V
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
