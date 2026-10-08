// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package stream declares fixtures with fields that have the tag option
// stream, so that kanon writes a stream decoder for each of their struct
// types: the types of the test vectors of the stream decoder, canonical
// types whose unions have members on both sides of a streamed field, with
// and without fields that the decode tracks, and types that accept every
// encoding of a value, with tracked fields, a union, unknown fields, named
// byte slice and string types, and slices of structs with a kanon codec of
// this package and of another. The generated tests run the stream checks of
// kanontest over each of them.
//
// # Dependency position
//
// stream imports the fixture package external, and its generated code the
// kanon runtime.
package stream
