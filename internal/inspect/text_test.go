// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect_test

import (
	"errors"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/inspect"
)

// errWrite is the error of every write of a failingWriter.
var errWrite = errors.New("inspect_test: the write fails")

// failingWriter is a writer whose every write fails with errWrite.
type failingWriter struct{}

// Write returns errWrite.
func (failingWriter) Write([]byte) (int, error) { return 0, errWrite }

// text returns what WriteText writes for the fields of the struct encoding
// that the hexadecimal digits in give spell.
func text(t *testing.T, give string) string {
	t.Helper()
	fields, err := inspect.Parse(unhex(t, give), depth)
	assert.NoError(t, err, "Parse reads the input")
	var b strings.Builder
	assert.NoError(t, inspect.WriteText(&b, fields), "WriteText writes the fields")
	return b.String()
}

func TestText(t *testing.T) {
	t.Parallel()
	t.Run("WriteText", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give string
			want string
		}{
			{name: "writes nothing for no fields", give: "", want: ""},
			{name: "writes a varint of 0 without a comment", give: "08 00", want: "1: 0\n"},
			{name: "writes the zigzag value of a varint in a comment", give: "10 0e", want: "2: 14  # zigzag 7\n"},
			{
				name: "writes the largest varint as unsigned",
				give: "08 ffffffffffffffffff 01",
				want: "1: 18446744073709551615  # zigzag -9223372036854775808\n",
			},
			{
				name: "writes the float value of a fixed64 in a comment",
				give: "21 01 00 00 00 00 00 00 00",
				want: "4: 1i64  # float 5e-324\n",
			},
			{
				name: "writes a fixed64 of 0 without a signed value",
				give: "21 00 00 00 00 00 00 00 00",
				want: "4: 0i64  # float 0\n",
			},
			{
				name: "writes the signed value of a fixed64 with the high bit set",
				give: "21 ffffffffffffffff",
				want: "4: 18446744073709551615i64  # signed -1, float NaN\n",
			},
			{
				name: "writes the float value of a fixed32 in a comment",
				give: "2d 00 00 c0 3f",
				want: "5: 1069547520i32  # float 1.5\n",
			},
			{
				name: "writes the signed value of a fixed32 with the high bit set",
				give: "2d ffffffff",
				want: "5: 4294967295i32  # signed -1, float NaN\n",
			},
			{name: "writes an empty bytes value as braces", give: "1a 00", want: "3: {}\n"},
			{name: "writes text as a quoted string", give: "0a 01 78", want: "1: {\"x\"}\n"},
			{
				name: "writes text with escape sequences",
				give: "22 05 61 22 5c 09 0a",
				want: "4: {\"a\\\"\\\\\\x09\\n\"}\n",
			},
			{name: "writes text in UTF-8 as its bytes", give: "0a 02 c3 a9", want: "1: {\"é\"}\n"},
			{
				name: "writes bytes with a control character as hexadecimal digits",
				give: "12 01 01",
				want: "2: {`01`}  # varints 1, zigzag -1\n",
			},
			{
				name: "writes the varints of hexadecimal digits in a comment",
				give: "12 03 02 04 06",
				want: "2: {`020406`}  # varints 2 4 6, zigzag 1 2 3\n",
			},
			{
				name: "writes bytes that end inside a varint without varints",
				give: "12 02 02 ff",
				want: "2: {`02ff`}\n",
			},
			{
				name: "writes bytes with a varint of more than 64 bits without varints",
				give: "12 0a ffffffffffffffffff 7f",
				want: "2: {`ffffffffffffffffff7f`}\n",
			},
			{
				name: "writes the struct reading of a bytes value between braces",
				give: "0a 03 08 96 01",
				want: "1: {\n  1: 150  # zigzag 75\n}\n",
			},
			{
				name: "writes the text of a struct reading in a comment",
				give: "0a 02 68 69",
				want: "1: {  # \"hi\"\n  13: 105  # zigzag -53\n}\n",
			},
			{
				name: "indents each level of struct readings by two spaces",
				give: "0a 04 0a 02 08 01",
				want: "1: {\n  1: {\n    1: 1  # zigzag -1\n  }\n}\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, text(t, tt.give), tt.want, "WriteText writes the text notation")
			})
		}
		t.Run("returns the error of the writer", func(t *testing.T) {
			t.Parallel()
			fields, err := inspect.Parse(unhex(t, "08 00"), depth)
			assert.NoError(t, err, "Parse reads the input")
			assert.ErrorIs(t, inspect.WriteText(failingWriter{}, fields), errWrite, "WriteText fails as the write")
		})
	})
}
