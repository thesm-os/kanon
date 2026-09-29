// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/batch"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// The lengths of the batch layout and its flag of 8-byte offsets, which the
// fuzz target checks a batch against.
const (
	// headerSize is the length of the version and the flags.
	headerSize = 2
	// countSize is the length of the count.
	countSize = 4
	// narrowOffset is the length of an offset without flagWide.
	narrowOffset = 4
	// wideOffset is the length of an offset with flagWide.
	wideOffset = 8
	// flagWide is the bit of the flags byte that selects 8-byte offsets.
	flagWide = 1
)

// Vectors of the batch layout. Counts and offsets are little-endian.
var (
	// emptyBatch has no messages and 4-byte offsets.
	emptyBatch = []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
	// emptyWide has no messages and 8-byte offsets.
	emptyWide = []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00}
	// oneEmpty has one empty encoding and 4-byte offsets.
	oneEmpty = []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
	// twoEmpty has two empty encodings and 4-byte offsets.
	twoEmpty = []byte{
		0x01, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x00,
	}
	// oneRecord has the encoding 08 02, of number.Skipped{First: 1}, and
	// 4-byte offsets.
	oneRecord = []byte{0x01, 0x00, 0x08, 0x02, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}
	// oneEmptyWide has one empty encoding and 8-byte offsets.
	oneEmptyWide = []byte{
		0x01, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00,
	}
	// threeItems has the encodings of the view.Item values of items at the
	// offsets 0, 3 and 3.
	threeItems = []byte{
		0x01, 0x00,
		0x0a, 0x01, 0x61,
		0x0a, 0x02, 0x62, 0x63,
		0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x00, 0x00,
	}
	// badMiddle is threeItems with the empty encoding replaced by 0a 05, a
	// string whose length, at offset 6 of the batch, runs past the
	// encoding.
	badMiddle = []byte{
		0x01, 0x00,
		0x0a, 0x01, 0x61,
		0x0a, 0x05,
		0x0a, 0x02, 0x62, 0x63,
		0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00,
		0x03, 0x00, 0x00, 0x00,
	}
)

// items are the messages of threeItems.
var items = []view.Item{{Name: "a"}, {}, {Name: "bc"}}

// records returns the records of b.
func records(b batch.Batch) [][]byte {
	out := make([][]byte, b.Len())
	for i := range out {
		out[i] = b.Record(i)
	}
	return out
}

func TestBatch(t *testing.T) {
	t.Parallel()
	vectors := []struct {
		name string
		in   []byte
		want [][]byte
	}{
		{name: "an empty batch", in: emptyBatch, want: [][]byte{}},
		{name: "an empty batch with 8-byte offsets", in: emptyWide, want: [][]byte{}},
		{name: "one empty encoding", in: oneEmpty, want: [][]byte{{}}},
		{name: "two empty encodings", in: twoEmpty, want: [][]byte{{}, {}}},
		{name: "one encoding", in: oneRecord, want: [][]byte{{0x08, 0x02}}},
		{name: "one empty encoding with 8-byte offsets", in: oneEmptyWide, want: [][]byte{{}}},
		{
			name: "encodings at nondecreasing offsets", in: threeItems,
			want: [][]byte{{0x0a, 0x01, 0x61}, {}, {0x0a, 0x02, 0x62, 0x63}},
		},
	}
	failures := []struct {
		name string
		in   []byte
		want error
	}{
		{name: "returns ErrLayout for a version byte alone", in: []byte{0x01}, want: batch.ErrLayout},
		{name: "returns ErrVersion for a version other than 1", in: []byte{0x02, 0x00}, want: batch.ErrVersion},
		{
			name: "returns ErrFlags for a flag above bit 0",
			in:   []byte{0x01, 0x02, 0x00, 0x00, 0x00, 0x00}, want: batch.ErrFlags,
		},
		{name: "returns ErrLayout for a header without a count", in: []byte{0x01, 0x00}, want: batch.ErrLayout},
		{
			name: "returns ErrLayout for a count of 2^32-1 without an index",
			in:   []byte{0x01, 0x00, 0xff, 0xff, 0xff, 0xff}, want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for an index of 8-byte offsets longer than the batch",
			in:   []byte{0x01, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}, want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for a first offset past the encodings",
			in:   []byte{0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}, want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for a first offset other than 0 within the encodings",
			in:   []byte{0x01, 0x00, 0x0a, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00}, want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for a decreasing offset",
			in: []byte{
				0x01, 0x00, 0x0a, 0x00,
				0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
				0x03, 0x00, 0x00, 0x00,
			},
			want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for an offset past the encodings",
			in: []byte{
				0x01, 0x00, 0x0a,
				0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00,
				0x02, 0x00, 0x00, 0x00,
			},
			want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for an 8-byte offset of 2^64-1",
			in: []byte{
				0x01, 0x01,
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
				0x02, 0x00, 0x00, 0x00,
			},
			want: batch.ErrLayout,
		},
		{
			name: "returns ErrLayout for encodings in a batch without messages",
			in:   []byte{0x01, 0x00, 0x0a, 0x00, 0x00, 0x00, 0x00}, want: batch.ErrLayout,
		},
	}
	t.Run("Parse", func(t *testing.T) {
		t.Parallel()
		for _, tt := range vectors {
			t.Run("reads the vector of "+tt.name, func(t *testing.T) {
				t.Parallel()
				b, err := batch.Parse(tt.in)
				assert.NoError(t, err, "Parse reads the batch")
				assert.Equal(t, records(b), tt.want, "the batch has the encodings of the vector")
			})
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := batch.Parse(tt.in)
				assert.ErrorIs(t, err, tt.want, "Parse fails for the reason of the batch")
			})
		}
		t.Run("returns records that do not alias the input", func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(threeItems)
			b, err := batch.Parse(in)
			assert.NoError(t, err, "Parse reads the batch")
			in[4] = 'z'
			assert.Equal(t, b.Record(0), []byte{0x0a, 0x01, 0x61}, "the record keeps its bytes")
		})
		t.Run("returns a Batch whose decoded strings do not alias the input", func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(threeItems)
			b, err := batch.Parse(in)
			assert.NoError(t, err, "Parse reads the batch")
			var got view.Item
			assert.NoError(t, b.Decode(0, &got), "Decode decodes the first message")
			in[4] = 'z'
			assert.Equal(t, got.Name, "a", "the decoded string keeps its bytes")
		})
	})
	t.Run("ParseAlias", func(t *testing.T) {
		t.Parallel()
		for _, tt := range vectors {
			t.Run("reads the vector of "+tt.name, func(t *testing.T) {
				t.Parallel()
				b, err := batch.ParseAlias(tt.in)
				assert.NoError(t, err, "ParseAlias reads the batch")
				assert.Equal(t, records(b), tt.want, "the batch has the encodings of the vector")
			})
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := batch.ParseAlias(tt.in)
				assert.ErrorIs(t, err, tt.want, "ParseAlias fails for the reason of the batch")
			})
		}
		t.Run("returns records that alias the input", func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(threeItems)
			b, err := batch.ParseAlias(in)
			assert.NoError(t, err, "ParseAlias reads the batch")
			in[4] = 'z'
			assert.Equal(t, b.Record(0), []byte{0x0a, 0x01, 'z'}, "the record shows the change to the input")
		})
		t.Run("returns a Batch whose decoded strings alias the input", func(t *testing.T) {
			t.Parallel()
			in := slices.Clone(threeItems)
			b, err := batch.ParseAlias(in)
			assert.NoError(t, err, "ParseAlias reads the batch")
			var got view.Item
			assert.NoError(t, b.Decode(0, &got), "Decode decodes the first message")
			in[4] = 'z'
			assert.Equal(t, got.Name, "z", "the decoded string shows the change to the input")
		})
	})
	t.Run("Len", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the number of messages", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			assert.Equal(t, b.Len(), 3, "the batch has three messages")
		})
		t.Run("returns 0 for the zero Batch", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, batch.Batch{}.Len(), 0, "the zero Batch has no messages")
		})
	})
	t.Run("Width", func(t *testing.T) {
		t.Parallel()
		widths := []struct {
			name string
			in   []byte
			want int
		}{
			{name: "returns 4 for a batch without the flag of 8-byte offsets", in: oneEmpty, want: narrowOffset},
			{name: "returns 8 for a batch with the flag of 8-byte offsets", in: oneEmptyWide, want: wideOffset},
		}
		for _, tt := range widths {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				b, err := batch.ParseAlias(tt.in)
				assert.NoError(t, err, "ParseAlias reads the batch")
				assert.Equal(t, b.Width(), tt.want, "Width returns the length of an offset")
			})
		}
		t.Run("returns 0 for the zero Batch", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, batch.Batch{}.Width(), 0, "the zero Batch has no offsets")
		})
	})
	t.Run("Offset", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the offset of each encoding in the batch", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			offsets := make([]int, b.Len())
			for i := range offsets {
				offsets[i] = b.Offset(i)
			}
			assert.Equal(t, offsets, []int{2, 5, 5}, "the encodings follow the header at the offsets of the index")
		})
		t.Run("returns the offset of an encoding with 8-byte offsets", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(oneEmptyWide)
			assert.NoError(t, err, "ParseAlias reads the batch")
			assert.Equal(t, b.Offset(0), headerSize, "the encoding follows the header")
		})
		t.Run("panics for an index equal to Len", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			got := assert.Panics(t, func() { b.Offset(3) }, "Offset panics")
			assert.Equal(t, fmt.Sprint(got), "runtime error: index out of range [3] with length 3",
				"the panic is that of a slice index")
		})
	})
	t.Run("Record", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a record without capacity past its end", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			record := b.Record(0)
			assert.Equal(t, cap(record), len(record), "an append to the record cannot change the next one")
		})
		panics := []struct {
			name  string
			index int
			want  string
		}{
			{
				name: "panics for an index equal to Len", index: 3,
				want: "runtime error: index out of range [3] with length 3",
			},
			{name: "panics for a negative index", index: -1, want: "runtime error: index out of range [-1]"},
		}
		for _, tt := range panics {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				b, err := batch.ParseAlias(threeItems)
				assert.NoError(t, err, "ParseAlias reads the batch")
				got := assert.Panics(t, func() { b.Record(tt.index) }, "Record panics")
				assert.Equal(t, fmt.Sprint(got), tt.want, "the panic is that of a slice index")
			})
		}
		t.Run("panics for any index of the zero Batch", func(t *testing.T) {
			t.Parallel()
			got := assert.Panics(t, func() { batch.Batch{}.Record(0) }, "Record panics")
			assert.Equal(t, fmt.Sprint(got), "runtime error: index out of range [0] with length 0",
				"the panic is that of a slice index")
		})
	})
	t.Run("Decode", func(t *testing.T) {
		t.Parallel()
		t.Run("decodes each message", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			got := make([]view.Item, b.Len())
			for i := range got {
				assert.NoError(t, b.Decode(i, &got[i]), "Decode decodes the message")
			}
			assert.Equal(t, got, items, "the messages are those of the batch")
		})
		t.Run("returns ErrNilMessage for a nil message", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			assert.ErrorIs(t, b.Decode(0, nil), batch.ErrNilMessage, "Decode rejects nil")
		})
		t.Run("returns ErrNilMessage for a nil pointer", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			assert.ErrorIs(t, b.Decode(0, (*view.Item)(nil)), batch.ErrNilMessage, "Decode rejects a nil pointer")
		})
		t.Run("panics for an index out of range", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(threeItems)
			assert.NoError(t, err, "ParseAlias reads the batch")
			var got view.Item
			panicked := assert.Panics(t, func() { _ = b.Decode(3, &got) }, "Decode panics")
			assert.Equal(t, fmt.Sprint(panicked), "runtime error: index out of range [3] with length 3",
				"the panic is that of a slice index")
		})
		t.Run("returns the error of the decode with offsets in the batch", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(badMiddle)
			assert.NoError(t, err, "ParseAlias reads the batch")
			var got view.Item
			decodeErr := assert.ErrorAs[*kanon.DecodeError](t, b.Decode(1, &got), "Decode returns the decode error")
			assert.Equal(t, decodeErr.Offset, 6, "the error is at the length of the string in the batch")
		})
		t.Run("decodes the other messages of a batch with a malformed encoding", func(t *testing.T) {
			t.Parallel()
			b, err := batch.ParseAlias(badMiddle)
			assert.NoError(t, err, "ParseAlias reads the batch")
			var first, last view.Item
			assert.NoError(t, b.Decode(0, &first), "Decode decodes the first message")
			assert.NoError(t, b.Decode(2, &last), "Decode decodes the last message")
			assert.Equal(t, []view.Item{first, last}, []view.Item{items[0], items[2]}, "both messages decode")
		})
	})
}

func TestBatchAllocs(t *testing.T) {
	t.Run("ParseAlias", func(t *testing.T) {
		t.Run("allocates nothing with Width, Offset, Record and Decode", func(t *testing.T) {
			got := make([]view.Item, len(items))
			failed := 0
			read := func() {
				b, err := batch.ParseAlias(threeItems)
				if err != nil || b.Width() != narrowOffset {
					failed++
				}
				for i := range b.Len() {
					_ = b.Offset(i)
					_ = b.Record(i)
					if b.Decode(i, &got[i]) != nil {
						failed++
					}
				}
			}
			assert.MaxAllocs(t, read, 0, "ParseAlias, Width, Offset, Record and Decode allocate nothing")
			assert.Equal(t, failed, 0, "every measured call reads the batch")
			assert.Equal(t, got, items, "the decodes return the messages")
		})
	})
	t.Run("Parse", func(t *testing.T) {
		t.Run("allocates the copy of the batch", func(t *testing.T) {
			failed := 0
			parse := func() {
				if _, err := batch.Parse(threeItems); err != nil {
					failed++
				}
			}
			assert.MaxAllocs(t, parse, 1, "Parse allocates one copy")
			assert.Equal(t, failed, 0, "every measured call reads the batch")
		})
	})
}

func FuzzParse(f *testing.F) {
	for _, seed := range [][]byte{emptyBatch, emptyWide, oneEmpty, twoEmpty, oneRecord, oneEmptyWide, threeItems} {
		f.Add(seed)
	}
	known := []error{batch.ErrVersion, batch.ErrFlags, batch.ErrLayout}
	f.Fuzz(func(t *testing.T, data []byte) {
		aliased, err := batch.ParseAlias(data)
		copied, copyErr := batch.Parse(data)
		assert.True(t, errors.Is(copyErr, err), "Parse and ParseAlias return the same error")
		if err != nil {
			isKnown := slices.ContainsFunc(known, func(e error) bool { return errors.Is(err, e) })
			assert.True(t, isKnown, "ParseAlias fails with an error of the package")
			return
		}
		assert.Equal(t, records(copied), records(aliased), "Parse and ParseAlias return the same records")
		width := narrowOffset
		if data[1]&flagWide != 0 {
			width = wideOffset
		}
		assert.Equal(t, aliased.Width(), width, "Width returns the width that the flags select")
		length := headerSize + aliased.Len()*width + countSize
		offset := headerSize
		for i := range aliased.Len() {
			assert.Equal(t, aliased.Offset(i), offset, "each record starts where the one before it ends")
			offset += len(aliased.Record(i))
			length += len(aliased.Record(i))
			var got view.Item
			_ = aliased.Decode(i, &got)
		}
		assert.Equal(t, length, len(data), "the header, the records, the index and the count tile the batch")
	})
}
