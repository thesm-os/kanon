// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"testing/iotest"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// Names of the checks of the stream decoder of T that [Checks] returns.
const (
	streamSamplesCheck = "Next/decodes the encoding of each sample as the reference decode"
	streamCutsCheck    = "Next/returns io.ErrUnexpectedEOF at the offset at which the reader ends"
	streamLimitCheck   = "Next/returns kanon.ErrLimit under a buffer limit one byte below the encoding"
	streamSizeCheck    = "Next/returns kanon.ErrStreamSize for a negative size"
	streamAllocsCheck  = "Next/allocates nothing for an encoding that the stream decoded before"
	streamLengthsCheck = "Len/returns the length that the encoding declares for each streamed value"
)

// How a run of the checks reads the value of a streamed byte slice or string:
// through Read into a buffer of a length above 0, or through WriteTo.
const (
	// readChunk is the length of the buffer of Read, which is shorter than the
	// read buffer of a stream, so that a value of more bytes takes two or more
	// calls of Read.
	readChunk = 512
	// writeTo makes the run copy the value through WriteTo.
	writeTo = 0
)

// cloneKanonName names the copy method of a struct type with a kanon codec,
// through which the checks copy a decoded element of a streamed slice.
const cloneKanonName = "CloneKanon"

// Stream is the stream decoder of the struct type T, which kanon generates
// for a struct type with a field that has the tag option stream, as the test
// file that kanon generates adapts it for the checks. Next returns the
// number of the streamed field in place of its constant. The adapter also
// implements io.Reader and io.WriterTo when T streams a byte slice or a
// string, and Element and Decode when T streams a slice: Decode decodes the
// next element of the streamed slice with the number num into e, a pointer
// to an element.
//
// # Concurrency
//
// A Stream is not safe for concurrent use.
type Stream[T any] interface {
	// Reset makes the stream read the encoding of a T, size bytes, from r
	// into m.
	Reset(r io.Reader, size int64, m *T)
	// Next moves to the next streamed field and returns its number, and
	// io.EOF at the end of the encoding.
	Next() (int, error)
	// Len returns the declared length of the value of the current field.
	Len() int64
}

// elements is the method set of the adapter of a stream decoder that streams
// a slice, as [Stream] states it.
type elements interface {
	Element() (int64, error)
	Decode(num int, e any) error
}

// streamFields returns the fields of l with the tag option stream, and fails
// when spec does not describe the stream decoder of T: a Stream without such
// a field, such a field without a Stream, and such a field of a type that does
// not stream: a byte slice and a string, and a slice of a struct type with a
// kanon codec.
func streamFields[T any](spec Spec[T], l *layout) ([]*field, error) {
	var out []*field
	for _, f := range l.fields {
		if !f.Stream {
			continue
		}
		s := f.shape
		if s.kind != kindString && s.kind != kindBytes && (s.kind != kindSlice || s.elem.kind != kindStruct) {
			return nil, fmt.Errorf("kanontest: the Spec streams field %s of %s, whose type %s does not stream",
				f.Name, l.typ, s.typ)
		}
		out = append(out, f)
	}
	if spec.Stream != nil && len(out) == 0 {
		return nil, fmt.Errorf("kanontest: the Spec names a stream decoder of %s, and streams no field", l.typ)
	}
	if spec.Stream == nil && len(out) > 0 {
		return nil, fmt.Errorf("kanontest: the Spec streams field %s of %s, and names no stream decoder",
			out[0].Name, l.typ)
	}
	return out, nil
}

// readValue reads the value of the current field f of st, a byte slice or a
// string, through Read chunk bytes at a time, or through WriteTo when chunk
// is writeTo, and returns it as a value of the type of f. It fails tb when
// the value has another length than Len declares.
func readValue[T any](tb assert.TB, st Stream[T], f *field, chunk int) (reflect.Value, error) {
	tb.Helper()
	declared := st.Len()
	var b []byte
	if chunk == writeTo {
		var buf bytes.Buffer
		wt, _ := st.(io.WriterTo)
		if _, err := wt.WriteTo(&buf); err != nil {
			return reflect.Value{}, err
		}
		b = buf.Bytes()
	} else {
		rd, _ := st.(io.Reader)
		p := make([]byte, chunk)
		for {
			n, err := rd.Read(p)
			b = append(b, p[:n]...)
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return reflect.Value{}, err
			}
		}
	}
	assert.Equal(tb, int64(len(b)), declared, f.Name+": the stream returns the bytes of the length that Len declares")
	x := reflect.New(f.shape.typ).Elem()
	if f.shape.kind == kindString {
		x.SetString(string(b))
	} else {
		x.SetBytes(b)
	}
	return x, nil
}

// decodeElements decodes the elements of the current field f of st, a slice,
// after their lengths, appends a copy of each to acc, a slice of the type of
// f, and returns it.
func decodeElements[T any](st Stream[T], f *field, acc reflect.Value) (reflect.Value, error) {
	el, _ := st.(elements)
	if !acc.IsValid() {
		acc = reflect.New(f.shape.typ).Elem()
	}
	for {
		_, err := el.Element()
		if errors.Is(err, io.EOF) {
			return acc, nil
		}
		if err != nil {
			return acc, err
		}
		e := reflect.New(f.shape.elem.typ)
		if err := el.Decode(f.Number, e.Interface()); err != nil {
			return acc, err
		}
		acc = reflect.Append(acc, e.MethodByName(cloneKanonName).Call(nil)[0].Elem())
	}
}

// streamChecks returns the checks of the stream decoder of T: the decode of
// the encoding of each sample, the end of the reader at each offset of the
// encoding of each sample, the buffer limit, a negative size, and the
// allocations. The checks of the probes and the property also run the
// stream decoder on every input, as [suite.streams] runs it.
func (s *suite[T, P]) streamChecks() []Check {
	return []Check{
		{Name: streamAllocsCheck, Serial: true, Run: s.streamAllocs},
		{Name: streamSamplesCheck, Run: func(tb assert.TB) {
			tb.Helper()
			for _, x := range s.encodable() {
				s.streams(tb, probe{name: x.name, data: x.enc})
			}
		}},
		{Name: streamCutsCheck, Run: func(tb assert.TB) {
			tb.Helper()
			s.streamCuts(tb, s.samples)
		}},
		{Name: streamLimitCheck, Run: func(tb assert.TB) {
			tb.Helper()
			s.streamLimits(tb, s.all())
		}},
		{Name: streamSizeCheck, Run: s.streamSize},
		{Name: streamLengthsCheck, Run: func(tb assert.TB) {
			tb.Helper()
			s.streamLengths(tb, s.all())
		}},
	}
}

// streams checks that the stream decoder of T decodes the probe p as the
// reference decode, with the depth limit of p and without its slab, whose
// stream reads the bytes of p from the first: from a reader that returns all
// of them, from a reader that returns one byte per call, with each streamed
// byte slice and string copied through WriteTo, and with every streamed
// value skipped, which leaves the streamed fields of the receiver empty.
func (s *suite[T, P]) streams(tb assert.TB, p probe) {
	tb.Helper()
	opts := kanon.StreamOptions{Depth: p.opts.Depth}
	want, wantErr := s.reference(probe{data: p.data, opts: kanon.Options{Depth: p.opts.Depth}})
	got, err := s.streamed(tb, bytes.NewReader(p.data), len(p.data), opts, readChunk, false)
	s.same(tb, p.name+": the stream decoder", got, err, want, wantErr)
	got, err = s.streamed(tb, iotest.OneByteReader(bytes.NewReader(p.data)), len(p.data), opts, 1, false)
	s.same(tb, p.name+": the stream decoder over a reader of one byte per call", got, err, want, wantErr)
	got, err = s.streamed(tb, bytes.NewReader(p.data), len(p.data), opts, writeTo, false)
	s.same(tb, p.name+": the stream decoder that copies the streamed bytes through WriteTo", got, err, want,
		wantErr)
	got, err = s.streamed(tb, bytes.NewReader(p.data), len(p.data), opts, readChunk, true)
	v := reflect.ValueOf(&want).Elem()
	for _, f := range s.streamFields {
		f.of(v).SetZero()
	}
	s.same(tb, p.name+": the stream decoder that skips the streamed values", got, err, want, wantErr)
}

// streamCuts checks that the stream decoder of T fails with a
// *kanon.DecodeError that wraps io.ErrUnexpectedEOF at offset k for the
// encoding of each sample of xs that decodes, from a reader that returns its
// first k bytes, for every k below its length. It reads each streamed byte
// slice and string through Read, and again through WriteTo.
func (s *suite[T, P]) streamCuts(tb assert.TB, xs []sample[T]) {
	tb.Helper()
	runs := []struct {
		chunk int
		how   string
	}{
		{chunk: readChunk, how: ", read through Read"},
		{chunk: writeTo, how: ", read through WriteTo"},
	}
	for _, x := range s.decodable(xs) {
		for k := range len(x.enc) {
			for _, run := range runs {
				name := x.name + " cut at byte " + strconv.Itoa(k) + run.how
				_, err := s.streamed(tb, bytes.NewReader(x.enc[:k]), len(x.enc), kanon.StreamOptions{}, run.chunk,
					false)
				assert.ErrorIs(tb, err, io.ErrUnexpectedEOF, name+": the stream decoder reports the end of the reader")
				got := assert.ErrorAs[*kanon.DecodeError](tb, err,
					name+": the stream decoder returns a *kanon.DecodeError")
				assert.Equal(tb, got.Offset, k,
					name+": the error locates the first byte that the reader does not return")
			}
		}
	}
}

// streamLimits checks the buffer limit of the stream decoder of T on the
// encoding of each sample of xs that decodes: under the buffer limit that the
// encoding needs, its decoded fields together or its longest element, as
// [suite.streamParts] measures them, the stream decodes it as the reference
// decode, and under one byte less it fails with an error that wraps
// kanon.ErrLimit. A limit of 0 selects kanon.DefaultBuffer, so an encoding
// that needs one byte or none has no lower limit.
func (s *suite[T, P]) streamLimits(tb assert.TB, xs []sample[T]) {
	tb.Helper()
	for _, x := range s.decodable(xs) {
		want, wantErr := s.reference(probe{data: x.enc})
		decoded, longest, _ := s.streamParts(x.enc)
		limit := max(decoded, longest)
		got, err := s.streamed(tb, bytes.NewReader(x.enc), len(x.enc), kanon.StreamOptions{Buffer: limit}, readChunk,
			false)
		s.same(tb, x.name+": the stream decoder under the buffer limit that the encoding needs", got, err, want,
			wantErr)
		if limit < 2 {
			continue
		}
		_, err = s.streamed(tb, bytes.NewReader(x.enc), len(x.enc), kanon.StreamOptions{Buffer: limit - 1}, readChunk,
			false)
		assert.ErrorIs(tb, err, kanon.ErrLimit,
			x.name+": the stream decoder fails under a buffer limit one byte below the encoding")
	}
}

// streamLengths checks that Len of the stream decoder of T returns, after
// each call of Next that returns a streamed field, the length that the
// encoding of each sample of xs that decodes declares for that occurrence of
// the field, as [suite.streamParts] reads it.
func (s *suite[T, P]) streamLengths(tb assert.TB, xs []sample[T]) {
	tb.Helper()
	for _, x := range s.decodable(xs) {
		_, _, want := s.streamParts(x.enc)
		var v T
		st := s.stream(bytes.NewReader(x.enc), int64(len(x.enc)), &v, kanon.StreamOptions{})
		var got []int64
		for {
			if _, err := st.Next(); err != nil {
				break
			}
			got = append(got, st.Len())
		}
		assert.Equal(tb, got, want, x.name+": Len returns the length that the encoding declares for each value",
			assert.EquateEmpty())
	}
}

// streamSize checks that the stream decoder of T fails with
// kanon.ErrStreamSize for a negative size, at every call of Next.
func (s *suite[T, P]) streamSize(tb assert.TB) {
	tb.Helper()
	var v T
	st := s.stream(bytes.NewReader(nil), -1, &v, kanon.StreamOptions{})
	_, err := st.Next()
	assert.ErrorIs(tb, err, kanon.ErrStreamSize, "Next returns kanon.ErrStreamSize for a negative size")
	_, err = st.Next()
	assert.ErrorIs(tb, err, kanon.ErrStreamSize, "Next returns kanon.ErrStreamSize again")
}

// streamAllocs checks that the stream decoder of T, after Reset, reads the
// encoding of every sample that the decode allocation check measures, which
// it read before, without an allocation, as [suite.drainer] reads it.
func (s *suite[T, P]) streamAllocs(tb assert.TB) {
	tb.Helper()
	for _, x := range s.measured(s.r.decodeAllocates) {
		var v T
		r := bytes.NewReader(x.enc)
		pass := s.drainer(s.stream(r, int64(len(x.enc)), &v, kanon.StreamOptions{}), r, x.enc, &v)
		_ = pass()
		msg := x.name + ": the stream decoder allocates nothing for an encoding that it decoded before"
		assert.MaxAllocs(tb, func() { _ = pass() }, 0, msg)
	}
}

// streamed decodes data, size bytes from r, with the stream decoder of T and
// opts, as its caller decodes it, and returns the receiver and the first
// error of the stream, nil at its end. Per streamed field it reads the bytes
// of a byte slice or a string through Read chunk bytes at a time, or through
// WriteTo when chunk is writeTo, and decodes every element of a slice after
// its length into a new element, which it copies with CloneKanon, since the
// strings of an element alias the buffers of the stream. It then sets each
// streamed field to the value that the decode sets: the bytes of the last
// occurrence of a byte slice or a string, and the elements of every
// occurrence of a slice. With skip set it reads no streamed value and leaves
// every streamed field of the receiver empty. It fails tb when the stream
// returns another number of bytes than Len declares.
func (s *suite[T, P]) streamed(
	tb assert.TB,
	r io.Reader,
	size int,
	opts kanon.StreamOptions,
	chunk int,
	skip bool,
) (T, error) {
	tb.Helper()
	var got T
	st := s.stream(r, int64(size), &got, opts)
	values := make(map[*field]reflect.Value)
	for {
		num, err := st.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return got, err
		}
		f := s.l.byNumber(uint64(num))
		if skip {
			continue
		}
		if f.shape.kind == kindSlice {
			values[f], err = decodeElements(st, f, values[f])
		} else {
			values[f], err = readValue(tb, st, f, chunk)
		}
		if err != nil {
			return got, err
		}
	}
	v := reflect.ValueOf(&got).Elem()
	for f, x := range values {
		f.of(v).Set(x)
	}
	return got, nil
}

// drainer returns the function that reads enc to its end with st, as the
// generated code and its caller read it, over r, which it resets to enc, into
// m: the bytes of each streamed byte slice and string into a buffer of
// readChunk bytes, and each element of each streamed slice into one element
// per slice, whose decode method reads the length of the element itself.
// The function returns the error that ends the encoding: io.EOF, or the first
// error of st. It allocates nothing beyond the allocations of st.
func (s *suite[T, P]) drainer(st Stream[T], r *bytes.Reader, enc []byte, m *T) func() error {
	rd, _ := st.(io.Reader)
	el, _ := st.(elements)
	p := make([]byte, readChunk)
	var slices []int
	var elems []any
	for _, f := range s.streamFields {
		if f.shape.kind == kindSlice {
			slices = append(slices, f.Number)
			elems = append(elems, reflect.New(f.shape.elem.typ).Interface())
		}
	}
	return func() error {
		r.Reset(enc)
		st.Reset(r, int64(len(enc)), m)
		for {
			num, err := st.Next()
			if err != nil {
				return err
			}
			for rd != nil {
				if _, err := rd.Read(p); err != nil {
					break
				}
			}
			for k, n := range slices {
				for n == num {
					if err := el.Decode(n, elems[k]); err != nil {
						break
					}
				}
			}
		}
	}
}

// streamParts measures enc, an encoding of T that decodes, as the stream
// decoder of T reads it: it returns the length of its decoded fields
// together, each with its tag, the length of its longest element of a
// streamed slice, and the length that it declares for each occurrence of a
// streamed field, in the order of the encoding.
func (s *suite[T, P]) streamParts(enc []byte) (int, int, []int64) {
	decoded, longest := 0, 0
	var lengths []int64
	for i := 0; i < len(enc); {
		tag, n := wire.Uvarint(enc[i:])
		k, _ := wire.Skip(enc[i+n:], tag, s.l.name, 0, i)
		f := s.l.byNumber(tag >> 3)
		if f == nil || !f.Stream {
			decoded += n + k
			i += n + k
			continue
		}
		l, ln := wire.Uvarint(enc[i+n:])
		lengths = append(lengths, int64(l))
		for j := i + n + ln; f.shape.kind == kindSlice && j < i+n+ln+int(l); {
			el, en := wire.Uvarint(enc[j:])
			longest = max(longest, int(el))
			j += en + int(el)
		}
		i += n + k
	}
	return decoded, longest, lengths
}

// decodable returns the samples of xs that encode and whose encoding the
// reference decode decodes, in their order.
func (s *suite[T, P]) decodable(xs []sample[T]) []sample[T] {
	var out []sample[T]
	for _, x := range encodes(xs) {
		if _, err := s.reference(probe{data: x.enc}); err == nil {
			out = append(out, x)
		}
	}
	return out
}
