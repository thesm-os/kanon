// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"cmp"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"unsafe"

	"go.thesmos.sh/kanon"
)

// Sizes of the reading of a Stream.
const (
	// bufferSize is the length of the read buffer of a Stream: the most bytes
	// that it asks of the reader with one call, except for the rest of a
	// value longer than the buffer, which it reads into the destination of
	// the value.
	bufferSize = 4096
	// maxHeader is the most bytes that the tag of a field and the varint after
	// it take: two varints of binary.MaxVarintLen64 bytes. A field whose
	// first maxHeader bytes are in the buffer has its tag, its length or its
	// varint, and a value of a fixed width in the buffer.
	maxHeader = 2 * binary.MaxVarintLen64
	// maxEmptyReads is the number of calls of the reader that return no byte
	// and no error after which one read of a Stream fails with
	// io.ErrNoProgress, as a fill of a bufio.Reader does.
	maxEmptyReads = 100
)

// StreamField is a field that a stream decoder returns to its caller instead
// of decoding it into its receiver: a byte slice or a string, whose bytes the
// caller reads, or a slice of a struct type with a kanon codec, whose
// elements the caller decodes one at a time.
type StreamField struct {
	// Tag is the tag of the field: its number shifted left by three bits,
	// with the wire format Bytes in the low bits.
	Tag uint64
	// Loc is the location of the field in errors: "Type.Field".
	Loc string
	// Elem names the struct type of the elements of a slice, as the errors of
	// the decode of the slice name it, and is empty for a byte slice and a
	// string.
	Elem string
	// Max is the bound of the tag option max of a slice: the most elements
	// that the slice takes over every occurrence of the field. It is 0 for a
	// slice without the option, a byte slice and a string.
	Max int
}

// StreamSchema describes the struct type of a stream decoder to a [Stream]:
// its location in errors, the rules of its decode, and its streamed fields.
// The generated code of the type declares it once, as a package variable.
type StreamSchema struct {
	// Loc names the struct type in errors.
	Loc string
	// Canonical reports that the decode of the type accepts only the
	// canonical encoding of a value.
	Canonical bool
	// Fields lists the streamed fields of the type.
	Fields []StreamField
}

// Run is a part of an encoding that the generated code of a stream decoder
// decodes with one call: a run of the fields that it decodes into its
// receiver, which [Stream.Next] returns, or an element of a streamed slice,
// which [Stream.Value] returns. The generated code decodes Data with the
// slab that [Run.Slab] returns, at offset 0 of the slab, and passes an error
// of the decode to [Stream.Fail], which locates it in the encoding.
//
// A Run takes four machine words, so the compiler keeps a Run that a call
// returns in registers.
type Run struct {
	// Data is the run.
	Data []byte
	// Depth is the number of levels that the decode of the run enters at
	// most: the levels below the struct type for a run of fields, and below
	// the struct of an element for an element.
	Depth int
}

// Slab returns the bytes of Data as a string that aliases them, the slab of
// the decode of the run, so that the decoded strings are substrings of Data
// and the decode copies nothing for them. The strings are valid as long as
// the bytes of Data: until the next Reset for a run of fields, and until the
// next call of Next, Element or Value for an element.
func (r Run) Slab() string {
	return unsafe.String(unsafe.SliceData(r.Data), len(r.Data))
}

// Stream is the state that the stream decoders of the kanon generator share:
// the reader of an encoding of a known length, the read buffer, the buffer
// of the decoded fields, the buffer of a long element, the streamed field
// that the caller reads, and the first error.
//
// The generated code of a struct type calls [Stream.Init] once, and
// [Stream.Reset] for each encoding. It then calls [Stream.Next], decodes the
// [Run] that Next returns, passes an error of the decode to [Stream.Fail],
// and calls [Stream.Open], which opens the streamed field after the run or
// ends the encoding. The caller of the generated code reads the value of the
// field that Open opened through [Stream.Read] or [Stream.WriteTo], or
// through [Stream.Element] and [Stream.Value], whose element the generated
// code decodes. [Stream.More] reports an element that the caller left
// unread, which the generated code decodes before it calls Next.
//
// # Reading
//
// A Stream reads the encoding from an io.LimitedReader of the length of the
// encoding, so it reads no byte past the encoding, into a read buffer of
// 4096 bytes. It parses the tags, the lengths and the values of the decoded
// fields, and the lengths of the elements, in the read buffer. It finds the
// end of each decoded field from the wire format of its tag, as the decode
// of an unknown field skips it, and copies the decoded fields that the read
// buffer contains whole with one copy. It reads the rest of a value longer
// than the read buffer into the destination of the value. WriteTo writes the
// value of a byte slice or a string to its writer from the read buffer, one
// read buffer at a time. A Stream checks a declared length against the rest
// of the encoding, and the length of an element against the rest of its
// field, before it reads the value or allocates for it.
//
// # Limits
//
// The buffer of the decoded fields grows to the decoded fields of the
// encoding together, with at most 20 bytes more of one field that the
// decode of the type rejects. A field that would take the decoded fields
// past the buffer limit of the stream fails with kanon.ErrLimit at its tag,
// once the decode of the run before the field succeeds. An element longer
// than the buffer limit fails with kanon.ErrLimit at its length, after the
// checks of the decode of the slice that precede the decode of the element.
// An element of up to 4096 bytes is a slice of the read buffer, and a longer
// one is in the buffer of the long element, which grows to the longest
// element. A slice whose [StreamField] has a Max counts its elements over
// every occurrence of the field, and its element past that bound fails with
// kanon.ErrMax before Element reads its length.
//
// # Errors
//
// A reader that ends before the length of the encoding fails the call that
// needs the next byte with a *kanon.DecodeError that wraps
// io.ErrUnexpectedEOF at the offset of that byte. The error locates a byte
// of a decoded field or of a tag at the struct type, and a byte of a
// streamed value at the streamed field. A method returns any other error of
// the reader unchanged, and io.ErrNoProgress when 100 calls of the reader
// return no byte and no error before the bytes that one read needs. The
// checks of a length and of a tag return the errors that the decode of the
// type returns for the same encoding, at the same offsets. After an error
// every method returns that error again. After the end of the encoding every
// method returns io.EOF, and WriteTo returns nil. WriteTo returns an error
// of its writer unchanged, and the error does not become the error of the
// Stream.
//
// # Allocation contract
//
// Init allocates the counts of the elements of the streamed slices when a
// field of the schema has a Max. The first Reset allocates the read buffer.
// Next and Value append to the
// buffer of the decoded fields and to the buffer of the long element, which
// grow to the decoded fields of an encoding and to its longest element of
// more than 4096 bytes, and allocate nothing once they have. Reset, Open,
// Read, Element, More, Len and Fail do not allocate, and WriteTo allocates
// only what its writer allocates. Errors allocate.
//
// # Concurrency
//
// A Stream is not safe for concurrent use.
type Stream struct {
	lr io.LimitedReader
	// buf is the read buffer, and ahead the bytes of the encoding from the
	// offset off that the stream read into it and has not used, a slice of
	// buf.
	buf, ahead []byte
	// schema describes the struct type that the stream decodes, and marks
	// has the bit of the number modulo 64 of each of its streamed fields, so
	// that a field whose bit is clear is no streamed field.
	schema *StreamSchema
	marks  uint64
	// run contains the decoded fields of the encoding, which the strings of
	// the receiver alias until the next Reset.
	run []byte
	// elem contains the bytes of the current element when it is longer than
	// the read buffer. The strings of the element alias them until the next
	// element.
	elem []byte
	// err is the first error, and io.EOF after the end of the encoding.
	err error
	// pending is the error of the field after the last run that takes the
	// decoded fields past the buffer limit, and nil without one.
	pending error
	// cur is the streamed field that Open opened last, and field its index in
	// the schema.
	cur   StreamField
	field int
	// counts has, per streamed field of the schema, the number of the
	// elements that Value returned over every occurrence of the field, when a
	// field of the schema has a Max, and is nil otherwise.
	counts []int
	// size is the length of the encoding, and off the offset of the next
	// byte.
	size, off int64
	// shift is the offset in the encoding of the last run or element that
	// Next or Value returned, which Fail adds to the offset of an error of
	// its decode.
	shift int64
	// left is the number of the bytes of the value of cur that the stream
	// has not read, and length the declared length of that value.
	left, length int64
	// elen is the length of the current element, and eat the offset of that
	// length. head reports that Element read the length of an element whose
	// bytes Value has not read.
	elen, eat int64
	head      bool
	// next is the index in the schema of the streamed field after the last
	// run, whose tag is at the offset tagAt, and -1 when no streamed field
	// follows the run.
	next  int
	tagAt int64
	// prior is the number of the streamed field that Open opened last, and 0
	// before the first.
	prior uint64
	// limit and depth are the buffer limit and the nesting limit.
	limit, depth int
}

// Init sets the schema of s, which describes the struct type that s decodes,
// and the limits of opts. The constructor of a generated stream decoder calls
// it once, before the first Reset. A Buffer of 0 or less selects
// kanon.DefaultBuffer, and the Depth selects the nesting limit that
// kanon.Options.Limit returns for it. Init allocates the counts of the
// elements of the streamed slices when a field of the schema has a Max.
func (s *Stream) Init(schema *StreamSchema, opts kanon.StreamOptions) {
	s.schema = schema
	bounded := false
	for _, f := range schema.Fields {
		s.marks |= 1 << ((f.Tag >> 3) & 63)
		bounded = bounded || f.Max > 0
	}
	if bounded {
		s.counts = make([]int, len(schema.Fields))
	}
	s.limit = cmp.Or(max(opts.Buffer, 0), kanon.DefaultBuffer)
	s.depth = kanon.Options{Depth: opts.Depth}.Limit()
}

// Reset makes s read the encoding of size bytes from r, from its first byte,
// and keeps the buffers of s for it. It sets the counts of the elements of
// the streamed slices to 0. A size below 0, or above math.MaxInt, the largest
// offset that a kanon.DecodeError states, makes every later call fail with
// kanon.ErrStreamSize.
func (s *Stream) Reset(r io.Reader, size int64) {
	if s.buf == nil {
		s.buf = make([]byte, bufferSize)
	}
	s.lr = io.LimitedReader{R: r, N: size}
	s.ahead, s.run = s.buf[:0], s.run[:0]
	clear(s.counts)
	s.size, s.off, s.shift, s.left, s.length, s.head, s.prior, s.err = size, 0, 0, 0, 0, false, 0, nil
	if uint64(size) > math.MaxInt {
		s.err = kanon.ErrStreamSize
	}
}

// Next moves past the rest of the current field, a byte slice or a string,
// by discarding the bytes that the caller has not read. The generated code
// decodes the elements of a slice that the caller has not decoded before it
// calls Next. Next then appends the decoded fields after the current field
// to the buffer of the decoded fields, up to the tag of the next streamed
// field or the end of the encoding, and returns them as a [Run].
//
// Next stops at a field that the decode of the type rejects, so that the
// decode of the run returns its error: after a tag that does not read, of
// field number 0, or of the number of a streamed field without its tag, and
// after a value that does not end within the encoding or whose tag has an
// invalid wire format. In a canonical decode it also stops after a tag that
// is not in its shortest form or not above the field before it. Such a tag
// at the start of a run, after a streamed field whose number is not below
// it, fails Next itself, since the decode of the run does not know the
// streamed field before it.
func (s *Stream) Next() (Run, error) {
	if s.err != nil {
		return Run{}, s.err
	}
	if err := s.discard(s.left); err != nil {
		return Run{}, s.fail(err, s.cur.Loc, int(s.cur.Tag>>3), s.off)
	}
	s.next, s.pending, s.shift = -1, nil, s.off
	start, prior := len(s.run), s.prior
	for s.off < s.size {
		failed := s.fill(int(min(maxHeader, s.size-s.off)))
		done, err := s.frame(failed, start, &prior)
		if err != nil {
			return Run{}, err
		}
		if done {
			break
		}
	}
	return Run{Data: s.run[start:], Depth: s.depth}, nil
}

// Open opens the streamed field after the last run, which [Stream.Next]
// found, and returns its number. The generated code calls it after the
// decode of the run succeeds. It reads the length of the field, and returns
// the error of the decode of the type in these cases, in this order:
//
//   - A length that does not read or that runs past the encoding fails at
//     the offset of the length.
//   - In a canonical decode, a length longer than its shortest form fails at
//     the same offset.
//   - A slice under a negative Depth, which rejects every nested value,
//     fails at the offset of the value.
//   - In a canonical decode, a value of no bytes, which the encoding leaves
//     out, fails at the offset of the tag.
//
// After a run without a streamed field, Open returns the error of the field
// after the run that takes the decoded fields past the buffer limit, and
// otherwise io.EOF at the end of the encoding. Every later call returns the
// same error.
func (s *Stream) Open() (int, error) {
	if s.next < 0 {
		if s.pending != nil {
			return 0, s.stop(s.pending)
		}
		return 0, s.stop(io.EOF)
	}
	f := s.schema.Fields[s.next]
	num, at := int(f.Tag>>3), s.off
	failed := s.fill(int(min(binary.MaxVarintLen64, s.size-s.off)))
	l, n := Uvarint(s.ahead)
	if n == 0 && failed != nil {
		return 0, s.fail(failed, f.Loc, num, s.off+int64(len(s.ahead)))
	}
	if n <= 0 || l > uint64(s.size-s.off-int64(n)) {
		return 0, s.stop(ReadError(n, f.Loc, num, int(at)))
	}
	if s.schema.Canonical && n > 1 && s.ahead[n-1] == 0 {
		return 0, s.stop(LongFormError(n, f.Loc, num, int(at)))
	}
	if f.Elem != "" && s.depth < 1 {
		return 0, s.stop(DepthError(f.Loc, num, int(at)+n))
	}
	if s.schema.Canonical && l == 0 {
		return 0, s.stop(AbsentError(f.Loc, num, int(s.tagAt)))
	}
	s.skip(n)
	s.cur, s.field, s.left, s.length, s.prior = f, s.next, int64(l), int64(l), uint64(num)
	return num, nil
}

// Fail records err, the error of the decode of the last run or element that
// Next or Value returned, as the error of s, which every later call returns,
// and returns it. The decode reads the run or the element at offset 0 of its
// slab, so Fail adds the offset of the run or the element in the encoding to
// the Offset of the first *kanon.DecodeError in the chain of err. It returns
// an error without a DecodeError unchanged.
func (s *Stream) Fail(err error) error {
	if d, ok := errors.AsType[*kanon.DecodeError](err); ok {
		d.Offset += int(s.shift)
	}
	return s.stop(err)
}

// More reports whether the current field is the streamed slice with the
// number num and has an element that Value has not read.
func (s *Stream) More(num int) bool {
	return s.err == nil && int(s.cur.Tag>>3) == num && (s.head || s.left > 0)
}

// Len returns the declared length in bytes of the value of the current
// field, and 0 before the first call of Next and after an error or the end
// of the encoding.
func (s *Stream) Len() int64 {
	if s.err != nil {
		return 0
	}
	return s.length
}

// Read reads the next bytes of the value of the current field, a byte slice
// or a string, into p: at most len(p) bytes, and at most the bytes that
// remain of the value. It copies the bytes that the read buffer contains,
// and reads a p of 4096 bytes or more from the reader itself when the read
// buffer is empty. It returns io.EOF when no byte of the value remains, and
// when the current field is a slice. It returns 0 and no error for an empty
// p.
func (s *Stream) Read(p []byte) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	if s.cur.Elem != "" || s.left == 0 {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p = p[:min(int64(len(p)), s.left)]
	if len(s.ahead) == 0 && len(p) >= len(s.buf) {
		n, err := s.readAtLeast(p, 1)
		if err != nil {
			return 0, s.fail(err, s.cur.Loc, int(s.cur.Tag>>3), s.off)
		}
		s.off, s.left = s.off+int64(n), s.left-int64(n)
		return n, nil
	}
	if err := s.fill(1); err != nil {
		return 0, s.fail(err, s.cur.Loc, int(s.cur.Tag>>3), s.off)
	}
	n := copy(p, s.ahead)
	s.skip(n)
	s.left -= int64(n)
	return n, nil
}

// WriteTo writes the rest of the value of the current field, a byte slice or
// a string, to w. It returns the number of bytes that w took. WriteTo writes
// the bytes that the read buffer contains, then refills the read buffer from
// the reader and writes again until the value ends. io.Copy from s therefore
// allocates nothing. WriteTo writes nothing and returns 0 and nil when no byte
// of the value remains, when the current field is a slice, and after the end
// of the encoding.
//
// It returns the error of w, and io.ErrShortWrite when w takes fewer bytes
// than it passes without an error. A count from w below 0 counts as 0. A count
// above the bytes that it passes counts as those bytes. An error of the reader
// becomes the error of s, as in Read.
func (s *Stream) WriteTo(w io.Writer) (int64, error) {
	if errors.Is(s.err, io.EOF) {
		return 0, nil
	}
	if s.err != nil {
		return 0, s.err
	}
	if s.cur.Elem != "" {
		return 0, nil
	}
	var written int64
	for s.left > 0 {
		if err := s.fill(1); err != nil {
			return written, s.fail(err, s.cur.Loc, int(s.cur.Tag>>3), s.off)
		}
		p := s.ahead[:min(int64(len(s.ahead)), s.left)]
		n, err := w.Write(p)
		n = min(max(n, 0), len(p))
		s.skip(n)
		s.left -= int64(n)
		written += int64(n)
		if err != nil {
			return written, err
		}
		if n < len(p) {
			return written, io.ErrShortWrite
		}
	}
	return written, nil
}

// Element reads the length of the next element of the current field, a
// slice, and returns it without reading the element. It returns the same
// length until Value reads the element, and io.EOF after the last element
// and when the current field is a byte slice or a string. It returns the
// error of the decode of the slice at the offset of the length: for an
// element past the Max of the field, before it reads the length, then for a
// length that does not read or that runs past the field, and in a canonical
// decode for a length longer than its shortest form.
func (s *Stream) Element() (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	if s.head {
		return s.elen, nil
	}
	if s.cur.Elem == "" || s.left == 0 {
		return 0, io.EOF
	}
	at, num := s.off, int(s.cur.Tag>>3)
	if s.cur.Max > 0 && s.counts[s.field] >= s.cur.Max {
		return 0, s.stop(MaxError(s.cur.Loc, num, int(at), s.cur.Max))
	}
	failed := s.fill(int(min(binary.MaxVarintLen64, s.left)))
	l, n := Uvarint(s.ahead[:min(int64(len(s.ahead)), s.left)])
	if n == 0 && failed != nil {
		return 0, s.fail(failed, s.cur.Loc, num, s.off+int64(len(s.ahead)))
	}
	if n <= 0 || l > uint64(s.left-int64(n)) {
		return 0, s.stop(ReadError(n, s.cur.Loc, num, int(at)))
	}
	if s.schema.Canonical && n > 1 && s.ahead[n-1] == 0 {
		return 0, s.stop(LongFormError(n, s.cur.Loc, num, int(at)))
	}
	s.skip(n)
	s.left -= int64(n)
	s.head, s.elen, s.eat = true, int64(l), at
	return s.elen, nil
}

// Value reads the next element of the current field, when the current field
// is the streamed slice with the number num, and returns it as a [Run]. It
// reads the length of the element first when Element has not, as Element
// reads it. It checks the nesting limit of the struct of the element, at the
// offset of its first byte, and then the length of the element against the
// buffer limit, at the offset of the length. It returns io.EOF after the last
// element and when the current field is another field. It counts the element
// that it returns for the Max of the field. The bytes of the run are valid
// until the next call of Next, Element or Value: a slice of the read buffer
// for an element of up to 4096 bytes, and of the buffer of the long element
// for a longer one.
func (s *Stream) Value(num int) (Run, error) {
	if s.err == nil && int(s.cur.Tag>>3) != num {
		return Run{}, io.EOF
	}
	l, err := s.Element()
	if err != nil {
		return Run{}, err
	}
	if s.depth < 2 {
		return Run{}, s.stop(DepthError(s.cur.Elem, 0, int(s.off)))
	}
	if l > int64(s.limit) {
		return Run{}, s.stop(LimitError(s.cur.Loc, num, int(s.eat), s.limit))
	}
	s.shift = s.off
	var elem []byte
	if l <= int64(len(s.buf)) {
		if err = s.fill(int(l)); err != nil {
			return Run{}, s.fail(err, s.cur.Loc, num, s.off+int64(len(s.ahead)))
		}
		elem = s.ahead[:l]
		s.skip(int(l))
	} else {
		if s.elem, err = s.appendN(s.elem[:0], int(l)); err != nil {
			return Run{}, s.fail(err, s.cur.Loc, num, s.off)
		}
		elem = s.elem
	}
	s.left, s.head = s.left-l, false
	if s.counts != nil {
		s.counts[s.field]++
	}
	return Run{Data: elem, Depth: s.depth - 2}, nil
}

// frame appends the decoded fields at the start of the read buffer to the
// buffer of the decoded fields, as [Stream.Next] frames them, and moves past
// them. The read buffer has at least maxHeader bytes, or the rest of the
// encoding, unless the reader ended or failed before them with failed, its
// error, which is nil otherwise. start is the length of the buffer of the
// decoded fields at the start of the run, and prior the number of the field
// before the next one.
//
// frame reports the end of the run: before the tag of a streamed field,
// after a field that the decode of the type rejects, and before a field
// that would take the decoded fields past the buffer limit, whose error it
// records as pending. It reads a value that continues past the read buffer
// from the reader. It returns the error of the reader for a field that needs
// the bytes after the read buffer, and the error of a canonical tag at the
// start of a run whose number is not above the streamed field before it.
func (s *Stream) frame(failed error, start int, prior *uint64) (bool, error) {
	w, canonical := s.ahead, s.schema.Canonical
	end := s.off + int64(len(w))
	ends := end == s.size
	i := 0
	for i < len(w) {
		b := w[i:]
		if len(b) < maxHeader && !ends && failed == nil {
			break
		}
		at := s.off + int64(i)
		tag, n := Uvarint(b)
		if n == 0 && !ends {
			return false, s.fail(failed, s.schema.Loc, 0, end)
		}
		if n <= 0 {
			// The decode rejects a tag that the encoding ends inside, with the
			// rest of the encoding, and a tag that exceeds 64 bits, with its 10
			// bytes.
			keep := len(w)
			if n < 0 {
				keep = i + binary.MaxVarintLen64
			}
			s.take(keep, 0)
			return true, nil
		}
		num, short := tag>>3, n == SizeUvarint(tag)
		bad := canonical && (!short || num <= *prior)
		if bad && short && num != 0 && len(s.run)+i == start {
			return false, s.stop(OrderError(num, *prior, s.schema.Loc, 0, int(at)))
		}
		k := s.streamed(num)
		if k >= 0 && tag == s.schema.Fields[k].Tag && !bad {
			s.take(i, n)
			s.next, s.tagAt = k, at
			return true, nil
		}
		if k >= 0 || num == 0 || bad {
			s.take(i+n, 0)
			return true, nil
		}
		*prior = num
		head, rest, kind := extent(b, tag, n, s.size-at)
		if kind == fieldShort {
			if !ends {
				return false, s.fail(failed, s.schema.Loc, 0, end)
			}
			s.take(len(w), 0)
			return true, nil
		}
		if kind == fieldBad {
			s.take(i+head, 0)
			return true, nil
		}
		if uint64(len(s.run)+i+head)+rest > uint64(s.limit) {
			s.pending = LimitError(s.schema.Loc, 0, int(at), s.limit)
			s.take(i, 0)
			return true, nil
		}
		if kind == fieldLong {
			s.take(i+head, 0)
			run, err := s.appendN(s.run, int(rest))
			s.run = run
			if err != nil {
				return false, s.fail(err, s.schema.Loc, 0, s.off)
			}
			return false, nil
		}
		i += head
	}
	if i == len(w) && !ends && failed != nil {
		return false, s.fail(failed, s.schema.Loc, 0, end)
	}
	s.take(i, 0)
	return false, nil
}

// streamed returns the index in the schema of the streamed field with the
// number num, and -1 when no streamed field has that number. It looks for
// the field only when the bit of num in the marks of s is set.
func (s *Stream) streamed(num uint64) int {
	if s.marks&(1<<(num&63)) == 0 {
		return -1
	}
	for k, f := range s.schema.Fields {
		if f.Tag>>3 == num {
			return k
		}
	}
	return -1
}

// take appends the first keep bytes of the read buffer to the buffer of the
// decoded fields, and moves past them and the extra bytes after them, which
// belong to no run: the tag of a streamed field.
func (s *Stream) take(keep, extra int) {
	s.run = append(s.run, s.ahead[:keep]...)
	s.skip(keep + extra)
}

// skip moves past the next n bytes of the encoding, which the read buffer
// contains.
func (s *Stream) skip(n int) {
	s.ahead = s.ahead[n:]
	s.off += int64(n)
}

// fill reads from the reader until the read buffer has need bytes, need at
// most the length of the buffer and of the rest of the encoding, as refill
// reads them. It returns at once when the read buffer has them.
func (s *Stream) fill(need int) error {
	if len(s.ahead) >= need {
		return nil
	}
	return s.refill(need)
}

// refill moves the bytes of the read buffer to its start, and then reads from
// the reader until the read buffer has need bytes, as many bytes as the
// reader returns with each call. It returns the error of the reader,
// io.ErrUnexpectedEOF for its end, when the reader ends or fails before need
// bytes.
func (s *Stream) refill(need int) error {
	n := copy(s.buf, s.ahead)
	k, err := s.readAtLeast(s.buf[n:], need-n)
	s.ahead = s.buf[:n+k]
	return err
}

// appendN appends the next n bytes of the encoding to dst: the bytes of the
// read buffer, and then the bytes that it reads from the reader into dst. It
// returns the error of the reader, io.ErrUnexpectedEOF for its end, with the
// bytes before it, when the reader ends or fails before them.
func (s *Stream) appendN(dst []byte, n int) ([]byte, error) {
	dst = slices.Grow(dst, n)
	k := copy(dst[len(dst):len(dst)+n], s.ahead)
	s.skip(k)
	m, err := s.readAtLeast(dst[len(dst)+k:len(dst)+n], n-k)
	s.off += int64(m)
	return dst[:len(dst)+k+m], err
}

// discard reads and drops the next n bytes of the encoding. It returns the
// error of the reader, io.ErrUnexpectedEOF for its end, when the reader ends
// or fails before them.
func (s *Stream) discard(n int64) error {
	for n > 0 {
		if err := s.fill(1); err != nil {
			return err
		}
		k := int(min(n, int64(len(s.ahead))))
		s.skip(k)
		n -= int64(k)
	}
	return nil
}

// readAtLeast reads from the reader into p until it has read need bytes, and
// returns the number of the bytes that it read, as io.ReadAtLeast does. It
// returns the error of the reader when the reader ends or fails before them:
// io.ErrUnexpectedEOF for its end, since the encoding continues, and
// io.ErrNoProgress after maxEmptyReads calls that return no byte and no
// error.
func (s *Stream) readAtLeast(p []byte, need int) (int, error) {
	n, empty := 0, 0
	for n < need {
		k, err := s.lr.Read(p[n:])
		n += k
		if n >= need {
			break
		}
		if err != nil {
			return n, unexpected(err)
		}
		if k == 0 {
			empty++
			if empty == maxEmptyReads {
				return n, io.ErrNoProgress
			}
		}
	}
	return n, nil
}

// fail records the error of a read of s, as stop records it, and returns
// it: a *kanon.DecodeError that wraps io.ErrUnexpectedEOF at off, the
// offset of the first byte that the reader did not return, which loc and num
// locate, for a reader that ended before the length of the encoding, and any
// other error unchanged.
func (s *Stream) fail(err error, loc string, num int, off int64) error {
	if errors.Is(err, io.ErrUnexpectedEOF) {
		err = ReadError(0, loc, num, int(off))
	}
	return s.stop(err)
}

// stop records err, an error that locates its input in the encoding, as the
// error of s, which every later call returns, and returns it.
func (s *Stream) stop(err error) error {
	s.err = err
	return err
}

// fieldKind is how far the decoded field at the start of the read buffer of
// a stream extends, as [extent] measures it.
type fieldKind uint8

// The extents of a decoded field. The zero fieldKind is invalid.
const (
	// fieldWhole is a field whose bytes the read buffer contains.
	fieldWhole fieldKind = 1
	// fieldLong is a field of the wire format Bytes whose value continues past
	// the read buffer, after its tag and its length.
	fieldLong fieldKind = 2
	// fieldShort is a field that the read buffer ends inside, before the end
	// of its length, of a varint or of a value of a fixed width.
	fieldShort fieldKind = 3
	// fieldBad is a field that the decode rejects at its tag or its length:
	// a varint that exceeds 64 bits, a length that runs past the encoding,
	// and an invalid wire format.
	fieldBad fieldKind = 4
)

// extent measures the decoded field at the start of b, whose tag is tag, of
// n bytes, and whose encoding continues for left bytes from its tag, as the
// wire format of the tag lays it out. It returns the kind of its extent with
// these lengths: for fieldWhole the length of the field, for fieldLong the
// length of its tag and its length followed by the length of its value, and
// for fieldBad the number of its bytes that the decode reads before it
// rejects the field.
func extent(b []byte, tag uint64, n int, left int64) (int, uint64, fieldKind) {
	v := b[n:]
	switch tag & 7 {
	case Varint:
		_, vn := Uvarint(v)
		if vn == 0 {
			return 0, 0, fieldShort
		}
		if vn < 0 {
			return n + binary.MaxVarintLen64, 0, fieldBad
		}
		return n + vn, 0, fieldWhole
	case Fixed64, Fixed32:
		width := 8
		if tag&7 == Fixed32 {
			width = 4
		}
		if len(v) < width {
			return 0, 0, fieldShort
		}
		return n + width, 0, fieldWhole
	case Bytes:
		l, ln := Uvarint(v)
		if ln == 0 {
			return 0, 0, fieldShort
		}
		if ln < 0 {
			return n + binary.MaxVarintLen64, 0, fieldBad
		}
		if l > uint64(left-int64(n+ln)) {
			return n + ln, 0, fieldBad
		}
		if l > uint64(len(v)-ln) {
			return n + ln, l, fieldLong
		}
		return n + ln + int(l), 0, fieldWhole
	default:
		return n, 0, fieldBad
	}
}

// unexpected returns io.ErrUnexpectedEOF for io.EOF, which a read of a Stream
// meets only when its reader ends before the length of the encoding, and err
// otherwise.
func unexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
