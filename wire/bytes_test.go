// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

func BenchmarkBytes(b *testing.B) {
	values := []byte{0x03, 'a', 'b', 'c', 0x00, 0x02, 'd', 'e'}
	buf := make([]byte, 16)
	b.Run("SizeBytes", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.SizeBytes(200)
		}
	})
	b.Run("PutRaw/string", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutRaw(buf, len(buf), "0123456789abcdef")
		}
	})
	b.Run("PutRaw/byte slice", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutRaw(buf, len(buf), values)
		}
	})
	b.Run("CountValues", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.CountValues(values)
		}
	})
}

// TestBytesAllocs checks that the length functions allocate nothing. It
// runs serially: testing.AllocsPerRun panics while a parallel test runs.
func TestBytesAllocs(t *testing.T) {
	values := []byte{0x03, 'a', 'b', 'c', 0x00, 0x02, 'd', 'e'}
	buf := make([]byte, 16)
	cases := []struct {
		name string
		fn   func()
	}{
		{name: "SizeBytes/allocates nothing", fn: func() { sinkInt = wire.SizeBytes(200) }},
		{name: "PutRaw/allocates nothing for a string", fn: func() { sinkInt = wire.PutRaw(buf, len(buf), "abc") }},
		{name: "PutRaw/allocates nothing for bytes", fn: func() { sinkInt = wire.PutRaw(buf, len(buf), values) }},
		{name: "CountValues/allocates nothing", fn: func() { sinkInt = wire.CountValues(values) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.MaxAllocs(t, c.fn, 0, "the function allocates nothing")
		})
	}
}

func TestBytes(t *testing.T) {
	t.Parallel()
	t.Run("SizeBytes", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			n    int
			want int
		}{
			{name: "sizes an empty value as its length byte", n: 0, want: 1},
			{name: "sizes 127 bytes with a one-byte length", n: 127, want: 128},
			{name: "sizes 128 bytes with a two-byte length", n: 128, want: 130},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.SizeBytes(c.n), c.want, "SizeBytes adds the length of the length prefix")
			})
		}
	})
	t.Run("PutRaw", func(t *testing.T) {
		t.Parallel()
		t.Run("writes a string that ends at i", func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, 5)
			i := wire.PutRaw(buf, 4, "abc")
			assert.Equal(t, i, 1, "PutRaw returns the offset of the first byte")
			assert.Equal(t, string(buf), "\x00abc\x00", "PutRaw writes the bytes without a length")
		})
		t.Run("writes a byte slice that ends at i", func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, 3)
			i := wire.PutRaw(buf, 3, []byte{1, 2})
			assert.Equal(t, i, 1, "PutRaw returns the offset of the first byte")
			assert.Equal(t, buf, []byte{0, 1, 2}, "PutRaw writes the bytes without a length")
		})
	})
	t.Run("CountValues", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			data []byte
			want int
		}{
			{name: "counts none in empty data", data: nil, want: 0},
			{name: "counts an empty value and a value of two bytes", data: []byte{0x00, 0x02, 'a', 'b'}, want: 2},
			{name: "counts empty values of one byte each", data: []byte{0x00, 0x00, 0x00}, want: 3},
			{name: "stops at a length that runs past data", data: []byte{0x01, 'a', 0x03, 'b'}, want: 1},
			{name: "stops at a length that does not end", data: []byte{0x00, 0x80}, want: 1},
			{name: "stops at a length that overflows", data: ones(9, 0x02), want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.CountValues(c.data), c.want, "CountValues counts the complete values")
			})
		}
	})
}
