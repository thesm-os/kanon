// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame

import "errors"

// Errors of a [Reader] and a [Writer]. A stream that ends before the first
// byte of a length returns io.EOF, and one that ends inside a length or
// before the declared length returns io.ErrUnexpectedEOF. Every other error
// is one of these, the error of the stream, or the error of the message's
// encode or decode, so that a caller compares with errors.Is.
var (
	// ErrVersion marks a frame whose version byte is not 1.
	ErrVersion = errors.New("frame: unsupported version")
	// ErrFlags marks a frame whose flags byte sets a bit above bit 0.
	ErrFlags = errors.New("frame: unknown flag")
	// ErrTooLarge marks a length above the limit of a Reader, and a frame
	// longer than math.MaxInt bytes for a Writer.
	ErrTooLarge = errors.New("frame: longer than the limit")
	// ErrChecksum marks a frame whose checksum differs from the CRC-32C of
	// its version through its payload.
	ErrChecksum = errors.New("frame: checksum mismatch")
	// ErrUnknown marks a type ID that a Registry has no constructor for. A
	// caller returns it when [Registry.New] reports false.
	ErrUnknown = errors.New("frame: type not registered")
	// ErrMalformed marks a length uvarint that runs past 10 bytes or past
	// 64 bits, a frame too short for its version and flags, a type uvarint
	// that runs past the frame or past 64 bits, and a frame too short for
	// the checksum that its flags select.
	ErrMalformed = errors.New("frame: malformed length or frame")
	// ErrNoFrame marks a call of [Reader.Decode] before the first
	// successful call of [Reader.Next] or after a failed one.
	ErrNoFrame = errors.New("frame: no current frame")
	// ErrNilMessage marks a nil message, a nil pointer in an interface
	// included, that a Writer cannot encode and a Reader cannot fill.
	ErrNilMessage = errors.New("frame: nil message")
)
