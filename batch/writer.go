// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"encoding/binary"
	"math"
	"slices"

	"go.thesmos.sh/kanon"
)

// Writer builds a batch. Append encodes each message at the end of the
// buffer of the batch and records its offset in the index, and Bytes
// appends the index and the count to the buffer. The zero Writer is empty
// and ready to use. A Writer is not safe for concurrent use.
type Writer struct {
	// Wide selects 8-byte offsets, for a batch whose encodings are longer
	// than 2^32-1 bytes. The writer reads it at the first successful
	// Append, or at Bytes for a batch without messages, and keeps that
	// width until Reset.
	Wide bool

	// buf is the header and the encodings, and after Bytes the index and
	// the count. It is empty before the first successful Append or Bytes,
	// and after Reset.
	buf []byte
	// index is the offsets of the encodings, in the width of the batch.
	index []byte
	// count is the number of messages.
	count int
	// flags is the flags byte that the first successful Append or Bytes
	// wrote.
	flags byte
	// done reports that Bytes has appended the index and the count.
	done bool
}

// Append encodes m at the end of the buffer and records its offset.
//
// Append returns [ErrFinalized] after Bytes until Reset, [ErrNilMessage] for
// a nil m, a nil pointer in an interface included, [ErrTooBig] when the
// batch with m, its header, index and count included, would not fit its
// offsets or an int, and the error of the encode when m fails to encode. A
// failed Append changes neither the length nor the bytes of the writer,
// and does not fix the width of the offsets.
func (w *Writer) Append(m kanon.Message) error {
	if w.done {
		return ErrFinalized
	}
	if isNil(m) {
		return ErrNilMessage
	}
	flags := w.header()
	width := offsetSize(flags)
	buf := w.buf
	if w.count == 0 {
		buf = append(buf[:0], version, flags)
	}
	offset := len(buf) - headerSize
	size := m.SizeKanon()
	// total is the length of the batch with m, which adds one offset per
	// message and the count to the buffer up to the end of m. The sums fit a
	// uint64 for any int. With 4-byte offsets, the encodings end below 2^32.
	end := uint64(len(buf)) + uint64(size)
	total := end + uint64(w.count+1)*uint64(width) + countSize
	pastNarrow := width == narrow && (uint64(offset)+uint64(size))>>32 != 0
	if pastNarrow || uint64(w.count) == maxCount || total&^math.MaxInt != 0 {
		return ErrTooBig
	}
	buf = slices.Grow(buf, size)[:len(buf)+size]
	if _, err := m.EncodeKanon(buf); err != nil {
		return err
	}
	w.buf = buf
	if width == wide {
		w.index = binary.LittleEndian.AppendUint64(w.index, uint64(offset))
	} else {
		w.index = binary.LittleEndian.AppendUint32(w.index, uint32(offset))
	}
	w.flags = flags
	w.count++
	return nil
}

// Len returns the number of messages that Append added since Reset.
func (w *Writer) Len() int {
	return w.count
}

// Bytes appends the index and the count to the buffer at its first call
// and returns the batch, which aliases the buffer until Reset. A later call
// returns the same bytes without a second index. For a writer without
// messages, Bytes reads Wide and returns a batch without messages.
func (w *Writer) Bytes() []byte {
	if w.done {
		return w.buf
	}
	if w.count == 0 {
		w.flags = w.header()
		w.buf = append(w.buf[:0], version, w.flags)
	}
	w.buf = append(w.buf, w.index...)
	w.buf = binary.LittleEndian.AppendUint32(w.buf, uint32(w.count))
	w.done = true
	return w.buf
}

// Reset empties the writer and keeps the memory of its buffers for the
// next batch. The bytes that Bytes returned before are no longer valid, and
// the next Append or Bytes reads Wide again.
func (w *Writer) Reset() {
	w.buf = w.buf[:0]
	w.index = w.index[:0]
	w.count = 0
	w.done = false
}

// header returns the flags byte of the batch: the one that the first
// successful Append wrote, or the one that Wide selects before it.
func (w *Writer) header() byte {
	if w.count > 0 {
		return w.flags
	}
	if w.Wide {
		return flagWide
	}
	return 0
}
