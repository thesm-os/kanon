// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

// DefaultBuffer is the buffer limit of a stream decoder whose [StreamOptions]
// leave Buffer at 0 or less: 16 MiB, the default limit of a frame.
const DefaultBuffer = 16777216

// StreamOptions set the limits of a stream decoder, which the kanon generator
// writes for a struct type with a field that has the tag option stream. The
// zero StreamOptions apply DefaultDepth and DefaultBuffer.
//
// # Concurrency
//
// StreamOptions is a value without references, safe to share.
type StreamOptions struct {
	// Depth is the number of levels that the decode enters below the top
	// struct at most, as the Depth of [Options] sets it. 0 means
	// DefaultDepth, and a negative Depth rejects every nested value.
	Depth int
	// Buffer is the most bytes of the encoding that a stream decoder has in
	// memory for the fields that do not stream, together, and for one
	// element of a streamed slice. A value that would take the decoder past
	// it fails with ErrLimit. 0 or less means DefaultBuffer.
	Buffer int
}
