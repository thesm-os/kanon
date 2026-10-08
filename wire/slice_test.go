// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"

	"go.thesmos.sh/kanon/wire"
)

// Counts of the elements of a slice around the first allocation of 10 MiB.
const (
	// fitWords is the number of 8-byte elements that fit in 10 MiB.
	fitWords = 1310720
	// fewWords is a number of 8-byte elements that take less than 10 MiB.
	fewWords = 1000
)

// Element types that take more than 10 MiB, and exactly 10 MiB.
type (
	huge  [10485761]byte
	chunk [10485760]byte
)

func TestSlice(t *testing.T) {
	t.Parallel()
	t.Run("SliceCap", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			got  int
			want int
		}{
			{
				name: "returns n for elements that take less than 10 MiB",
				got:  wire.SliceCap[int64](fewWords),
				want: fewWords,
			},
			{name: "returns n for elements that take 10 MiB", got: wire.SliceCap[int64](fitWords), want: fitWords},
			{
				name: "returns the elements that fit in 10 MiB for elements that take more",
				got:  wire.SliceCap[int64](fitWords + 1), want: fitWords,
			},
			{name: "returns 1 for elements of 10 MiB each", got: wire.SliceCap[chunk](3), want: 1},
			{name: "returns 1 for elements of more than 10 MiB each", got: wire.SliceCap[huge](3), want: 1},
			{name: "returns 0 for no elements of more than 10 MiB each", got: wire.SliceCap[huge](0), want: 0},
			{
				name: "returns n for elements that take no memory",
				got:  wire.SliceCap[struct{}](math.MaxInt),
				want: math.MaxInt,
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.got, c.want, "SliceCap returns the capacity of the first allocation")
			})
		}
	})
}

// TestSliceAllocs checks that the capacity of a first allocation takes no
// allocation. It runs serially: testing.AllocsPerRun panics while a parallel
// test runs.
func TestSliceAllocs(t *testing.T) {
	t.Run("SliceCap/allocates nothing", func(t *testing.T) {
		n := fitWords + 1
		assert.MaxAllocs(t, func() { sinkInt = wire.SliceCap[huge](n) }, 0, "SliceCap allocates nothing")
	})
}

func BenchmarkSlice(b *testing.B) {
	b.Run("SliceCap", func(b *testing.B) {
		n := fitWords + 1
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = wire.SliceCap[int64](n)
		}
		assert.Equal(b, got, fitWords, "SliceCap returns the elements that fit in 10 MiB")
	})
}
