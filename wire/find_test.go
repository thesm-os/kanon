// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The field that the Find cases look for: field 2, a string, of the struct
// Order.
const (
	findLoc   = "Order.Name"
	findType  = "Order"
	findField = "Name"
	findTag   = 2<<3 | wire.Bytes
)

func TestFind(t *testing.T) {
	t.Parallel()
	t.Run("Find", func(t *testing.T) {
		t.Parallel()
		offsets := []struct {
			name string
			data []byte
			want int
		}{
			{
				name: "returns the offset of the length of the value",
				data: []byte{0x08, 0x01, 0x12, 0x01, 'a'},
				want: 3,
			},
			{
				name: "returns the offset of the last occurrence",
				data: []byte{0x12, 0x00, 0x12, 0x01, 'b'},
				want: 3,
			},
			{
				name: "returns the offset of a field after unknown fields",
				data: []byte{0x19, 1, 2, 3, 4, 5, 6, 7, 8, 0x25, 1, 2, 3, 4, 0x12, 0x00},
				want: 15,
			},
			{name: "returns -1 for data without the field", data: []byte{0x08, 0x01}, want: -1},
			{name: "returns -1 for empty data", data: nil, want: -1},
		}
		for _, c := range offsets {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				at, err := wire.Find(c.data, findTag, findLoc)
				assert.NoError(t, err, "Find reads a well-formed encoding")
				assert.Equal(t, at, c.want, "Find returns the offset of the value")
			})
		}
		failures := []struct {
			name string
			data []byte
			want *kanon.DecodeError
		}{
			{
				name: "returns io.ErrUnexpectedEOF for a tag that ends early",
				data: []byte{0x08, 0x01, 0x80},
				want: &kanon.DecodeError{Type: findType, Offset: 2, Err: io.ErrUnexpectedEOF},
			},
			{
				name: "returns kanon.ErrMalformed for a tag that overflows",
				data: ones(9, 0x02),
				want: &kanon.DecodeError{
					Type: findType, Offset: 0, Detail: "varint overflows 64 bits", Err: kanon.ErrMalformed,
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF for a value that ends early",
				data: []byte{0x12, 0x05, 'a'},
				want: &kanon.DecodeError{Type: findType, Offset: 0, Err: io.ErrUnexpectedEOF},
			},
			{
				name: "returns kanon.ErrMalformed for the field with another wire format",
				data: []byte{0x08, 0x01, 0x10, 0x01},
				want: &kanon.DecodeError{
					Type: findType, Field: findField, Number: 2, Offset: 2, Detail: "wire format varint, want bytes",
					Err: kanon.ErrMalformed,
				},
			},
		}
		for _, c := range failures {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				at, err := wire.Find(c.data, findTag, findLoc)
				assert.Equal(t, at, -1, "Find returns no offset with an error")
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "Find returns a *kanon.DecodeError")
				assert.Equal(t, got, c.want, "Find locates the error")
			})
		}
	})
	t.Run("FindOne", func(t *testing.T) {
		t.Parallel()
		offsets := []struct {
			name string
			data []byte
			want int
		}{
			{
				name: "returns the offset of the length of the value",
				data: []byte{0x08, 0x01, 0x12, 0x01, 'a'},
				want: 3,
			},
			{name: "returns -1 for data without the field", data: []byte{0x08, 0x01}, want: -1},
		}
		for _, c := range offsets {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				at, err := wire.FindOne(c.data, findTag, findLoc)
				assert.NoError(t, err, "FindOne reads a well-formed encoding")
				assert.Equal(t, at, c.want, "FindOne returns the offset of the value")
			})
		}
		failures := []struct {
			name string
			data []byte
			tag  uint64
			loc  string
			want *kanon.DecodeError
		}{
			{
				name: "returns kanon.ErrRepeatedView at the tag of a second occurrence",
				data: []byte{0x12, 0x00, 0x08, 0x01, 0x12, 0x01, 'b'},
				tag:  findTag,
				loc:  findLoc,
				want: &kanon.DecodeError{
					Type: findType, Field: findField, Number: 2, Offset: 4, Err: kanon.ErrRepeatedView,
				},
			},
			{
				name: "returns kanon.ErrRepeatedView at offset 5 for the record encoding of the wire format",
				data: []byte{0x72, 0x03, 0x0a, 0x01, 'a', 0x72, 0x02, 0x10, 0x01},
				tag:  14<<3 | wire.Bytes,
				loc:  "Record.Item",
				want: &kanon.DecodeError{
					Type: "Record", Field: "Item", Number: 14, Offset: 5, Err: kanon.ErrRepeatedView,
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF for a value that ends early",
				data: []byte{0x12, 0x05, 'a'},
				tag:  findTag,
				loc:  findLoc,
				want: &kanon.DecodeError{Type: findType, Offset: 0, Err: io.ErrUnexpectedEOF},
			},
		}
		for _, c := range failures {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				at, err := wire.FindOne(c.data, c.tag, c.loc)
				assert.Equal(t, at, -1, "FindOne returns no offset with an error")
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "FindOne returns a *kanon.DecodeError")
				assert.Equal(t, got, c.want, "FindOne locates the error")
			})
		}
	})
}

func BenchmarkFind(b *testing.B) {
	data := []byte{0x08, 0x01, 0x12, 0x01, 'a'}
	b.ReportAllocs()
	for b.Loop() {
		sinkInt, sinkErr = wire.Find(data, findTag, findLoc)
	}
}

// TestFindAllocs checks that Find and FindOne allocate nothing. It runs
// serially: testing.AllocsPerRun panics while a parallel test runs.
func TestFindAllocs(t *testing.T) {
	data := []byte{0x08, 0x01, 0x12, 0x01, 'a'}
	t.Run("Find/allocates nothing", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkInt, sinkErr = wire.Find(data, findTag, findLoc) }, 0,
			"Find allocates nothing for a well-formed encoding")
	})
	t.Run("FindOne/allocates nothing", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkInt, sinkErr = wire.FindOne(data, findTag, findLoc) }, 0,
			"FindOne allocates nothing for a well-formed encoding")
	})
}
