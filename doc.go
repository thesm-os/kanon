// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package kanon declares the method set, the decode options, the errors and
// the version constants of the binary codecs that the kanon generator writes
// for Go struct types. A struct type T that a //go:generate go tool kanon
// -type=T directive names gets the methods of [Message] and [Cloner] on *T,
// and code written against those interfaces encodes, decodes and copies any
// generated type. A named type that is not a struct, which such a directive
// names, gets the method of [Validator], and encodes as its underlying type.
//
// # Encoding
//
// A struct encodes as a sequence of fields, each a tag and a value. The tag
// is a varint of the field number shifted left by three bits, with the wire
// format of the value in the low three bits. Integers are varints, nested
// values carry a length, map keys are sorted, and absent fields are left
// out, so equal values encode to identical bytes. A decoder skips a field
// number it does not know, so a struct can gain and lose fields.
//
// # Types that encode themselves
//
// A value of a type with binary, gob or text methods encodes as the bytes
// that the first of those families returns. For such a type whose ==
// compares every bit, a field at the zero value is absent, and the generated
// code leaves it out without a call of a method of the type. It calls the
// method IsZero() bool in place of == when the type declares it, which must
// report true for the zero value alone. A [Sizer] sizes its values with
// SizeKanon in place of an encode.
//
// # Decoding without a copy
//
// Every decoded string is a substring of the slab that [Options] name, a
// string of which the input is a substring. With the zero Options a decode
// copies its input once and uses the copy as the slab. A caller that owns
// the input buffer passes the buffer itself as the slab, and the decode
// then allocates nothing for its strings.
//
// # Errors
//
// A decode returns a [*DecodeError] that names the struct type, the field,
// the field number and the offset of the malformed input, and that wraps
// one of [io.ErrUnexpectedEOF], [ErrMalformed], [ErrRange], [ErrDepth],
// [ErrUnknownType], [ErrInvalidKey], [ErrAmbiguousKey] and
// [ErrRepeatedView], the error of a type that decodes itself, or the error
// of the ValidateKanon of a [Validator]. An encode returns a [*EncodeError]
// that wraps [ErrUnlistedType], [ErrInvalidKey], [ErrAmbiguousKey] or
// [ErrSize], the error of a type that encodes itself, or the error of
// ValidateKanon.
// errors.Is and errors.As classify both.
//
// # Versions
//
// Every generated file declares two [EnforceVersion] constants for the
// version of the generator that wrote it. A file whose version lies outside
// [MinVersion] and [MaxVersion] of the runtime it compiles against fails to
// compile.
//
// # Dependency position
//
// kanon imports encoding, errors, io, strconv and strings from the standard
// library. Generated code and the packages go.thesmos.sh/kanon/wire,
// go.thesmos.sh/kanon/frame, go.thesmos.sh/kanon/batch and
// go.thesmos.sh/kanon/kanontest import it.
package kanon
