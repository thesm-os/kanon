// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"cmp"
	"errors"

	"go.thesmos.sh/kanon"
)

// Slab is the slab of the strings of one decode, which the generated code
// passes to the functions of the decode by pointer. A decode whose
// kanon.Options have a Slab takes every string from that Slab. A decode whose
// options have an empty Slab starts without a slab. It keeps each string of
// the receiver that the input repeats at its place, and copies its input once,
// for the first decoded string that is not empty and that the receiver does
// not have at its place.
//
// The zero Slab is the slab of the decode of a struct without strings, which
// reads no string. Its Nested passes no slab to a nested decode.
//
// # Allocation contract
//
// String allocates the copy of the input once per Reset. Reset, Nested and
// Locate allocate nothing.
//
// # Concurrency
//
// A Slab is not safe for concurrent use. The functions of one decode use it
// one at a time.
type Slab struct {
	// text is the slab: the Slab of the options, or the copy of in once
	// String makes it, and "" before.
	text string
	// in is the input of a decode whose options have an empty Slab, which the
	// copy copies, and nil for options with a Slab.
	in []byte
}

// Reset makes s the slab of a decode of data with opts, and returns the
// offset of data in the slab. For a Slab in opts, s takes its strings from
// that Slab, and the offset is the Offset of opts. For an empty Slab, s
// copies data for the first string that changes, and the offset is 0. Reset
// drops the copy that s made for a decode before.
func (s *Slab) Reset(data []byte, opts kanon.Options) int {
	if opts.Slab != "" {
		s.text, s.in = opts.Slab, nil
		return opts.Offset
	}
	s.text, s.in = "", data
	return 0
}

// String returns the string that the decode sets at a place whose string is
// old, for the bytes b of the input at offset off of the slab. With a Slab in
// the options of the decode, it returns the substring of that Slab at off.
// Without one, it returns old when b equals old, the empty string when b is
// empty, and otherwise the substring at off of the copy of the input, which
// the first such call makes.
func (s *Slab) String(old string, b []byte, off int) string {
	if s.in == nil {
		return s.text[off : off+len(b)]
	}
	if string(b) == old {
		return old
	}
	if len(b) == 0 {
		return ""
	}
	if s.text == "" {
		s.text = string(s.in)
	}
	return s.text[off : off+len(b)]
}

// Nested returns the options of the decode of a struct of another package
// that starts at offset off of the slab and that enters depth levels below the
// struct at most, which is 0 or more. Once s has a slab, the options pass the
// slab and off. Before, they pass an empty Slab and the offset 0, so that the
// nested decode keeps its own strings and copies its own encoding. A depth of
// 0 becomes a Depth of -1, as [Nested] makes it.
func (s *Slab) Nested(off, depth int) kanon.Options {
	if s.text == "" {
		return kanon.Options{Depth: cmp.Or(depth, -1)}
	}
	return kanon.Options{Slab: s.text, Offset: off, Depth: cmp.Or(depth, -1)}
}

// Locate returns err, the error of the decode of a struct of another package
// that starts at offset off of the slab, with the options that Nested
// returned. When those options passed no slab, the nested decode counted its
// offsets from 0, and Locate adds off to the Offset of the first
// *kanon.DecodeError in the chain of err, as [Stream.Fail] adds the offset of
// a run. It returns nil for a nil err.
func (s *Slab) Locate(err error, off int) error {
	if err == nil || s.text != "" {
		return err
	}
	if d, ok := errors.AsType[*kanon.DecodeError](err); ok {
		d.Offset += off
	}
	return err
}
