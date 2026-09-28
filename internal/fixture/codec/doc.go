// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package codec declares the types of the fixtures that encode themselves,
// one per family of methods that kanon tries: [Token] through AppendBinary,
// [Ticket] through GobEncode, [Grade] through AppendText, [Word] through
// MarshalText, and [Stamp] through AppendBinary with a pointer receiver.
// [Note] appends text of any length, on both sides of the stack array that
// the generated code appends into. [Parity] encodes itself through
// MarshalBinary to one bit, so that two map keys of it can have one
// projection. The encoding of each fails for one value, and the decoding
// for malformed data, so that the generated tests reach the error branches
// of the code that calls them. [Codecs] has a field of each, and
// [Appenders] a field of each with an append method.
//
// # Dependency position
//
// codec imports encoding/binary, errors, math, strings and unicode/utf8
// from the standard library, and its generated code the kanon runtime. The
// fixtures import it.
package codec
