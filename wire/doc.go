// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package wire provides the primitives that the code of the kanon generator
// calls: varints, fixed-width values, lengths, times, the skipping of
// unknown fields, the order of map keys, the reader of a stream decoder, and
// the errors of an encode and a decode. It is exported because generated
// code in other modules imports it. Applications use package
// go.thesmos.sh/kanon instead.
//
// # Writing backward
//
// An encode writes the last byte of an encoding first, so that the length
// of a nested value is known when its prefix is written. Every Put function
// takes a buffer and an offset i, writes its value into the bytes that end
// at i, and returns the offset of the first byte it wrote. The caller sizes
// the buffer first, and a Put function does not check the room.
//
// # Reading
//
// A read of a value at the start of data returns the value and its length
// n in bytes. n is 0 when data ends inside the value, and -1 when a varint
// exceeds 64 bits. The generated code turns a non-positive n into the error
// of [ReadError].
//
// # Inlining
//
// [Uvarint], [PutUvarint], [PutTag], [PutBool], [Uint32], [Uint64],
// [PutUint32], [PutUint64], [PutRaw], [SizeUvarint], [SizeBytes], [Zigzag],
// [Unzigzag] and [Nested] fit the compiler's inlining budget, so a call in
// a generated method compiles to the body of the function, and a one-byte
// varint costs one comparison. The other functions run once per value at
// most, or only on malformed input.
//
// # Errors
//
// The error functions return a *kanon.DecodeError or a *kanon.EncodeError.
// [MustExact] panics with a *kanon.EncodeError instead, since the generated
// code checks the guarantee of a kanon.Exact type without an error path. It
// panics only for a type that breaks the guarantee.
// They take the location of the value as loc, "Type.Field" for a field and
// "Type" alone for the struct itself, with num, the field number, 0 for the
// struct. Type names contain dots, and field names do not, so the last dot
// of loc separates the two.
//
// # Dependency position
//
// wire imports go.thesmos.sh/kanon for the errors, and cmp, encoding/binary,
// errors, fmt, io, math, math/bits, slices, strconv, strings, sync/atomic,
// time and unsafe from the standard library. Generated code imports it.
package wire
