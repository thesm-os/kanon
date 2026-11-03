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
//   - a value of a type that encodes itself into more than 128 bytes and
//     is not a [Sizer];
//   - a map of more than 16 keys other than bools, whose keys sort in a
//     slice;
//   - a value of a [Validator] whose ValidateKanon allocates for it.
//
// DecodeKanon and MergeKanon into a receiver that decoded the same input
// before do not allocate, with a slab in their [Options] or without one,
// except for these values:
//
//   - a value of a type that decodes itself;
//   - a time in a zone that is neither UTC, the local zone at that instant,
//     nor a whole number of quarter hours from UTC-12 to UTC+14, an offset
//     that every zone in use has;
//   - a map of more than 16 entries whose values refer to memory, and a map
//     key that refers to memory;
//   - a value that an interface stores by value;
//   - a value of a [Validator] whose ValidateKanon allocates for it;
//   - without a slab, a map key that contains a string that is not empty,
//     and a map value that contains such a string and does not refer to
//     memory.
//
// Without a slab, a decode keeps each string of the receiver that the input
// repeats at its place, and copies its input once, for the first string that
// changes, as [Options] states. The strings of the exceptions without a slab
// take a substring of that copy.
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
	// when a value fails to encode itself or encodes to another length than
	// the SizeKanon of a [Sizer] ([ErrSize]), the ValidateKanon of a
	// [Validator] rejects a value, an interface stores a type that its list
	// does not name, or a map has a key with a NaN component ([ErrInvalidKey])
	// or two keys of one projection ([ErrAmbiguousKey]). After an error, buf does
	// not contain a valid encoding, and AppendBinary returns its buffer
	// unchanged.
	EncodeKanon(buf []byte) (int, error)

	// DecodeKanon sets the receiver to the value encoded in data and reuses
	// the memory of the receiver: the capacity of its slices, the entries of
	// its maps, the values its pointers point at, its nested structs and its
	// strings. It first clears the fields that the encoding leaves out, as
	// Reset does. With a slab in opts, every decoded string is a substring of
	// the slab. Without one, the decode keeps each string of the receiver
	// that data repeats at its place, and takes every other string from one
	// copy of data, as [Options] states. It returns a
	// [*DecodeError] for malformed input, and for a value that the
	// ValidateKanon of a [Validator] rejects. The decode of a type whose
	// directive has the -canonical flag also returns a [*DecodeError] that
	// wraps [ErrNotCanonical] for input that is not the canonical encoding of
	// its value. After an error the receiver contains the fields decoded
	// before it, and the failed field has an unspecified value.
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

// Validator is the method set of a named type T that is not a struct: a
// bool, a number, a string, a slice, an array or a map. A //go:generate go
// tool kanon -type=T directive generates the method, and a method written
// by hand with this signature on a value receiver works alike.
//
// # Encoding
//
// The generated code encodes a value of a Validator as a value of its
// underlying type, ahead of the binary, gob and text methods of the type.
// Adding ValidateKanon to a type with such methods, or removing it, changes
// the encoding of every field of the type, which the wire format lists as
// incompatible in both directions. kanon treats a named
// interface type whose method set has ValidateKanon as any other interface,
// and encodes the value of the interface by its concrete type. When the
// generated code calls ValidateKanon, kanon rejects a Validator whose only
// value is its zero value, such as an array of no elements, whose encoding
// is a constant.
//
// # Validation
//
// The generated code calls ValidateKanon on every value of the type that it
// encodes or decodes, except a zero value that the encoding leaves out. An
// error fails the encode with an [*EncodeError], and the decode with a
// [*DecodeError] at the offset of the value, whose cause is that error. A
// view method calls it on the value of a field that the encoding contains,
// and returns the zero value for a field that the encoding leaves out.
// SizeKanon, Reset and CloneKanon do not call it.
//
// The generated code does not call the ValidateKanon of a type whose kanon
// directive has no -validate, since that method returns nil for every value.
// A field of such a type encodes without an error path. The generator reads
// the directive from the source of the package of the type when it
// generates the code of a struct. A module must generate its code again
// after it upgrades a dependency whose directive adds -validate, since code
// generated before does not call the method.
//
// # Allocation contract
//
// A ValidateKanon that allocates for a value makes the encode and the
// decode of the value allocate, which the allocation contract of [Message]
// lists as an exception. The conformance test of a type that a -type flag
// names fails when its ValidateKanon allocates for a value that it accepts.
type Validator interface {
	// ValidateKanon returns nil for a value that kanon encodes and decodes,
	// and the reason that it rejects any other value.
	ValidateKanon() error
}

// Sizer is the method of a type that encodes itself through its binary, gob
// or text methods and knows the length of that encoding without encoding.
// SizeKanon has the meaning that it has in [Message]: the length of the
// bytes that kanon writes for the value, without a tag and a length, which
// for such a type are the bytes that the encode method of its family
// returns.
//
// # Encoding
//
// The generated code sizes a value of a Sizer with SizeKanon in place of a
// call of its encode method, and appends the encoding into the room that
// SizeKanon sized, through the append method of the type when it has one.
// It still calls the encode method of every value that it writes, and fails
// the encode with an [*EncodeError] that wraps [ErrSize] when the method
// returns another length. For a type whose == does not compare every bit, a
// field is present when its SizeKanon is not 0.
//
// # Allocation contract
//
// A Sizer with an append method encodes a value of any length without an
// allocation. SizeKanon must not allocate.
type Sizer interface {
	// SizeKanon returns the length of the encoding of the receiver that the
	// encode method of its family returns.
	SizeKanon() int
}

// Exact is the method set of a [Sizer] that states two guarantees of its
// methods. It encodes itself through an append method, AppendBinary or
// AppendText, and == compares every bit of it, so that a field leaves out
// its zero value. The guarantees:
//
//   - SizeKanon returns no negative value. The append method returns no
//     error for a value other than the zero value, and appends SizeKanon
//     bytes whenever it returns no error. The append method can fail for
//     the zero value.
//   - The decode method of its family accepts a byte string only when the
//     append method writes that byte string for the value that the decode
//     method sets.
//
// The second guarantee is the guarantee of [Strict], so an Exact type need
// not declare StrictKanon.
//
// # Encoding
//
// The generated code writes a field of an Exact type without an error path,
// and a canonical decode does not encode a decoded value again. A value in
// any other position, such as an element or a union member, which the
// encoding writes at the zero value too, returns the error of the append
// method, and the generated code checks neither the room nor the length of
// its encoding. An [Appender] has no error path in any position. An append
// method that fails for the value of a field, or that appends another length
// than SizeKanon in any position, panics the encode with an [*EncodeError]
// that wraps [ErrExact]. The generation fails for a type that declares
// ExactKanon without an append method, without SizeKanon, or with an == that
// does not compare every bit.
//
// [go.thesmos.sh/kanon/kanontest.RunExact] checks both guarantees, in the
// package that declares the type.
type Exact interface {
	Sizer
	// ExactKanon marks the type. No code calls it.
	ExactKanon()
}

// Appender is the method set of an [Exact] type whose append method returns
// no error for any value, its zero value included. AppendKanon appends the
// bytes that the append method of the family of the type appends, which are
// SizeKanon bytes, and has no error result.
//
// # Encoding
//
// The generated code writes a value of an Appender through AppendKanon in
// every position, the zero value of an element included, without an error
// path, so a struct whose other fields cannot fail encodes without an error.
// The encode panics with an [*EncodeError] that wraps [ErrExact] for an
// AppendKanon that appends another length than SizeKanon. The generation
// fails for a type that declares AppendKanon without ExactKanon, and for an
// AppendKanon of another signature.
//
// [go.thesmos.sh/kanon/kanontest.RunExact] checks that AppendKanon appends
// the bytes of the append method of the family, for every value that it
// checks.
type Appender interface {
	Exact
	// AppendKanon appends the encoding of the receiver to dst and returns the
	// extended slice.
	AppendKanon(dst []byte) []byte
}

// Strict is the method set of a type that encodes itself through its binary,
// gob or text methods, and whose decode method accepts a byte string only
// when the encode method of its family writes that byte string for the value
// that the decode method sets. The guarantee is the second guarantee of
// [Exact], without the first, so the encode method can fail for any value.
// An enumeration whose text methods write and parse the names of its
// variants keeps it, and fails to encode a value outside its variants.
//
// # Encoding
//
// A canonical decode does not encode a decoded value of a Strict type again,
// in any position. The encode returns the error of the encode method, as it
// does for every type that encodes itself and is not Exact. The generation
// fails for a type that declares StrictKanon and does not encode itself
// through the methods of a family, such as a [Validator] or a struct with a
// kanon codec.
//
// [go.thesmos.sh/kanon/kanontest.RunStrict] checks the guarantee, in the
// package that declares the type.
type Strict interface {
	// StrictKanon marks the type. No code calls it.
	StrictKanon()
}

// Options set the slab and the nesting limit of a decode. With the zero
// Options, a decode keeps the strings of its receiver that the input
// repeats, copies its input once for the strings that change, and applies
// [DefaultDepth].
//
// # Strings without a slab
//
// A decode with an empty Slab keeps each string of the receiver that equals
// the decoded string at its place. The places are a field of a struct, an
// element in the capacity of a slice, an element of an array, the value that
// a pointer points at, a map value that the decode takes from the values of
// the map, and the value of an interface that stores the same concrete type.
// The decode copies its input once, for the first decoded string that is not
// empty and that the receiver does not have at its place. Every later such
// string is a substring of the copy. The strings of a map key, and of a map
// value that does not refer to memory, take substrings of the copy, since the
// decode clears a map before it reads its entries. A nested struct of
// another package decodes with the copy as its
// slab when the decode has made the copy, and with an empty Slab before, so
// that it copies its own encoding for a string that changes.
//
// A kept string refers to the memory that it referred to before the decode.
// When an earlier decode took a string of the receiver from a Slab over a
// buffer that the caller reuses, the kept string changes with the buffer.
// Reset clears the strings of a receiver, so a decode after Reset does not
// keep such a string.
type Options struct {
	// Slab is a string of which the input is the substring that starts at
	// Offset. Every decoded string is a substring of Slab, so the decode
	// allocates nothing for its strings and does not keep a string of the
	// receiver. A Slab that aliases the input's memory, as unsafe.String over
	// the input returns it, makes every decoded string alias that memory, and
	// the caller must not change the memory while the decoded value is in
	// use. An empty Slab makes the decode keep the strings of the receiver
	// that the input repeats, and copy the input for the strings that change.
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
// empty. The code of generator versions 1 to 5 takes the slab of every
// decode from it.
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
