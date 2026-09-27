// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import "encoding"

// DefaultDepth is the nesting limit of a decode whose [Options] leave Depth
// at 0.
const DefaultDepth = 100

// Message is the method set that the kanon generator writes for a struct
// type T. *T implements it.
//
// # Allocation contract
//
// SizeKanon, EncodeKanon, and AppendBinary into a buffer with SizeKanon
// bytes of spare capacity do not allocate, except for these values:
//
//   - a value of a type that encodes itself and has no append method;
//   - a value of a type that encodes itself into more than 128 bytes;
//   - a map of more than 16 keys other than bools, whose keys sort in a
//     slice.
//
// DecodeKanon and MergeKanon with a slab in their [Options], into a receiver
// that decoded the same input before, do not allocate, except for these
// values:
//
//   - a value of a type that decodes itself;
//   - a time in a zone that is neither UTC, the local zone at that instant,
//     nor a whole number of hours from UTC-12 to UTC+14;
//   - a map of more than 16 entries whose values refer to memory, and a map
//     key that refers to memory;
//   - a value that an interface stores by value.
//
// Without a slab, a decode allocates one copy of its input when T contains
// a string.
//
// # Concurrency
//
// The methods do not synchronize. Encoding methods read the receiver, and
// decoding methods and Reset write it, so concurrent calls on one value
// need the caller's synchronization.
type Message interface {
	encoding.BinaryAppender
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// SizeKanon returns the length of the encoding in bytes.
	SizeKanon() int

	// EncodeKanon writes the encoding into the last SizeKanon bytes of buf
	// and returns their count. It returns io.ErrShortBuffer without writing
	// when buf is shorter than SizeKanon bytes. It returns an [*EncodeError]
	// when a value fails to encode itself or an interface stores a type that
	// its list does not name.
	EncodeKanon(buf []byte) (int, error)

	// DecodeKanon sets the receiver to the value encoded in data and reuses
	// the memory of the receiver: the capacity of its slices, the entries of
	// its maps, the values its pointers point at and its nested structs.
	// Every decoded string is a substring of the slab of opts. It returns a
	// [*DecodeError] for malformed input. After an error the receiver
	// contains the fields decoded before it, and the failed field has an
	// unspecified value.
	DecodeKanon(data []byte, opts Options) error

	// MergeKanon decodes data into the receiver without resetting it first,
	// with the options and the errors of DecodeKanon. A slice appends the
	// elements in data, a map adds its entries, a nested struct merges, and
	// any other field takes the value in data.
	MergeKanon(data []byte, opts Options) error

	// Reset sets the receiver to its zero value and keeps the memory that a
	// decode reuses: the capacity of its slices, with the values that the
	// pointers in that capacity point at, and the entries of its maps.
	Reset()
}

// Cloner is the method set of a generated *T with its copy method, whose
// result type names T and so is not part of [Message].
type Cloner[T any] interface {
	Message

	// CloneKanon returns a deep copy of the receiver. The copy shares no
	// memory with the receiver or with the slab the receiver decoded from:
	// its strings, slices, maps and the values of its pointers and
	// interfaces are copies. A value of a type that encodes itself is copied
	// through its own encode and decode methods when it refers to memory,
	// and by assignment when it does not or when those methods fail. A nil
	// receiver returns nil.
	CloneKanon() *T
}

// Options set the slab and the nesting limit of a decode, and with the zero
// Options a decode copies its input once, uses the copy as the slab and
// applies [DefaultDepth].
type Options struct {
	// Slab is a string of which the input is the substring that starts at
	// Offset. Every decoded string is a substring of Slab, so the decode
	// allocates nothing for its strings. A Slab that aliases the input's
	// memory, as unsafe.String over the input returns it, makes every
	// decoded string alias that memory, and the caller must not change the
	// memory while the decoded value is in use. An empty Slab makes the
	// decode copy the input.
	Slab string
	// Offset is the index in Slab of the first byte of the input. Offsets in
	// a [DecodeError] are offsets in Slab.
	Offset int
	// Depth is the number of values that the decode enters below the top
	// struct at most: nested structs, slices, arrays, maps, the values of
	// pointers and the values of interfaces. 0 means DefaultDepth, and a
	// negative Depth rejects every nested value.
	Depth int
}

// Source returns the slab and the offset of data in it for a decode of
// data: Slab and Offset, or a copy of data as a string and 0 when Slab is
// empty. The copy is the one allocation of a decode without a slab.
func (o Options) Source(data []byte) (string, int) {
	if o.Slab == "" {
		return string(data), 0
	}
	return o.Slab, o.Offset
}

// Limit returns the nesting limit of a decode: Depth, or DefaultDepth when
// Depth is 0.
func (o Options) Limit() int {
	if o.Depth == 0 {
		return DefaultDepth
	}
	return o.Depth
}
