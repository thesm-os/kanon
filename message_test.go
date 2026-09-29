// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/iface"
	"go.thesmos.sh/kanon/internal/fixture/union"
	"go.thesmos.sh/kanon/internal/fixture/validate"
)

// defaultDepth pins the nesting limit of the zero Options.
const defaultDepth = 100

// Vectors of the repeated fields of the wire format specification.
var (
	// unionTwice is field 1 of union.Containers, the union member Slice,
	// twice: [1] and then [2].
	unionTwice = []byte{0x0a, 0x01, 0x02, 0x0a, 0x01, 0x04}
	// interfaceTwice is field 8 of iface.Variants, the interface Nested,
	// twice with the concrete type []int32 of number 1: [1] and then [2].
	interfaceTwice = []byte{0x42, 0x03, 0x01, 0x01, 0x02, 0x42, 0x03, 0x01, 0x01, 0x04}
)

// levelAbove is field 1 of validate.Values, the Level 4, which its
// ValidateKanon rejects: the tag of field 1 and the varint 4.
var levelAbove = []byte{0x08, 0x04}

// Names of the field of levelAbove, which the errors of ValidateKanon name.
const (
	valuesType  = "Values"
	levelField  = "Level"
	levelNumber = 1
	// levelOffset is the offset of the value of levelAbove, after its tag.
	levelOffset = 1
)

func TestMessage(t *testing.T) {
	t.Parallel()
	t.Run("DecodeKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("replaces a union member with its second occurrence", func(t *testing.T) {
			t.Parallel()
			var c union.Containers
			assert.NoError(t, c.DecodeKanon(unionTwice, kanon.Options{}), "DecodeKanon decodes the vector")
			assert.Equal(t, c, union.Containers{Kind: union.ContainerKindSlice, Slice: []int32{2}},
				"the second occurrence of the member replaces the union")
		})
		t.Run("merges a second occurrence of an interface that stores the same concrete type", func(t *testing.T) {
			t.Parallel()
			var v iface.Variants
			assert.NoError(t, v.DecodeKanon(interfaceTwice, kanon.Options{}), "DecodeKanon decodes the vector")
			assert.Equal(t, v.Nested, any([]int32{1, 2}), "the slices of the two occurrences merge")
		})
	})
	t.Run("MergeKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("merges an encoding as a decode of the encoding of the receiver followed by it", func(t *testing.T) {
			t.Parallel()
			half := len(interfaceTwice) / 2
			var merged, whole iface.Variants
			assert.NoError(t, merged.DecodeKanon(interfaceTwice[:half], kanon.Options{}),
				"DecodeKanon decodes the first occurrence")
			assert.NoError(t, merged.MergeKanon(interfaceTwice[half:], kanon.Options{}),
				"MergeKanon merges the second occurrence")
			assert.NoError(t, whole.DecodeKanon(interfaceTwice, kanon.Options{}), "DecodeKanon decodes both")
			assert.Equal(t, merged.Nested, whole.Nested, "the merge equals the decode of the concatenation")
		})
	})
}

func TestValidator(t *testing.T) {
	t.Parallel()
	t.Run("MarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns an EncodeError with the error of ValidateKanon for a value that it rejects", func(t *testing.T) {
			t.Parallel()
			_, err := (&validate.Values{Level: validate.LevelMax + 1}).MarshalBinary()
			e := assert.ErrorAs[*kanon.EncodeError](t, err, "MarshalBinary returns an EncodeError")
			want := kanon.EncodeError{
				Type:   valuesType,
				Field:  levelField,
				Number: levelNumber,
				Err:    validate.ErrLevel,
			}
			assert.Equal(t, *e, want, "the EncodeError names the field and wraps the error of ValidateKanon")
		})
	})
	t.Run("DecodeKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a DecodeError with the error of ValidateKanon at the offset of a value that it rejects",
			func(t *testing.T) {
				t.Parallel()
				var v validate.Values
				err := v.DecodeKanon(levelAbove, kanon.Options{})
				e := assert.ErrorAs[*kanon.DecodeError](t, err, "DecodeKanon returns a DecodeError")
				want := kanon.DecodeError{
					Type:   valuesType,
					Field:  levelField,
					Number: levelNumber,
					Offset: levelOffset,
					Err:    validate.ErrLevel,
				}
				assert.Equal(t, *e, want,
					"the DecodeError names the field and the offset of the value, and wraps the error of ValidateKanon")
			})
	})
}

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("Source", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a copy of data at offset 0 without a slab", func(t *testing.T) {
			t.Parallel()
			data := []byte("abc")
			slab, off := kanon.Options{Offset: 7}.Source(data)
			data[0] = 'x'
			assert.Equal(t, slab, "abc", "the slab is a copy that a later write to data does not change")
			assert.Equal(t, off, 0, "data starts the copy")
		})
		t.Run("returns Slab and Offset", func(t *testing.T) {
			t.Parallel()
			slab, off := kanon.Options{Slab: "xxabc", Offset: 2}.Source([]byte("abc"))
			assert.Equal(t, slab, "xxabc", "the slab is Slab")
			assert.Equal(t, off, 2, "the offset is Offset")
		})
	})
	t.Run("Limit", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			depth int
			want  int
		}{
			{name: "returns DefaultDepth for 0", depth: 0, want: defaultDepth},
			{name: "returns a positive Depth", depth: 3, want: 3},
			{name: "returns 0 for a negative Depth", depth: -1, want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, kanon.Options{Depth: c.depth}.Limit(), c.want, "Limit returns the nesting limit")
			})
		}
	})
	t.Run("DefaultDepth", func(t *testing.T) {
		t.Parallel()
		t.Run("is 100", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.DefaultDepth, defaultDepth, "the default nesting limit is 100 levels")
		})
	})
}
