// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/external"
	"go.thesmos.sh/kanon/internal/fixture/stream"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of a stream decoder.
const (
	streamSamplesCheck = "Next/decodes the encoding of each sample as the reference decode"
	streamCutsCheck    = "Next/returns io.ErrUnexpectedEOF at the offset at which the reader ends"
	streamLimitCheck   = "Next/returns kanon.ErrLimit under a buffer limit one byte below the encoding"
	streamSizeCheck    = "Next/returns kanon.ErrStreamSize for a negative size"
	streamAllocsCheck  = "Next/allocates nothing for an encoding that the stream decoded before"
	streamLengthsCheck = "Len/returns the length that the encoding declares for each streamed value"
)

// errStream is the error that a broken stream decoder returns in place of
// the error of the stream.
var errStream = errors.New("stream: broken")

// bundleFields describes the fields of stream.Bundle.
var bundleFields = []kanontest.Field{
	{Name: "Name", Number: 1},
	{Name: "Data", Number: 2, Stream: true},
	{Name: "Members", Number: 3, Stream: true},
}

// bundleSpec describes stream.Bundle, a canonical struct type with a streamed
// byte slice and a streamed slice, and its stream decoder.
var bundleSpec = kanontest.Spec[stream.Bundle]{
	Fields:    bundleFields,
	Canonical: true,
	Stream: func(r io.Reader, size int64, m *stream.Bundle, opts kanon.StreamOptions) kanontest.Stream[stream.Bundle] {
		return bundleStream{stream.NewBundleStream(r, size, m, opts)}
	},
}

// recordStreamSpec describes stream.Record, a struct type with a streamed
// string, byte slice and slices, tracked fields, a union and unknown fields,
// and its stream decoder.
var recordStreamSpec = kanontest.Spec[stream.Record]{
	Fields: []kanontest.Field{
		{Name: "ID", Number: 1},
		{Name: "Note", Number: 2, Stream: true},
		{Name: "Head", Number: 3},
		{Name: "Body", Number: 4, Stream: true},
		{Name: "Text", Number: 5, Union: "Kind", Case: stream.RecordKindText},
		{Name: "Parts", Number: 6, Stream: true},
		{Name: "Labels", Number: 7, Stream: true},
		{Name: "Number", Number: 8, Union: "Kind", Case: stream.RecordKindNumber},
		{Name: "Meta", Number: 9},
	},
	Unknown: "Rest",
	Stream: func(r io.Reader, size int64, m *stream.Record, opts kanon.StreamOptions) kanontest.Stream[stream.Record] {
		return recordStream{stream.NewRecordStream(r, size, m, opts)}
	},
}

// bundleStream adapts a stream.BundleStream to kanontest.Stream, as the test
// file that kanon generates adapts it.
type bundleStream struct {
	*stream.BundleStream
}

// Next returns the number of the next streamed field of stream.Bundle.
func (s bundleStream) Next() (int, error) {
	f, err := s.BundleStream.Next()
	return int(f), err
}

// Decode decodes the next element of Members into e.
func (s bundleStream) Decode(_ int, e any) error {
	return s.DecodeMembers(e.(*stream.Inner))
}

// recordStream adapts a stream.RecordStream to kanontest.Stream, as the test
// file that kanon generates adapts it.
type recordStream struct {
	*stream.RecordStream
}

// Next returns the number of the next streamed field of stream.Record.
func (s recordStream) Next() (int, error) {
	f, err := s.RecordStream.Next()
	return int(f), err
}

// Decode decodes the next element of the streamed slice num into e.
func (s recordStream) Decode(num int, e any) error {
	if num == int(stream.RecordFieldParts) {
		return s.DecodeParts(e.(*stream.Part))
	}
	return s.DecodeLabels(e.(*external.Label))
}

// readShort is a stream decoder of stream.Bundle whose Read drops the last
// byte of every read.
type readShort struct {
	bundleStream
}

// Read reads into p and reports one byte less than it read.
func (s readShort) Read(p []byte) (int, error) {
	n, err := s.bundleStream.Read(p)
	return max(n-1, 0), err
}

// writeShort is a stream decoder of stream.Bundle whose WriteTo drops the
// last byte of the value.
type writeShort struct {
	bundleStream
}

// WriteTo writes the value but its last byte to w.
func (s writeShort) WriteTo(w io.Writer) (int64, error) {
	var buf bytes.Buffer
	if _, err := s.bundleStream.WriteTo(&buf); err != nil {
		return 0, err
	}
	n, err := w.Write(buf.Bytes()[:max(buf.Len()-1, 0)])
	return int64(n), err
}

// elementSkip is a stream decoder of stream.Bundle whose Decode decodes two
// elements and returns the second.
type elementSkip struct {
	bundleStream
}

// Decode decodes the next element into e, and then the element after it,
// when the slice has one.
func (s elementSkip) Decode(num int, e any) error {
	if err := s.bundleStream.Decode(num, e); err != nil {
		return err
	}
	if _, err := s.Element(); errors.Is(err, io.EOF) {
		return nil
	}
	return s.bundleStream.Decode(num, e)
}

// writeErrors is a stream decoder of stream.Bundle whose WriteTo returns
// errStream in place of every error.
type writeErrors struct {
	bundleStream
}

// WriteTo writes the value to w, and returns errStream in place of an error.
func (s writeErrors) WriteTo(w io.Writer) (int64, error) {
	n, err := s.bundleStream.WriteTo(w)
	if err != nil {
		return n, errStream
	}
	return n, nil
}

// plainErrors is a stream decoder of stream.Bundle whose Next returns
// errStream in place of every error other than io.EOF.
type plainErrors struct {
	bundleStream
}

// Next returns the next streamed field, and errStream in place of an error.
func (s plainErrors) Next() (int, error) {
	n, err := s.bundleStream.Next()
	if err != nil && !errors.Is(err, io.EOF) {
		return 0, errStream
	}
	return n, err
}

// lenOff is a stream decoder of stream.Bundle whose Len counts one byte too
// many.
type lenOff struct {
	bundleStream
}

// Len returns one more than the declared length.
func (s lenOff) Len() int64 {
	return s.bundleStream.Len() + 1
}

// allocatingStream is a stream decoder of stream.Bundle whose Next allocates.
type allocatingStream struct {
	bundleStream
}

// Next returns the next streamed field, and allocates.
func (s allocatingStream) Next() (int, error) {
	escaped = make([]byte, 1)
	return s.bundleStream.Next()
}

// brokenBundle returns the Spec of stream.Bundle whose stream decoder the
// function adapt makes from the generated one.
func brokenBundle(adapt func(bundleStream) kanontest.Stream[stream.Bundle]) kanontest.Spec[stream.Bundle] {
	spec := bundleSpec
	spec.Stream = func(r io.Reader, size int64, m *stream.Bundle, opts kanon.StreamOptions) kanontest.Stream[stream.Bundle] {
		return adapt(bundleStream{stream.NewBundleStream(r, size, m, opts)})
	}
	return spec
}

func TestStream(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns checks that a generated canonical stream decoder passes", func(t *testing.T) {
			t.Parallel()
			passes(t, bundleSpec)
		})
		t.Run("returns checks that a generated stream decoder of every streamed kind passes", func(t *testing.T) {
			t.Parallel()
			passes(t, recordStreamSpec)
		})
		specs := []struct {
			name string
			spec func() []kanontest.Check
			want string
		}{
			{
				name: "returns one check for a Spec with a stream decoder and no streamed field",
				spec: func() []kanontest.Check {
					return kanontest.Checks(kanontest.Spec[stream.Bundle]{
						Fields: fields("Name", "Data", "Members"), Canonical: true, Stream: bundleSpec.Stream,
					})
				},
				want: "kanontest: the Spec names a stream decoder of stream.Bundle, and streams no field",
			},
			{
				name: "returns one check for a Spec with a streamed field and no stream decoder",
				spec: func() []kanontest.Check {
					return kanontest.Checks(kanontest.Spec[stream.Bundle]{Fields: bundleFields, Canonical: true})
				},
				want: "kanontest: the Spec streams field Data of stream.Bundle, and names no stream decoder",
			},
			{
				name: "returns one check for a Spec that streams a field whose type does not stream",
				spec: func() []kanontest.Check {
					spec := recordStreamSpec
					spec.Fields = append([]kanontest.Field{{Name: "ID", Number: 1, Stream: true}}, spec.Fields[1:]...)
					return kanontest.Checks(spec)
				},
				want: "kanontest: the Spec streams field ID of stream.Record, whose type int64 does not stream",
			},
		}
		for _, tt := range specs {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				checks := tt.spec()
				assert.Length(t, checks, 1, "Checks returns the check of the Spec alone")
				failures := assert.Rejects(t, checks[0].Name, checks[0].Run)
				assert.Length(t, failures, 1, "the check of the Spec fails once")
				got, _ := failures[0].Got()
				err, _ := got.(error)
				assert.Equal(t, err.Error(), tt.want, "the check states why the Spec does not describe the stream")
			})
		}
	})
	t.Run("Next", func(t *testing.T) {
		t.Parallel()
		failures := []struct {
			name  string
			spec  kanontest.Spec[stream.Bundle]
			check string
			want  string
		}{
			{
				name: "fails for a stream decoder whose Read drops a byte",
				spec: brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
					return readShort{s}
				}),
				check: streamSamplesCheck,
				want:  "the stream returns the bytes of the length that Len declares",
			},
			{
				name: "fails for a stream decoder whose WriteTo drops a byte",
				spec: brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
					return writeShort{s}
				}),
				check: streamSamplesCheck,
				want:  "the stream returns the bytes of the length that Len declares",
			},
			{
				name: "fails for a stream decoder that skips an element",
				spec: brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
					return elementSkip{s}
				}),
				check: streamSamplesCheck,
				want:  "the stream decoder decodes the value of the reference decode",
			},
			{
				name: "fails for a stream decoder that returns another error for the end of the reader",
				spec: brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
					return plainErrors{s}
				}),
				check: streamCutsCheck,
				want:  "the stream decoder reports the end of the reader",
			},
			{
				name: "fails for a stream decoder whose WriteTo returns another error for the end of the reader",
				spec: brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
					return writeErrors{s}
				}),
				check: streamCutsCheck,
				want:  "read through WriteTo: the stream decoder reports the end of the reader",
			},
			{
				name: "fails for a stream decoder that ignores the buffer limit",
				spec: func() kanontest.Spec[stream.Bundle] {
					spec := bundleSpec
					spec.Stream = func(r io.Reader, size int64, m *stream.Bundle, _ kanon.StreamOptions) kanontest.Stream[stream.Bundle] {
						return bundleStream{stream.NewBundleStream(r, size, m, kanon.StreamOptions{})}
					}
					return spec
				}(),
				check: streamLimitCheck,
				want:  "the stream decoder fails under a buffer limit one byte below the encoding",
			},
			{
				name: "fails for a stream decoder that reads a negative size as 0",
				spec: func() kanontest.Spec[stream.Bundle] {
					spec := bundleSpec
					spec.Stream = func(r io.Reader, size int64, m *stream.Bundle, opts kanon.StreamOptions) kanontest.Stream[stream.Bundle] {
						return bundleStream{stream.NewBundleStream(r, max(size, 0), m, opts)}
					}
					return spec
				}(),
				check: streamSizeCheck,
				want:  "Next returns kanon.ErrStreamSize for a negative size",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				rejects(t, tt.spec, tt.check, tt.want)
			})
		}
	})
	t.Run("Len", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a stream decoder whose Len counts one byte too many", func(t *testing.T) {
			t.Parallel()
			spec := brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] { return lenOff{s} })
			rejects(t, spec, streamLengthsCheck, "Len returns the length that the encoding declares for each value")
		})
	})
	t.Run("Bench", func(t *testing.T) {
		t.Parallel()
		t.Run("measures a generated stream decoder", func(t *testing.T) {
			t.Parallel()
			r := testing.Benchmark(func(b *testing.B) {
				b.Helper()
				kanontest.Bench(b, recordStreamSpec)
			})
			assert.InRange(t, r.N, 1, 1<<63, "Bench runs its sub-benchmarks")
		})
	})
}

// TestStreamAllocs runs the check of a stream decoder that counts
// allocations, which counts them while no parallel test runs, so that it
// and its subtests do not call t.Parallel.
func TestStreamAllocs(t *testing.T) {
	t.Run("Checks", func(t *testing.T) {
		t.Run("passes a generated stream decoder", func(t *testing.T) {
			counts(t, recordStreamSpec)
		})
		t.Run("fails for a stream decoder whose Next allocates", func(t *testing.T) {
			countsAllocations(t)
			spec := brokenBundle(func(s bundleStream) kanontest.Stream[stream.Bundle] {
				return allocatingStream{s}
			})
			rejects(t, spec, streamAllocsCheck, "the stream decoder allocates nothing")
		})
	})
}

func FuzzStream(f *testing.F) {
	kanontest.Fuzz(f, recordStreamSpec)
}
