// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"encoding/binary"
	"go/types"
	"slices"
)

// value is the encoding of one Go value: a field, an element of a slice or
// an array, the value that a pointer points at, a map key or a map value,
// or the value of a concrete type that an interface stores.
//
// A value and every value it refers to belong to the type tree of one
// field. A named type that contains itself refers to its own value, so the
// graph of values can have cycles: a walk over it marks the values it
// visits.
type value struct {
	// typ is the Go type of the value: the named type when the field or the
	// element declares one, so that the generated code converts through it.
	typ types.Type
	// id identifies the encoding, as [classifier.id] computes it. Two values
	// with one id encode alike and share the functions of the code file.
	id string
	// elem is the element of a slice, an array or a pointer, and the value
	// of a map.
	elem *value
	// key is the key of a map.
	key *value
	// inline is the target of a kindStruct value of a struct type without a
	// kanon codec, whose encoding the code file writes with functions of its
	// own. It is nil for a struct with a kanon codec, and for a struct that a
	// classifier in lookup mode did not find.
	inline *target
	// list is the list of the concrete types of a kindInterface value, and
	// variants lists the ones that the value can store, in tag order.
	list     *typeList
	variants []variant
	// size is the length of a kindByteArray or kindArray value.
	size int64
	kind kind
	// self names the methods through which a kindBinary value encodes itself.
	self selfCodec
	// fails reports that encoding the value can fail: it is or contains a
	// kindBinary value, a kanon.Validator whose ValidateKanon the generated
	// code calls, or an interface. The value of a field that [exactField]
	// reports cannot fail.
	fails bool
	// validate reports that the value is of a kanon.Validator, a named type
	// that is not a struct and encodes as its underlying type, and that the
	// generated code calls its ValidateKanon on every value that it writes or
	// reads, as [classifier.checks] reports. It is false for a type whose
	// generated ValidateKanon returns nil for every value.
	validate bool
	// indirect reports that a kindStruct value refers to memory outside
	// itself: a pointer, a slice, a map or an interface among its encoded
	// fields, at any depth of the structs it nests by value.
	indirect bool
	// nilable reports that the code sizes and encodes a kindInterface value
	// that can be nil, as [markNilable] finds it.
	nilable bool
	// hollow reports that a kindStruct value of a struct with a kanon codec
	// that kanon generates has one value, as [classifier.hollow] finds it.
	hollow bool
	// hidden reports that a kindStruct value can contain state that its
	// encoding leaves out, as [classifier.hidden] finds it.
	hidden bool
}

// holdsMemory reports whether a decode can reuse memory that v refers to:
// v is a pointer, a byte slice, a slice, a map or an interface, a struct
// that refers to memory, or an array of values that refer to memory.
func (v *value) holdsMemory() bool {
	switch v.kind {
	case kindPointer, kindBytes, kindSlice, kindMap, kindInterface:
		return true
	case kindStruct:
		return v.indirect
	case kindArray:
		return v.size > 0 && v.elem.holdsMemory()
	default:
		return false
	}
}

// zeroOnly reports whether the zero value is the only value of v: an array
// of no elements, an array of elements of one value, an inline struct
// without a union, without unknown fields and without a field of more than
// one value, and a hollow struct with a kanon codec. A field of v is never
// present, and a value of v encodes to the constant bytes that
// [value.constant] returns.
func (v *value) zeroOnly() bool {
	switch v.kind {
	case kindByteArray:
		return v.size == 0
	case kindArray:
		return v.size == 0 || v.elem.zeroOnly()
	case kindStruct:
		m := v.inline
		if m == nil {
			return v.hollow
		}
		if m.unknown != nil || len(m.discriminators) > 0 {
			return false
		}
		for _, f := range m.fields {
			if !f.val.zeroOnly() {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// inert reports whether a value of v is always the zero value of its type:
// an array of no elements or of inert elements, and a struct of one encoded
// value, as [value.zeroOnly] reports, without state that its encoding leaves
// out. Reset, a decode and a clone write nothing for an inert value. A
// struct that zeroOnly reports and that has a skipped field is not inert,
// since the skipped field can have any value.
func (v *value) inert() bool {
	switch v.kind {
	case kindByteArray:
		return v.size == 0
	case kindArray:
		return v.size == 0 || v.elem.inert()
	case kindStruct:
		return v.zeroOnly() && !v.hidden
	default:
		return false
	}
}

// constant returns the encoding of the zero value of v, a value that
// [value.zeroOnly] reports: the length 0 of a struct and of an array of no
// elements, and the length and the encodings of the elements of any other
// array.
func (v *value) constant() []byte {
	if v.kind != kindArray || v.size == 0 {
		return []byte{0}
	}
	var content []byte
	for range v.size {
		content = append(content, v.elem.constant()...)
	}
	return append(binary.AppendUvarint(nil, uint64(len(content))), content...)
}

// merges reports whether a repeated occurrence of v merges into the value
// instead of replacing it: a slice appends its elements, a map adds its
// entries, a struct merges, and a pointer to a struct merges into the struct
// that it points at.
func (v *value) merges() bool {
	switch v.kind {
	case kindSlice, kindMap, kindStruct:
		return true
	case kindPointer:
		return v.elem.kind == kindStruct
	default:
		return false
	}
}

// variantMerges reports whether v, an interface, can store a concrete type
// that merges, as [value.merges] reports, so that a repeated occurrence of
// v can merge into its value.
func (v *value) variantMerges() bool {
	return slices.ContainsFunc(v.variants, func(c variant) bool { return c.val.merges() })
}

// unframed reports whether the encoding of v begins with neither a length
// nor a fixed width of its own: a pointer begins with its presence byte, and
// an interface with its type number. A field whose value is one of them, or
// points at one, puts a varint length before the encoding.
func (v *value) unframed() bool {
	return v.kind == kindPointer || v.kind == kindInterface
}

// signed reports whether v, an integer, has a signed Go type.
func (v *value) signed() bool {
	b, _ := v.typ.Underlying().(*types.Basic)
	return b.Info()&types.IsUnsigned == 0
}

// narrow reports whether the decode of v, a varint of an integer type,
// checks the range of its Go type: the type is neither int64 nor uint64.
// int, uint and uintptr are 32 bits wide on some platforms.
func (v *value) narrow() bool {
	b, _ := v.typ.Underlying().(*types.Basic)
	return b.Kind() != types.Int64 && b.Kind() != types.Uint64
}

// native reports whether v is an integer of a Go type whose width is the
// platform's: int, uint or uintptr, which are 32 bits wide on some
// platforms and 64 bits on the wire.
func (v *value) native() bool {
	b, _ := v.typ.Underlying().(*types.Basic)
	switch b.Kind() {
	case types.Int, types.Uint, types.Uintptr:
		return true
	default:
		return false
	}
}

// basicName returns the name of the basic type that underlies v, a bool, a
// number or a string: the Go type through which the value passes between
// the wire and its field.
func (v *value) basicName() string {
	b, _ := v.typ.Underlying().(*types.Basic)
	return b.Name()
}
