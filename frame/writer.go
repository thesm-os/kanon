// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"encoding/binary"
	"io"
	"math"
	"slices"

	"go.thesmos.sh/kanon"
)

// Writer writes frames to a stream with one buffer, which grows to the
// longest frame written. A Writer is not safe for concurrent use.
type Writer struct {
	// Checksum adds a CRC-32C to every frame.
	Checksum bool

	// w is the stream.
	w io.Writer
	// buf is the buffer of the frames, which each Write overwrites.
	buf []byte
}

// NewWriter returns a Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// Write writes m as one frame of type id: it sizes m, encodes it into the
// buffer after the header, appends the checksum when Checksum is set, and
// writes the complete frame with one call of the stream.
//
// Write returns [ErrNilMessage] for a nil m, a nil pointer in an interface
// included, and [ErrTooLarge] when the frame, its length included, is longer
// than math.MaxInt bytes. When m fails to encode, Write returns the error of
// the encode and does not write to the stream. When the stream returns an
// error, Write returns it. When the stream writes fewer bytes than the frame
// without an error, Write returns io.ErrShortWrite. Write does not retry,
// since a short write can already have put a prefix of the frame on the
// stream.
func (w *Writer) Write(id uint64, m kanon.Message) error {
	if isNil(m) {
		return ErrNilMessage
	}
	size := m.SizeKanon()
	var flags byte
	tail := 0
	if w.Checksum {
		flags, tail = flagChecksum, checksumSize
	}
	// The header, the checksum and the length add at most 26 bytes to the
	// payload, so the total of any int fits a uint64, and a total with a
	// bit above math.MaxInt does not fit an int.
	length := uint64(fixedHeader+uvarintLen(id)+tail) + uint64(size)
	total := uint64(uvarintLen(length)) + length
	if total&^math.MaxInt != 0 {
		return ErrTooLarge
	}
	buf := slices.Grow(w.buf[:0], int(total))[:total]
	w.buf = buf
	start := binary.PutUvarint(buf, length)
	buf[start], buf[start+1] = version, flags
	i := start + fixedHeader
	i += binary.PutUvarint(buf[i:], id)
	if _, err := m.EncodeKanon(buf[:i+size]); err != nil {
		return err
	}
	i += size
	if w.Checksum {
		binary.LittleEndian.PutUint32(buf[i:], checksum(buf[start:i]))
	}
	written, err := w.w.Write(buf)
	if err != nil {
		return err
	}
	if written != len(buf) {
		return io.ErrShortWrite
	}
	return nil
}

// uvarintLen returns the length in bytes of the uvarint of x.
func uvarintLen(x uint64) int {
	var b [maxVarint]byte
	return binary.PutUvarint(b[:], x)
}
