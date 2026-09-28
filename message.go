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
//
// # Map keys
//
// A map key encodes as its projection: the encoding as a value of the key
// with every float component of -0.0 replaced by +0.0, so that a struct
// field of -0.0 is absent from it. The projection leaves out the
// identity of a pointer, the name of a zone, a monotonic clock reading and
// the fields that the encoding leaves out, so a decoded key has the
// projection of the encoded key and not its identity. Keys of one
// projection decode to one entry, that of the later key.
type Message interface {
	encoding.BinaryAppender
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// SizeKanon returns the length of the encoding in bytes, for a value
	// whose encode succeeds.
	SizeKanon() int

	// EncodeKanon writes the encoding into the last SizeKanon bytes of buf
	// and returns their count. It returns io.ErrShortBuffer without writing
	// when buf is shorter than SizeKanon bytes. It returns an [*EncodeError]
	// when a value fails to encode itself, an interface stores a type that
	// its list does not name, or a map has a key with a NaN component
	// ([ErrInvalidKey]) or two keys of one projection ([ErrAmbiguousKey]).
	// After an error, buf does not contain a valid encoding, and
	// AppendBinary returns its buffer unchanged.
	EncodeKanon(buf []byte) (int, error)

	// DecodeKanon sets the receiver to the value encoded in data and reuses
	// the memory of the receiver: the capacity of its slices, the entries of
	// its maps, the values its pointers point at and its nested structs. It
	// first clears the fields that the encoding leaves out, as Reset does.
	// Every decoded string is a substring of the slab of opts. It returns a
	// [*DecodeError] for malformed input. After an error the receiver
	// contains the fields decoded before it, and the failed field has an
	// unspecified value.
	DecodeKanon(data []byte, opts Options) error

	// MergeKanon decodes data into the receiver without resetting it first,
	// with the options and the errors of DecodeKanon. A slice appends the
	// elements in data, a map adds its entries, a nested struct merges, an
	// interface that stores the concrete type in data merges as that type
	// merges, a union member replaces the union, and any other field takes
	// the value in data. The fields that the encoding leaves out keep their
	// values. A map of the receiver with two keys of one projection fails
	// the merge with [ErrAmbiguousKey] before the map changes.
	MergeKanon(data []byte, opts Options) error

	// Reset clears every field of the receiver, including the fields that
	// the encoding leaves out: skipped fields, unexported fields without a
	// kanon tag, and fields of a function or a channel type or of a pointer
	// to one. An encoded scalar becomes its zero value, and an encoded
	// pointer or interface becomes nil. An encoded slice or map becomes
	// empty and keeps its storage for a decode
	// to reuse: every element in the capacity of a slice is reset, a pointer
	// element keeps its target with its value reset, and the entries of a
	// map are deleted. A skipped slice or map becomes nil. The receiver need
	// not be reflect.DeepEqual to a new zero value, but no decoded content
	// remains in its values or in the storage that it keeps.
	Reset()
}

// Cloner is the method set of a generated *T with its copy method, whose
// result type names T and so is not part of [Message].
type Cloner[T any] interface {
	Message

	// CloneKanon returns a copy of the receiver that does not share memory
	// with the receiver or with the slab the receiver decoded from: its
	// strings, slices, maps and the values of its pointers and interfaces are
	// copies. The copy has the fields of the encoding, the union
	// discriminators and the unknown fields of the receiver, and the zero
	// value in every other field, map keys included. A value of a type that
	// encodes itself is copied through its own encode and decode methods when
	// it refers to memory, and by assignment when it does not or when those
	// methods fail. A map with two keys of one projection, which does not
	// encode, can have one entry for them in the copy. A nil receiver returns
	// nil.
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
	// a [DecodeError] are offsets in Slab. The decode of a struct that
	// contains no string reads nothing from Slab, and counts the offsets of
	// its errors from Offset also when Slab is empty.
	Offset int
	// Depth is the number of levels that the decode enters below the top
	// struct at most. Each struct, slice, map, pointer and interface value
	// below the top struct is one level, so that the elements of a field of
	// type []*T, for a struct type T, are three levels below it. 0 means
	// DefaultDepth, and a negative Depth rejects every nested value.
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

// Limit returns the number of levels that a decode enters below the top
// struct at most: Depth, DefaultDepth when Depth is 0, and 0 when Depth is
// negative.
func (o Options) Limit() int {
	if o.Depth == 0 {
		return DefaultDepth
	}
	return max(o.Depth, 0)
}
