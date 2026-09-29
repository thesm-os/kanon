// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/internal/inspect"
	"go.thesmos.sh/kanon/wire"
)

// depth is the depth of the struct readings in the tests, the default depth
// of a decode.
const depth = kanon.DefaultDepth

// inputLoc is the location that the errors of Parse name.
const inputLoc = "input"

// unhex returns the bytes that the hexadecimal digits of s spell, with
// spaces between them.
func unhex(tb testing.TB, s string) []byte {
	tb.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	assert.NoError(tb, err, "the hexadecimal digits decode")
	return b
}

func TestField(t *testing.T) {
	t.Parallel()
	t.Run("Parse", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name  string
			give  string
			depth int
			want  []inspect.Field
		}{
			{name: "returns no fields for empty input", depth: depth},
			{
				name:  "returns a varint with the bytes of its varint",
				give:  "08 96 01",
				depth: depth,
				want:  []inspect.Field{{Number: 1, Wire: wire.Varint, Value: []byte{0x96, 0x01}}},
			},
			{
				name:  "returns a varint that is longer than its shortest form",
				give:  "08 80 00",
				depth: depth,
				want:  []inspect.Field{{Number: 1, Wire: wire.Varint, Value: []byte{0x80, 0x00}}},
			},
			{
				name:  "returns a fixed64 with its eight bytes",
				give:  "21 01 02 03 04 05 06 07 08",
				depth: depth,
				want: []inspect.Field{
					{Number: 4, Wire: wire.Fixed64, Value: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}},
				},
			},
			{
				name:  "returns a fixed32 with its four bytes",
				give:  "2d 01 02 03 04",
				depth: depth,
				want:  []inspect.Field{{Number: 5, Wire: wire.Fixed32, Value: []byte{0x01, 0x02, 0x03, 0x04}}},
			},
			{
				name:  "returns a bytes value without its length",
				give:  "0a 01 78",
				depth: depth,
				want:  []inspect.Field{{Number: 1, Wire: wire.Bytes, Value: []byte{0x78}}},
			},
			{
				name:  "returns an empty bytes value without a struct reading",
				give:  "0a 00",
				depth: depth,
				want:  []inspect.Field{{Number: 1, Wire: wire.Bytes, Value: []byte{}}},
			},
			{
				name:  "returns the struct reading of a bytes value that parses to its end",
				give:  "0a 02 08 01",
				depth: depth,
				want: []inspect.Field{{
					Number: 1, Wire: wire.Bytes, Value: []byte{0x08, 0x01},
					Fields: []inspect.Field{{Number: 1, Wire: wire.Varint, Value: []byte{0x01}}},
				}},
			},
			{
				name:  "returns no struct reading at depth 0",
				give:  "0a 02 08 01",
				depth: 0,
				want:  []inspect.Field{{Number: 1, Wire: wire.Bytes, Value: []byte{0x08, 0x01}}},
			},
			{
				name:  "returns struct readings down to depth levels below the input",
				give:  "0a 04 0a 02 08 01",
				depth: 1,
				want: []inspect.Field{{
					Number: 1, Wire: wire.Bytes, Value: []byte{0x0a, 0x02, 0x08, 0x01},
					Fields: []inspect.Field{{Number: 1, Wire: wire.Bytes, Value: []byte{0x08, 0x01}}},
				}},
			},
			{
				name:  "returns the offsets of the tags in the input",
				give:  "0a 01 78 10 0e",
				depth: depth,
				want: []inspect.Field{
					{Number: 1, Wire: wire.Bytes, Value: []byte{0x78}},
					{Number: 2, Wire: wire.Varint, Offset: 3, Value: []byte{0x0e}},
				},
			},
			{
				name:  "returns the offsets of the fields of a struct reading in its value",
				give:  "0a 04 08 01 10 02",
				depth: depth,
				want: []inspect.Field{{
					Number: 1, Wire: wire.Bytes, Value: []byte{0x08, 0x01, 0x10, 0x02},
					Fields: []inspect.Field{
						{Number: 1, Wire: wire.Varint, Value: []byte{0x01}},
						{Number: 2, Wire: wire.Varint, Offset: 2, Value: []byte{0x02}},
					},
				}},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := inspect.Parse(unhex(t, tt.give), tt.depth)
				assert.NoError(t, err, "Parse reads the input")
				assert.Equal(t, got, tt.want, "Parse returns the fields")
			})
		}
		failures := []struct {
			name string
			give string
			// wantFields is the number of fields before the malformed one.
			wantFields int
			wantOffset int
			wantErr    error
		}{
			{
				name: "returns io.ErrUnexpectedEOF for input that ends inside a tag",
				give: "80", wantErr: io.ErrUnexpectedEOF,
			},
			{
				name: "returns kanon.ErrMalformed for a tag of more than 64 bits",
				give: "80 80 80 80 80 80 80 80 80 02", wantErr: kanon.ErrMalformed,
			},
			{name: "returns kanon.ErrMalformed for field number 0", give: "00 00", wantErr: kanon.ErrMalformed},
			{name: "returns kanon.ErrMalformed for wire format 3", give: "0b 00", wantErr: kanon.ErrMalformed},
			{
				name: "returns io.ErrUnexpectedEOF for a varint that ends early",
				give: "08 01 10 80", wantFields: 1, wantOffset: 2, wantErr: io.ErrUnexpectedEOF,
			},
			{
				name: "returns io.ErrUnexpectedEOF for a fixed64 of fewer than eight bytes",
				give: "21 01 02 03 04 05 06 07", wantErr: io.ErrUnexpectedEOF,
			},
			{
				name: "returns io.ErrUnexpectedEOF for a fixed32 of fewer than four bytes",
				give: "2d 01 02 03", wantErr: io.ErrUnexpectedEOF,
			},
			{
				name: "returns io.ErrUnexpectedEOF for a length that runs past the input",
				give: "0a 02 78", wantErr: io.ErrUnexpectedEOF,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := inspect.Parse(unhex(t, tt.give), depth)
				assert.ErrorIs(t, err, tt.wantErr, "Parse fails for the malformed field")
				decodeErr := assert.ErrorAs[*kanon.DecodeError](t, err, "Parse returns a decode error")
				assert.Equal(t, decodeErr.Offset, tt.wantOffset, "the error is at the tag of the malformed field")
				assert.Equal(t, decodeErr.Type, inputLoc, "the error names the input")
				assert.Length(t, got, tt.wantFields, "Parse returns the fields before the malformed one")
			})
		}
	})
}

func FuzzField(f *testing.F) {
	for _, seed := range []string{
		"0a 01 78 10 0e", "0a 04 0a 02 08 01", "21 01 02 03 04 05 06 07 08", "2d ffffffff", "0a 02 68 69",
		"12 03 02 04 06", "08 80 80 80 80 80 80 80 80 80 01", "0b",
	} {
		f.Add(unhex(f, seed))
	}
	record := view.Record{Text: "kanon", Children: []int32{1, -1}, Item: view.Item{Name: "x", Count: 7}}
	enc, err := record.MarshalBinary()
	assert.NoError(f, err, "the record encodes")
	f.Add(enc)
	f.Fuzz(func(t *testing.T, data []byte) {
		fields, err := inspect.Parse(data, depth)
		var text, js bytes.Buffer
		assert.NoError(t, inspect.WriteText(&text, fields), "WriteText writes the fields")
		assert.NoError(t, inspect.WriteJSON(&js, fields), "WriteJSON writes the fields")
		assert.True(t, json.Valid(js.Bytes()), "WriteJSON writes valid JSON")
		if err == nil {
			offsets := make([]int, len(fields))
			for k, fd := range fields {
				offsets[k] = fd.Offset
			}
			assert.True(t, slices.IsSorted(offsets), "Parse returns the fields in the order of the input")
		}
		var got view.Record
		if got.UnmarshalBinary(data) != nil {
			return
		}
		canonical, err := got.MarshalBinary()
		if err != nil {
			return
		}
		fields, err = inspect.Parse(canonical, depth)
		assert.NoError(t, err, "Parse reads the encoding that the generated code writes")
		numbers := make([]uint64, len(fields))
		for k, fd := range fields {
			numbers[k] = fd.Number
		}
		assert.True(t, slices.IsSorted(numbers), "Parse returns the fields of an encoder in ascending field number")
	})
}
