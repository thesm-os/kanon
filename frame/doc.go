// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package frame writes kanon messages to a byte stream and reads them back,
// one frame per message. A frame is a uvarint length, a version byte, a
// flags byte, a uvarint type ID, the kanon encoding of the message and, when
// bit 0 of the flags is set, a CRC-32C of the version through the payload. A
// [Writer] writes frames with one reused buffer, a [Reader] reads them with
// one reused buffer and decodes a frame into a message without a copy, and a
// [Registry] maps the type IDs that the application assigns to
// constructors.
//
// # Reading
//
// [Reader.Next] reads one frame. For a length within the limit, it reads the
// whole frame before it checks the version, the flags, the type and the
// checksum. After [ErrVersion], [ErrFlags] and [ErrChecksum], the next call
// reads the next frame, so a caller can skip the frame. After [ErrMalformed]
// and [ErrTooLarge], the position in the stream can be inside a frame, and
// the caller closes the stream. A stream that ends between frames returns
// io.EOF, and one that ends inside a frame returns io.ErrUnexpectedEOF.
//
// NewReader wraps a stream that does not implement io.ByteReader in a
// bufio.Reader, as encoding/gob does, so that reading a length costs no call
// of the stream per byte.
//
// # Aliasing
//
// The payload that Next returns aliases the reader's buffer, and so do the
// strings of a message that [Reader.Decode] decodes, since Decode passes the
// payload as the slab of the decode. The next call of Next can overwrite
// them. A caller that keeps a payload copies it, and one that keeps a
// message after the next frame copies it with CloneKanon.
//
// # Allocation contract
//
// [Writer.Write] does not allocate when an earlier frame of the writer was
// at least as long. [Reader.Next] and [Reader.Decode] do not allocate when
// an earlier frame of the reader was at least as long, for a message whose
// decode with a slab does not allocate.
//
// # Concurrency
//
// A Writer and a Reader are not safe for concurrent use. A Registry is safe
// for concurrent calls of [Registry.New] once every call of
// [Registry.Register] has returned.
//
// # Dependency position
//
// frame imports go.thesmos.sh/kanon, and bufio, encoding/binary, errors,
// fmt, hash/crc32, io, math, reflect, slices and unsafe from the standard
// library.
package frame
