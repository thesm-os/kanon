// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package codec declares the types of the fixtures that encode themselves,
// one per family of methods that kanon tries: [Token] through AppendBinary,
// [Ticket] through GobEncode, [Grade] through AppendText, [Word] through
// MarshalText, and [Stamp] through AppendBinary with a pointer receiver.
// [Note] appends text of any length, on both sides of the stack array that
// the generated code appends into. [Parity] encodes itself through
// MarshalBinary to one bit, so that two map keys of it can have one
// projection. [Seal] has no encoding for its zero value, which a field of
// it leaves out, nor for [SealVoid], which a field encodes. == reports -0.0
// equal to +0.0 in a [Reading] and does not apply to a [Blob]. A field of
// either is present when its encoding has bytes. [Hash], [Clip] and
// [Reading] are kanon.Sizer types, which a codec sizes with SizeKanon: a
// Hash tests its presence with IsZero, and a Reading has no append method.
// A Clip counts 4 bytes at most, a Hash its Size above the 8 bytes that it
// appends and a Reading 16 for -Inf, so that their encodes fail for such a
// value with kanon.ErrSize, and a Clip with a NUL byte and a Reading of +Inf
// return a SizeKanon of -1. The encoding of each type fails for a value in every way
// that the generated code handles, and the decoding for malformed data, so
// that the generated tests run the error branches of the code that calls
// them. [Codecs] has a field of each, and [Appenders] a field of each with
// an append method.
//
// # Dependency position
//
// codec imports encoding/binary, errors, math, slices, strings and
// unicode/utf8 from the standard library, and its generated code the kanon
// runtime. The fixtures import it.
package codec
