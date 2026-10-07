// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch_test

import (
	"math"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/batch"
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/number"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// oversized is a view.Item whose SizeKanon returns size, as the SizeKanon of
// a message with an encoding of size bytes does. The methods of view.Item
// implement the rest of kanon.Message.
type oversized struct {
	view.Item

	size int
}

// SizeKanon returns o.size.
func (o *oversized) SizeKanon() int { return o.size }

// failing is a message whose encode fails.
func failing() kanon.Message {
	return &codec.Codecs{Token: codec.TokenRevoked}
}

// write returns a Writer with Wide set to wide after it appended messages.
func write(t *testing.T, wide bool, messages ...kanon.Message) *batch.Writer {
	t.Helper()
	w := &batch.Writer{Wide: wide}
	assert.Total(t, w.Append, messages, "Append appends the message")
	return w
}

func TestWriter(t *testing.T) {
	t.Parallel()
	t.Run("Append", func(t *testing.T) {
		t.Parallel()
		t.Run("returns ErrFinalized after Bytes", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{})
			w.Bytes()
			assert.ErrorIs(t, w.Append(&view.Item{}), batch.ErrFinalized, "Append fails after Bytes")
			assert.Equal(t, w.Bytes(), oneEmpty, "the batch has one message")
		})
		t.Run("returns ErrNilMessage for a nil message", func(t *testing.T) {
			t.Parallel()
			w := write(t, false)
			assert.ErrorIs(t, w.Append(nil), batch.ErrNilMessage, "Append rejects nil")
			assert.Equal(t, w.Len(), 0, "the writer has no messages")
		})
		t.Run("returns ErrNilMessage for a nil pointer", func(t *testing.T) {
			t.Parallel()
			w := write(t, false)
			assert.ErrorIs(t, w.Append((*view.Item)(nil)), batch.ErrNilMessage, "Append rejects a nil pointer")
			assert.Equal(t, w.Len(), 0, "the writer has no messages")
		})
		t.Run("returns the error of an encode that fails without a change to the batch", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{})
			assert.ErrorIs(t, w.Append(failing()), codec.ErrTokenRevoked, "Append returns the error of the encode")
			assert.Equal(t, w.Bytes(), oneEmpty, "the batch has the message before the failure")
		})
		t.Run("returns ErrTooBig for encodings longer than 2^32-1 bytes with 4-byte offsets", func(t *testing.T) {
			t.Parallel()
			w := write(t, false)
			err := w.Append(&oversized{size: min(math.MaxInt, 1<<32)})
			assert.ErrorIs(t, err, batch.ErrTooBig, "Append rejects the message")
			assert.Equal(t, w.Bytes(), emptyBatch, "the batch has no messages")
		})
		t.Run("returns ErrTooBig for a batch longer than math.MaxInt bytes", func(t *testing.T) {
			t.Parallel()
			w := write(t, true)
			// With the header, one 8-byte offset and the count, the batch is
			// math.MaxInt+1 bytes long.
			err := w.Append(&oversized{size: math.MaxInt - 13})
			assert.ErrorIs(t, err, batch.ErrTooBig, "Append rejects the message")
			assert.Equal(t, w.Bytes(), emptyWide, "the batch has no messages")
		})
		t.Run("reads Wide at the first successful Append", func(t *testing.T) {
			t.Parallel()
			w := write(t, true)
			assert.ErrorIs(t, w.Append(failing()), codec.ErrTokenRevoked, "the first Append fails")
			w.Wide = false
			assert.NoError(t, w.Append(&view.Item{}), "Append appends the message")
			assert.Equal(t, w.Bytes(), oneEmpty, "the batch has 4-byte offsets")
		})
		t.Run("keeps the width of the first successful Append", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{})
			w.Wide = true
			assert.NoError(t, w.Append(&view.Item{}), "Append appends the message")
			assert.Equal(t, w.Bytes(), twoEmpty, "the batch has 4-byte offsets")
		})
	})
	t.Run("Len", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the number of appended messages", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{}, &view.Item{})
			assert.Equal(t, w.Len(), 2, "the writer has two messages")
		})
	})
	t.Run("Bytes", func(t *testing.T) {
		t.Parallel()
		vectors := []struct {
			name     string
			wide     bool
			messages []kanon.Message
			want     []byte
		}{
			{name: "an empty batch", want: emptyBatch},
			{name: "an empty batch with 8-byte offsets", wide: true, want: emptyWide},
			{name: "one empty encoding", messages: []kanon.Message{&view.Item{}}, want: oneEmpty},
			{name: "two empty encodings", messages: []kanon.Message{&view.Item{}, &view.Item{}}, want: twoEmpty},
			{name: "one encoding", messages: []kanon.Message{&number.Skipped{First: 1}}, want: oneRecord},
			{
				name: "one empty encoding with 8-byte offsets", wide: true,
				messages: []kanon.Message{&view.Item{}}, want: oneEmptyWide,
			},
			{
				name:     "encodings at nondecreasing offsets",
				messages: []kanon.Message{&items[0], &items[1], &items[2]}, want: threeItems,
			},
		}
		for _, tt := range vectors {
			t.Run("returns the vector of "+tt.name, func(t *testing.T) {
				t.Parallel()
				w := write(t, tt.wide, tt.messages...)
				assert.Equal(t, w.Bytes(), tt.want, "the batch has the bytes of the vector")
			})
		}
		t.Run("returns the same batch on a second call", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{})
			w.Bytes()
			assert.Equal(t, w.Bytes(), oneEmpty, "the batch has one index")
		})
		t.Run("returns a batch that Parse reads back", func(t *testing.T) {
			t.Parallel()
			w := write(t, true, &items[0], &items[1], &items[2])
			b, err := batch.Parse(w.Bytes())
			assert.NoError(t, err, "Parse reads the batch")
			got := make([]view.Item, b.Len())
			for i := range got {
				assert.NoError(t, b.Decode(i, &got[i]), "Decode decodes the message")
			}
			assert.Equal(t, got, items, "the batch returns the appended messages")
		})
	})
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()
		t.Run("empties the writer", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{}, &view.Item{})
			w.Bytes()
			w.Reset()
			assert.Equal(t, w.Len(), 0, "the writer has no messages")
			assert.Equal(t, w.Bytes(), emptyBatch, "the batch has no messages")
		})
		t.Run("starts a batch that Append adds to after Bytes", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{}, &view.Item{})
			w.Bytes()
			w.Reset()
			assert.NoError(t, w.Append(&view.Item{}), "Append appends the message")
			assert.Equal(t, w.Bytes(), oneEmpty, "the batch has the message after Reset")
		})
		t.Run("reads Wide again at the next Append", func(t *testing.T) {
			t.Parallel()
			w := write(t, false, &view.Item{})
			w.Reset()
			w.Wide = true
			assert.NoError(t, w.Append(&view.Item{}), "Append appends the message")
			assert.Equal(t, w.Bytes(), oneEmptyWide, "the batch has 8-byte offsets")
		})
	})
}

func TestWriterAllocs(t *testing.T) {
	t.Run("Append", func(t *testing.T) {
		t.Run("allocates nothing with Bytes after a batch at least as long", func(t *testing.T) {
			var w batch.Writer
			failed := 0
			build := func() {
				w.Reset()
				for i := range items {
					if w.Append(&items[i]) != nil {
						failed++
					}
				}
				w.Bytes()
			}
			assert.MaxAllocs(t, build, 0, "Append and Bytes allocate nothing")
			assert.Equal(t, failed, 0, "every measured Append appends the message")
			assert.Equal(t, w.Bytes(), threeItems, "the writer builds the batch")
		})
	})
}
