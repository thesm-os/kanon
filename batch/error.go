// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch

import "errors"

// Errors of a [Writer], [Parse], [ParseAlias] and [Batch.Decode]. Every
// error is one of these or the error of the encode or the decode of a
// message, so that a caller compares with errors.Is.
var (
	// ErrVersion marks a batch whose version byte is not 1.
	ErrVersion = errors.New("batch: unsupported version")
	// ErrFlags marks a batch whose flags byte sets a bit above bit 0.
	ErrFlags = errors.New("batch: unknown flag")
	// ErrLayout marks a batch too short for its header and count, an index
	// longer than the batch, a first offset other than 0, a decreasing
	// offset, an offset past the encodings, and bytes of encodings in a
	// batch without messages.
	ErrLayout = errors.New("batch: index does not match the data")
	// ErrTooBig marks an Append after which the batch would not fit its
	// offsets or an int: encodings longer than 2^32-1 bytes with 4-byte
	// offsets, more than 2^32-1 messages, or a batch longer than
	// math.MaxInt bytes.
	ErrTooBig = errors.New("batch: the batch does not fit its offsets or an int")
	// ErrFinalized marks an Append after Bytes and before Reset.
	ErrFinalized = errors.New("batch: append after Bytes")
	// ErrNilMessage marks a nil message, a nil pointer in an interface
	// included, that a Writer cannot encode and a Batch cannot fill.
	ErrNilMessage = errors.New("batch: nil message")
)
