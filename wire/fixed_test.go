// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/wire"
)

// sinkUint32 keeps the compiler from dropping the results of benchmarked
// calls.
var sinkUint32 uint32

func BenchmarkFixed(b *testing.B) {
	data, buf := make([]byte, 8), make([]byte, 8)
	b.Run("Uint32", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkUint32, sinkInt = wire.Uint32(data)
		}
	})
	b.Run("Uint64", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkUint64, sinkInt = wire.Uint64(data)
		}
	})
	b.Run("PutUint32", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutUint32(buf, len(buf), 0x01020304)
		}
	})
	b.Run("PutUint64", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			sinkInt = wire.PutUint64(buf, len(buf), 0x0102030405060708)
		}
	})
}

func TestFixed(t *testing.T) {
	t.Parallel()
	t.Run("Uint32", func(t *testing.T) {
		t.Parallel()
		t.Run("reads the last four little-endian bytes", func(t *testing.T) {
			t.Parallel()
			v, n := wire.Uint32([]byte{0x04, 0x03, 0x02, 0x01})
			assert.Equal(t, v, uint32(0x01020304), "Uint32 reads the bytes least significant first")
			assert.Equal(t, n, 4, "Uint32 reads four bytes")
		})
		t.Run("returns length 0 for three bytes", func(t *testing.T) {
			t.Parallel()
			v, n := wire.Uint32([]byte{0x04, 0x03, 0x02})
			assert.Equal(t, v, uint32(0), "Uint32 reads no value from short data")
			assert.Equal(t, n, 0, "Uint32 reports truncation")
		})
	})
	t.Run("Uint64", func(t *testing.T) {
		t.Parallel()
		t.Run("reads the last eight little-endian bytes", func(t *testing.T) {
			t.Parallel()
			v, n := wire.Uint64([]byte{0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01})
			assert.Equal(t, v, uint64(0x0102030405060708), "Uint64 reads the bytes least significant first")
			assert.Equal(t, n, 8, "Uint64 reads eight bytes")
		})
		t.Run("returns length 0 for seven bytes", func(t *testing.T) {
			t.Parallel()
			v, n := wire.Uint64(make([]byte, 7))
			assert.Equal(t, v, uint64(0), "Uint64 reads no value from short data")
			assert.Equal(t, n, 0, "Uint64 reports truncation")
		})
	})
	t.Run("PutUint32", func(t *testing.T) {
		t.Parallel()
		t.Run("writes four little-endian bytes that end at i", func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, 6)
			i := wire.PutUint32(buf, 5, 0x01020304)
			assert.Equal(t, i, 1, "PutUint32 returns the offset of the first byte")
			assert.Equal(t, buf, []byte{0, 0x04, 0x03, 0x02, 0x01, 0}, "PutUint32 writes the bytes before i")
		})
	})
	t.Run("PutUint64", func(t *testing.T) {
		t.Parallel()
		t.Run("writes eight little-endian bytes that end at i", func(t *testing.T) {
			t.Parallel()
			buf := make([]byte, 10)
			i := wire.PutUint64(buf, 9, 0x0102030405060708)
			assert.Equal(t, i, 1, "PutUint64 returns the offset of the first byte")
			assert.Equal(t, buf, []byte{0, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0},
				"PutUint64 writes the bytes before i")
		})
	})
}
