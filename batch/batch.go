// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"encoding/binary"
	"math"
	"reflect"
	"slices"
	"unsafe"

	"go.thesmos.sh/kanon"
)

// These constants give the values and the lengths of the batch layout, in
// which the header precedes the encodings and the index of their offsets
// precedes the count.
const (
	// version is the value of the version byte of this layout.
	version = 1
	// flagWide is the bit of the flags byte that selects 8-byte offsets.
	flagWide = 1
	// headerSize is the length in bytes of the version and the flags.
	headerSize = 2
	// countSize is the length in bytes of the count.
	countSize = 4
	// narrow is the length in bytes of an offset without flagWide.
	narrow = 4
	// wide is the length in bytes of an offset with flagWide.
	wide = 8
	// maxCount is the largest count, which the 4 bytes of the count limit.
	maxCount = math.MaxUint32
)

// Batch reads the messages of one batch. [Parse] and [ParseAlias] check its
// layout once, so that its methods check only the index of a message. The
// zero Batch has no messages. A Batch is safe for concurrent use, since its
// methods only read.
type Batch struct {
	// data is the batch: the copy that Parse made, or the input of
	// ParseAlias.
	data []byte
	// start is the offset in data of the index, and so the end of the
	// encodings.
	start int
	// count is the number of messages.
	count int
	// width is the length in bytes of an offset: narrow or wide.
	width int
}

// Parse checks the layout of data as [ParseAlias] does and returns its
// Batch, whose slab is one copy of data that Parse makes after the check.
// The records and the decoded strings of the Batch alias the copy, and not
// data.
func Parse(data []byte) (Batch, error) {
	b, err := ParseAlias(data)
	if err != nil {
		return Batch{}, err
	}
	b.data = slices.Clone(data)
	return b, nil
}

// ParseAlias checks the layout of data and returns its Batch, with data as
// the slab. The records and the decoded strings of the Batch alias data,
// which must not change while they are in use.
//
// ParseAlias returns [ErrVersion] for a version other than 1, [ErrFlags]
// for a flag above bit 0, and [ErrLayout] for data too short for the
// header and the count, an index longer than the data, and offsets that are
// not nondecreasing from 0 within the encodings. It checks the count
// against the length of data in uint64 before it slices data, so that no
// count and no offset overflows an int, and it does not allocate.
func ParseAlias(data []byte) (Batch, error) {
	if len(data) < headerSize {
		return Batch{}, ErrLayout
	}
	if data[0] != version {
		return Batch{}, ErrVersion
	}
	flags := data[1]
	if flags&^flagWide != 0 {
		return Batch{}, ErrFlags
	}
	if len(data) < headerSize+countSize {
		return Batch{}, ErrLayout
	}
	width := offsetSize(flags)
	end := len(data) - countSize
	count := binary.LittleEndian.Uint32(data[end:])
	// count*width is at most 2^35, which fits a uint64.
	if uint64(count)*uint64(width) > uint64(end-headerSize) {
		return Batch{}, ErrLayout
	}
	b := Batch{data: data, count: int(count), width: width}
	b.start = end - b.count*width
	if err := b.check(); err != nil {
		return Batch{}, err
	}
	return b, nil
}

// Len returns the number of messages.
func (b Batch) Len() int {
	return b.count
}

// Record returns the encoding of message i, which aliases the slab of b and
// has no capacity past its end. It panics when i is out of range, as an
// index into a slice of the messages does.
func (b Batch) Record(i int) []byte {
	start, end := b.bounds(i)
	return b.data[start:end:end]
}

// Decode decodes message i into m, with the slab of b and the offset of the
// message in it as the kanon.Options, and kanon.DefaultDepth as the depth,
// so that the offsets of its errors are offsets in the batch. It panics when
// i is out of range, as an index into a slice of the messages does, and
// returns [ErrNilMessage] for a nil m, a nil pointer in an interface
// included, and the error of the decode. The check of the layout leaves
// the encodings unread, so a malformed encoding fails the decode of its
// message alone.
func (b Batch) Decode(i int, m kanon.Message) error {
	start, end := b.bounds(i)
	if isNil(m) {
		return ErrNilMessage
	}
	slab := unsafe.String(unsafe.SliceData(b.data), len(b.data))
	return m.DecodeKanon(b.data[start:end], kanon.Options{Slab: slab, Offset: start})
}

// check returns [ErrLayout] unless the offsets of b are nondecreasing from 0
// and end within the encodings, and a batch without messages has no bytes
// of encodings.
func (b Batch) check() error {
	region := uint64(b.start - headerSize)
	if b.count > 0 && b.offset(0) != 0 {
		return ErrLayout
	}
	var prev uint64
	for i := range b.count {
		o := b.offset(i)
		if o < prev || o > region {
			return ErrLayout
		}
		prev = o
	}
	if b.count == 0 && region != 0 {
		return ErrLayout
	}
	return nil
}

// bounds returns the start and the end in data of the encoding of message
// i. It panics when i is out of range, as an index into a slice of the
// messages does.
func (b Batch) bounds(i int) (start, end int) {
	// The index has count offsets of at least one byte, so its first count
	// bytes are a slice of count elements, which i indexes.
	_ = b.data[b.start : b.start+b.count][i]
	start = headerSize + int(b.offset(i))
	end = b.start
	if i+1 < b.count {
		end = headerSize + int(b.offset(i+1))
	}
	return start, end
}

// offset returns the offset of the encoding of message i from the index,
// without a check of i.
func (b Batch) offset(i int) uint64 {
	at := b.data[b.start+i*b.width:]
	if b.width == wide {
		return binary.LittleEndian.Uint64(at)
	}
	return uint64(binary.LittleEndian.Uint32(at))
}

// offsetSize returns the length in bytes of an offset of a batch with
// flags.
func offsetSize(flags byte) int {
	if flags&flagWide != 0 {
		return wide
	}
	return narrow
}

// isNil reports whether m is nil or stores a nil pointer, which a batch
// cannot encode and a decode cannot fill.
func isNil(m kanon.Message) bool {
	if m == nil {
		return true
	}
	v := reflect.ValueOf(m)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
