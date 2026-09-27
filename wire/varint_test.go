// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"encoding/binary"
	"math"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

// maxTag is the smallest tag that PutTag does not write: the tag of field
// number 2048, which takes three bytes.
const maxTag = 1 << 14

// ones returns n bytes of 0xff, the continuation bit and seven bits of
// ones, followed by last.
func ones(n int, last byte) []byte {
	b := make([]byte, n, n+1)
	for i := range b {
		b[i] = 0xff
	}
	return append(b, last)
}

func TestVarint(t *testing.T) {
	t.Parallel()
	t.Run("Uvarint", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			data  []byte
			value uint64
			n     int
		}{
			{name: "reads 0 from one byte", data: []byte{0x00}, value: 0, n: 1},
			{name: "reads 127 from one byte", data: []byte{0x7f}, value: 127, n: 1},
			{name: "reads 128 from two bytes", data: []byte{0x80, 0x01}, value: 128, n: 2},
			{name: "reads 300 from two bytes", data: []byte{0xac, 0x02}, value: 300, n: 2},
			{name: "reads the bits of every group", data: []byte{0xd5, 0xaa, 0x55}, value: 0x155555, n: 3},
			{name: "stops at the end of the varint", data: []byte{0x05, 0xff}, value: 5, n: 1},
			{name: "reads a leading zero group", data: []byte{0x80, 0x00}, value: 0, n: 2},
			{name: "reads leading zero groups up to 10 bytes", data: []byte{
				0x81, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00,
			}, value: 1, n: 10},
			{name: "reads the largest value from 10 bytes", data: ones(9, 0x01), value: math.MaxUint64, n: 10},
			{name: "returns length 0 for empty data", data: nil, value: 0, n: 0},
			{name: "returns length 0 for data that ends inside the varint", data: []byte{0x80}, value: 0, n: 0},
			{name: "returns length 0 for 9 bytes with continuation bits", data: ones(8, 0xff), value: 0, n: 0},
			{name: "rejects a 10th byte above 1", data: ones(9, 0x02), value: 0, n: -1},
			{name: "rejects an 11th byte", data: ones(10, 0x00), value: 0, n: -1},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				v, n := wire.Uvarint(c.data)
				assert.Equal(t, v, c.value, "Uvarint returns the value of the varint")
				assert.Equal(t, n, c.n, "Uvarint returns the length of the varint, 0 for truncation, -1 for overflow")
			})
		}
	})
	t.Run("SizeUvarint", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			v    uint64
			want int
		}{
			{name: "sizes 0 as one byte", v: 0, want: 1},
			{name: "sizes 127 as one byte", v: 127, want: 1},
			{name: "sizes 128 as two bytes", v: 128, want: 2},
			{name: "sizes 16383 as two bytes", v: 16383, want: 2},
			{name: "sizes 16384 as three bytes", v: 16384, want: 3},
			{name: "sizes the largest value as ten bytes", v: math.MaxUint64, want: 10},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.SizeUvarint(c.v), c.want, "SizeUvarint returns the length of the shortest varint")
			})
		}
	})
	t.Run("PutUvarint", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			v    uint64
		}{
			{name: "writes 0 in one byte", v: 0},
			{name: "writes 127 in one byte", v: 127},
			{name: "writes 128 in two bytes", v: 128},
			{name: "writes 300 in two bytes", v: 300},
			{name: "writes the largest value in ten bytes", v: math.MaxUint64},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				buf := make([]byte, 12)
				i := wire.PutUvarint(buf, len(buf), c.v)
				want := binary.AppendUvarint(nil, c.v)
				assert.Equal(t, buf[i:], want, "PutUvarint writes the shortest varint so that it ends at i")
				assert.Equal(t, buf[:i], make([]byte, i), "PutUvarint writes nothing before the varint")
			})
		}
	})
	t.Run("PutTag", func(t *testing.T) {
		t.Parallel()
		t.Run("writes every tag of one or two bytes as PutUvarint writes it", func(t *testing.T) {
			t.Parallel()
			for tag := range uint64(maxTag) {
				got, want := make([]byte, 3), make([]byte, 3)
				gi, wi := wire.PutTag(got, len(got), tag), wire.PutUvarint(want, len(want), tag)
				assert.Equal(t, gi, wi, "PutTag returns the offset of the tag's first byte")
				assert.Equal(t, got, want, "PutTag writes the varint of the tag and nothing else")
			}
		})
	})
	t.Run("Zigzag", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			v    int64
			want uint64
		}{
			{name: "maps 0 to 0", v: 0, want: 0},
			{name: "maps -1 to 1", v: -1, want: 1},
			{name: "maps 1 to 2", v: 1, want: 2},
			{name: "maps -2 to 3", v: -2, want: 3},
			{name: "maps 2 to 4", v: 2, want: 4},
			{name: "maps the largest int64 to the largest even uint64", v: math.MaxInt64, want: math.MaxUint64 - 1},
			{name: "maps the smallest int64 to the largest uint64", v: math.MinInt64, want: math.MaxUint64},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.Zigzag(c.v), c.want, "Zigzag interleaves negative and positive values")
				assert.Equal(t, wire.Unzigzag(c.want), c.v, "Unzigzag inverts Zigzag")
			})
		}
	})
	t.Run("CountVarints", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			data []byte
			want int
		}{
			{name: "counts none in empty data", data: nil, want: 0},
			{name: "counts varints of one and two bytes", data: []byte{0x01, 0x80, 0x01, 0x7f}, want: 3},
			{name: "counts no varint that data cuts off", data: []byte{0x80}, want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.CountVarints(c.data), c.want, "CountVarints counts the bytes that end a varint")
			})
		}
	})
}

// Sinks keep the compiler from dropping the results of benchmarked calls.
var (
	sinkUint64 uint64
	sinkInt64  int64
	sinkInt    int
)

func BenchmarkVarint(b *testing.B) {
	oneByte, tenBytes := []byte{0x7f}, ones(9, 0x01)
	buf := make([]byte, 16)
	b.Run("Uvarint/one byte", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkUint64, sinkInt = wire.Uvarint(oneByte)
		}
	})
	b.Run("Uvarint/ten bytes", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkUint64, sinkInt = wire.Uvarint(tenBytes)
		}
	})
	b.Run("SizeUvarint", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.SizeUvarint(math.MaxUint64)
		}
	})
	b.Run("PutUvarint/one byte", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutUvarint(buf, len(buf), 0x7f)
		}
	})
	b.Run("PutUvarint/ten bytes", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutUvarint(buf, len(buf), math.MaxUint64)
		}
	})
	b.Run("PutTag/two bytes", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutTag(buf, len(buf), maxTag-1)
		}
	})
	b.Run("Zigzag", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkUint64 = wire.Zigzag(math.MinInt64)
		}
	})
	b.Run("Unzigzag", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt64 = wire.Unzigzag(math.MaxUint64)
		}
	})
	b.Run("CountVarints", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.CountVarints(tenBytes)
		}
	})
}

func FuzzUvarint(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0xac, 0x02})
	f.Add(ones(9, 0x01))
	f.Add(ones(9, 0x02))
	f.Add(ones(10, 0x00))
	f.Fuzz(func(t *testing.T, data []byte) {
		v, n := wire.Uvarint(data)
		want, wn := binary.Uvarint(data)
		switch {
		case wn > 0:
			assert.Equal(t, v, want, "Uvarint reads the value binary.Uvarint reads")
			assert.Equal(t, n, wn, "Uvarint reads the length binary.Uvarint reads")
		case wn == 0:
			assert.Equal(t, n, 0, "Uvarint reports truncation where binary.Uvarint does")
		default:
			assert.Equal(t, n, -1, "Uvarint reports overflow where binary.Uvarint does")
		}
	})
}
