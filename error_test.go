// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"errors"
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// Messages of the causes, pinned because callers log them.
const (
	malformedText = "kanon: malformed input"
	rangeText     = "kanon: value outside the range of the type"
	depthText     = "kanon: nested deeper than the limit"
	unknownText   = "kanon: interface type number not listed"
	unlistedText  = "kanon: type not listed in the tag option types"
	canonicalText = "kanon: input is not the canonical encoding"
)

func TestDecodeError(t *testing.T) {
	t.Parallel()
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			err  *kanon.DecodeError
			want string
		}{
			{
				name: "names the type, the field, its number, the offset and the detail",
				err: &kanon.DecodeError{
					Type: "Order", Field: "Count", Number: 2, Offset: 12, Detail: "varint overflows 64 bits",
					Err: kanon.ErrMalformed,
				},
				want: "kanon: Order.Count (field 2) at offset 12: varint overflows 64 bits",
			},
			{
				name: "names the type alone for an error without a field",
				err: &kanon.DecodeError{
					Type: "shop.Line", Offset: 3, Detail: "field number 0", Err: kanon.ErrMalformed,
				},
				want: "kanon: shop.Line at offset 3: field number 0",
			},
			{
				name: "states the cause without its prefix when the detail is empty",
				err:  &kanon.DecodeError{Type: "Order", Field: "ID", Number: 1, Offset: 5, Err: io.ErrUnexpectedEOF},
				want: "kanon: Order.ID (field 1) at offset 5: unexpected EOF",
			},
			{
				name: "drops the prefix of a kanon cause",
				err:  &kanon.DecodeError{Type: "Order", Offset: 0, Err: kanon.ErrDepth},
				want: "kanon: Order at offset 0: nested deeper than the limit",
			},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.err.Error(), c.want, "Error renders the location, then the detail")
			})
		}
	})
	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the cause for errors.Is and errors.As", func(t *testing.T) {
			t.Parallel()
			var err error = &kanon.DecodeError{Type: "Order", Err: io.ErrUnexpectedEOF}
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF, "errors.Is finds the cause")
			assert.ErrorIsNot(t, err, kanon.ErrMalformed, "errors.Is does not find another cause")
			got := assert.ErrorAs[*kanon.DecodeError](t, err, "errors.As finds the DecodeError")
			assert.Equal(t, got.Type, "Order", "errors.As returns the error itself")
		})
	})
}

func TestEncodeError(t *testing.T) {
	t.Parallel()
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		t.Run("names the field and states the cause without its prefix", func(t *testing.T) {
			t.Parallel()
			err := &kanon.EncodeError{Type: "Order", Field: "Shape", Number: 7, Err: kanon.ErrUnlistedType}
			assert.Equal(t, err.Error(), "kanon: Order.Shape (field 7): type not listed in the tag option types",
				"Error renders the field, then the cause")
		})
		t.Run("keeps the message of another cause", func(t *testing.T) {
			t.Parallel()
			err := &kanon.EncodeError{Type: "Order", Field: "Token", Number: 3, Err: errors.New("token: empty")}
			assert.Equal(t, err.Error(), "kanon: Order.Token (field 3): token: empty",
				"Error renders the message of a cause without a kanon prefix unchanged")
		})
	})
	t.Run("Unwrap", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the cause for errors.Is", func(t *testing.T) {
			t.Parallel()
			var err error = &kanon.EncodeError{Type: "Order", Field: "Shape", Number: 7, Err: kanon.ErrUnlistedType}
			assert.ErrorIs(t, err, kanon.ErrUnlistedType, "errors.Is finds the cause")
		})
	})
}

func TestCauses(t *testing.T) {
	t.Parallel()
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			err  error
			want string
		}{
			{name: "states ErrMalformed", err: kanon.ErrMalformed, want: malformedText},
			{name: "states ErrRange", err: kanon.ErrRange, want: rangeText},
			{name: "states ErrDepth", err: kanon.ErrDepth, want: depthText},
			{name: "states ErrUnknownType", err: kanon.ErrUnknownType, want: unknownText},
			{name: "states ErrUnlistedType", err: kanon.ErrUnlistedType, want: unlistedText},
			{name: "states ErrNotCanonical", err: kanon.ErrNotCanonical, want: canonicalText},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, c.err.Error(), c.want, "the cause has a fixed message")
			})
		}
	})
}
