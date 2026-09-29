// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package inspect prints kanon encodings without the Go types that wrote
// them, for the inspect subcommand of the kanon command. [Parse] reads the
// fields of a struct encoding from their tags, as a decode skips unknown
// fields. Each field has its number, its wire format and the bytes of its
// value, and a bytes value that parses as a struct encoding has its fields
// too. [WriteText] writes fields in the text notation of protoscope, and
// [WriteJSON] writes them as JSON. [Command] runs the subcommand. It also
// reads a stream of frames with a frame.Reader and a batch with
// batch.ParseAlias.
//
// # Readings
//
// An encoding does not contain its schema. The output gives each value every
// reading that its wire format allows:
//
//   - a varint as unsigned and as zigzag;
//   - a fixed64 and a fixed32 as unsigned, as signed and as a float;
//   - a bytes value as a struct encoding, as text, as hexadecimal digits and
//     as a run of varints.
//
// A bytes value that happens to parse as a struct encoding prints as one.
// A short string can. The text shows one reading of a bytes value and gives
// the others in a comment. JSON gives every reading.
//
// # Failure semantics
//
// Parse returns the fields before the first malformed field. For that field
// it returns the *kanon.DecodeError of package wire at the offset of its
// tag. Command prints a frame that fails its version, its flags or its
// checksum with its error, and continues after it. It does the same for a
// payload or a record that does not parse. A malformed struct encoding and a
// truncated frame end the output, and Command writes their error to its
// standard error. Command returns 1 when any part of the input fails.
//
// # Dependency position
//
// inspect imports go.thesmos.sh/kanon, go.thesmos.sh/kanon/wire,
// go.thesmos.sh/kanon/frame and go.thesmos.sh/kanon/batch, and bytes, cmp,
// encoding/hex, encoding/json, errors, flag, fmt, io, math, os, strconv,
// strings, unicode and unicode/utf8 from the standard library. The kanon
// command imports it.
package inspect
