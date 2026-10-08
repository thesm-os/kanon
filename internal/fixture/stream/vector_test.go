// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package stream_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/stream"
)

// bundle is the encoding of the Bundle of the test vectors of the stream
// decoder: {Name: "b", Data: {1, 2, 3}, Members: {{Label: "a"}, {Count: 1}}}.
var bundle = []byte{
	0x0a, 0x01, 0x62, 0x12, 0x03, 0x01, 0x02, 0x03, 0x1a, 0x07, 0x03, 0x0a, 0x01, 0x61, 0x02, 0x10, 0x02,
}

// vector is the value of the test vectors of the stream decoder, whose
// encoding is bundle.
var vector = stream.Bundle{
	Name:    "b",
	Data:    []byte{1, 2, 3},
	Members: []stream.Inner{{Label: "a"}, {Count: 1}},
}

// results are the results of the calls of a stream over bundle, as
// [transcript] records them.
var results = []string{
	"Next 2", "Len 3", "Read 010203", "Next 3", "Len 7", "Element 3", "DecodeMembers {a 0}", "Element 2",
	"DecodeMembers { 1}",
}

func TestVector(t *testing.T) {
	t.Parallel()
	t.Run("Bundle", func(t *testing.T) {
		t.Parallel()
		t.Run("MarshalBinary", func(t *testing.T) {
			t.Parallel()
			t.Run("writes the encoding of the test vectors", func(t *testing.T) {
				t.Parallel()
				got, err := vector.MarshalBinary()
				assert.NoError(t, err, "MarshalBinary encodes the value")
				assert.Equal(t, got, bundle, "MarshalBinary writes the 17 bytes of the test vectors")
			})
		})
	})
	t.Run("BundleStream", func(t *testing.T) {
		t.Parallel()
		t.Run("Next", func(t *testing.T) {
			t.Parallel()
			t.Run("returns the results of the test vectors", func(t *testing.T) {
				t.Parallel()
				var b stream.Bundle
				got, err := transcript(stream.NewBundleStream(bytes.NewReader(bundle), int64(len(bundle)), &b,
					kanon.StreamOptions{}))
				assert.NoError(t, err, "the stream decodes the encoding to its end")
				assert.Equal(t, got, results, "the calls of the stream return the results of the test vectors")
				assert.Equal(t, b, stream.Bundle{Name: "b"}, "the receiver has the name and no streamed value")
			})
			failures := []struct {
				name   string
				enc    []byte
				reader io.Reader
				buffer int
				want   []string
				decode bool
			}{
				{
					name:   "returns io.ErrUnexpectedEOF for a reader that returns 10 bytes",
					enc:    bundle,
					reader: bytes.NewReader(bundle[:10]),
					want: []string{
						"Next 2", "Len 3", "Read 010203", "Next 3", "Len 7",
						"Element: kanon: Bundle.Members (field 3) at offset 10: unexpected EOF",
					},
				},
				{
					name: "returns io.ErrUnexpectedEOF for a length of Members that runs past the encoding",
					enc: []byte{
						0x0a, 0x01, 0x62, 0x12, 0x03, 0x01, 0x02, 0x03, 0x1a,
						0x08, 0x03, 0x0a, 0x01, 0x61, 0x02, 0x10, 0x02,
					},
					want: []string{
						"Next 2", "Len 3", "Read 010203",
						"Next: kanon: Bundle.Members (field 3) at offset 9: unexpected EOF",
					},
					decode: true,
				},
				{
					name: "returns io.ErrUnexpectedEOF for the length of an element that runs past Members",
					enc: []byte{
						0x0a, 0x01, 0x62, 0x12, 0x03, 0x01, 0x02, 0x03, 0x1a,
						0x07, 0x03, 0x0a, 0x01, 0x61, 0x03, 0x10, 0x02,
					},
					want: []string{
						"Next 2", "Len 3", "Read 010203", "Next 3", "Len 7", "Element 3", "DecodeMembers {a 0}",
						"Element: kanon: Bundle.Members (field 3) at offset 14: unexpected EOF",
					},
					decode: true,
				},
				{
					name: "returns kanon.ErrNotCanonical for Data at a value that the encoding leaves out",
					enc:  []byte{0x0a, 0x01, 0x62, 0x12, 0x00, 0x1a, 0x07, 0x03, 0x0a, 0x01, 0x61, 0x02, 0x10, 0x02},
					want: []string{
						"Next: kanon: Bundle.Data (field 2) at offset 3: field at a value that the encoding leaves out",
					},
					decode: true,
				},
				{
					name: "returns kanon.ErrNotCanonical for Data after Members, after the elements of Members",
					enc: []byte{
						0x0a, 0x01, 0x62, 0x1a, 0x07, 0x03, 0x0a, 0x01, 0x61,
						0x02, 0x10, 0x02, 0x12, 0x03, 0x01, 0x02, 0x03,
					},
					want: []string{
						"Next 3", "Len 7", "Element 3", "DecodeMembers {a 0}", "Element 2", "DecodeMembers { 1}",
						"Next: kanon: Bundle at offset 12: field 2 after field 3",
					},
					decode: true,
				},
				{
					name:   "returns kanon.ErrLimit for the run of Name under a buffer of 2 bytes",
					enc:    bundle,
					buffer: 2,
					want:   []string{"Next: kanon: Bundle at offset 0: value takes the buffer past 2 bytes"},
				},
				{
					name:   "returns kanon.ErrLimit for the first element under a buffer of 2 bytes",
					enc:    []byte{0x1a, 0x07, 0x03, 0x0a, 0x01, 0x61, 0x02, 0x10, 0x02},
					buffer: 2,
					want: []string{
						"Next 3", "Len 7", "Element 3",
						"DecodeMembers: kanon: Bundle.Members (field 3) at offset 2: " +
							"value takes the buffer past 2 bytes",
					},
				},
			}
			for _, tt := range failures {
				t.Run(tt.name, func(t *testing.T) {
					t.Parallel()
					r := tt.reader
					if r == nil {
						r = bytes.NewReader(tt.enc)
					}
					var b stream.Bundle
					got, err := transcript(stream.NewBundleStream(r, int64(len(tt.enc)), &b,
						kanon.StreamOptions{Buffer: tt.buffer}))
					assert.Equal(t, got, tt.want, "the stream fails at the offset of the test vectors")
					if tt.decode {
						assert.Equal(t, new(stream.Bundle).UnmarshalBinary(tt.enc), err,
							"DecodeKanon returns the error of the stream")
					}
				})
			}
			t.Run("returns the results of the test vectors from a reader of one byte per call", func(t *testing.T) {
				t.Parallel()
				var b stream.Bundle
				got, err := transcript(stream.NewBundleStream(iotest.OneByteReader(bytes.NewReader(bundle)),
					int64(len(bundle)), &b, kanon.StreamOptions{}))
				assert.NoError(t, err, "the stream decodes the encoding to its end")
				assert.Equal(t, got, results, "the calls of the stream return the results of the test vectors")
			})
		})
		t.Run("WriteTo", func(t *testing.T) {
			t.Parallel()
			t.Run("writes the bytes of Data of the test vectors through io.Copy", func(t *testing.T) {
				t.Parallel()
				var b stream.Bundle
				s := stream.NewBundleStream(bytes.NewReader(bundle), int64(len(bundle)), &b, kanon.StreamOptions{})
				f, err := s.Next()
				assert.NoError(t, err, "Next opens the first streamed field")
				assert.Equal(t, f, stream.BundleFieldData, "the first streamed field is Data")
				var data bytes.Buffer
				n, err := io.Copy(&data, s)
				assert.NoError(t, err, "io.Copy copies the value of Data")
				expect.Equal(t, n, int64(3), "io.Copy copies the 3 bytes of Data")
				expect.Equal(t, data.Bytes(), []byte{1, 2, 3}, "io.Copy copies the bytes of Data")
			})
		})
	})
}

// transcript reads the encoding of a Bundle with s to its end, as a caller
// reads it: per field, the field that Next returns and its length, the bytes
// of Data that Read returns, and the length of each element of Members that
// Element returns with the element that DecodeMembers decodes. It returns the
// results and the first error, with the method that returned it, and nil at
// the end of the encoding.
func transcript(s *stream.BundleStream) ([]string, error) {
	var out []string
	for {
		f, err := s.Next()
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return append(out, "Next: "+err.Error()), err
		}
		out = append(out, fmt.Sprintf("Next %d", f), fmt.Sprintf("Len %d", s.Len()))
		if f == stream.BundleFieldData {
			data, err := io.ReadAll(s)
			if err != nil {
				return append(out, "Read: "+err.Error()), err
			}
			out = append(out, fmt.Sprintf("Read %x", data))
			continue
		}
		for {
			l, err := s.Element()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return append(out, "Element: "+err.Error()), err
			}
			var e stream.Inner
			if err := s.DecodeMembers(&e); err != nil {
				return append(out, fmt.Sprintf("Element %d", l), "DecodeMembers: "+err.Error()), err
			}
			out = append(out, fmt.Sprintf("Element %d", l), fmt.Sprintf("DecodeMembers %v", e))
		}
	}
}
