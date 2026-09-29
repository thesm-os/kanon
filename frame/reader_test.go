// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/frame"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// Inputs of the tests of Next, one for each check of a frame.
var (
	// cutLength is a length whose continuation bit is set at the end of
	// the stream.
	cutLength = []byte{0x80}
	// cutFrame declares 3 bytes after its length and has 2.
	cutFrame = []byte{0x03, 0x01, 0x00}
	// tenContinued is ten length bytes with the continuation bit set.
	tenContinued = bytes.Repeat([]byte{0x80}, 10)
	// tenthAboveOne is a length whose tenth byte is 0x7f, above 64 bits.
	tenthAboveOne = append(bytes.Repeat([]byte{0x80}, 9), 0x7f)
	// tenthOne is a length of 2^63, whose tenth byte is 1.
	tenthOne = append(bytes.Repeat([]byte{0x80}, 9), 0x01)
	// threeByteLength is the length 16512, 2^14+2^7, in three bytes.
	threeByteLength = []byte{0x80, 0x81, 0x01}
	// defaultLength is the length DefaultMaxSize, 2^24, without a frame.
	defaultLength = []byte{0x80, 0x80, 0x80, 0x08}
	// aboveDefaultLength is the length DefaultMaxSize+1 without a frame.
	aboveDefaultLength = []byte{0x81, 0x80, 0x80, 0x08}
	// noFlags is a frame of a version byte alone.
	noFlags = []byte{0x01, 0x01}
	// versionTwo is a frame of version 2 without a type.
	versionTwo = []byte{0x02, 0x02, 0x00}
	// badVersion is emptyFrame with version 2.
	badVersion = []byte{0x03, 0x02, 0x00, 0x01}
	// badFlags is emptyFrame with flag bit 1 set.
	badFlags = []byte{0x03, 0x01, 0x02, 0x01}
	// cutType is a frame whose type uvarint runs past the frame.
	cutType = []byte{0x03, 0x01, 0x00, 0x80}
	// wideType is a frame whose type uvarint has a tenth byte of 2, above
	// 64 bits.
	wideType = slices.Concat([]byte{0x0c, 0x01, 0x00}, bytes.Repeat([]byte{0x80}, 9), []byte{0x02})
	// shortChecksum is a checksummed frame with 1 byte for the checksum.
	shortChecksum = []byte{0x04, 0x01, 0x01, 0x01, 0x00}
	// badChecksum is checkedFrame with the last byte of its checksum
	// changed.
	badChecksum = []byte{0x07, 0x01, 0x01, 0x01, 0x70, 0x2a, 0xec, 0x25}
	// cutPayload is a frame whose payload view.Item cannot decode: a
	// string whose length, at offset 1 of the payload, runs past the
	// payload.
	cutPayload = []byte{0x05, 0x01, 0x00, 0x01, 0x0a, 0x05}
)

// failingReader is a stream whose every read fails with errStream.
type failingReader struct{}

// Read returns errStream.
func (failingReader) Read([]byte) (int, error) { return 0, errStream }

// ReadByte returns errStream.
func (failingReader) ReadByte() (byte, error) { return 0, errStream }

// loop is a stream that repeats one frame without end, and allocates
// nothing to read.
type loop struct {
	frame []byte
	i     int
}

// Read copies the next bytes of the repeated frame into p.
func (l *loop) Read(p []byte) (int, error) {
	for k := range p {
		p[k], _ = l.ReadByte()
	}
	return len(p), nil
}

// ReadByte returns the next byte of the repeated frame.
func (l *loop) ReadByte() (byte, error) {
	b := l.frame[l.i]
	l.i = (l.i + 1) % len(l.frame)
	return b, nil
}

func TestReader(t *testing.T) {
	t.Parallel()
	t.Run("NewReader", func(t *testing.T) {
		t.Parallel()
		t.Run("reads an io.ByteReader up to the end of the frame", func(t *testing.T) {
			t.Parallel()
			in := bytes.NewReader(slices.Concat(emptyFrame, itemFrame))
			_, _, err := frame.NewReader(in).Next()
			assert.NoError(t, err, "Next reads the first frame")
			assert.Equal(t, in.Len(), len(itemFrame), "the stream is at the second frame")
		})
		t.Run("reads an io.Reader through a buffer that reads ahead", func(t *testing.T) {
			t.Parallel()
			in := bytes.NewReader(slices.Concat(emptyFrame, itemFrame))
			r := frame.NewReader(struct{ io.Reader }{in})
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the first frame")
			assert.Equal(t, in.Len(), 0, "the buffer has read the second frame from the stream")
			_, payload, err := r.Next()
			assert.NoError(t, err, "Next reads the second frame from the buffer")
			assert.Equal(t, payload, []byte{0x0a, 0x01, 0x61}, "the second frame has its payload")
		})
	})
	t.Run("Next", func(t *testing.T) {
		t.Parallel()
		failures := []struct {
			name    string
			in      []byte
			maxSize int
			want    error
		}{
			{name: "returns io.EOF at the end of the stream between frames", want: io.EOF},
			{
				name: "returns io.ErrUnexpectedEOF for a stream that ends inside a length",
				in:   cutLength, want: io.ErrUnexpectedEOF,
			},
			{
				name: "returns io.ErrUnexpectedEOF for a stream that ends before the declared length",
				in:   cutFrame, want: io.ErrUnexpectedEOF,
			},
			{
				name: "returns ErrMalformed for ten length bytes with the continuation bit",
				in:   tenContinued, want: frame.ErrMalformed,
			},
			{name: "returns ErrMalformed for a tenth length byte above 1", in: tenthAboveOne, want: frame.ErrMalformed},
			{name: "returns ErrTooLarge for a length of 2^63", in: tenthOne, want: frame.ErrTooLarge},
			{
				name: "returns ErrTooLarge for a length above MaxSize",
				in:   emptyFrame, maxSize: 2, want: frame.ErrTooLarge,
			},
			{
				name: "returns ErrTooLarge for a length of three bytes above MaxSize",
				in:   threeByteLength, maxSize: 16511, want: frame.ErrTooLarge,
			},
			{
				name: "returns ErrTooLarge for a length above DefaultMaxSize for a negative MaxSize",
				in:   aboveDefaultLength, maxSize: -1, want: frame.ErrTooLarge,
			},
			{name: "returns ErrMalformed for a frame without flags", in: noFlags, want: frame.ErrMalformed},
			{name: "returns ErrVersion for a frame of a version other than 1", in: versionTwo, want: frame.ErrVersion},
			{name: "returns ErrFlags for a flag above bit 0", in: badFlags, want: frame.ErrFlags},
			{name: "returns ErrMalformed for a type that runs past the frame", in: cutType, want: frame.ErrMalformed},
			{name: "returns ErrMalformed for a type above 64 bits", in: wideType, want: frame.ErrMalformed},
			{
				name: "returns ErrMalformed for a frame too short for its checksum",
				in:   shortChecksum, want: frame.ErrMalformed,
			},
			{name: "returns ErrChecksum for a checksum that differs", in: badChecksum, want: frame.ErrChecksum},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := frame.NewReader(bytes.NewReader(tt.in))
				r.MaxSize = tt.maxSize
				_, _, err := r.Next()
				assert.ErrorIs(t, err, tt.want, "Next fails for the reason of the frame")
			})
		}
		after := []struct {
			name string
			bad  []byte
			want error
		}{
			{name: "reads the next frame after ErrVersion", bad: badVersion, want: frame.ErrVersion},
			{name: "reads the next frame after ErrFlags", bad: badFlags, want: frame.ErrFlags},
			{name: "reads the next frame after ErrChecksum", bad: badChecksum, want: frame.ErrChecksum},
		}
		for _, tt := range after {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := frame.NewReader(bytes.NewReader(slices.Concat(tt.bad, itemFrame)))
				_, _, err := r.Next()
				assert.ErrorIs(t, err, tt.want, "Next fails for the first frame")
				_, payload, err := r.Next()
				assert.NoError(t, err, "Next reads the second frame")
				assert.Equal(t, payload, []byte{0x0a, 0x01, 0x61}, "the second frame has its own payload")
			})
		}
		t.Run("reads a frame of length MaxSize", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(emptyFrame))
			r.MaxSize = 3
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads a frame at the limit")
		})
		t.Run("reads a frame of length DefaultMaxSize for a MaxSize of 0", func(t *testing.T) {
			t.Parallel()
			_, _, err := frame.NewReader(bytes.NewReader(defaultLength)).Next()
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "Next reads past the length and finds no frame")
		})
		t.Run("returns the error of the stream", func(t *testing.T) {
			t.Parallel()
			_, _, err := frame.NewReader(failingReader{}).Next()
			assert.ErrorIs(t, err, errStream, "Next returns the error of the stream")
		})
		t.Run("returns the error of the stream inside a frame", func(t *testing.T) {
			t.Parallel()
			in := io.MultiReader(bytes.NewReader([]byte{0x03, 0x01}), failingReader{})
			_, _, err := frame.NewReader(in).Next()
			assert.ErrorIs(t, err, errStream, "Next returns the error of the stream")
		})
	})
	t.Run("Decode", func(t *testing.T) {
		t.Parallel()
		t.Run("returns ErrNoFrame before the first frame", func(t *testing.T) {
			t.Parallel()
			var got view.Item
			err := frame.NewReader(bytes.NewReader(itemFrame)).Decode(&got)
			assert.ErrorIs(t, err, frame.ErrNoFrame, "Decode has no frame to decode")
		})
		t.Run("returns ErrNoFrame after a failed Next", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(slices.Concat(itemFrame, badChecksum)))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the first frame")
			_, _, err = r.Next()
			assert.ErrorIs(t, err, frame.ErrChecksum, "Next fails for the second frame")
			var got view.Item
			assert.ErrorIs(t, r.Decode(&got), frame.ErrNoFrame, "Decode has no frame to decode")
		})
		t.Run("returns ErrNilMessage for a nil message", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(itemFrame))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the frame")
			assert.ErrorIs(t, r.Decode(nil), frame.ErrNilMessage, "Decode rejects nil")
		})
		t.Run("returns ErrNilMessage for a nil pointer", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(itemFrame))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the frame")
			assert.ErrorIs(t, r.Decode((*view.Item)(nil)), frame.ErrNilMessage, "Decode rejects a nil pointer")
		})
		t.Run("decodes the payload of the current frame twice", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(itemFrame))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the frame")
			var first, second view.Item
			assert.NoError(t, r.Decode(&first), "Decode decodes the payload")
			assert.NoError(t, r.Decode(&second), "Decode decodes the payload again")
			assert.Equal(t, second, first, "both decodes return the message")
		})
		t.Run("returns the error of the decode with offsets in the payload", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(cutPayload))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the frame")
			var got view.Item
			decodeErr := assert.ErrorAs[*kanon.DecodeError](t, r.Decode(&got), "Decode returns the decode error")
			assert.Equal(t, decodeErr.Offset, 1, "the error is at the length of the string in the payload")
		})
		t.Run("aliases the strings of the message to the buffer until the next frame", func(t *testing.T) {
			t.Parallel()
			second := []byte{0x06, 0x01, 0x00, 0x01, 0x0a, 0x01, 0x62}
			r := frame.NewReader(bytes.NewReader(slices.Concat(itemFrame, second)))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the first frame")
			var got view.Item
			assert.NoError(t, r.Decode(&got), "Decode decodes the first payload")
			_, _, err = r.Next()
			assert.NoError(t, err, "Next reads the second frame")
			assert.Equal(t, got.Name, "b", "the next frame overwrites the string of the first message")
		})
	})
	t.Run("Checksummed", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			in   []byte
			want bool
		}{
			{name: "reports true for a frame with a checksum", in: checkedFrame, want: true},
			{name: "reports false for a frame without a checksum", in: emptyFrame, want: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := frame.NewReader(bytes.NewReader(tt.in))
				_, _, err := r.Next()
				assert.NoError(t, err, "Next reads the frame")
				assert.Equal(t, r.Checksummed(), tt.want, "Checksummed reports the checksum of the frame")
			})
		}
		t.Run("reports false before the first frame", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(checkedFrame))
			assert.False(t, r.Checksummed(), "Checksummed reports no frame")
		})
		t.Run("reports false after a failed Next", func(t *testing.T) {
			t.Parallel()
			r := frame.NewReader(bytes.NewReader(slices.Concat(checkedFrame, badChecksum)))
			_, _, err := r.Next()
			assert.NoError(t, err, "Next reads the first frame")
			_, _, err = r.Next()
			assert.ErrorIs(t, err, frame.ErrChecksum, "Next fails for the second frame")
			assert.False(t, r.Checksummed(), "Checksummed reports no frame")
		})
	})
}

func TestReaderAllocs(t *testing.T) {
	t.Run("Next", func(t *testing.T) {
		t.Run("allocates nothing with Decode after a frame at least as long", func(t *testing.T) {
			item := view.Item{Name: "kanon", Count: 7}
			var out bytes.Buffer
			w := frame.NewWriter(&out)
			w.Checksum = true
			assert.NoError(t, w.Write(typeID, &item), "Write writes the frame")
			r := frame.NewReader(&loop{frame: out.Bytes()})
			var got view.Item
			failed := 0
			read := func() {
				if _, _, err := r.Next(); err != nil {
					failed++
				}
				if r.Decode(&got) != nil {
					failed++
				}
			}
			assert.MaxAllocs(t, read, 0, "Next and Decode allocate nothing")
			assert.Equal(t, failed, 0, "every measured call reads and decodes the frame")
			assert.Equal(t, got, item, "the decodes return the message")
		})
	})
}

func FuzzReader(f *testing.F) {
	for _, seed := range [][]byte{emptyFrame, itemFrame, checkedFrame, checkedItemFrame, badChecksum, cutPayload} {
		f.Add(seed)
	}
	known := []error{
		io.EOF, io.ErrUnexpectedEOF, frame.ErrMalformed, frame.ErrTooLarge, frame.ErrVersion, frame.ErrFlags,
		frame.ErrChecksum,
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		r := frame.NewReader(bytes.NewReader(data))
		// A limit of the input length bounds the buffer by the input.
		r.MaxSize = len(data)
		// Each frame that Next reads takes a byte of data at least, so the
		// stream ends within len(data)+1 calls.
		for range len(data) + 1 {
			_, _, err := r.Next()
			if err != nil {
				isKnown := slices.ContainsFunc(known, func(e error) bool { return errors.Is(err, e) })
				assert.True(t, isKnown, "Next fails with an error of the package or of the stream")
				return
			}
			var got view.Item
			_ = r.Decode(&got)
		}
		t.Fatal("Next reads more frames than the input has bytes")
	})
}
