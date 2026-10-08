// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/bound"
)

// Details of the errors of the test vectors, which state the bound of the
// field.
const (
	itemsDetail = "more elements than the max of 2"
	metaDetail  = "more elements than the max of 1"
)

func TestPage(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		decodes := []struct {
			name string
			give []byte
			want bound.Page
		}{
			{
				name: "decodes two items, the bound of Items",
				give: []byte{0x0a, 0x04, 0x01, 0x61, 0x01, 0x62},
				want: bound.Page{Items: [][]byte{{0x61}, {0x62}}},
			},
			{
				name: "decodes one entry, the bound of Meta",
				give: []byte{0x12, 0x03, 0x01, 0x78, 0x01},
				want: bound.Page{Meta: map[string]uint64{"x": 1}},
			},
		}
		for _, tt := range decodes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var got bound.Page
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
				name: "returns kanon.ErrMax at the third item",
				give: []byte{0x0a, 0x06, 0x01, 0x61, 0x01, 0x62, 0x01, 0x63},
				want: &kanon.DecodeError{
					Type: "Page", Field: "Items", Number: 1, Offset: 6, Detail: itemsDetail, Err: kanon.ErrMax,
				},
			},
			{
				name: "returns kanon.ErrMax at the third empty item",
				give: []byte{0x0a, 0x03, 0x00, 0x00, 0x00},
				want: &kanon.DecodeError{
					Type: "Page", Field: "Items", Number: 1, Offset: 4, Detail: itemsDetail, Err: kanon.ErrMax,
				},
			},
			{
				name: "returns kanon.ErrMax before the length of a third item that runs past the field",
				give: []byte{0x0a, 0x05, 0x01, 0x61, 0x01, 0x62, 0x05},
				want: &kanon.DecodeError{
					Type: "Page", Field: "Items", Number: 1, Offset: 6, Detail: itemsDetail, Err: kanon.ErrMax,
				},
			},
			{
				name: "returns kanon.ErrMax at the key of the second entry",
				give: []byte{0x12, 0x06, 0x01, 0x78, 0x01, 0x01, 0x79, 0x02},
				want: &kanon.DecodeError{
					Type: "Page", Field: "Meta", Number: 2, Offset: 5, Detail: metaDetail, Err: kanon.ErrMax,
				},
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := new(bound.Page).UnmarshalBinary(tt.give)
				got := assert.ErrorAs[*kanon.DecodeError](t, err, "UnmarshalBinary returns a *kanon.DecodeError")
				assert.Equal(t, got, tt.want, "the error names the field, the offset and the bound of the test vector")
			})
		}
	})
	t.Run("MarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns kanon.ErrMax for three items", func(t *testing.T) {
			t.Parallel()
			_, err := (&bound.Page{Items: [][]byte{{0x61}, {0x62}, {0x63}}}).MarshalBinary()
			got := assert.ErrorAs[*kanon.EncodeError](t, err, "MarshalBinary returns a *kanon.EncodeError")
			assert.Equal(t, got, &kanon.EncodeError{Type: "Page", Field: "Items", Number: 1, Err: kanon.ErrMax},
				"the error names the field past its bound")
		})
	})
}
