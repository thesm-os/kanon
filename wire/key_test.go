// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"cmp"
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// Offsets of the zones of the DecodedTime cases, in seconds east of UTC: a
// minute, which added to the offset of a zone of our time, a multiple of
// 15 minutes, makes one whose zone neither time.FixedZone nor the decode
// shares, and a whole hour, which every such zone has.
const (
	oddMinute = 60
	wholeHour = 3600
)

// The map that the key cases check: field 3 of the struct Order, whose
// entries start at offset 9.
const (
	keyLoc    = "Order.Index"
	keyType   = "Order"
	keyField  = "Index"
	keyNumber = 3
	keyOff    = 9
)

// key is a map key of the key cases: its projection is p, and n tells
// apart two keys of one projection.
type key struct{ p, n int }

// compareKeys orders keys by projection.
func compareKeys(a, b key) int { return cmp.Compare(a.p, b.p) }

// canonicalKey reports whether k is the one key of its projection that a
// decode yields: the key whose n is 0.
func canonicalKey(k key) bool { return k.n == 0 }

// ambiguous returns the *kanon.EncodeError of a map with two keys of one
// projection, located at the key map.
func ambiguous() *kanon.EncodeError {
	return &kanon.EncodeError{Type: keyType, Field: keyField, Number: keyNumber, Err: kanon.ErrAmbiguousKey}
}

func TestKey(t *testing.T) {
	t.Parallel()
	t.Run("KeyFloat32", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give float32
			want uint32
		}{
			{name: "returns the bits of +0.0 for -0.0", give: float32(math.Copysign(0, -1)), want: 0},
			{name: "returns the bits of a float that is not zero", give: 1.5, want: math.Float32bits(1.5)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.KeyFloat32(tt.give), tt.want, "KeyFloat32 returns the bits of the key")
			})
		}
	})
	t.Run("KeyFloat64", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give float64
			want uint64
		}{
			{name: "returns the bits of +0.0 for -0.0", give: math.Copysign(0, -1), want: 0},
			{name: "returns the bits of a float that is not zero", give: -2.5, want: math.Float64bits(-2.5)},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.KeyFloat64(tt.give), tt.want, "KeyFloat64 returns the bits of the key")
			})
		}
	})
	t.Run("KeyTies", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give []key
			want error
		}{
			{name: "returns nil for keys of distinct projections", give: []key{{p: 1}, {p: 2}, {p: 3}}},
			{name: "returns nil for no key", give: nil},
			{
				name: "returns ErrAmbiguousKey for two keys of one projection next to each other",
				give: []key{{p: 1}, {p: 2}, {p: 2, n: 1}},
				want: ambiguous(),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.KeyTies(tt.give, compareKeys, keyLoc, keyNumber), tt.want,
					"KeyTies returns the error of the keys")
			})
		}
	})
	t.Run("PairTies", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give []wire.Pair[key, string]
			want error
		}{
			{
				name: "returns nil for the keys of distinct projections",
				give: []wire.Pair[key, string]{{Key: key{p: 1}, Value: "a"}, {Key: key{p: 2}, Value: "a"}},
			},
			{
				name: "returns ErrAmbiguousKey for two keys of one projection next to each other",
				give: []wire.Pair[key, string]{
					{Key: key{p: 1}, Value: "a"}, {Key: key{p: 3}, Value: "b"}, {Key: key{p: 3, n: 1}, Value: "c"},
				},
				want: ambiguous(),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.PairTies(tt.give, compareKeys, keyLoc, keyNumber), tt.want,
					"PairTies returns the error of the keys")
			})
		}
	})
	t.Run("OneKey", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give int
			want error
		}{
			{name: "returns nil for one key", give: 1},
			{name: "returns nil for no key", give: 0},
			{name: "returns ErrAmbiguousKey for two keys", give: 2, want: ambiguous()},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.OneKey(tt.give, keyLoc, keyNumber), tt.want,
					"OneKey returns the error of the keys")
			})
		}
	})
	t.Run("DecodedTime", func(t *testing.T) {
		t.Parallel()
		at := time.Unix(1713400000, 5)
		_, local := at.In(time.Local).Zone()
		shared := 0
		if local == 0 {
			shared = wholeHour
		}
		quarters, _ := wire.Time(timeWithZone(quartersWest), timeLoc, timeNumber, timeOff)
		tests := []struct {
			name string
			give time.Time
			want bool
		}{
			{name: "reports true for a time in UTC", give: at.UTC(), want: true},
			{name: "reports true for a time in the local zone", give: at.In(time.Local), want: true},
			{
				name: "reports true for a time in a fixed zone that time.FixedZone shares",
				give: at.In(time.FixedZone("", shared)),
				want: true,
			},
			{
				name: "reports true for a time that Time decodes at a whole number of quarter hours",
				give: quarters,
				want: true,
			},
			{
				name: "reports false for a time in a fixed zone of quarter hours that time.FixedZone allocates",
				give: quarters.In(time.FixedZone("", quartersWest)),
				want: false,
			},
			{name: "reports false for a time with a monotonic clock reading", give: time.Now(), want: false},
			{
				name: "reports false for a time in a fixed zone of the local offset",
				give: at.In(time.FixedZone("", local)),
				want: false,
			},
			{
				name: "reports false for a time in a named zone",
				give: at.In(time.FixedZone("CET", shared)),
				want: false,
			},
			{
				name: "reports false for a time in a fixed zone that time.FixedZone allocates",
				give: at.In(time.FixedZone("", local+oddMinute)),
				want: false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.DecodedTime(tt.give), tt.want, "DecodedTime reports a time that Time decodes")
			})
		}
	})
	t.Run("MergeKeys", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			give     map[key]string
			wantKeys []key
		}{
			{
				name: "returns no key for keys that a decode yields",
				give: map[key]string{{p: 1}: "a", {p: 2}: "b"},
			},
			{
				name:     "returns every key in order for a key that no decode yields",
				give:     map[key]string{{p: 3}: "a", {p: 1, n: 1}: "b", {p: 2}: "c"},
				wantKeys: []key{{p: 1, n: 1}, {p: 2}, {p: 3}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				keys, err := wire.MergeKeys(tt.give, nil, canonicalKey, compareKeys, keyLoc, keyNumber, keyOff)
				assert.NoError(t, err, "MergeKeys accepts keys of distinct projections")
				assert.Equal(t, keys, tt.wantKeys, "MergeKeys returns the keys in the order of compare",
					assert.EquateEmpty())
			})
		}
		t.Run("returns ErrAmbiguousKey at the entries of the map for two keys of one projection", func(t *testing.T) {
			t.Parallel()
			start := []key{{p: 9}}
			keys, err := wire.MergeKeys(map[key]string{{p: 1}: "a", {p: 1, n: 1}: "b"}, start, canonicalKey,
				compareKeys, keyLoc, keyNumber, keyOff)
			got := assert.ErrorAs[*kanon.DecodeError](t, err, "MergeKeys returns a *kanon.DecodeError")
			assert.Equal(t, got, &kanon.DecodeError{
				Type: keyType, Field: keyField, Number: keyNumber, Offset: keyOff, Err: kanon.ErrAmbiguousKey,
			}, "MergeKeys locates the error at the entries of the map")
			assert.Equal(t, keys, start, "MergeKeys returns the keys that it was given")
		})
		t.Run("appends the keys of the map to keys", func(t *testing.T) {
			t.Parallel()
			start := []key{{p: 9}}
			keys, err := wire.MergeKeys(map[key]string{{p: 1, n: 1}: "a"}, start, canonicalKey, compareKeys, keyLoc,
				keyNumber, keyOff)
			assert.NoError(t, err, "MergeKeys accepts one key")
			assert.Equal(t, keys, []key{{p: 9}, {p: 1, n: 1}}, "MergeKeys appends after the keys it is given")
		})
	})
	t.Run("KeepLastKeys", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give map[key]string
			keys []key
			want map[key]string
		}{
			{
				name: "deletes the earlier of two keys of one projection",
				give: map[key]string{{p: 1, n: 1}: "a", {p: 1, n: 2}: "b", {p: 2}: "c"},
				keys: []key{{p: 1, n: 1}, {p: 2}, {p: 1, n: 2}},
				want: map[key]string{{p: 1, n: 2}: "b", {p: 2}: "c"},
			},
			{
				name: "keeps the last of three keys of one projection",
				give: map[key]string{{p: 1, n: 3}: "a", {p: 1, n: 1}: "b", {p: 1, n: 2}: "c"},
				keys: []key{{p: 1, n: 3}, {p: 1, n: 1}, {p: 1, n: 2}},
				want: map[key]string{{p: 1, n: 2}: "c"},
			},
			{
				name: "keeps a key that the map took twice when it is the last",
				give: map[key]string{{p: 1}: "a", {p: 1, n: 1}: "b"},
				keys: []key{{p: 1}, {p: 1, n: 1}, {p: 1}},
				want: map[key]string{{p: 1}: "a"},
			},
			{
				name: "keeps keys of distinct projections",
				give: map[key]string{{p: 2}: "a", {p: 1}: "b"},
				keys: []key{{p: 2}, {p: 1}},
				want: map[key]string{{p: 2}: "a", {p: 1}: "b"},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				wire.KeepLastKeys(tt.give, tt.keys, compareKeys)
				assert.Equal(t, tt.give, tt.want, "KeepLastKeys keeps the last key of each projection")
			})
		}
	})
}

// TestKeyAllocs checks that the key functions allocate nothing for keys
// that a decode yields and for keys of distinct projections. It runs
// serially: testing.AllocsPerRun panics while a parallel test runs.
func TestKeyAllocs(t *testing.T) {
	at := time.Unix(1, 0).UTC()
	quarters, _ := wire.Time(timeWithZone(quartersWest), timeLoc, timeNumber, timeOff)
	decoded := map[key]string{{p: 1}: "a", {p: 2}: "b"}
	sorted := []key{{p: 1}, {p: 2}}
	pairs := []wire.Pair[key, string]{{Key: key{p: 1}}, {Key: key{p: 2}}}
	var buf [4]key
	tests := []struct {
		name string
		fn   func()
	}{
		{name: "DecodedTime/allocates nothing for a time in UTC", fn: func() { sinkBool = wire.DecodedTime(at) }},
		{
			name: "DecodedTime/allocates nothing for a time in a zone that the decode shares",
			fn:   func() { sinkBool = wire.DecodedTime(quarters) },
		},
		{
			name: "KeyTies/allocates nothing for keys of distinct projections",
			fn:   func() { sinkErr = wire.KeyTies(sorted, compareKeys, keyLoc, keyNumber) },
		},
		{
			name: "PairTies/allocates nothing for keys of distinct projections",
			fn:   func() { sinkErr = wire.PairTies(pairs, compareKeys, keyLoc, keyNumber) },
		},
		{name: "OneKey/allocates nothing for one key", fn: func() { sinkErr = wire.OneKey(1, keyLoc, keyNumber) }},
		{
			name: "MergeKeys/allocates nothing for keys that a decode yields",
			fn: func() {
				_, sinkErr = wire.MergeKeys(decoded, buf[:0], canonicalKey, compareKeys, keyLoc, keyNumber, keyOff)
			},
		},
		{
			name: "KeepLastKeys/allocates nothing",
			fn: func() {
				keys := append(buf[:0], key{p: 2}, key{p: 1})
				wire.KeepLastKeys(decoded, keys, compareKeys)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.MaxAllocs(t, tt.fn, 0, "the function allocates nothing")
		})
	}
}
