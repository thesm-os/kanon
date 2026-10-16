// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound_test

import (
	"bytes"
	"io"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/expect"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/bound"
)

// Details of the errors of the test vectors of Note. Each detail contains the
// bound of its field.
const (
	labelDetail = "longer than the max of 2"
	bodyDetail  = "longer than the max of 3"
)

// labelTwice is the encoding of a Note with two occurrences of Label. The
// first has three bytes and is longer than the bound of 2. The second has one
// byte.
var labelTwice = []byte{0x0a, 0x03, 0x61, 0x62, 0x63, 0x0a, 0x01, 0x61}

// errLabelPast is the error of a Label of three bytes at the start of an
// encoding. The byte past the bound is at offset 4.
var errLabelPast = &kanon.DecodeError{
	Type: "Note", Field: "Label", Number: 1, Offset: 4, Detail: labelDetail, Err: kanon.ErrMax,
}

// errBodyPast is the error of a Body of four bytes at the start of an
// encoding. The byte past the bound is at offset 5.
var errBodyPast = &kanon.DecodeError{
	Type: "Note", Field: "Body", Number: 2, Offset: 5, Detail: bodyDetail, Err: kanon.ErrMax,
}

func TestNote(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		decodes := []struct {
			name string
			give []byte
			want bound.Note
		}{
			{
				name: "decodes a label of two bytes, the bound of Label",
				give: []byte{0x0a, 0x02, 0x61, 0x62},
				want: bound.Note{Label: "ab"},
			},
			{
				name: "decodes a body of three bytes, the bound of Body",
				give: []byte{0x12, 0x03, 0x01, 0x02, 0x03},
				want: bound.Note{Body: []byte{0x01, 0x02, 0x03}},
			},
		}
		for _, tt := range decodes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got bound.Note
				assert.NoError(t, got.UnmarshalBinary(tt.give), "UnmarshalBinary decodes the test vector")
				assert.Equal(t, got, tt.want, "UnmarshalBinary decodes the value of the test vector")
			})
		}
		failures := []struct {
			name string
			give []byte
			want *kanon.DecodeError
		}{
			{
				name: "returns kanon.ErrMax at the third byte of a label",
				give: []byte{0x0a, 0x03, 0x61, 0x62, 0x63},
				want: errLabelPast,
			},
			{
				name: "returns io.ErrUnexpectedEOF for a label whose length runs past the input",
				give: []byte{0x0a, 0x05, 0x61, 0x62, 0x63},
				want: &kanon.DecodeError{Type: "Note", Field: "Label", Number: 1, Offset: 1, Err: io.ErrUnexpectedEOF},
			},
			{
				name: "returns kanon.ErrMax at the fourth byte of a body",
				give: []byte{0x12, 0x04, 0x01, 0x02, 0x03, 0x04},
				want: errBodyPast,
			},
			{
				name: "returns kanon.ErrMax for a label past its bound before a later label within it",
				give: labelTwice,
				want: errLabelPast,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := new(bound.Note).UnmarshalBinary(tt.give)
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "UnmarshalBinary returns a *kanon.DecodeError")
				assert.Equal(t, got, tt.want, "the error names the field, the offset and the bound of the test vector")
			})
		}
	})
	t.Run("MarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns kanon.ErrMax for a label of three bytes", func(t *testing.T) {
			t.Parallel()
			_, err := (&bound.Note{Label: "abc"}).MarshalBinary()
			got := assert.ErrorAs[*kanon.EncodeError](t, err, "MarshalBinary returns a *kanon.EncodeError")
			assert.Equal(t, got, &kanon.EncodeError{Type: "Note", Field: "Label", Number: 1, Err: kanon.ErrMax},
				"the error names the field past its bound")
		})
	})
	t.Run("NoteView", func(t *testing.T) {
		t.Parallel()
		t.Run("Label", func(t *testing.T) {
			t.Parallel()
			t.Run("returns kanon.ErrMax at the third byte of a label", func(t *testing.T) {
				t.Parallel()
				_, err := bound.NoteView([]byte{0x0a, 0x03, 0x61, 0x62, 0x63}).Label()
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "Label returns a *kanon.DecodeError")
				assert.Equal(t, got, errLabelPast, "the error names the field, the offset and the bound")
			})
			t.Run("returns the last label of an encoding whose earlier label is past its bound", func(t *testing.T) {
				t.Parallel()
				got, err := bound.NoteView(labelTwice).Label()
				expect.NoError(t, err, "Label reads the last occurrence of the field")
				expect.Equal(t, got, []byte{0x61}, "Label returns the bytes of the last occurrence")
			})
		})
		t.Run("IndexKanon", func(t *testing.T) {
			t.Parallel()
			t.Run("returns kanon.ErrMax for a label past its bound before a later label within it", func(t *testing.T) {
				t.Parallel()
				_, err := bound.NoteView(labelTwice).IndexKanon()
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "IndexKanon returns a *kanon.DecodeError")
				assert.Equal(t, got, errLabelPast, "the error names the field, the offset and the bound")
			})
		})
	})
	t.Run("NoteStream", func(t *testing.T) {
		t.Parallel()
		t.Run("Next", func(t *testing.T) {
			t.Parallel()
			t.Run("returns kanon.ErrMax for a body past its bound before it returns the field", func(t *testing.T) {
				t.Parallel()
				give := []byte{0x12, 0x04, 0x01, 0x02, 0x03, 0x04}
				var m bound.Note
				d := bound.NewNoteStream(bytes.NewReader(give), int64(len(give)), &m, kanon.StreamOptions{})
				f, err := d.Next()
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "Next returns a *kanon.DecodeError")
				expect.Equal(t, got, errBodyPast, "the error names the field, the offset and the bound")
				expect.Equal(t, f, bound.NoteField(0), "Next returns no field")
			})
		})
	})
}
