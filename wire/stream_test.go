// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"testing"
	"testing/iotest"
	"unsafe"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"
	"go.dokimi.dev/assert/expect"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The streamed fields of Box, the struct type of the stream cases: the byte
// slice Data and the slice Items of the struct Item, and with the tag option
// max the slice Parts of the struct Part.
const (
	dataNum  = 2
	itemsNum = 4
	partsNum = 6
)

// The schemas of Box: with the decode that accepts every encoding of a
// value, with the canonical decode, and with the bounds 2 on Items and 1 on
// Parts.
var (
	boxSchema = wire.StreamSchema{Loc: "Box", Fields: []wire.StreamField{
		{Tag: dataNum<<3 | wire.Bytes, Loc: "Box.Data"},
		{Tag: itemsNum<<3 | wire.Bytes, Loc: "Box.Items", Elem: "Item"},
	}}
	canonicalSchema = wire.StreamSchema{Loc: "Box", Canonical: true, Fields: boxSchema.Fields}
	boundedSchema   = wire.StreamSchema{Loc: "Box", Fields: []wire.StreamField{
		{Tag: dataNum<<3 | wire.Bytes, Loc: "Box.Data"},
		{Tag: itemsNum<<3 | wire.Bytes, Loc: "Box.Items", Elem: "Item", Max: 2},
		{Tag: partsNum<<3 | wire.Bytes, Loc: "Box.Parts", Elem: "Part", Max: 1},
	}}
)

// farSchema is the schema of Far, whose byte slice Data is field 64, whose
// number is 0 modulo 64.
var farSchema = wire.StreamSchema{Loc: "Far", Fields: []wire.StreamField{{Tag: 64<<3 | wire.Bytes, Loc: "Far.Data"}}}

// canonicalBox is a canonical encoding of Box: field 1, Data, field 3,
// Items with one element, and field 5.
var canonicalBox = []byte{
	0x08, 0x01, 0x12, 0x01, 0x61, 0x18, 0x01, 0x22, 0x03, 0x02, 0x08, 0x01, 0x28, 0x01,
}

// valueLength is the length of a value longer than the buffer of the reader
// of a stream, which has 4096 bytes.
const valueLength = 5000

// Encodings of Box with values longer than the buffer of the reader: field
// 1, a byte slice of valueLength bytes, and then field 1 as a varint; and
// Items with one element of valueLength bytes, at offset 5.
var (
	longValue = slices.Concat(
		[]byte{0x0a}, binary.AppendUvarint(nil, valueLength), bytes.Repeat([]byte{0x61}, valueLength),
		[]byte{0x08, 0x01},
	)
	longElement = slices.Concat(
		[]byte{0x22}, binary.AppendUvarint(nil, valueLength+2), binary.AppendUvarint(nil, valueLength),
		bytes.Repeat([]byte{0x61}, valueLength),
	)
)

// overflow is a varint that exceeds 64 bits: nine bytes of the continuation
// bit alone, and a tenth byte of 2.
var overflow = slices.Concat(bytes.Repeat([]byte{0x80}, 9), []byte{0x02})

// longData is an encoding of Box with Data of 10000 bytes, more than two read
// buffers, at offset 3.
var longData = slices.Concat([]byte{0x12}, binary.AppendUvarint(nil, 10000), bytes.Repeat([]byte{0x62}, 10000))

// emptyReader is a reader that returns no byte and no error for every call.
type emptyReader struct{}

// Read returns 0 and no error.
func (emptyReader) Read([]byte) (int, error) {
	return 0, nil
}

// countWriter is a writer that reports count(len(p)) bytes for each write of
// p and returns err: a writer that fails, that takes fewer bytes than it gets,
// or that reports a count outside the length of p.
type countWriter struct {
	count func(int) int
	err   error
}

// Write returns the count of w for p and the error of w.
func (w countWriter) Write(p []byte) (int, error) {
	return w.count(len(p)), w.err
}

// errBroken is the error of a reader or a writer that fails.
var errBroken = errors.New("disk: broken")

// streamCase is the encoding of Box that a stream reads, the reader that
// returns it, and the results of the calls that [transcript] records.
type streamCase struct {
	name   string
	schema *wire.StreamSchema
	opts   kanon.StreamOptions
	enc    []byte
	// cut makes the reader return the first ends bytes of enc, and broken
	// makes it fail with errBroken after them. The stream reads enc as an
	// encoding of len(enc) bytes either way. single makes the reader return
	// one byte per call, and eof makes it return io.EOF with its last bytes.
	cut, broken, single, eof bool
	ends                     int
	// skip makes the caller skip every streamed value.
	skip bool
	want []string
}

func TestStream(t *testing.T) {
	t.Parallel()
	t.Run("Next", func(t *testing.T) {
		t.Parallel()
		runCases(t, []streamCase{
			{
				name: "returns the encoding as one run without a streamed field",
				enc: []byte{
					0x08, 0x96, 0x01, 0x1a, 0x02, 0x61, 0x62, 0x29, 1, 2, 3, 4, 5, 6, 7, 8, 0x35, 1, 2, 3, 4,
				},
				want: []string{
					"run [08 96 01 1a 02 61 62 29 01 02 03 04 05 06 07 08 35 01 02 03 04] at 0 end", "Open: EOF",
				},
			},
			{name: "returns an empty run for an empty encoding", want: []string{"run [] at 0 end", "Open: EOF"}},
			{
				name: "stops before the tag of a streamed field",
				enc:  []byte{0x08, 0x01, 0x12, 0x03, 0x61, 0x62, 0x63, 0x08, 0x02},
				want: []string{
					"run [08 01] at 0", "open 2", "len 3", "read [61 62 63]", "run [08 02] at 7 end", "Open: EOF",
				},
			},
			{
				name: "discards the bytes of a streamed field that the caller skips",
				enc:  []byte{0x12, 0x03, 0x61, 0x62, 0x63, 0x08, 0x01},
				skip: true,
				want: []string{"run [] at 0", "open 2", "len 3", "run [08 01] at 5 end", "Open: EOF"},
			},
			{
				name: "returns a streamed field each time that it occurs",
				enc:  []byte{0x12, 0x01, 0x61, 0x08, 0x01, 0x12, 0x01, 0x62},
				want: []string{
					"run [] at 0", "open 2", "len 1", "read [61]", "run [08 01] at 3", "open 2", "len 1", "read [62]",
					"run [] at 8 end", "Open: EOF",
				},
			},
			{
				name: "returns a streamed field whose tag is longer than its shortest form",
				enc:  []byte{0x92, 0x00, 0x01, 0x61},
				want: []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [] at 4 end", "Open: EOF"},
			},
			{
				name:   "stops before the tag of a streamed field of a number above 63",
				schema: &farSchema,
				enc:    []byte{0x90, 0x04, 0x01, 0x82, 0x04, 0x01, 0x61},
				want: []string{
					"run [90 04 01] at 0", "open 64", "len 1", "read [61]", "run [] at 7 end", "Open: EOF",
				},
			},
			{
				name:   "returns a decoded field whose number is the number of a streamed field modulo 64",
				schema: &farSchema,
				enc:    []byte{0x80, 0x08, 0x01, 0x82, 0x04, 0x01, 0x61},
				want: []string{
					"run [80 08 01] at 0", "open 64", "len 1", "read [61]", "run [] at 7 end", "Open: EOF",
				},
			},
			{
				name: "stops after the tag of the number of a streamed field in another wire format",
				enc:  []byte{0x08, 0x01, 0x10, 0x05},
				want: []string{"run [08 01 10] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after the tag of field number 0",
				enc:  []byte{0x08, 0x01, 0x00, 0x05},
				want: []string{"run [08 01 00] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a tag that does not end within the encoding",
				enc:  []byte{0x08, 0x01, 0x80},
				want: []string{"run [08 01 80] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a tag that exceeds 64 bits",
				enc:  slices.Concat(overflow, []byte{0x08, 0x01}),
				want: []string{"run [80 80 80 80 80 80 80 80 80 02] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a tag with an invalid wire format",
				enc:  []byte{0x08, 0x01, 0x0b, 0x05},
				want: []string{"run [08 01 0b] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a varint that does not end within the encoding",
				enc:  []byte{0x08, 0x80},
				want: []string{"run [08 80] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a varint that exceeds 64 bits",
				enc:  slices.Concat([]byte{0x08}, overflow, []byte{0x08, 0x01}),
				want: []string{"run [08 80 80 80 80 80 80 80 80 80 02] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after eight bytes that the encoding cuts short",
				enc:  []byte{0x09, 0x01, 0x02, 0x03},
				want: []string{"run [09 01 02 03] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after four bytes that the encoding cuts short",
				enc:  []byte{0x0d, 0x01, 0x02},
				want: []string{"run [0d 01 02] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a length that runs past the encoding",
				enc:  []byte{0x0a, 0x05, 0x61},
				want: []string{"run [0a 05] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a length that does not end within the encoding",
				enc:  []byte{0x0a, 0x80},
				want: []string{"run [0a 80] at 0 end", "Open: EOF"},
			},
			{
				name: "stops after a length that exceeds 64 bits",
				enc:  slices.Concat([]byte{0x0a}, overflow, []byte{0x61}),
				want: []string{"run [0a 80 80 80 80 80 80 80 80 80 02] at 0 end", "Open: EOF"},
			},
			{
				name: "stops before a field that takes the decoded fields past the buffer limit",
				opts: kanon.StreamOptions{Buffer: 4},
				enc:  []byte{0x0a, 0x02, 0x61, 0x62, 0x08, 0x01},
				want: []string{
					"run [0a 02 61 62] at 0", "Open: kanon: Box at offset 4: value takes the buffer past 4 bytes",
				},
			},
			{
				name: "reads no byte of a value whose length takes the decoded fields past the buffer limit",
				opts: kanon.StreamOptions{Buffer: 3},
				enc:  []byte{0x08, 0x01, 0x0a, 0x02, 0x61, 0x62},
				cut:  true, ends: 4,
				want: []string{"run [08 01] at 0", "Open: kanon: Box at offset 2: value takes the buffer past 3 bytes"},
			},
			{
				name: "fills the buffer limit with the decoded fields",
				opts: kanon.StreamOptions{Buffer: 6},
				enc:  []byte{0x08, 0x01, 0x0a, 0x02, 0x61, 0x62},
				want: []string{"run [08 01 0a 02 61 62] at 0 end", "Open: EOF"},
			},
			{
				name: "applies the buffer limit to a varint",
				opts: kanon.StreamOptions{Buffer: 3},
				enc:  []byte{0x08, 0x01, 0x08, 0x01},
				want: []string{"run [08 01] at 0", "Open: kanon: Box at offset 2: value takes the buffer past 3 bytes"},
			},
			{
				name: "applies the buffer limit to eight bytes",
				opts: kanon.StreamOptions{Buffer: 8},
				enc:  []byte{0x09, 1, 2, 3, 4, 5, 6, 7, 8},
				want: []string{"run [] at 0", "Open: kanon: Box at offset 0: value takes the buffer past 8 bytes"},
			},
			{
				name: "counts the decoded fields of every run against the buffer limit",
				opts: kanon.StreamOptions{Buffer: 3},
				enc:  []byte{0x08, 0x01, 0x12, 0x01, 0x61, 0x08, 0x02},
				want: []string{
					"run [08 01] at 0", "open 2", "len 1", "read [61]", "run [] at 5",
					"Open: kanon: Box at offset 5: value takes the buffer past 3 bytes",
				},
			},
			{
				name: "stops before a field over the buffer limit and the streamed field after it",
				opts: kanon.StreamOptions{Buffer: 3},
				enc:  []byte{0x0a, 0x02, 0x61, 0x62, 0x12, 0x01, 0x61},
				want: []string{"run [] at 0", "Open: kanon: Box at offset 0: value takes the buffer past 3 bytes"},
			},
			{
				name: "keeps 20 bytes past the buffer limit of a field that the decode rejects",
				opts: kanon.StreamOptions{Buffer: 2},
				enc: slices.Concat(
					[]byte{0x08, 0x01, 0x8a, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00},
					overflow, []byte{0x61},
				),
				want: []string{
					"run [08 01 8a 80 80 80 80 80 80 80 80 00 80 80 80 80 80 80 80 80 80 02] at 0 end", "Open: EOF",
				},
			},
			{
				name:   "returns the fields of a canonical encoding",
				schema: &canonicalSchema,
				enc:    canonicalBox,
				want: []string{
					"run [08 01] at 0", "open 2", "len 1", "read [61]", "run [18 01] at 5", "open 4", "len 3",
					"element 2", "value [08 01] at 10", "run [28 01] at 12 end", "Open: EOF",
				},
			},
			{
				name:   "returns the fields of a canonical encoding from a reader of one byte per call",
				schema: &canonicalSchema,
				enc:    canonicalBox,
				single: true,
				want: []string{
					"run [08 01] at 0", "open 2", "len 1", "read [61]", "run [18 01] at 5", "open 4", "len 3",
					"element 2", "value [08 01] at 10", "run [28 01] at 12 end", "Open: EOF",
				},
			},
			{
				name:   "returns the fields of a canonical encoding from a reader that returns io.EOF with its last bytes",
				schema: &canonicalSchema,
				enc:    canonicalBox,
				eof:    true,
				want: []string{
					"run [08 01] at 0", "open 2", "len 1", "read [61]", "run [18 01] at 5", "open 4", "len 3",
					"element 2", "value [08 01] at 10", "run [28 01] at 12 end", "Open: EOF",
				},
			},
			{
				name:   "returns kanon.ErrNotCanonical for a tag at the start of a run below the streamed field before it",
				schema: &canonicalSchema,
				enc:    []byte{0x22, 0x03, 0x02, 0x08, 0x01, 0x12, 0x01, 0x61},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2", "value [08 01] at 3",
					"Next: kanon: Box at offset 5: field 2 after field 4",
				},
			},
			{
				name:   "returns kanon.ErrNotCanonical for a streamed field that repeats at the start of a run",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x01, 0x61, 0x12, 0x01, 0x62},
				want: []string{
					"run [] at 0", "open 2", "len 1", "read [61]",
					"Next: kanon: Box at offset 3: field 2 after field 2",
				},
			},
			{
				name:   "stops after a canonical tag that is not above the field before it",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x01, 0x61, 0x18, 0x01, 0x08, 0x01},
				want:   []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [18 01 08] at 3 end", "Open: EOF"},
			},
			{
				name:   "stops after a canonical decoded field that repeats",
				schema: &canonicalSchema,
				enc:    []byte{0x0a, 0x01, 0x61, 0x0a, 0x01, 0x62},
				want:   []string{"run [0a 01 61 0a] at 0 end", "Open: EOF"},
			},
			{
				name:   "stops after a canonical streamed field that repeats after a decoded field",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x01, 0x61, 0x18, 0x01, 0x12, 0x01, 0x62},
				want:   []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [18 01 12] at 3 end", "Open: EOF"},
			},
			{
				name:   "stops after a canonical tag that is longer than its shortest form",
				schema: &canonicalSchema,
				enc:    []byte{0x88, 0x00, 0x01},
				want:   []string{"run [88 00] at 0 end", "Open: EOF"},
			},
			{
				name:   "stops after a canonical tag that is longer than its shortest form at the start of a run",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x01, 0x61, 0x88, 0x00, 0x01},
				want:   []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [88 00] at 3 end", "Open: EOF"},
			},
			{
				name:   "stops after a canonical tag of field number 0 at the start of a run",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x01, 0x61, 0x00, 0x01},
				want:   []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [00] at 3 end", "Open: EOF"},
			},
			{
				name:   "stops after the canonical tag of a streamed field that is longer than its shortest form",
				schema: &canonicalSchema,
				enc:    []byte{0x92, 0x00, 0x01, 0x61},
				want:   []string{"run [92 00] at 0 end", "Open: EOF"},
			},
			{
				name:   "stops after the canonical tag of the number of a streamed field in another wire format",
				schema: &canonicalSchema,
				enc:    []byte{0x10, 0x05},
				want:   []string{"run [10] at 0 end", "Open: EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a tag",
				enc:  []byte{0x08, 0x01, 0x88, 0x01, 0x01},
				cut:  true, ends: 3,
				want: []string{"Next: kanon: Box at offset 3: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a varint",
				enc:  []byte{0x08, 0x96, 0x01},
				cut:  true, ends: 2,
				want: []string{"Next: kanon: Box at offset 2: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside eight bytes",
				enc:  []byte{0x09, 1, 2, 3, 4, 5, 6, 7, 8},
				cut:  true, ends: 5,
				want: []string{"Next: kanon: Box at offset 5: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a length",
				enc:  slices.Concat([]byte{0x0a, 0x81, 0x01}, make([]byte, 129)),
				cut:  true, ends: 2,
				want: []string{"Next: kanon: Box at offset 2: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a value",
				enc:  []byte{0x0a, 0x02, 0x61, 0x62},
				cut:  true, ends: 3,
				want: []string{"Next: kanon: Box at offset 3: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside the bytes that the caller skips",
				enc:  []byte{0x12, 0x03, 0x61, 0x62, 0x63, 0x08, 0x01},
				cut:  true, ends: 3,
				skip: true,
				want: []string{
					"run [] at 0", "open 2", "len 3", "Next: kanon: Box.Data (field 2) at offset 3: unexpected EOF",
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a value longer than its buffer",
				enc:  longValue,
				cut:  true, ends: 3000,
				want: []string{"Next: kanon: Box at offset 3000: unexpected EOF"},
			},
			{
				name: "applies the buffer limit to a value longer than the buffer of the reader",
				opts: kanon.StreamOptions{Buffer: valueLength},
				enc:  longValue,
				want: []string{"run [] at 0", "Open: kanon: Box at offset 0: value takes the buffer past 5000 bytes"},
			},
			{
				name:   "returns the error of the reader",
				enc:    []byte{0x08, 0x01},
				broken: true,
				want:   []string{"Next: disk: broken"},
			},
			{
				name:   "returns the error of the reader for the bytes that the caller skips",
				enc:    []byte{0x12, 0x03, 0x61, 0x62, 0x63},
				broken: true, ends: 3,
				skip: true,
				want: []string{"run [] at 0", "open 2", "len 3", "Next: disk: broken"},
			},
		})
		ends := []struct {
			name   string
			single bool
		}{
			{name: "returns io.ErrUnexpectedEOF at each offset at which the reader ends"},
			{
				name:   "returns io.ErrUnexpectedEOF at each offset at which a reader of one byte per call ends",
				single: true,
			},
		}
		for _, c := range ends {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				for k := range len(canonicalBox) {
					s := start(streamCase{
						schema: &canonicalSchema, enc: canonicalBox, cut: true, ends: k, single: c.single,
					})
					_, err := transcript(s, &canonicalSchema, false)
					assert.ErrorIs(t, err, io.ErrUnexpectedEOF,
						"the stream reports the end of the reader as truncation")
					got := assert.ErrorAs[*kanon.DecodeError](t, err, "the stream returns a *kanon.DecodeError")
					assert.Equal(t, got.Offset, k, "the error locates the first byte that the reader does not return")
				}
			})
		}
		t.Run("returns io.ErrNoProgress for a reader that returns no byte and no error", func(t *testing.T) {
			t.Parallel()
			var s wire.Stream
			s.Init(&boxSchema, kanon.StreamOptions{})
			s.Reset(emptyReader{}, 2)
			_, err := s.Next()
			assert.Equal(t, err, io.ErrNoProgress, "Next returns io.ErrNoProgress", assert.ByIdentity())
		})
		runs := []struct {
			name string
			c    streamCase
		}{
			{
				name: "returns decoded fields that continue past the buffer of the reader",
				c:    streamCase{enc: bytes.Repeat([]byte{0x08, 0x01}, 3000)},
			},
			{
				name: "returns decoded fields that continue past the buffer of a reader of one byte per call",
				c:    streamCase{enc: bytes.Repeat([]byte{0x08, 0x01}, 3000), single: true},
			},
			{name: "returns a decoded value longer than the buffer of the reader", c: streamCase{enc: longValue}},
		}
		for _, c := range runs {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				s := start(c.c)
				r, err := s.Next()
				assert.NoError(t, err, "Next reads the encoding")
				expect.Equal(t, r.Data, c.c.enc, "the run is the encoding")
				_, err = s.Open()
				expect.Equal(t, err, io.EOF, "Open ends the encoding after the run", expect.ByIdentity())
			})
		}
		depths := []struct {
			name  string
			depth int
			want  int
		}{
			{name: "returns a run of DefaultDepth levels for a Depth of 0", depth: 0, want: kanon.DefaultDepth},
			{name: "returns a run of the levels of a positive Depth", depth: 3, want: 3},
			{name: "returns a run of no level for a negative Depth", depth: -1, want: 0},
		}
		for _, c := range depths {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				s := start(streamCase{opts: kanon.StreamOptions{Depth: c.depth}, enc: []byte{0x08, 0x01}})
				r, err := s.Next()
				assert.NoError(t, err, "Next returns the run")
				assert.Equal(t, r.Depth, c.want, "the run has the levels below the struct type")
			})
		}
	})
	t.Run("Run", func(t *testing.T) {
		t.Parallel()
		t.Run("takes four machine words", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, unsafe.Sizeof(wire.Run{}), 4*unsafe.Sizeof(uintptr(0)),
				"a Run has the size of the values that a call returns in registers")
		})
		t.Run("Slab", func(t *testing.T) {
			t.Parallel()
			slabs := []struct {
				name string
				data []byte
				want string
			}{
				{name: "returns the bytes of Data", data: []byte("ab"), want: "ab"},
				{name: "returns an empty string for a run without data", data: nil, want: ""},
			}
			for _, c := range slabs {
				t.Run(c.name, func(t *testing.T) {
					t.Parallel()
					assert.Equal(t, wire.Run{Data: c.data}.Slab(), c.want, "Slab returns the bytes of the run")
				})
			}
		})
	})
	t.Run("Fail", func(t *testing.T) {
		t.Parallel()
		t.Run("returns an error without a *kanon.DecodeError unchanged", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x08, 0x01}})
			assert.Equal(t, s.Fail(errBroken), errBroken, "Fail returns its error", assert.ByIdentity())
		})
		shifts := []struct {
			name string
			enc  []byte
			// value makes the stream read the element of Items at the start of
			// enc, and otherwise the run after Data at the start of enc.
			value bool
			err   error
			want  int
		}{
			{
				name: "adds the offset of the last run to the offset of a *kanon.DecodeError",
				enc:  []byte{0x12, 0x01, 0x61, 0x08, 0x01},
				err:  wire.DepthError("Box", 0, 1),
				want: 4,
			},
			{
				name:  "adds the offset of the last element to the offset of a *kanon.DecodeError",
				enc:   []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				value: true,
				err:   wire.DepthError("Item", 0, 1),
				want:  4,
			},
			{
				name:  "adds the offset of the last element to a *kanon.DecodeError that another error wraps",
				enc:   []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				value: true,
				err:   fmt.Errorf("decode: %w", wire.DepthError("Item", 0, 1)),
				want:  4,
			},
		}
		for _, c := range shifts {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				s := opened(t, streamCase{enc: c.enc})
				var err error
				if c.value {
					_, err = s.Value(itemsNum)
				} else {
					_, err = s.Next()
				}
				assert.NoError(t, err, "the stream reads the run or the element")
				got := assert.ErrorAs[*kanon.DecodeError](t, s.Fail(c.err), "Fail returns the *kanon.DecodeError")
				assert.Equal(t, got.Offset, c.want, "Fail locates the error in the encoding")
			})
		}
		t.Run("makes Next return its error", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x08, 0x01}})
			_ = s.Fail(errBroken)
			_, err := s.Next()
			assert.Equal(t, err, errBroken, "Next returns the error that Fail records", assert.ByIdentity())
		})
	})
	t.Run("Open", func(t *testing.T) {
		t.Parallel()
		runCases(t, []streamCase{
			{
				name: "returns io.ErrUnexpectedEOF for a length that runs past the encoding",
				enc:  []byte{0x12, 0x05, 0x61},
				want: []string{"run [] at 0", "Open: kanon: Box.Data (field 2) at offset 1: unexpected EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF for a length that does not end within the encoding",
				enc:  []byte{0x12, 0x80},
				want: []string{"run [] at 0", "Open: kanon: Box.Data (field 2) at offset 1: unexpected EOF"},
			},
			{
				name: "returns kanon.ErrMalformed for a length that exceeds 64 bits",
				enc:  slices.Concat([]byte{0x12}, overflow),
				want: []string{"run [] at 0", "Open: kanon: Box.Data (field 2) at offset 1: varint overflows 64 bits"},
			},
			{
				name:   "returns kanon.ErrNotCanonical for a canonical length that is longer than its shortest form",
				schema: &canonicalSchema,
				enc:    []byte{0x12, 0x81, 0x00, 0x61},
				want: []string{
					"run [] at 0", "Open: kanon: Box.Data (field 2) at offset 1: varint of 2 bytes has a shorter form",
				},
			},
			{
				name: "opens a field whose length is longer than its shortest form",
				enc:  []byte{0x12, 0x81, 0x00, 0x61},
				want: []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [] at 4 end", "Open: EOF"},
			},
			{
				name: "returns kanon.ErrDepth for a slice under a negative Depth",
				opts: kanon.StreamOptions{Depth: -1},
				enc:  []byte{0x22, 0x02, 0x01, 0x08},
				want: []string{
					"run [] at 0", "Open: kanon: Box.Items (field 4) at offset 2: nested deeper than the limit",
				},
			},
			{
				name: "opens a byte slice under a negative Depth",
				opts: kanon.StreamOptions{Depth: -1},
				enc:  []byte{0x12, 0x01, 0x61},
				want: []string{"run [] at 0", "open 2", "len 1", "read [61]", "run [] at 3 end", "Open: EOF"},
			},
			{
				name:   "returns kanon.ErrNotCanonical for a canonical value of no bytes at the offset of its tag",
				schema: &canonicalSchema,
				enc:    []byte{0x08, 0x01, 0x12, 0x00},
				want: []string{
					"run [08 01] at 0",
					"Open: kanon: Box.Data (field 2) at offset 2: field at a value that the encoding leaves out",
				},
			},
			{
				name:   "returns kanon.ErrDepth before kanon.ErrNotCanonical for a canonical slice of no bytes",
				schema: &canonicalSchema,
				opts:   kanon.StreamOptions{Depth: -1},
				enc:    []byte{0x22, 0x00},
				want: []string{
					"run [] at 0", "Open: kanon: Box.Items (field 4) at offset 2: nested deeper than the limit",
				},
			},
			{
				name: "opens a value of no bytes",
				enc:  []byte{0x12, 0x00},
				want: []string{"run [] at 0", "open 2", "len 0", "read []", "run [] at 2 end", "Open: EOF"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside a length",
				enc:  slices.Concat([]byte{0x12, 0x81, 0x01}, make([]byte, 129)),
				cut:  true, ends: 2,
				want: []string{"run [] at 0", "Open: kanon: Box.Data (field 2) at offset 2: unexpected EOF"},
			},
			{
				name:   "returns the error of the reader",
				enc:    []byte{0x12, 0x01, 0x61},
				broken: true, ends: 1,
				want: []string{"run [] at 0", "Open: disk: broken"},
			},
			{
				name: "returns the error of a decoded field over the buffer limit once",
				opts: kanon.StreamOptions{Buffer: 1},
				enc:  []byte{0x08, 0x01},
				want: []string{"run [] at 0", "Open: kanon: Box at offset 0: value takes the buffer past 1 bytes"},
			},
		})
		t.Run("makes every later call return io.EOF at the end of the encoding", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x08, 0x01}})
			_, err := s.Next()
			assert.NoError(t, err, "Next returns the run")
			_, err = s.Open()
			assert.Equal(t, err, io.EOF, "Open returns io.EOF at the end of the encoding", assert.ByIdentity())
			_, err = s.Next()
			assert.Equal(t, err, io.EOF, "Next returns io.EOF after the end of the encoding", assert.ByIdentity())
		})
	})
	t.Run("Read", func(t *testing.T) {
		t.Parallel()
		runCases(t, []streamCase{
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends",
				enc:  []byte{0x12, 0x03, 0x61, 0x62, 0x63},
				cut:  true, ends: 3,
				want: []string{
					"run [] at 0", "open 2", "len 3", "read [61]",
					"Read: kanon: Box.Data (field 2) at offset 3: unexpected EOF",
				},
			},
			{
				name:   "returns the error of the reader",
				enc:    []byte{0x12, 0x03, 0x61, 0x62, 0x63},
				broken: true, ends: 3,
				want: []string{"run [] at 0", "open 2", "len 3", "read [61]", "Read: disk: broken"},
			},
		})
		t.Run("reads at most len(p) bytes", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63}})
			p := make([]byte, 2)
			n, err := s.Read(p)
			assert.NoError(t, err, "Read reads the value")
			assert.Equal(t, p[:n], []byte("ab"), "Read reads len(p) bytes of a longer value")
		})
		t.Run("reads no byte past the value", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61, 0x08, 0x01}})
			p := make([]byte, 4)
			n, err := s.Read(p)
			assert.NoError(t, err, "Read reads the value")
			assert.Equal(t, p[:n], []byte("a"), "Read reads the bytes of the value alone")
		})
		t.Run("returns io.EOF for a slice", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_, err := s.Read(make([]byte, 1))
			assert.Equal(t, err, io.EOF, "Read reads no element of a slice", assert.ByIdentity())
		})
		t.Run("returns io.EOF before the first call of Next", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_, err := s.Read(make([]byte, 1))
			assert.Equal(t, err, io.EOF, "Read reads nothing before a field is open", assert.ByIdentity())
		})
		t.Run("returns the error of the stream", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_ = s.Fail(errBroken)
			_, err := s.Read(make([]byte, 1))
			assert.Equal(t, err, errBroken, "Read returns the error of the stream", assert.ByIdentity())
		})
		t.Run("returns 0 and no error for an empty p", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61}})
			n, err := s.Read(nil)
			expect.NoError(t, err, "Read returns no error for an empty p")
			expect.Equal(t, n, 0, "Read reads no byte into an empty p")
		})
		t.Run("reads a value longer than the read buffer", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: longData})
			var got []byte
			p := make([]byte, 2*4096)
			for {
				n, err := s.Read(p)
				got = append(got, p[:n]...)
				if errors.Is(err, io.EOF) {
					break
				}
				assert.NoError(t, err, "Read reads the value")
			}
			assert.Equal(t, got, longData[3:], "Read reads every byte of the value")
		})
		t.Run("returns io.ErrUnexpectedEOF where the reader ends inside a read into p", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: longData, cut: true, ends: 5000})
			p := make([]byte, 2*4096)
			var err error
			for err == nil {
				_, err = s.Read(p)
			}
			assert.Equal(t, err.Error(), "kanon: Box.Data (field 2) at offset 5000: unexpected EOF",
				"Read locates the first byte that the reader does not return")
		})
	})
	t.Run("WriteTo", func(t *testing.T) {
		t.Parallel()
		values := []struct {
			name string
			enc  []byte
			want string
		}{
			{name: "writes the rest of the value", enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63, 0x08, 0x01}, want: "abc"},
			{name: "writes a value longer than the read buffer", enc: longData, want: string(longData[3:])},
		}
		for _, c := range values {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				s := opened(t, streamCase{enc: c.enc})
				var buf bytes.Buffer
				n, err := s.WriteTo(&buf)
				assert.NoError(t, err, "WriteTo writes the value")
				expect.Equal(t, n, int64(len(c.want)), "WriteTo counts the bytes of the value")
				expect.Equal(t, buf.String(), c.want, "WriteTo writes the bytes of the value")
			})
		}
		t.Run("writes the bytes that Read has not read", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63}})
			_, err := s.Read(make([]byte, 1))
			assert.NoError(t, err, "Read reads the first byte")
			var buf bytes.Buffer
			_, err = s.WriteTo(&buf)
			assert.NoError(t, err, "WriteTo writes the rest of the value")
			assert.Equal(t, buf.String(), "bc", "WriteTo writes the bytes after the byte that Read read")
		})
		t.Run("moves to the field after the value", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63, 0x08, 0x01}})
			_, err := s.WriteTo(io.Discard)
			assert.NoError(t, err, "WriteTo writes the value")
			got, _ := transcript(s, &boxSchema, false)
			assert.Equal(t, got, []string{"run [08 01] at 5 end", "Open: EOF"}, "Next reads the field after the value")
		})
		t.Run("writes nothing for a slice", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			n, err := s.WriteTo(io.Discard)
			expect.NoError(t, err, "WriteTo returns no error for a slice")
			expect.Equal(t, n, int64(0), "WriteTo writes no element of a slice")
			expect.True(t, s.More(itemsNum), "the element of the slice remains")
		})
		t.Run("writes nothing before the first call of Next", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x12, 0x01, 0x61}})
			n, err := s.WriteTo(io.Discard)
			expect.NoError(t, err, "WriteTo returns no error before a field is open")
			expect.Equal(t, n, int64(0), "WriteTo writes nothing before a field is open")
		})
		t.Run("writes nothing after the end of the encoding", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_, err := transcript(s, &boxSchema, false)
			assert.Equal(t, err, io.EOF, "the stream reads the encoding to its end", assert.ByIdentity())
			n, err := s.WriteTo(io.Discard)
			expect.NoError(t, err, "WriteTo returns no error after the end of the encoding")
			expect.Equal(t, n, int64(0), "WriteTo writes nothing after the end of the encoding")
		})
		t.Run("returns the error of the stream", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_ = s.Fail(errBroken)
			_, err := s.WriteTo(io.Discard)
			assert.Equal(t, err, errBroken, "WriteTo returns the error of the stream", assert.ByIdentity())
		})
		t.Run("returns io.ErrUnexpectedEOF where the reader ends", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: longData, cut: true, ends: 5000})
			n, err := s.WriteTo(io.Discard)
			assert.HasError(t, err, "WriteTo fails where the reader ends")
			expect.Equal(t, n, int64(4997), "WriteTo writes the bytes of the value that the reader returns")
			expect.Equal(t, err.Error(), "kanon: Box.Data (field 2) at offset 5000: unexpected EOF",
				"WriteTo locates the first byte that the reader does not return")
			_, err = s.Next()
			expect.Equal(t, err.Error(), "kanon: Box.Data (field 2) at offset 5000: unexpected EOF",
				"the error of the reader becomes the error of the stream")
		})
		t.Run("returns the error of the reader", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63}, broken: true, ends: 3})
			n, err := s.WriteTo(io.Discard)
			expect.Equal(t, err, errBroken, "WriteTo returns the error of the reader", expect.ByIdentity())
			expect.Equal(t, n, int64(1), "WriteTo writes the byte that the reader returns")
		})
		writers := []struct {
			name string
			w    countWriter
			want int64
			err  error
			rest string
		}{
			{
				name: "returns the error of the writer",
				w:    countWriter{count: func(n int) int { return n - 1 }, err: errBroken},
				want: 2, err: errBroken, rest: "c",
			},
			{
				name: "returns io.ErrShortWrite for a writer that takes fewer bytes without an error",
				w:    countWriter{count: func(n int) int { return n - 1 }},
				want: 2, err: io.ErrShortWrite, rest: "c",
			},
			{
				name: "counts no byte for a negative count of the writer",
				w:    countWriter{count: func(int) int { return -1 }},
				want: 0, err: io.ErrShortWrite, rest: "abc",
			},
			{
				name: "counts the bytes that it passes for a count of the writer above them",
				w:    countWriter{count: func(n int) int { return n + 1 }},
				want: 3, rest: "",
			},
		}
		for _, c := range writers {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				s := opened(t, streamCase{enc: []byte{0x12, 0x03, 0x61, 0x62, 0x63}})
				n, err := s.WriteTo(c.w)
				expect.Equal(t, err, c.err, "WriteTo returns the error of the write", expect.ByIdentity())
				expect.Equal(t, n, c.want, "WriteTo counts the bytes that the writer takes")
				rest, err := io.ReadAll(s)
				expect.NoError(t, err, "Read reads the bytes that the writer does not take")
				expect.Equal(t, string(rest), c.rest, "the bytes that the writer does not take remain")
			})
		}
	})
	t.Run("Element", func(t *testing.T) {
		t.Parallel()
		runCases(t, []streamCase{
			{
				name: "returns the elements of a streamed slice",
				enc:  []byte{0x22, 0x04, 0x02, 0x08, 0x01, 0x00},
				want: []string{
					"run [] at 0", "open 4", "len 4", "element 2", "value [08 01] at 3", "element 0", "value [] at 6",
					"run [] at 6 end", "Open: EOF",
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF for a length that runs past the field",
				enc:  []byte{0x22, 0x02, 0x05, 0x08},
				want: []string{
					"run [] at 0", "open 4", "len 2", "Element: kanon: Box.Items (field 4) at offset 2: unexpected EOF",
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF for a length that does not end within the field",
				enc:  []byte{0x22, 0x01, 0x80, 0x08},
				want: []string{
					"run [] at 0", "open 4", "len 1", "Element: kanon: Box.Items (field 4) at offset 2: unexpected EOF",
				},
			},
			{
				name: "returns kanon.ErrMalformed for a length that exceeds 64 bits",
				enc:  slices.Concat([]byte{0x22, 0x0a}, overflow),
				want: []string{
					"run [] at 0", "open 4", "len 10",
					"Element: kanon: Box.Items (field 4) at offset 2: varint overflows 64 bits",
				},
			},
			{
				name:   "returns kanon.ErrNotCanonical for a canonical length that is longer than its shortest form",
				schema: &canonicalSchema,
				enc:    []byte{0x22, 0x03, 0x81, 0x00, 0x08},
				want: []string{
					"run [] at 0", "open 4", "len 3",
					"Element: kanon: Box.Items (field 4) at offset 2: varint of 2 bytes has a shorter form",
				},
			},
			{
				name: "returns a length that is longer than its shortest form",
				enc:  []byte{0x22, 0x03, 0x81, 0x00, 0x08},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 1", "value [08] at 4", "run [] at 5 end", "Open: EOF",
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends",
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				cut:  true, ends: 2,
				want: []string{
					"run [] at 0", "open 4", "len 3", "Element: kanon: Box.Items (field 4) at offset 2: unexpected EOF",
				},
			},
			{
				name:   "returns the error of the reader",
				enc:    []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				broken: true, ends: 2,
				want: []string{"run [] at 0", "open 4", "len 3", "Element: disk: broken"},
			},
			{
				name:   "returns the elements of a slice up to its bound",
				schema: &boundedSchema,
				enc:    []byte{0x22, 0x02, 0x00, 0x00},
				want: []string{
					"run [] at 0", "open 4", "len 2", "element 0", "value [] at 3", "element 0", "value [] at 4",
					"run [] at 4 end", "Open: EOF",
				},
			},
			{
				name:   "returns kanon.ErrMax at the length of the element past the bound",
				schema: &boundedSchema,
				enc:    []byte{0x22, 0x03, 0x00, 0x00, 0x05},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 0", "value [] at 3", "element 0", "value [] at 4",
					"Element: kanon: Box.Items (field 4) at offset 4: more elements than the max of 2",
				},
			},
			{
				name:   "returns kanon.ErrMax for the elements of every occurrence of the field together",
				schema: &boundedSchema,
				enc:    []byte{0x22, 0x02, 0x00, 0x00, 0x22, 0x01, 0x00},
				want: []string{
					"run [] at 0", "open 4", "len 2", "element 0", "value [] at 3", "element 0", "value [] at 4",
					"run [] at 4", "open 4", "len 1",
					"Element: kanon: Box.Items (field 4) at offset 6: more elements than the max of 2",
				},
			},
			{
				name:   "counts the elements of each bounded slice apart",
				schema: &boundedSchema,
				enc:    []byte{0x22, 0x02, 0x00, 0x00, 0x32, 0x01, 0x00},
				want: []string{
					"run [] at 0", "open 4", "len 2", "element 0", "value [] at 3", "element 0", "value [] at 4",
					"run [] at 4", "open 6", "len 1", "element 0", "value [] at 7", "run [] at 7 end", "Open: EOF",
				},
			},
		})
		t.Run("returns the same length until Value reads the element", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x03, 0x02, 0x08, 0x01}})
			assert.Deterministic(t, func(struct{}) (int64, error) { return s.Element() }, struct{}{},
				"Element returns the same length before Value reads the element")
		})
		t.Run("returns io.EOF for a byte slice", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_, err := s.Element()
			assert.Equal(t, err, io.EOF, "Element reads no element of a byte slice", assert.ByIdentity())
		})
		t.Run("returns the error of the stream", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_ = s.Fail(errBroken)
			_, err := s.Element()
			assert.Equal(t, err, errBroken, "Element returns the error of the stream", assert.ByIdentity())
		})
	})
	t.Run("Value", func(t *testing.T) {
		t.Parallel()
		runCases(t, []streamCase{
			{
				name: "reads each element of a streamed slice that the caller skips",
				enc:  []byte{0x22, 0x04, 0x02, 0x08, 0x01, 0x00},
				skip: true,
				want: []string{
					"run [] at 0", "open 4", "len 4", "value [08 01] at 3", "value [] at 6", "run [] at 6 end",
					"Open: EOF",
				},
			},
			{
				name:   "returns kanon.ErrMax for an element past the bound that the caller skips",
				schema: &boundedSchema,
				enc:    []byte{0x22, 0x03, 0x00, 0x00, 0x00},
				skip:   true,
				want: []string{
					"run [] at 0", "open 4", "len 3", "value [] at 3", "value [] at 4",
					"Value: kanon: Box.Items (field 4) at offset 4: more elements than the max of 2",
				},
			},
			{
				name: "returns kanon.ErrDepth for an element under a Depth of 1",
				opts: kanon.StreamOptions{Depth: 1},
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2",
					"Value: kanon: Item at offset 3: nested deeper than the limit",
				},
			},
			{
				name: "returns kanon.ErrLimit for an element longer than the buffer limit at its length",
				opts: kanon.StreamOptions{Buffer: 1},
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2",
					"Value: kanon: Box.Items (field 4) at offset 2: value takes the buffer past 1 bytes",
				},
			},
			{
				name: "returns kanon.ErrDepth before kanon.ErrLimit",
				opts: kanon.StreamOptions{Depth: 1, Buffer: 1},
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2",
					"Value: kanon: Item at offset 3: nested deeper than the limit",
				},
			},
			{
				name: "reads an element of the length of the buffer limit",
				opts: kanon.StreamOptions{Buffer: 2},
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2", "value [08 01] at 3", "run [] at 5 end", "Open: EOF",
				},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends",
				enc:  []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				cut:  true, ends: 4,
				want: []string{
					"run [] at 0", "open 4", "len 3", "element 2",
					"Value: kanon: Box.Items (field 4) at offset 4: unexpected EOF",
				},
			},
			{
				name:   "returns the error of the reader",
				enc:    []byte{0x22, 0x03, 0x02, 0x08, 0x01},
				broken: true, ends: 3,
				want: []string{"run [] at 0", "open 4", "len 3", "element 2", "Value: disk: broken"},
			},
			{
				name: "returns io.ErrUnexpectedEOF where the reader ends inside an element longer than its buffer",
				enc:  longElement,
				cut:  true, ends: 3000,
				want: []string{
					"run [] at 0", "open 4", "len 5002", "element 5000",
					"Value: kanon: Box.Items (field 4) at offset 3000: unexpected EOF",
				},
			},
		})
		t.Run("returns an element longer than the buffer of the reader", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: longElement})
			r, err := s.Value(itemsNum)
			assert.NoError(t, err, "Value reads the element")
			expect.Equal(t, r.Data, longElement[5:], "the run is the element")
			expect.Equal(t, offset(s), 5, "the run is at the offset of the element")
		})
		t.Run("returns elements that continue past the buffer of the reader", func(t *testing.T) {
			t.Parallel()
			elements := bytes.Repeat([]byte{0x02, 0x08, 0x01}, 2000)
			s := opened(t, streamCase{enc: slices.Concat([]byte{0x22}, binary.AppendUvarint(nil, 6000), elements)})
			var got [][]byte
			for s.More(itemsNum) {
				r, err := s.Value(itemsNum)
				assert.NoError(t, err, "Value reads the element")
				got = append(got, slices.Clone(r.Data))
			}
			assert.Equal(t, got, slices.Repeat([][]byte{{0x08, 0x01}}, 2000), "Value returns every element")
		})
		t.Run("returns the depth of the struct of an element", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{opts: kanon.StreamOptions{Depth: 3}, enc: []byte{0x22, 0x01, 0x00}})
			r, err := s.Value(itemsNum)
			assert.NoError(t, err, "Value reads the element")
			assert.Equal(t, r.Depth, 1, "the element has the depth two levels below the struct type")
		})
		t.Run("returns io.EOF for another field", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_, err := s.Value(dataNum)
			assert.Equal(t, err, io.EOF, "Value reads no element for the number of another field", assert.ByIdentity())
		})
		t.Run("returns io.EOF after the last element", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_, err := s.Value(itemsNum)
			assert.NoError(t, err, "Value reads the element")
			_, err = s.Value(itemsNum)
			assert.Equal(t, err, io.EOF, "Value reads no element after the last", assert.ByIdentity())
		})
		t.Run("returns the error of the stream for another field", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_ = s.Fail(errBroken)
			_, err := s.Value(dataNum)
			assert.Equal(t, err, errBroken, "Value returns the error of the stream", assert.ByIdentity())
		})
	})
	t.Run("More", func(t *testing.T) {
		t.Parallel()
		t.Run("reports true for an element that Value has not read", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			assert.True(t, s.More(itemsNum), "More reports the element")
		})
		t.Run("reports true for an empty element whose length Element read", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_, err := s.Element()
			assert.NoError(t, err, "Element reads the length")
			assert.True(t, s.More(itemsNum), "More reports the element whose length Element read")
		})
		t.Run("reports false after the last element", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_, err := s.Value(itemsNum)
			assert.NoError(t, err, "Value reads the element")
			assert.False(t, s.More(itemsNum), "More reports no element after the last")
		})
		t.Run("reports false for another field", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			assert.False(t, s.More(dataNum), "More reports no element of another field")
		})
		t.Run("reports false after an error", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x01, 0x00}})
			_ = s.Fail(errBroken)
			assert.False(t, s.More(itemsNum), "More reports no element after an error")
		})
	})
	t.Run("Len", func(t *testing.T) {
		t.Parallel()
		t.Run("returns 0 before the first call of Next", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x12, 0x01, 0x61}})
			assert.Equal(t, s.Len(), int64(0), "Len returns 0 before a field is open")
		})
		t.Run("returns 0 after the end of the encoding", func(t *testing.T) {
			t.Parallel()
			s := start(streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_, err := transcript(s, &boxSchema, false)
			assert.Equal(t, err, io.EOF, "the stream reads the encoding to its end", assert.ByIdentity())
			assert.Equal(t, s.Len(), int64(0), "Len returns 0 after the end of the encoding")
		})
		t.Run("returns 0 after an error", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x12, 0x01, 0x61}})
			_ = s.Fail(errBroken)
			assert.Equal(t, s.Len(), int64(0), "Len returns 0 after an error")
		})
	})
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()
		t.Run("makes every call return kanon.ErrStreamSize for a negative size", func(t *testing.T) {
			t.Parallel()
			var s wire.Stream
			s.Init(&boxSchema, kanon.StreamOptions{})
			s.Reset(bytes.NewReader(nil), -1)
			_, err := s.Next()
			assert.Equal(t, err, kanon.ErrStreamSize, "Next returns kanon.ErrStreamSize", assert.ByIdentity())
			_, err = s.Next()
			assert.Equal(t, err, kanon.ErrStreamSize, "Next returns kanon.ErrStreamSize again", assert.ByIdentity())
		})
		t.Run("reads an encoding of the largest size", func(t *testing.T) {
			t.Parallel()
			var s wire.Stream
			s.Init(&boxSchema, kanon.StreamOptions{})
			s.Reset(bytes.NewReader(nil), math.MaxInt)
			_, err := s.Next()
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "Next reads the reader, which ends before the encoding")
		})
		t.Run("reads the next encoding from its first byte", func(t *testing.T) {
			t.Parallel()
			s := opened(t, streamCase{enc: []byte{0x22, 0x03, 0x02, 0x08, 0x01}})
			_, err := s.Element()
			assert.NoError(t, err, "Element reads the length")
			enc := []byte{0x08, 0x01, 0x12, 0x01, 0x61}
			s.Reset(bytes.NewReader(enc), int64(len(enc)))
			expect.Equal(t, s.Len(), int64(0), "Len returns 0 after Reset")
			_, err = s.Element()
			expect.Equal(t, err, io.EOF, "Element reads no element after Reset", expect.ByIdentity())
			got, _ := transcript(s, &boxSchema, false)
			want := []string{"run [08 01] at 0", "open 2", "len 1", "read [61]", "run [] at 5 end", "Open: EOF"}
			expect.Equal(t, got, want, "the stream reads the next encoding from its first byte")
		})
		t.Run("sets the counts of the elements of the bounded slices to 0", func(t *testing.T) {
			t.Parallel()
			enc := []byte{0x22, 0x02, 0x00, 0x00}
			s := start(streamCase{schema: &boundedSchema, enc: enc})
			_, err := transcript(s, &boundedSchema, false)
			assert.Equal(t, err, io.EOF, "the stream reads the elements up to the bound", assert.ByIdentity())
			s.Reset(bytes.NewReader(enc), int64(len(enc)))
			_, err = transcript(s, &boundedSchema, false)
			assert.Equal(t, err, io.EOF, "the stream reads the elements up to the bound again after Reset",
				assert.ByIdentity())
		})
	})
	t.Run("Init", func(t *testing.T) {
		t.Parallel()
		header := []byte{0x0a, 0x80, 0x80, 0x80, 0x08}
		limits := []struct {
			name   string
			buffer int
			want   string
		}{
			{
				name: "applies DefaultBuffer for a Buffer of 0", buffer: 0,
				want: "kanon: Box at offset 0: value takes the buffer past 16777216 bytes",
			},
			{
				name: "applies DefaultBuffer for a negative Buffer", buffer: -1,
				want: "kanon: Box at offset 0: value takes the buffer past 16777216 bytes",
			},
			{
				name: "applies a Buffer above DefaultBuffer", buffer: kanon.DefaultBuffer + len(header),
				want: "kanon: Box at offset 5: unexpected EOF",
			},
		}
		for _, c := range limits {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				var s wire.Stream
				s.Init(&boxSchema, kanon.StreamOptions{Buffer: c.buffer})
				s.Reset(bytes.NewReader(header), int64(len(header)+kanon.DefaultBuffer))
				_, err := transcript(&s, &boxSchema, false)
				assert.Equal(t, err.Error(), c.want, "the stream applies the buffer limit")
			})
		}
	})
}

// TestStreamAllocs checks that a stream that read an encoding before reads
// it again without an allocation. It runs serially: testing.AllocsPerRun
// panics while a parallel test runs.
func TestStreamAllocs(t *testing.T) {
	p := make([]byte, 2)
	cases := []struct {
		name  string
		value func(*wire.Stream) error
	}{
		{
			name:  "Next/allocates nothing for an encoding that the stream read before",
			value: func(s *wire.Stream) error { return readAll(s, p) },
		},
		{
			name: "WriteTo/allocates nothing for an encoding that the stream read before",
			value: func(s *wire.Stream) error {
				_, err := s.WriteTo(io.Discard)
				return err
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := bytes.NewReader(canonicalBox)
			var s wire.Stream
			s.Init(&canonicalSchema, kanon.StreamOptions{})
			pass := func() {
				r.Reset(canonicalBox)
				s.Reset(r, int64(len(canonicalBox)))
				sinkErr = drain(&s, c.value)
			}
			pass()
			assert.MaxAllocs(t, pass, 0, "a stream reads an encoding that it read before without an allocation")
		})
	}
	t.Run("Next/allocates nothing for the elements of a bounded slice that the stream read before", func(t *testing.T) {
		enc := []byte{0x22, 0x02, 0x00, 0x00}
		r := bytes.NewReader(enc)
		var s wire.Stream
		s.Init(&boundedSchema, kanon.StreamOptions{})
		pass := func() {
			r.Reset(enc)
			s.Reset(r, int64(len(enc)))
			sinkErr = drain(&s, func(*wire.Stream) error { return nil })
		}
		pass()
		assert.MaxAllocs(t, pass, 0, "a stream counts the elements of a bounded slice without an allocation")
	})
	inits := []struct {
		name   string
		schema *wire.StreamSchema
		allocs uint64
	}{
		{name: "Init/allocates nothing for a schema without a bound", schema: &boxSchema, allocs: 0},
		{name: "Init/allocates the counts once for a schema with a bound", schema: &boundedSchema, allocs: 1},
	}
	for _, c := range inits {
		t.Run(c.name, func(t *testing.T) {
			assert.MaxAllocs(t, func() {
				var s wire.Stream
				s.Init(c.schema, kanon.StreamOptions{})
			}, c.allocs, "Init allocates the counts of the elements for a schema with a bound alone")
		})
	}
}

func BenchmarkStream(b *testing.B) {
	values := make([]byte, 0, 1<<12)
	for range 1 << 10 {
		values = append(values, 0x08, 0x01)
	}
	data := slices.Concat([]byte{0x12, 0x80, 0x80, 0x04}, make([]byte, 1<<16))
	items := slices.Concat([]byte{0x22, 0x80, 0x18}, bytes.Repeat([]byte{0x02, 0x08, 0x01}, 1<<10))
	p := make([]byte, 1<<12)
	read := func(s *wire.Stream) error { return readAll(s, p) }
	cases := []struct {
		name  string
		enc   []byte
		value func(*wire.Stream) error
	}{
		{name: "Next", enc: values, value: read},
		{name: "Read", enc: data, value: read},
		{
			name: "WriteTo",
			enc:  data,
			value: func(s *wire.Stream) error {
				_, err := s.WriteTo(io.Discard)
				return err
			},
		},
		{name: "Value", enc: items, value: read},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			r := bytes.NewReader(c.enc)
			var s wire.Stream
			s.Init(&boxSchema, kanon.StreamOptions{})
			var err error
			bc := bench.Start(b).MaxAllocs(0).Warmup(1)
			defer bc.End()
			for bc.Loop() {
				r.Reset(c.enc)
				s.Reset(r, int64(len(c.enc)))
				err = drain(&s, c.value)
			}
			assert.Equal(b, err, io.EOF, "the stream reads the encoding to its end", assert.ByIdentity())
			b.SetBytes(int64(len(c.enc)))
		})
	}
}

// runCases runs each of cases as a subtest of t: a stream reads the encoding
// of the case, and the calls that [transcript] makes return the results of
// the case.
func runCases(t *testing.T, cases []streamCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, _ := transcript(start(c), cmp.Or(c.schema, &boxSchema), c.skip)
			assert.Equal(t, got, c.want, "the calls of the stream return the results of the encoding")
		})
	}
}

// start returns a stream with the schema of c, the schema of Box when c
// names none, and the options of c, over the reader of c, before its first
// call of Next.
func start(c streamCase) *wire.Stream {
	var r io.Reader = bytes.NewReader(c.enc)
	if c.cut || c.broken {
		r = bytes.NewReader(c.enc[:c.ends])
	}
	if c.broken {
		r = io.MultiReader(r, iotest.ErrReader(errBroken))
	}
	if c.single {
		r = iotest.OneByteReader(r)
	}
	if c.eof {
		r = iotest.DataErrReader(r)
	}
	s := new(wire.Stream)
	s.Init(cmp.Or(c.schema, &boxSchema), c.opts)
	s.Reset(r, int64(len(c.enc)))
	return s
}

// opened returns the stream of c after the call of Open that opens the
// streamed field at the start of the encoding of c. It fails tb when Next or
// Open fails.
func opened(tb testing.TB, c streamCase) *wire.Stream {
	tb.Helper()
	s := start(c)
	_, err := s.Next()
	assert.NoError(tb, err, "Next reads the encoding up to the streamed field")
	_, err = s.Open()
	assert.NoError(tb, err, "Open opens the streamed field")
	return s
}

// offset returns the offset in the encoding of the last run or element that
// s returned, which Fail adds to the offset of an error of its decode. It
// calls Fail on a copy of s, so s keeps its state.
func offset(s *wire.Stream) int {
	c, d := *s, new(kanon.DecodeError)
	_ = c.Fail(d)
	return d.Offset
}

// transcript drives s, a stream over an encoding of the struct type of
// schema, as the generated code and its caller drive it, and returns the
// results of the calls and the error that ends them: per call of Next the
// run that it returns, at its offset in the encoding and marked as the end
// when Open returns io.EOF after it, then the field that Open opens and its
// length, and the bytes of the field that Read returns
// two at a time, or the length and the bytes of each element that Element
// and Value return. A caller that skips the streamed values reads the
// elements that the generated code reads before it calls Next, with More
// and Value. The last result is the error, with the method that returned
// it.
func transcript(s *wire.Stream, schema *wire.StreamSchema, skip bool) ([]string, error) {
	var out []string
	for {
		r, err := s.Next()
		if err != nil {
			return append(out, "Next: "+err.Error()), err
		}
		event := fmt.Sprintf("run [% x] at %d", r.Data, offset(s))
		num, err := s.Open()
		if errors.Is(err, io.EOF) {
			event += " end"
		}
		out = append(out, event)
		if err != nil {
			return append(out, "Open: "+err.Error()), err
		}
		out = append(out, fmt.Sprintf("open %d", num), fmt.Sprintf("len %d", s.Len()))
		k := slices.IndexFunc(schema.Fields, func(f wire.StreamField) bool { return int(f.Tag>>3) == num })
		var events []string
		switch {
		case skip && schema.Fields[k].Elem != "":
			for s.More(num) {
				events, err = values(s, num, false)
				out = append(out, events...)
				if err != nil {
					return out, err
				}
			}
		case skip:
		case schema.Fields[k].Elem != "":
			events, err = values(s, num, true)
			out = append(out, events...)
		default:
			events, err = readValue(s)
			out = append(out, events...)
		}
		if err != nil {
			return out, err
		}
	}
}

// readValue reads the value of the current field of s, a byte slice or a
// string, two bytes at a time, and returns the bytes that it read and the
// error of Read that ends them, with the error itself.
func readValue(s *wire.Stream) ([]string, error) {
	var got []byte
	p := make([]byte, 2)
	for {
		n, err := s.Read(p)
		got = append(got, p[:n]...)
		if errors.Is(err, io.EOF) {
			return []string{fmt.Sprintf("read [% x]", got)}, nil
		}
		if err != nil {
			return []string{fmt.Sprintf("read [% x]", got), "Read: " + err.Error()}, err
		}
	}
}

// values reads the elements of the current field of s, the streamed slice
// with the number num, and returns the length and the bytes of each, or of
// the next alone when all is false, and the error that ends them, with the
// error itself.
func values(s *wire.Stream, num int, all bool) ([]string, error) {
	var out []string
	for {
		if all {
			l, err := s.Element()
			if errors.Is(err, io.EOF) {
				return out, nil
			}
			if err != nil {
				return append(out, "Element: "+err.Error()), err
			}
			out = append(out, fmt.Sprintf("element %d", l))
		}
		r, err := s.Value(num)
		if err != nil {
			return append(out, "Value: "+err.Error()), err
		}
		out = append(out, fmt.Sprintf("value [% x] at %d", r.Data, offset(s)))
		if !all {
			return out, nil
		}
	}
}

// drain reads the encoding of s to its end, as the generated code and its
// caller read it: the bytes of each streamed byte slice and string with
// value, and each element of each streamed slice. It returns the error that
// ends the encoding: io.EOF, or the first error of s.
func drain(s *wire.Stream, value func(*wire.Stream) error) error {
	for {
		if _, err := s.Next(); err != nil {
			return err
		}
		num, err := s.Open()
		if err != nil {
			return err
		}
		if err := value(s); err != nil {
			return err
		}
		for s.More(num) {
			if _, err := s.Value(num); err != nil {
				return err
			}
		}
	}
}

// readAll reads the value of the current field of s, a byte slice or a
// string, into p until Read returns io.EOF. It returns nil then, and the
// error of Read otherwise.
func readAll(s *wire.Stream, p []byte) error {
	for {
		if _, err := s.Read(p); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
