// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"bytes"
	"errors"
	"io"
	"math"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/frame"
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// errStream is the error of the streams of the tests that fail.
var errStream = errors.New("frame_test: the stream fails")

// oversized is a view.Item whose SizeKanon returns size, as the SizeKanon of
// a message with an encoding of size bytes does. The methods of view.Item
// implement the rest of kanon.Message.
type oversized struct {
	view.Item

	size int
}

// SizeKanon returns o.size.
func (o *oversized) SizeKanon() int { return o.size }

// shortWriter is a stream that writes one byte less than it is given and
// returns no error.
type shortWriter struct{}

// Write reports one byte less than len(p).
func (shortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

// failingWriter is a stream whose every write fails with errStream.
type failingWriter struct{}

// Write returns errStream.
func (failingWriter) Write([]byte) (int, error) { return 0, errStream }

func TestWriter(t *testing.T) {
	t.Parallel()
	t.Run("Write", func(t *testing.T) {
		t.Parallel()
		t.Run("writes a length of two bytes for a frame of 128 bytes", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			name := strings.Repeat("a", 123)
			assert.NoError(t, frame.NewWriter(&out).Write(typeID, &view.Item{Name: name}), "Write writes the frame")
			want := slices.Concat([]byte{0x80, 0x01, 0x01, 0x00, 0x01, 0x0a, 0x7b}, []byte(name))
			assert.Equal(t, out.Bytes(), want, "the length 128 takes two bytes")
		})
		t.Run("writes a second frame over the buffer of the first", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			w := frame.NewWriter(&out)
			assert.NoError(t, w.Write(typeID, &view.Item{Name: "long"}), "Write writes the first frame")
			assert.NoError(t, w.Write(typeID, &view.Item{Name: "a"}), "Write writes the second frame")
			want := slices.Concat([]byte{0x09, 0x01, 0x00, 0x01, 0x0a, 0x04, 'l', 'o', 'n', 'g'}, itemFrame)
			assert.Equal(t, out.Bytes(), want, "each frame has its own bytes")
		})
		t.Run("returns ErrNilMessage for a nil message", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			assert.ErrorIs(t, frame.NewWriter(&out).Write(typeID, nil), frame.ErrNilMessage, "Write rejects nil")
			assert.Equal(t, out.Len(), 0, "Write writes nothing")
		})
		t.Run("returns ErrNilMessage for a nil pointer", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := frame.NewWriter(&out).Write(typeID, (*view.Item)(nil))
			assert.ErrorIs(t, err, frame.ErrNilMessage, "Write rejects a nil pointer")
			assert.Equal(t, out.Len(), 0, "Write writes nothing")
		})
		t.Run("returns the error of an encode that fails", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			err := frame.NewWriter(&out).Write(typeID, &codec.Codecs{Token: codec.TokenRevoked})
			assert.ErrorIs(t, err, codec.ErrTokenRevoked, "Write returns the error of the encode")
			assert.Equal(t, out.Len(), 0, "Write writes nothing")
		})
		t.Run("returns ErrTooLarge for a frame longer than math.MaxInt bytes", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			w := frame.NewWriter(&out)
			w.Checksum = true
			err := w.Write(typeID, &oversized{size: math.MaxInt - 10})
			assert.ErrorIs(t, err, frame.ErrTooLarge, "Write rejects the frame")
			assert.Equal(t, out.Len(), 0, "Write writes nothing")
		})
		t.Run("returns the error of the stream", func(t *testing.T) {
			t.Parallel()
			err := frame.NewWriter(failingWriter{}).Write(typeID, &view.Item{})
			assert.ErrorIs(t, err, errStream, "Write returns the error of the stream")
		})
		t.Run("returns io.ErrShortWrite for a stream that writes part of the frame", func(t *testing.T) {
			t.Parallel()
			err := frame.NewWriter(shortWriter{}).Write(typeID, &view.Item{})
			assert.ErrorIs(t, err, io.ErrShortWrite, "Write reports the short write")
		})
	})
}

func TestWriterAllocs(t *testing.T) {
	t.Run("Write", func(t *testing.T) {
		t.Run("allocates nothing after a frame at least as long", func(t *testing.T) {
			w := frame.NewWriter(io.Discard)
			w.Checksum = true
			item := view.Item{Name: "kanon", Count: 7}
			assert.NoError(t, w.Write(typeID, &item), "Write writes the first frame")
			failed := 0
			write := func() {
				if w.Write(typeID, &item) != nil {
					failed++
				}
			}
			assert.MaxAllocs(t, write, 0, "Write allocates nothing")
			assert.Equal(t, failed, 0, "every measured Write writes the frame")
		})
	})
}
