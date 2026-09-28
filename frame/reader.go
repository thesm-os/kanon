// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"slices"
	"unsafe"

	"go.thesmos.sh/kanon"
)

// DefaultMaxSize is the frame length that a Reader accepts when its
// MaxSize is 0 or less: 16 MiB.
const DefaultMaxSize = 16777216

// Reader reads frames from a stream with one buffer, which grows to the
// longest frame read. A Reader is not safe for concurrent use.
type Reader struct {
	// MaxSize is the longest frame length that the reader accepts. 0 or
	// less means DefaultMaxSize.
	MaxSize int

	// r is the stream, or a bufio.Reader over it when the stream does not
	// implement io.ByteReader.
	r byteReader
	// buf is the frame after its length, which each Next overwrites.
	buf []byte
	// payload is the payload of the current frame, a slice of buf.
	payload []byte
	// current reports that the last call of Next succeeded, so that
	// payload is the payload of a frame.
	current bool
}

// byteReader is a stream that reads byte by byte.
type byteReader interface {
	io.Reader
	io.ByteReader
}

// NewReader returns a Reader that reads from r. It wraps r in a
// bufio.Reader when r does not implement io.ByteReader, as encoding/gob
// does, so the Reader then reads ahead of the current frame.
func NewReader(r io.Reader) *Reader {
	br, ok := r.(byteReader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Reader{r: br}
}

// Next reads the next frame and returns its type ID and payload. The
// payload aliases the reader's buffer: a caller must not change it while
// Decode or a string that Decode returned is in use, and the next call of
// Next can overwrite it. A caller that keeps the payload longer copies it.
//
// Next returns io.EOF at the end of the stream between frames, and
// io.ErrUnexpectedEOF when the stream ends inside a frame. It returns
// [ErrMalformed] for a length uvarint that runs past 10 bytes or 64 bits,
// and [ErrTooLarge] for a length above MaxSize, before it allocates for
// the frame. For a length within MaxSize, it reads the whole frame before
// it checks it, and returns [ErrMalformed] for a frame too short for its
// version and flags, a type uvarint that runs past the frame or 64 bits,
// and a frame too short for its checksum, [ErrVersion], [ErrFlags] and
// [ErrChecksum].
func (r *Reader) Next() (id uint64, payload []byte, err error) {
	r.current = false
	length, err := r.length()
	if err != nil {
		return 0, nil, err
	}
	limit := r.MaxSize
	if limit <= 0 {
		limit = DefaultMaxSize
	}
	if length > uint64(limit) {
		return 0, nil, ErrTooLarge
	}
	r.buf = slices.Grow(r.buf[:0], int(length))[:length]
	if _, err = io.ReadFull(r.r, r.buf); err != nil {
		return 0, nil, truncated(err)
	}
	id, err = r.parse()
	if err != nil {
		return 0, nil, err
	}
	r.current = true
	return id, r.payload, nil
}

// Decode decodes the payload of the frame that the last successful call of
// Next read, an empty payload included, into m, with the payload as the
// slab of its kanon.Options and kanon.DefaultDepth as the depth. It can
// decode that payload more than once. The strings of m alias the reader's
// buffer until the next call of Next, and a value that is used after the
// next frame is copied with CloneKanon. Decode returns [ErrNoFrame] before
// the first successful Next and after a failed one, [ErrNilMessage] for a
// nil m, a nil pointer in an interface included, and the error of the
// decode, whose offsets are offsets in the payload.
func (r *Reader) Decode(m kanon.Message) error {
	if !r.current {
		return ErrNoFrame
	}
	if isNil(m) {
		return ErrNilMessage
	}
	slab := unsafe.String(unsafe.SliceData(r.payload), len(r.payload))
	return m.DecodeKanon(r.payload, kanon.Options{Slab: slab})
}

// length reads the length uvarint of the next frame as binary.ReadUvarint
// does, and returns [ErrMalformed] where ReadUvarint returns its unexported
// overflow error: for a uvarint that runs past 10 bytes or past 64 bits. It
// returns io.EOF when the stream ends before the first byte, and
// io.ErrUnexpectedEOF when it ends inside the uvarint.
func (r *Reader) length() (uint64, error) {
	var x uint64
	var shift uint
	for i := range maxVarint {
		b, err := r.r.ReadByte()
		if err != nil {
			if i > 0 {
				return 0, truncated(err)
			}
			return 0, err
		}
		if b < 0x80 {
			if i == maxVarint-1 && b > 1 {
				return 0, ErrMalformed
			}
			return x | uint64(b)<<shift, nil
		}
		x |= uint64(b&0x7f) << shift
		shift += 7
	}
	return 0, ErrMalformed
}

// parse checks the frame in buf and sets payload: the version, the flags,
// the type uvarint and the checksum, in that order. It returns the type ID.
func (r *Reader) parse() (uint64, error) {
	if len(r.buf) < fixedHeader {
		return 0, ErrMalformed
	}
	if r.buf[0] != version {
		return 0, ErrVersion
	}
	flags := r.buf[1]
	if flags&^flagChecksum != 0 {
		return 0, ErrFlags
	}
	id, n := binary.Uvarint(r.buf[fixedHeader:])
	if n <= 0 {
		return 0, ErrMalformed
	}
	start, end := fixedHeader+n, len(r.buf)
	if flags&flagChecksum != 0 {
		end -= checksumSize
		if end < start {
			return 0, ErrMalformed
		}
		if checksum(r.buf[:end]) != binary.LittleEndian.Uint32(r.buf[end:]) {
			return 0, ErrChecksum
		}
	}
	r.payload = r.buf[start:end]
	return id, nil
}

// truncated returns err of a read inside a frame: io.ErrUnexpectedEOF for
// the end of the stream, which a read inside a frame means truncation, and
// err otherwise.
func truncated(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
