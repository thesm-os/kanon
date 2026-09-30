// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/canonical"
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/iface"
	"go.thesmos.sh/kanon/internal/fixture/union"
	"go.thesmos.sh/kanon/internal/fixture/validate"
)

// codec.Hash and codec.Clip are the kanon.Sizer fixtures.
var (
	_ kanon.Sizer = codec.Hash{}
	_ kanon.Sizer = codec.Clip("")
)

// A codec.Clip longer than the 4 bytes that its SizeKanon counts, and the
// field of codec.Codecs that the error of its encode names.
const (
	clipLong   = "kanon"
	codecsType = "Codecs"
	clipField  = "Clip"
	clipNumber = 12
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

// encodingVectors lists the encoding vectors of the wire format specification
// in hex, each with a receiver of its vector type of package canonical, whose
// decode accepts only the canonical encoding.
var encodingVectors = []struct {
	msg  kanon.Message
	give string
}{
	{new(canonical.Inner), ""},
	{new(canonical.Inner), "0a 01 78 10 0e"},
	{new(canonical.Inner), "10 01"},
	{new(canonical.Inner), "10 d8 04"},
	{new(canonical.Numbers), "08 01 21 02 00 00 00 00 00 00 00 28 01 30 01"},
	{new(canonical.Numbers), "40 0d 48 01 51 02 00 00 00 00 00 00 00"},
	{new(canonical.Lists), "0a 03 01 00 02"},
	{new(canonical.Maps), "0a 08 01 61 01 31 01 62 01 32"},
	{new(canonical.Maps), "12 03 01 6e 03"},
	{new(canonical.Container), "12 00"},
	{new(canonical.Container), "1a 06 00 01 03 0a 01 63"},
	{new(canonical.Tree), "0a 01 72 12 05 01 03 0a 01 61"},
	{new(canonical.Times), "0a 02 08 02"},
	{new(canonical.Times), "0a 05 08 02 10 f4 03"},
	{new(canonical.Times), "0a 05 08 02 18 a0 38"},
	{new(canonical.Times), "0a 02 08 01"},
	{new(canonical.Times), "0a 00"},
	{new(canonical.Times), "0a 0a 08 ff db 8f f9 ce 03 18 a0 38"},
	{new(canonical.Times), "10 80 a8 d6 b9 07"},
	{new(canonical.Times), "1a 01 00"},
	{new(canonical.Times), "1a 08 07 08 ff db 8f f9 ce 03"},
	{new(canonical.Keys), "0a 0a 00 00 00 00 00 00 00 00 01 61"},
	{new(canonical.Bytes), "0a 03 01 02 03"},
	{new(canonical.Opaque), "0a 02 ab cd"},
	{new(canonical.Opaque), "12 04 00 00 00 01"},
	{new(canonical.Fixture), "0a 01 61 20 01 30 01"},
	{new(canonical.Fixture), "4a 04 01 62 01 61"},
	{new(canonical.Patch), "39 01 00 00 00 00 00 00 00"},
	{new(canonical.Patch), "42 20 01" + strings.Repeat(" 00", 31)},
	{new(canonical.Holder), "0a 0b 01 09 09 00 00 00 00 00 00 f8 3f"},
	{new(canonical.Holder), "0a 05 02 01 02 08 06"},
	{new(canonical.Holder), "0a 02 02 00"},
	{new(canonical.Holder), "12 01 74"},
	{new(canonical.Holder), "18 12"},
	{new(canonical.Holder), "18 00"},
	{new(canonical.Holder), "22 02 01 02"},
	{new(canonical.Holder), "22 01 00"},
	{new(canonical.Holder), "2a 02 00 0a"},
	{new(canonical.Holder), "32 0e 04 08 02 10 12 01 61 04 08 04 10 02 01 62"},
	{new(canonical.Holder), "3a 12 02 08 02 02 05 08 02 18 9f 38 06 05 08 02 18 a0 38 04"},
	{new(canonical.Holder), "42 00"},
	{new(canonical.Holder), "4a 07 03 05 01 01 73 02 01"},
	{new(canonical.Holder), "4a 02 01 00"},
}

// vector returns the bytes of s, a vector in hex whose bytes spaces
// separate, as the specifications write it.
func vector(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(s, " ", ""))
	assert.NoError(t, err, "the vector is hex")
	return b
}

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
		t.Run("decodes every encoding vector of the wire format to a value that encodes to the vector",
			func(t *testing.T) {
				t.Parallel()
				for _, v := range encodingVectors {
					data := vector(t, v.give)
					assert.NoError(t, v.msg.DecodeKanon(data, kanon.Options{}),
						v.give+": DecodeKanon decodes the vector")
					got, err := v.msg.MarshalBinary()
					assert.NoError(t, err, v.give+": MarshalBinary encodes the decoded value")
					assert.Equal(t, hex.EncodeToString(got), hex.EncodeToString(data),
						v.give+": the decoded value encodes to the vector")
				}
			})
		rejections := []struct {
			name string
			msg  kanon.Message
			give string
			off  int
		}{
			{
				name: "returns ErrNotCanonical at a tag that is not in its shortest form",
				msg:  new(canonical.Inner), give: "90 00 0e", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a value that is not in its shortest form",
				msg:  new(canonical.Inner), give: "10 8e 00", off: 1,
			},
			{
				name: "returns ErrNotCanonical at a length that is not in its shortest form",
				msg:  new(canonical.Inner), give: "0a 81 00 78", off: 1,
			},
			{
				name: "returns ErrNotCanonical at an element that is not in its shortest form",
				msg:  new(canonical.Lists), give: "0a 04 01 80 00 02", off: 3,
			},
			{
				name: "returns ErrNotCanonical at a field below the field before it",
				msg:  new(canonical.Inner), give: "10 0e 0a 01 78", off: 2,
			},
			{
				name: "returns ErrNotCanonical at the second occurrence of a field",
				msg:  new(canonical.Inner), give: "10 0e 10 0e", off: 2,
			},
			{
				name: "returns ErrNotCanonical at the second occurrence of a union member",
				msg:  new(canonical.Repeats), give: "0a 01 02 0a 01 04", off: 3,
			},
			{
				name: "returns ErrNotCanonical at the second occurrence of an interface",
				msg:  new(canonical.Repeats), give: "42 03 01 01 02 42 03 01 01 04", off: 5,
			},
			{
				name: "returns ErrNotCanonical at a field that the schema does not list",
				msg:  new(canonical.Inner), give: "18 01", off: 0,
			},
			{
				name: "returns ErrNotCanonical at the second member of a union",
				msg:  new(canonical.Holder), give: "12 01 74 18 12", off: 3,
			},
			{
				name: "returns ErrNotCanonical at a bool field of 2",
				msg:  new(canonical.Numbers), give: "28 02", off: 1,
			},
			{
				name: "returns ErrNotCanonical at a bool of 2 that a pointer points at",
				msg:  new(canonical.Numbers), give: "48 02", off: 1,
			},
			{
				name: "returns ErrNotCanonical at an integer field of 0",
				msg:  new(canonical.Inner), give: "10 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at an empty string field",
				msg:  new(canonical.Inner), give: "0a 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a bool field of false",
				msg:  new(canonical.Numbers), give: "28 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a byte array field of zero bytes",
				msg:  new(canonical.Fixture), give: "42 20" + strings.Repeat(" 00", 32), off: 0,
			},
			{
				name: "returns ErrNotCanonical at a struct field of no bytes",
				msg:  new(canonical.Nest), give: "0a 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a nil interface field",
				msg:  new(canonical.Holder), give: "0a 01 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a field of a type that encodes itself at its zero value",
				msg:  new(canonical.Opaque), give: "12 04 00 00 00 00", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a time field of the zero time in UTC",
				msg:  new(canonical.Times), give: "0a 07 08 ff db 8f f9 ce 03", off: 0,
			},
			{
				name: "returns ErrNotCanonical at a time field below the time field before it",
				msg:  new(canonical.Times), give: "0a 05 10 f4 03 08 02", off: 5,
			},
			{
				name: "returns ErrNotCanonical at a time field 1 of 0",
				msg:  new(canonical.Times), give: "0a 02 08 00", off: 2,
			},
			{
				name: "returns ErrNotCanonical at a time field 4",
				msg:  new(canonical.Times), give: "0a 02 20 01", off: 2,
			},
			{
				name: "returns ErrNotCanonical at a map key below the key before it",
				msg:  new(canonical.Maps), give: "0a 08 01 62 01 32 01 61 01 31", off: 6,
			},
			{
				name: "returns ErrNotCanonical at the second occurrence of a map key",
				msg:  new(canonical.Maps), give: "0a 08 01 61 01 31 01 61 01 32", off: 6,
			},
			{
				name: "returns ErrNotCanonical at a map key of -0.0",
				msg:  new(canonical.Keys), give: "0a 0a 00 00 00 00 00 00 00 80 01 61", off: 2,
			},
		}
		for _, tt := range rejections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := tt.msg.DecodeKanon(vector(t, tt.give), kanon.Options{})
				assert.ErrorIs(t, err, kanon.ErrNotCanonical, "DecodeKanon returns ErrNotCanonical")
				e := assert.ErrorAs[*kanon.DecodeError](t, err, "DecodeKanon returns a DecodeError")
				assert.Equal(t, e.Offset, tt.off, "the DecodeError names the offset of the rule that the input breaks")
			})
		}
		acceptances := []struct {
			name string
			msg  kanon.Message
			give string
			want kanon.Message
		}{
			{
				name: "decodes a pointer field to a zero value",
				msg:  new(canonical.Numbers), give: "48 00", want: &canonical.Numbers{I: new(false)},
			},
			{
				name: "decodes a pointer field to a struct of no bytes",
				msg:  new(canonical.Container), give: "12 00", want: &canonical.Container{Inner: &canonical.Inner{}},
			},
			{
				name: "decodes a selected union member at its zero value",
				msg:  new(canonical.Holder), give: "18 00", want: &canonical.Holder{Kind: canonical.HolderKindNum},
			},
			{
				name: "decodes an interface that stores a nil pointer",
				msg:  new(canonical.Holder), give: "0a 02 02 00",
				want: &canonical.Holder{Shape: (*canonical.Square)(nil)},
			},
			{
				name: "decodes a time field of the Unix epoch",
				msg:  new(canonical.Times), give: "0a 00", want: &canonical.Times{CreatedAt: time.Unix(0, 0).UTC()},
			},
			{
				name: "decodes the zero time as an element",
				msg:  new(canonical.Times), give: "1a 08 07 08 ff db 8f f9 ce 03",
				want: &canonical.Times{Stamps: []time.Time{{}}},
			},
			{
				name: "decodes a struct field whose fields are present",
				msg:  new(canonical.Nest), give: "0a 02 10 02", want: &canonical.Nest{Inner: canonical.Inner{Count: 1}},
			},
		}
		for _, tt := range acceptances {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, tt.msg.DecodeKanon(vector(t, tt.give), kanon.Options{}),
					"DecodeKanon decodes the canonical encoding")
				assert.Equal(t, tt.msg, tt.want, "DecodeKanon decodes the value of the encoding")
			})
		}
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

func TestSizer(t *testing.T) {
	t.Parallel()
	t.Run("MarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns an EncodeError with ErrSize for an encoding of another length than SizeKanon",
			func(t *testing.T) {
				t.Parallel()
				_, err := (&codec.Codecs{Clip: clipLong}).MarshalBinary()
				e := assert.ErrorAs[*kanon.EncodeError](t, err, "MarshalBinary returns an EncodeError")
				want := kanon.EncodeError{Type: codecsType, Field: clipField, Number: clipNumber, Err: kanon.ErrSize}
				assert.Equal(t, *e, want, "the EncodeError names the field and wraps ErrSize")
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
