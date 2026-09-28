// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"math"
	"slices"
	"time"

	"go.thesmos.sh/kanon"
)

// KeyFloat32 returns the bits that a map key writes for f: those of +0.0
// for -0.0, which the projection of a key does not tell apart from +0.0,
// and those of f otherwise.
func KeyFloat32(f float32) uint32 {
	if f == 0 {
		return 0
	}
	return math.Float32bits(f)
}

// KeyFloat64 returns the bits that a map key writes for f, as [KeyFloat32]
// does for a float32.
func KeyFloat64(f float64) uint64 {
	if f == 0 {
		return 0
	}
	return math.Float64bits(f)
}

// KeyTies returns the error of the encode of a map when keys, its keys in
// sorted order, contains two keys that compare orders equal, which have one
// projection and sort next to each other: a *kanon.EncodeError that wraps
// kanon.ErrAmbiguousKey, which loc and num locate at the field of the map.
// It returns nil for keys of distinct projections. compare orders keys by
// projection, as the compare function of a key type does.
func KeyTies[K any](keys []K, compare func(a, b K) int, loc string, num int) error {
	for i := 1; i < len(keys); i++ {
		if compare(keys[i-1], keys[i]) == 0 {
			return ambiguousKeyError(loc, num)
		}
	}
	return nil
}

// PairTies returns the error of [KeyTies] for the keys of pairs, the sorted
// pairs of a map.
func PairTies[K comparable, V any](pairs []Pair[K, V], compare func(a, b K) int, loc string, num int) error {
	for i := 1; i < len(pairs); i++ {
		if compare(pairs[i-1].Key, pairs[i].Key) == 0 {
			return ambiguousKeyError(loc, num)
		}
	}
	return nil
}

// OneKey returns the error of [KeyTies] for a map of n keys whose type has
// one projection, such as a struct whose encoded fields have one value each
// and whose other fields tell its keys apart: that error for more than one
// key, and nil otherwise.
func OneKey(n int, loc string, num int) error {
	if n > 1 {
		return ambiguousKeyError(loc, num)
	}
	return nil
}

// DecodedTime reports whether t is the time that [Time] decodes from the
// encoding of t: a time without a monotonic clock reading, in UTC, in the
// local zone when that zone has the offset of t at its instant, or else in
// the unnamed fixed zone of its offset, when time.FixedZone shares that
// zone between calls. Two times that DecodedTime accepts are equal exactly
// when they encode alike, so the key equality of a Go map merges two such
// keys of one encoding. A zone that time.FixedZone allocates for every
// call makes DecodedTime allocate once.
func DecodedTime(t time.Time) bool {
	if t != t.Round(0) {
		return false
	}
	loc := t.Location()
	if loc == time.UTC {
		return true
	}
	_, off := t.Zone()
	if _, local := t.In(time.Local).Zone(); local == off {
		return loc == time.Local
	}
	return loc == time.FixedZone("", off)
}

// MergeKeys returns keys with the keys of x that a merge into x compares
// with the keys that it decodes, and an error when two keys of x have one
// projection, which fails the merge before it changes x: a
// *kanon.DecodeError that wraps kanon.ErrAmbiguousKey, which loc, num and
// off locate at the entries of the map, with keys as it was. canonical
// reports whether a key is the key that a decode of its encoding yields,
// whose projection no other such key of x shares, and compare orders keys
// by projection, as the compare function of a key type does. When canonical
// rejects a key of x, MergeKeys appends every key of x to keys, in the order
// of compare, since such a key can have the projection of a key that the
// merge decodes. It returns keys unchanged otherwise.
func MergeKeys[K comparable, V any](
	x map[K]V,
	keys []K,
	canonical func(K) bool,
	compare func(a, b K) int,
	loc string,
	num, off int,
) ([]K, error) {
	found := false
	for k := range x {
		if !canonical(k) {
			found = true
			break
		}
	}
	if !found {
		return keys, nil
	}
	start := len(keys)
	for k := range x {
		keys = append(keys, k)
	}
	own := keys[start:]
	slices.SortFunc(own, compare)
	for i := 1; i < len(own); i++ {
		if compare(own[i-1], own[i]) == 0 {
			return keys[:start], decodeError(kanon.ErrAmbiguousKey, loc, num, off, "")
		}
	}
	return keys, nil
}

// KeepLastKeys deletes from the map x the entries of the keys whose
// projection a later key of keys repeats, so that x keeps one entry for
// the keys of one projection, that of the last. keys lists keys of x in the
// order of their insertion, a key that x kept twice included, and compare
// orders keys by projection. KeepLastKeys sorts keys.
func KeepLastKeys[K comparable, V any](x map[K]V, keys []K, compare func(a, b K) int) {
	slices.SortStableFunc(keys, compare)
	for i := 0; i < len(keys); {
		j := i + 1
		for j < len(keys) && compare(keys[i], keys[j]) == 0 {
			j++
		}
		last := keys[j-1]
		for _, k := range keys[i : j-1] {
			if k != last {
				delete(x, k)
			}
		}
		i = j
	}
}

// ambiguousKeyError returns the error of the encode of a map with two keys
// of one projection, which loc and num locate at the field of the map. It
// wraps kanon.ErrAmbiguousKey.
func ambiguousKeyError(loc string, num int) error {
	return MarshalError(kanon.ErrAmbiguousKey, loc, num)
}
