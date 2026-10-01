// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"strconv"
	"strings"
)

// errPrefix begins the message of every error of the package and of the
// generated code.
const errPrefix = "kanon: "

// Causes of a [DecodeError], which errors.Is matches through
// [DecodeError.Unwrap]. Input that ends inside a value wraps
// io.ErrUnexpectedEOF instead, so a stream reader that retries on
// truncation does not need a check of its own for kanon.
var (
	// ErrMalformed marks input that breaks the wire format: a varint longer
	// than 64 bits, an invalid tag, a known field with another wire format,
	// an array of another length, a presence byte other than 0 and 1, and
	// bytes after the value that a length announced.
	ErrMalformed = errors.New("kanon: malformed input")
	// ErrRange marks a value outside the range of its Go type: an integer
	// wider than its field, time nanoseconds above 999999999, and a time
	// zone offset outside 32 bits.
	ErrRange = errors.New("kanon: value outside the range of the type")
	// ErrDepth marks a value nested deeper than the Depth of [Options].
	ErrDepth = errors.New("kanon: nested deeper than the limit")
	// ErrUnknownType marks an interface type number that the field's list
	// of concrete types does not name.
	ErrUnknownType = errors.New("kanon: interface type number not listed")
	// ErrRepeatedView marks a second occurrence of a struct field in the
	// encoding that a view reads the field from. A decode merges the
	// occurrences, and one byte slice cannot hold the merge.
	ErrRepeatedView = errors.New("kanon: struct field occurs twice in a view")
	// ErrNotCanonical marks input that the decode of a type whose directive
	// has the -canonical flag rejects, and that the wire format lets a
	// decoder accept: input that is not the canonical encoding of the value
	// that it decodes to. Such input has a varint longer than its shortest
	// form, a field number that is not above the one before it or that the
	// schema does not list, a second member of a union, a bool above 1, a
	// field at a value that the encoding leaves out, a map key that is not
	// above the key before it or that has a float component of -0.0, a time
	// field of 0, or bytes of a type that encodes itself that its encode
	// method does not write for the decoded value.
	ErrNotCanonical = errors.New("kanon: input is not the canonical encoding")
)

// ErrUnlistedType, ErrSize and ErrExact are causes of an [EncodeError], which
// errors.Is matches through [EncodeError.Unwrap].
var (
	// ErrUnlistedType marks an interface that stores a value of a type that
	// the tag option types of its field does not list.
	ErrUnlistedType = errors.New("kanon: type not listed in the tag option types")
	// ErrSize marks a value of a [Sizer] whose encode method returns another
	// length than its SizeKanon, which sized the room of the value.
	ErrSize = errors.New("kanon: encoding length differs from SizeKanon")
	// ErrExact marks a value of an [Exact] type that breaks the guarantees of
	// Exact: an append method that appends another length than SizeKanon, or
	// that fails for the value of a field, which is never the zero value. The
	// encode panics with an [*EncodeError] that wraps it.
	ErrExact = errors.New("kanon: a type that declares ExactKanon breaks its guarantee")
)

// Causes of an [EncodeError] and of a [DecodeError] for a map key. The
// projection of a key is its encoding, with every float component of -0.0
// written as +0.0.
var (
	// ErrInvalidKey marks a map key with a NaN component, which the wire
	// format cannot order: the encode of a map with such a key fails, and so
	// does the decode of one.
	ErrInvalidKey = errors.New("kanon: map key has a NaN component")
	// ErrAmbiguousKey marks two keys of one map with the same projection:
	// the encode of such a map fails, and so does a merge into a receiver
	// whose map has two such keys.
	ErrAmbiguousKey = errors.New("kanon: two map keys encode alike")
)

// DecodeError is the error of a decode: the malformed input, where it
// starts, and why it is malformed. The generated code returns one for every
// failure, as a *DecodeError.
type DecodeError struct {
	// Type names the struct whose encoding contains the malformed input: a
	// struct type name, a package-qualified name for a struct of another
	// package, or the path of an anonymous struct type, such as Order.Meta.
	Type string
	// Field names the field of Type whose value is malformed. It is empty
	// for an error in the sequence of fields itself: a tag that does not
	// read, an invalid tag, and a struct nested deeper than the limit.
	Field string
	// Number is the field number of Field, and 0 when Field is empty.
	Number int
	// Offset is the offset of the malformed input in the slab of the
	// decode.
	Offset int
	// Detail states what is malformed, and is empty when Err states it.
	Detail string
	// Err is the cause: io.ErrUnexpectedEOF, [ErrMalformed], [ErrRange],
	// [ErrDepth], [ErrUnknownType], [ErrInvalidKey], [ErrAmbiguousKey],
	// [ErrRepeatedView], [ErrNotCanonical], the error of a type that decodes
	// itself, or the error of the ValidateKanon of a [Validator].
	Err error
}

// Error returns "kanon: Type.Field (field Number) at offset Offset: Detail",
// with "Type" alone for an error without a field, and the message of Err
// without its "kanon: " prefix when Detail is empty.
func (e *DecodeError) Error() string {
	var sb strings.Builder
	sb.WriteString(errPrefix)
	sb.WriteString(e.Type)
	if e.Field != "" {
		sb.WriteByte('.')
		sb.WriteString(e.Field)
		sb.WriteString(" (field ")
		sb.WriteString(strconv.Itoa(e.Number))
		sb.WriteByte(')')
	}
	sb.WriteString(" at offset ")
	sb.WriteString(strconv.Itoa(e.Offset))
	sb.WriteString(": ")
	sb.WriteString(detail(e.Detail, e.Err))
	return sb.String()
}

// Unwrap returns the cause, Err.
func (e *DecodeError) Unwrap() error {
	return e.Err
}

// EncodeError is the error of an encode: the field whose value failed to
// encode, and why. The generated code returns one as an *EncodeError.
type EncodeError struct {
	// Type names the struct that declares Field, as in [DecodeError].
	Type string
	// Field names the field whose value failed to encode.
	Field string
	// Number is the field number of Field.
	Number int
	// Err is the cause: [ErrUnlistedType], [ErrInvalidKey],
	// [ErrAmbiguousKey], [ErrSize], the error of a type that encodes itself,
	// or the error of the ValidateKanon of a [Validator].
	Err error
}

// Error returns "kanon: Type.Field (field Number): " followed by the
// message of Err without its "kanon: " prefix.
func (e *EncodeError) Error() string {
	return errPrefix + e.Type + "." + e.Field + " (field " + strconv.Itoa(e.Number) + "): " + detail("", e.Err)
}

// Unwrap returns the cause, Err.
func (e *EncodeError) Unwrap() error {
	return e.Err
}

// detail returns d, or the message of err without its "kanon: " prefix when
// d is empty.
func detail(d string, err error) string {
	if d != "" {
		return d
	}
	return strings.TrimPrefix(err.Error(), errPrefix)
}
