// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"go.thesmos.sh/kanon"
)

// ReadError returns the error for a value at offset off whose read returned
// the length n: truncation, which wraps io.ErrUnexpectedEOF, for n of 0 and
// for a positive n, the length of a prefix whose value runs past the input;
// and a varint longer than 64 bits, which wraps kanon.ErrMalformed, for a
// negative n.
func ReadError(n int, loc string, num, off int) error {
	if n < 0 {
		return decodeError(kanon.ErrMalformed, loc, num, off, "varint overflows 64 bits")
	}
	return decodeError(io.ErrUnexpectedEOF, loc, num, off, "")
}

// TagError returns the error for an invalid tag at offset off: a tag with
// field number 0, or with wire format 3, 4, 6 or 7. It wraps
// kanon.ErrMalformed.
func TagError(tag uint64, loc string, num, off int) error {
	detail := "field number 0"
	if tag>>3 != 0 {
		detail = "field " + strconv.FormatUint(tag>>3, 10) + " has " + wireName(tag&7)
	}
	return decodeError(kanon.ErrMalformed, loc, num, off, detail)
}

// FormatError returns the error for the tag at offset off of a known
// field, whose wire format is not want. It wraps kanon.ErrMalformed.
func FormatError(tag uint64, want int, loc string, off int) error {
	detail := "wire format " + wireName(tag&7) + ", want " + wireName(uint64(want))
	return decodeError(kanon.ErrMalformed, loc, int(tag>>3), off, detail)
}

// RangeError returns the error for the value v at offset off, which does
// not fit the integer type typ. It wraps kanon.ErrRange.
func RangeError[V int64 | uint64](v V, typ, loc string, num, off int) error {
	return decodeError(kanon.ErrRange, loc, num, off, fmt.Sprintf("value %d outside %s", v, typ))
}

// VarintError returns the error for the varint at offset off of a value of
// the integer type typ, whose read returned the length n and the value v:
// the error of [ReadError] for an n of 0 or less, and the error of
// [RangeError] otherwise. The generated code of an int, a uint and a
// uintptr, whose range only a platform with a 32-bit int checks, rejects
// both cases in one condition and calls it.
func VarintError[V int64 | uint64](n int, v V, typ, loc string, num, off int) error {
	if n <= 0 {
		return ReadError(n, loc, num, off)
	}
	return RangeError(v, typ, loc, num, off)
}

// LengthError returns the error for the value at offset off whose length
// is got and not want: a byte array, or a complex128. It wraps
// kanon.ErrMalformed.
func LengthError(got uint64, want int, loc string, num, off int) error {
	detail := "length " + strconv.FormatUint(got, 10) + ", want " + strconv.Itoa(want)
	return decodeError(kanon.ErrMalformed, loc, num, off, detail)
}

// TrailingError returns the error for n bytes at offset off after the value
// that a length announced: after the value of a pointer field or an
// interface field, and after the last element of an array. It wraps
// kanon.ErrMalformed.
func TrailingError(n int, loc string, num, off int) error {
	return decodeError(kanon.ErrMalformed, loc, num, off, strconv.Itoa(n)+" bytes after the value")
}

// PresenceError returns the error for the presence byte b at offset off of
// a pointer, which is neither 0 nor 1. It wraps kanon.ErrMalformed.
func PresenceError(b byte, loc string, num, off int) error {
	return decodeError(kanon.ErrMalformed, loc, num, off, "presence byte "+strconv.Itoa(int(b))+", want 0 or 1")
}

// TypeError returns the error for the interface type number t at offset
// off, which the list of the field's concrete types does not name. It
// wraps kanon.ErrUnknownType.
func TypeError(t uint64, loc string, num, off int) error {
	return decodeError(kanon.ErrUnknownType, loc, num, off, "type number "+strconv.FormatUint(t, 10))
}

// DepthError returns the error for the value at offset off that is nested
// deeper than the limit of the decode: a struct, which loc names alone with
// num 0, or a value of a field. It wraps kanon.ErrDepth.
func DepthError(loc string, num, off int) error {
	return decodeError(kanon.ErrDepth, loc, num, off, "")
}

// UnmarshalError returns the error for the value at offset off of a type
// that decodes itself, whose decode method failed with err, or of a
// kanon.Validator, whose ValidateKanon rejected the decoded value with err.
// It wraps err.
func UnmarshalError(err error, loc string, num, off int) error {
	return decodeError(err, loc, num, off, "")
}

// MarshalError returns the error for the value of a type that encodes
// itself, whose encode method failed with err, or of a kanon.Validator,
// whose ValidateKanon rejected the value with err. It wraps err.
func MarshalError(err error, loc string, num int) error {
	e := &kanon.EncodeError{Number: num, Err: err}
	e.Type, e.Field = split(loc)
	return e
}

// UnlistedError returns the error for the interface value v, whose type
// the tag option types of the field does not list. It wraps
// kanon.ErrUnlistedType, and its message names the type of v.
func UnlistedError(v any, loc string, num int) error {
	return MarshalError(fmt.Errorf("%w: %T", kanon.ErrUnlistedType, v), loc, num)
}

// InvalidKeyError returns the error for the encode of a map with a key
// that has a NaN component, which loc and num locate at the field of the
// map. It wraps kanon.ErrInvalidKey.
func InvalidKeyError(loc string, num int) error {
	return MarshalError(kanon.ErrInvalidKey, loc, num)
}

// KeyError returns the error for the decoded map key at offset off, which
// has a NaN component. It wraps kanon.ErrInvalidKey.
func KeyError(loc string, num, off int) error {
	return decodeError(kanon.ErrInvalidKey, loc, num, off, "")
}

// RepeatedError returns the error for the second occurrence of a struct
// field in the encoding that a view reads, at off, the offset of its tag: a
// decode merges the occurrences, and one byte slice cannot hold the merge.
// It wraps kanon.ErrRepeatedView.
func RepeatedError(loc string, num, off int) error {
	return decodeError(kanon.ErrRepeatedView, loc, num, off, "")
}

// LongFormError returns the error of a canonical decode for the varint of n
// bytes at offset off, which has a shorter form: its last byte is 0. It
// wraps kanon.ErrNotCanonical.
func LongFormError(n int, loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "varint of "+strconv.Itoa(n)+" bytes has a shorter form")
}

// OrderError returns the error of a canonical decode for the tag at offset
// off of field number field, which is not above prev, the number of the
// field before it: the field repeats or is out of order. loc and num name
// the struct, or the field of a time. It wraps kanon.ErrNotCanonical.
func OrderError(field, prev uint64, loc string, num, off int) error {
	detail := "field " + strconv.FormatUint(field, 10) + " after field " + strconv.FormatUint(prev, 10)
	return decodeError(kanon.ErrNotCanonical, loc, num, off, detail)
}

// UnknownFieldError returns the error of a canonical decode for the tag at
// offset off of field number field, which the schema does not list. loc and
// num name the struct, or the field of a time. It wraps
// kanon.ErrNotCanonical.
func UnknownFieldError(field uint64, loc string, num, off int) error {
	detail := "field " + strconv.FormatUint(field, 10) + " not in the schema"
	return decodeError(kanon.ErrNotCanonical, loc, num, off, detail)
}

// MemberError returns the error of a canonical decode for the tag at offset
// off of a member of the union of the discriminator disc, which follows
// another member of that union. It wraps kanon.ErrNotCanonical.
func MemberError(disc, loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "second member of the union of "+disc)
}

// BoolError returns the error of a canonical decode for the bool v at offset
// off, which is above 1. It wraps kanon.ErrNotCanonical.
func BoolError(v uint64, loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "bool "+strconv.FormatUint(v, 10)+", want 0 or 1")
}

// AbsentError returns the error of a canonical decode for the field whose
// tag is at offset off and whose value is one that the encoding leaves out,
// such as a zero integer or an empty string. It wraps kanon.ErrNotCanonical.
func AbsentError(loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "field at a value that the encoding leaves out")
}

// KeyOrderError returns the error of a canonical decode for the map key at
// offset off, which is not above the key before it in the order of map keys:
// the keys repeat a projection or are out of order. It wraps
// kanon.ErrNotCanonical.
func KeyOrderError(loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "key not above the key before it")
}

// NegativeZeroError returns the error of a canonical decode for the map key
// at offset off, which has a float component of -0.0, which the projection
// of a key writes as +0.0. It wraps kanon.ErrNotCanonical.
func NegativeZeroError(loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "key with a float component of -0.0")
}

// EncodingError returns the error of a canonical decode for the value at
// offset off of a type that encodes itself, whose bytes differ from the
// bytes that the encode method of its type writes for the decoded value. It
// wraps kanon.ErrNotCanonical.
func EncodingError(loc string, num, off int) error {
	return decodeError(kanon.ErrNotCanonical, loc, num, off, "bytes differ from the encoding of the decoded value")
}

// ExactError returns the error of the canonical decode of a field of a
// kanon.Exact type, whose decode method returned err or set the zero value:
// the error of [UnmarshalError] at the offset valueOff of the value for an
// err that is not nil, and the error of [AbsentError] at the offset tagOff
// of the tag of the field otherwise.
func ExactError(err error, loc string, num, tagOff, valueOff int) error {
	if err != nil {
		return UnmarshalError(err, loc, num, valueOff)
	}
	return AbsentError(loc, num, tagOff)
}

// SizeError returns the error for the encode of a value of a kanon.Sizer,
// whose encode method returned another length than its SizeKanon, which
// loc and num locate at the field of the value. It wraps kanon.ErrSize.
func SizeError(loc string, num int) error {
	return MarshalError(kanon.ErrSize, loc, num)
}

// MustExact checks enc, the encoding that the append method of a
// kanon.Exact type returned with err for a value whose SizeKanon is n.
// kanon.Exact guarantees that enc has n bytes when err is nil, and that err
// is nil for a value other than the zero value. The put function of a field,
// which never writes the zero value, passes err, and the put function of any
// other position returns err itself and passes nil. For a type that breaks
// the guarantee, MustExact panics with a *kanon.EncodeError that loc and num
// locate at the field of the value, and that wraps kanon.ErrExact and err.
func MustExact(enc []byte, err error, n int, loc string, num int) {
	if err != nil || len(enc) != n {
		panic(exactFailure(len(enc), err, n, loc, num))
	}
}

// exactFailure returns the error of [MustExact] for an append method that
// appended got bytes and returned err, for a value whose SizeKanon is n: the
// error of [MarshalError] for a cause that wraps kanon.ErrExact and err, or
// kanon.ErrExact alone and states got and n when err is nil.
func exactFailure(got int, err error, n int, loc string, num int) error {
	if err != nil {
		return MarshalError(fmt.Errorf("%w: %w", kanon.ErrExact, err), loc, num)
	}
	return MarshalError(fmt.Errorf("%w: appended %d bytes, want %d", kanon.ErrExact, got, n), loc, num)
}

// decodeError returns the *kanon.DecodeError of loc and num at offset off,
// which wraps cause and states detail.
func decodeError(cause error, loc string, num, off int, detail string) *kanon.DecodeError {
	e := &kanon.DecodeError{Type: loc, Number: num, Offset: off, Detail: detail, Err: cause}
	if num != 0 {
		e.Type, e.Field = split(loc)
	}
	return e
}

// split returns the struct type and the field that loc, "Type.Field",
// names. A field name has no dot, so the last dot of loc separates the two.
func split(loc string) (string, string) {
	k := strings.LastIndexByte(loc, '.')
	return loc[:k], loc[k+1:]
}
