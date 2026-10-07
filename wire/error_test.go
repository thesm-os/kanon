// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"errors"
	"io"
	"math"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The cases locate their errors at a field of a struct of another package,
// whose type name contains a dot.
const (
	errLoc    = "shop.Order.Count"
	errType   = "shop.Order"
	errField  = "Count"
	errNumber = 7
	errOff    = 12
	// errTagOff and errValueOff are the offsets of a tag and of a value that
	// wire.ExactError chooses between.
	errTagOff   = 3
	errValueOff = 20
)

// errOwn is the error of a type that encodes or decodes itself.
var errOwn = errors.New("token: empty")

// exactEncoding is the encoding of a value of a kanon.Exact type, which
// wire.MustExact checks.
var exactEncoding = []byte{1, 2, 3}

// hexagon is a type that no list of concrete types names.
type hexagon struct{}

func TestError(t *testing.T) {
	t.Parallel()
	located := func(cause error, detail string) *kanon.DecodeError {
		return &kanon.DecodeError{
			Type: errType, Field: errField, Number: errNumber, Offset: errOff, Detail: detail, Err: cause,
		}
	}
	cases := []struct {
		name string
		err  error
		want *kanon.DecodeError
	}{
		{
			name: "ReadError/reports truncation for length 0",
			err:  wire.ReadError(0, errLoc, errNumber, errOff),
			want: located(io.ErrUnexpectedEOF, ""),
		},
		{
			name: "ReadError/reports truncation for a length that runs past the input",
			err:  wire.ReadError(3, errLoc, errNumber, errOff),
			want: located(io.ErrUnexpectedEOF, ""),
		},
		{
			name: "ReadError/reports an overflow for length -1",
			err:  wire.ReadError(-1, errLoc, errNumber, errOff),
			want: located(kanon.ErrMalformed, "varint overflows 64 bits"),
		},
		{
			name: "TagError/locates an invalid tag at the struct",
			err:  wire.TagError(errNumber<<3|6, errType, 0, errOff),
			want: &kanon.DecodeError{
				Type: errType, Offset: errOff, Detail: "field 7 has invalid wire format 6", Err: kanon.ErrMalformed,
			},
		},
		{
			name: "FormatError/names the wire format and the one the field wants",
			err:  wire.FormatError(errNumber<<3|wire.Fixed64, wire.Varint, errLoc, errOff),
			want: located(kanon.ErrMalformed, "wire format fixed64, want varint"),
		},
		{
			name: "FormatError/names the formats fixed32 and bytes",
			err:  wire.FormatError(errNumber<<3|wire.Fixed32, wire.Bytes, errLoc, errOff),
			want: located(kanon.ErrMalformed, "wire format fixed32, want bytes"),
		},
		{
			name: "RangeError/names a signed value and the type",
			err:  wire.RangeError(int64(-129), "int8", errLoc, errNumber, errOff),
			want: located(kanon.ErrRange, "value -129 outside int8"),
		},
		{
			name: "RangeError/names an unsigned value and the type",
			err:  wire.RangeError(uint64(math.MaxUint64), "uint32", errLoc, errNumber, errOff),
			want: located(kanon.ErrRange, "value 18446744073709551615 outside uint32"),
		},
		{
			name: "VarintError/reports truncation for length 0",
			err:  wire.VarintError(0, int64(0), "int", errLoc, errNumber, errOff),
			want: located(io.ErrUnexpectedEOF, ""),
		},
		{
			name: "VarintError/reports an overflow for length -1",
			err:  wire.VarintError(-1, uint64(0), "uint", errLoc, errNumber, errOff),
			want: located(kanon.ErrMalformed, "varint overflows 64 bits"),
		},
		{
			name: "VarintError/names a value outside the type for a positive length",
			err:  wire.VarintError(5, int64(math.MaxInt32+1), "int", errLoc, errNumber, errOff),
			want: located(kanon.ErrRange, "value 2147483648 outside int"),
		},
		{
			name: "LengthError/names the length and the one the value wants",
			err:  wire.LengthError(3, 32, errLoc, errNumber, errOff),
			want: located(kanon.ErrMalformed, "length 3, want 32"),
		},
		{
			name: "TrailingError/counts the bytes after the value",
			err:  wire.TrailingError(2, errLoc, errNumber, errOff),
			want: located(kanon.ErrMalformed, "2 bytes after the value"),
		},
		{
			name: "PresenceError/names the presence byte",
			err:  wire.PresenceError(2, errLoc, errNumber, errOff),
			want: located(kanon.ErrMalformed, "presence byte 2, want 0 or 1"),
		},
		{
			name: "TypeError/names the type number",
			err:  wire.TypeError(9, errLoc, errNumber, errOff),
			want: located(kanon.ErrUnknownType, "type number 9"),
		},
		{
			name: "DepthError/locates a struct nested too deep at the struct",
			err:  wire.DepthError(errType, 0, errOff),
			want: &kanon.DecodeError{Type: errType, Offset: errOff, Err: kanon.ErrDepth},
		},
		{
			name: "DepthError/locates a value nested too deep at its field",
			err:  wire.DepthError(errLoc, errNumber, errOff),
			want: located(kanon.ErrDepth, ""),
		},
		{
			name: "UnmarshalError/wraps the error of the value",
			err:  wire.UnmarshalError(errOwn, errLoc, errNumber, errOff),
			want: located(errOwn, ""),
		},
		{
			name: "KeyError/locates a map key with a NaN component",
			err:  wire.KeyError(errLoc, errNumber, errOff),
			want: located(kanon.ErrInvalidKey, ""),
		},
		{
			name: "RepeatedError/locates the second occurrence of a struct field",
			err:  wire.RepeatedError(errLoc, errNumber, errOff),
			want: located(kanon.ErrRepeatedView, ""),
		},
		{
			name: "LongFormError/counts the bytes of the varint",
			err:  wire.LongFormError(3, errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "varint of 3 bytes has a shorter form"),
		},
		{
			name: "OrderError/names the field and the field before it at the struct",
			err:  wire.OrderError(2, 5, errType, 0, errOff),
			want: &kanon.DecodeError{
				Type: errType, Offset: errOff, Detail: "field 2 after field 5", Err: kanon.ErrNotCanonical,
			},
		},
		{
			name: "UnknownFieldError/names the field number at the struct",
			err:  wire.UnknownFieldError(9, errType, 0, errOff),
			want: &kanon.DecodeError{
				Type: errType, Offset: errOff, Detail: "field 9 not in the schema", Err: kanon.ErrNotCanonical,
			},
		},
		{
			name: "MemberError/names the discriminator of the union",
			err:  wire.MemberError("Kind", errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "second member of the union of Kind"),
		},
		{
			name: "BoolError/names the value of the bool",
			err:  wire.BoolError(2, errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "bool 2, want 0 or 1"),
		},
		{
			name: "AbsentError/locates a field at a value that the encoding leaves out",
			err:  wire.AbsentError(errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "field at a value that the encoding leaves out"),
		},
		{
			name: "KeyOrderError/locates a map key that does not ascend",
			err:  wire.KeyOrderError(errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "key not above the key before it"),
		},
		{
			name: "NegativeZeroError/locates a map key with a component of -0.0",
			err:  wire.NegativeZeroError(errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "key with a float component of -0.0"),
		},
		{
			name: "EncodingError/locates bytes that the encode method does not write",
			err:  wire.EncodingError(errLoc, errNumber, errOff),
			want: located(kanon.ErrNotCanonical, "bytes differ from the encoding of the decoded value"),
		},
		{
			name: "ExactError/wraps the error of the decode method at the offset of the value",
			err:  wire.ExactError(errOwn, errLoc, errNumber, errTagOff, errOff),
			want: located(errOwn, ""),
		},
		{
			name: "ExactError/locates the zero value at the offset of the tag",
			err:  wire.ExactError(nil, errLoc, errNumber, errOff, errValueOff),
			want: located(kanon.ErrNotCanonical, "field at a value that the encoding leaves out"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := assert.ErrorAs[*kanon.DecodeError](t, c.err, "the function returns a *kanon.DecodeError")
			assert.Equal(t, got, c.want, "the error locates the input and states the cause")
		})
	}
	t.Run("MarshalError", func(t *testing.T) {
		t.Parallel()
		t.Run("wraps the error of the value", func(t *testing.T) {
			t.Parallel()
			got := assert.ErrorAs[*kanon.EncodeError](t, wire.MarshalError(errOwn, errLoc, errNumber),
				"MarshalError returns a *kanon.EncodeError")
			assert.Equal(t, got, &kanon.EncodeError{Type: errType, Field: errField, Number: errNumber, Err: errOwn},
				"MarshalError locates the field and wraps the cause")
		})
	})
	t.Run("UnlistedError", func(t *testing.T) {
		t.Parallel()
		t.Run("wraps ErrUnlistedType and names the type", func(t *testing.T) {
			t.Parallel()
			err := wire.UnlistedError(hexagon{}, errLoc, errNumber)
			assert.ErrorIs(t, err, kanon.ErrUnlistedType, "UnlistedError wraps ErrUnlistedType")
			assert.Equal(t, err.Error(),
				"kanon: shop.Order.Count (field 7): type not listed in the tag option types: wire_test.hexagon",
				"UnlistedError names the field and the type of the value")
		})
	})
	t.Run("InvalidKeyError", func(t *testing.T) {
		t.Parallel()
		t.Run("wraps ErrInvalidKey at the field of the map", func(t *testing.T) {
			t.Parallel()
			got := assert.ErrorAs[*kanon.EncodeError](t, wire.InvalidKeyError(errLoc, errNumber),
				"InvalidKeyError returns a *kanon.EncodeError")
			assert.Equal(t, got,
				&kanon.EncodeError{Type: errType, Field: errField, Number: errNumber, Err: kanon.ErrInvalidKey},
				"InvalidKeyError locates the field and wraps the cause")
		})
	})
	t.Run("SizeError", func(t *testing.T) {
		t.Parallel()
		t.Run("wraps ErrSize at the field of the value", func(t *testing.T) {
			t.Parallel()
			got := assert.ErrorAs[*kanon.EncodeError](t, wire.SizeError(errLoc, errNumber),
				"SizeError returns a *kanon.EncodeError")
			assert.Equal(t, got,
				&kanon.EncodeError{Type: errType, Field: errField, Number: errNumber, Err: kanon.ErrSize},
				"SizeError locates the field and wraps the cause")
		})
	})
	t.Run("MustExact", func(t *testing.T) {
		t.Parallel()
		t.Run("returns for an encoding of SizeKanon bytes without an error", func(t *testing.T) {
			t.Parallel()
			assert.NotPanics(t, func() { wire.MustExact(exactEncoding, nil, len(exactEncoding), errLoc, errNumber) },
				"MustExact accepts the result that kanon.Exact guarantees")
		})
		t.Run("panics with ErrExact and the error of the append method", func(t *testing.T) {
			t.Parallel()
			err := exactPanic(t, func() { wire.MustExact(nil, errOwn, len(exactEncoding), errLoc, errNumber) })
			assert.That(t, err).
				ErrorIs(kanon.ErrExact, "the error wraps ErrExact").
				ErrorIs(errOwn, "the error wraps the error of the append method")
			assert.Equal(t, err.Error(),
				"kanon: shop.Order.Count (field 7): a type that declares ExactKanon breaks its guarantee: token: empty",
				"the error names the field and states the error of the append method")
		})
		t.Run("panics with ErrExact for an encoding of another length than SizeKanon", func(t *testing.T) {
			t.Parallel()
			err := exactPanic(t, func() { wire.MustExact(exactEncoding, nil, len(exactEncoding)+1, errLoc, errNumber) })
			assert.ErrorIs(t, err, kanon.ErrExact, "the error wraps ErrExact")
			assert.Equal(t, err.Error(),
				"kanon: shop.Order.Count (field 7): a type that declares ExactKanon breaks its guarantee: "+
					"appended 3 bytes, want 4",
				"the error names the field and states both lengths")
		})
	})
}

// TestErrorAllocs checks that the check of an encoding allocates nothing. It
// runs serially: testing.AllocsPerRun panics while a parallel test runs.
func TestErrorAllocs(t *testing.T) {
	t.Run("MustExact/allocates nothing for an encoding of SizeKanon bytes", func(t *testing.T) {
		assert.MaxAllocs(t, func() { wire.MustExact(exactEncoding, nil, len(exactEncoding), errLoc, errNumber) }, 0,
			"MustExact allocates nothing")
	})
}

func BenchmarkError(b *testing.B) {
	b.Run("MustExact", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			wire.MustExact(exactEncoding, nil, len(exactEncoding), errLoc, errNumber)
		}
	})
}

// exactPanic runs fn, which calls wire.MustExact, and returns the
// *kanon.EncodeError that it panics with. It fails t when fn does not panic
// with one.
func exactPanic(t *testing.T, fn func()) *kanon.EncodeError {
	t.Helper()
	reason := assert.Panics(t, fn, "MustExact panics")
	err, _ := reason.(error)
	return assert.ErrorAs[*kanon.EncodeError](t, err, "MustExact panics with a *kanon.EncodeError")
}
