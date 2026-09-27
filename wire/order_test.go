// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

func BenchmarkOrder(b *testing.B) {
	at := time.Unix(1, 0)
	zoned := at.In(time.FixedZone("", hourEast))
	b.Run("CompareBool", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.CompareBool(false, true)
		}
	})
	b.Run("CompareComplex", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.CompareComplex(complex(1, 2), complex(1, 3))
		}
	})
	b.Run("CompareTime", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.CompareTime(at.UTC(), zoned)
		}
	})
}

func TestOrder(t *testing.T) {
	t.Parallel()
	t.Run("CompareBool", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			a, b bool
			want int
		}{
			{name: "orders false before true", a: false, b: true, want: -1},
			{name: "orders true after false", a: true, b: false, want: 1},
			{name: "ties two falses", a: false, b: false, want: 0},
			{name: "ties two trues", a: true, b: true, want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.CompareBool(c.a, c.b), c.want, "CompareBool orders false before true")
			})
		}
	})
	t.Run("CompareComplex", func(t *testing.T) {
		t.Parallel()
		nan := math.NaN()
		cases := []struct {
			name string
			a, b complex128
			want int
		}{
			{name: "orders by the real parts first", a: complex(1, 9), b: complex(2, 0), want: -1},
			{name: "orders by the imaginary parts on equal real parts", a: complex(1, 2), b: complex(1, 1), want: 1},
			{name: "ties equal numbers", a: complex(1, 2), b: complex(1, 2), want: 0},
			{name: "orders a NaN real part first", a: complex(nan, 0), b: complex(-1, 0), want: -1},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.CompareComplex(c.a, c.b), c.want, "CompareComplex orders by real, then imaginary")
			})
		}
	})
	t.Run("CompareTime", func(t *testing.T) {
		t.Parallel()
		at := time.Unix(1, 0)
		east, west := time.FixedZone("", hourEast), time.FixedZone("", -hourEast)
		cases := []struct {
			name string
			a, b time.Time
			want int
		}{
			{name: "orders by instant first", a: at.In(east), b: time.Unix(2, 0).UTC(), want: -1},
			{name: "orders UTC before a zone at one instant", a: at.UTC(), b: at.In(west), want: -1},
			{name: "orders a zone after UTC at one instant", a: at.In(west), b: at.UTC(), want: 1},
			{name: "ties two times in UTC", a: at.UTC(), b: at.UTC(), want: 0},
			{name: "orders zones by offset at one instant", a: at.In(west), b: at.In(east), want: -1},
			{name: "ties two zones with one offset", a: at.In(east), b: at.In(time.FixedZone("E", hourEast)), want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.CompareTime(c.a, c.b), c.want, "CompareTime orders by instant, UTC, then offset")
			})
		}
	})
}
