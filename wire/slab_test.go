// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unsafe"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/bench"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The input of the Slab cases, the bytes of its string at slabAt, and the
// offset of the input in the slab of a caller, which starts with slabPrefix.
const (
	slabInput  = "abcdefgh"
	slabPrefix = "xyz"
	slabAt     = 2
	slabWord   = "cde"
	// slabNestedAt and slabDepth are the offset and the depth of the nested
	// struct of the Nested and Locate cases.
	slabNestedAt = 5
	slabDepth    = 3
	// slabErrorAt is the offset of the error of the nested decode of the
	// Locate cases, from the start of the encoding of the nested struct.
	slabErrorAt = 4
)

// callerOptions are the options of a decode of slabInput with the slab of a
// caller, which contains the input after slabPrefix.
var callerOptions = kanon.Options{Slab: slabPrefix + slabInput, Offset: len(slabPrefix)}

// Sinks of the allocation checks, which keep the compiler from dropping a
// result.
var (
	sinkString  string
	sinkOptions kanon.Options
)

func TestSlab(t *testing.T) {
	t.Parallel()
	t.Run("Reset", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the Offset of options with a slab", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			assert.Equal(t, s.Reset([]byte(slabInput), callerOptions), len(slabPrefix),
				"Reset returns the offset of the input in the slab")
		})
		t.Run("returns offset 0 for options with an empty Slab", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			assert.Equal(t, s.Reset([]byte(slabInput), kanon.Options{Offset: len(slabPrefix)}), 0,
				"Reset counts the offsets of a decode without a slab from 0")
		})
		t.Run("drops the copy of the decode before", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			_ = s.String("", data[:1], off)
			s.Reset(data, kanon.Options{})
			assert.Equal(t, s.Nested(slabNestedAt, slabDepth), kanon.Options{Depth: slabDepth},
				"the slab of the new decode has no copy")
		})
		t.Run("drops the input of the decode before", func(t *testing.T) {
			t.Parallel()
			data, opts := []byte(slabInput), callerOptions
			var s wire.Slab
			s.Reset(data, kanon.Options{})
			off := s.Reset(data, opts)
			got := s.String(slabWord, data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, unsafe.StringData(got), unsafe.StringData(opts.Slab[off+slabAt:]),
				"the slab of the new decode takes its strings from the slab of the caller", assert.ByIdentity())
		})
	})
	t.Run("String", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the substring of the slab of options with a slab", func(t *testing.T) {
			t.Parallel()
			data, opts := []byte(slabInput), callerOptions
			var s wire.Slab
			off := s.Reset(data, opts)
			got := s.String("", data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, unsafe.StringData(got), unsafe.StringData(opts.Slab[off+slabAt:]),
				"String returns the bytes at their offset of the slab", assert.ByIdentity())
			assert.Equal(t, got, slabWord, "String returns the decoded string")
		})
		t.Run("returns the substring of the slab for bytes that equal old", func(t *testing.T) {
			t.Parallel()
			data, opts := []byte(slabInput), callerOptions
			var s wire.Slab
			off := s.Reset(data, opts)
			got := s.String(strings.Clone(slabWord), data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, unsafe.StringData(got), unsafe.StringData(opts.Slab[off+slabAt:]),
				"String takes the string from the slab of the caller", assert.ByIdentity())
		})
		t.Run("returns old for bytes that equal it", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			old := strings.Clone(slabWord)
			got := s.String(old, data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, unsafe.StringData(got), unsafe.StringData(old), "String keeps the string of the place",
				assert.ByIdentity())
		})
		t.Run("returns the empty string for empty bytes", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			assert.Equal(t, s.String(slabWord, data[slabAt:slabAt], off+slabAt), "",
				"String returns the empty string for a value of no bytes")
		})
		t.Run("returns a copy of the bytes that differ from old", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			got := s.String("", data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, got, slabWord, "String returns the decoded string")
			assert.Pure(t, func() string { return got }, func() { clear(data) },
				"the decoded string does not change with the input")
		})
		t.Run("returns the strings that differ from one copy of the input", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			first := s.String("", data[:1], off)
			second := s.String("", data[slabAt:slabAt+len(slabWord)], off+slabAt)
			assert.Equal(t, unsafe.Pointer(unsafe.StringData(second)),
				unsafe.Add(unsafe.Pointer(unsafe.StringData(first)), slabAt),
				"String returns the second string from the copy of the first", assert.ByIdentity())
		})
	})
	t.Run("Nested", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the slab and the offset of options with a slab", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			s.Reset([]byte(slabInput), callerOptions)
			assert.Equal(t, s.Nested(slabNestedAt, slabDepth),
				kanon.Options{Slab: callerOptions.Slab, Offset: slabNestedAt, Depth: slabDepth},
				"Nested passes the slab of the caller and the offset of the nested struct")
		})
		t.Run("returns the copy and the offset once a string copied the input", func(t *testing.T) {
			t.Parallel()
			data := []byte(slabInput)
			var s wire.Slab
			off := s.Reset(data, kanon.Options{})
			first := s.String("", data[:1], off)
			got := s.Nested(slabNestedAt, slabDepth)
			assert.Equal(t, got, kanon.Options{Slab: slabInput, Offset: slabNestedAt, Depth: slabDepth},
				"Nested passes the copy of the input and the offset of the nested struct")
			assert.Equal(t, unsafe.StringData(got.Slab), unsafe.StringData(first),
				"Nested passes the copy that String made", assert.ByIdentity())
		})
		t.Run("returns options with an empty Slab before the copy", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			s.Reset([]byte(slabInput), kanon.Options{})
			assert.Equal(t, s.Nested(slabNestedAt, slabDepth), kanon.Options{Depth: slabDepth},
				"Nested passes neither a slab nor an offset")
		})
		t.Run("returns options with an empty Slab for the zero Slab", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			assert.Equal(t, s.Nested(slabNestedAt, slabDepth), kanon.Options{Depth: slabDepth},
				"Nested passes neither a slab nor an offset")
		})
		t.Run("returns the limit 0 for depth 0", func(t *testing.T) {
			t.Parallel()
			var s wire.Slab
			s.Reset([]byte(slabInput), callerOptions)
			assert.Equal(t, s.Nested(slabNestedAt, 0).Limit(), 0,
				"the nested decode enters no level below the struct")
		})
	})
	t.Run("Locate", func(t *testing.T) {
		t.Parallel()
		lazy := func() *wire.Slab {
			s := new(wire.Slab)
			s.Reset([]byte(slabInput), kanon.Options{})
			return s
		}
		copied := func() *wire.Slab {
			data := []byte(slabInput)
			s := new(wire.Slab)
			off := s.Reset(data, kanon.Options{})
			_ = s.String("", data[:1], off)
			return s
		}
		caller := func() *wire.Slab {
			s := new(wire.Slab)
			s.Reset([]byte(slabInput), callerOptions)
			return s
		}
		tests := []struct {
			name        string
			giveSlab    func() *wire.Slab
			giveWrapped bool
			wantAt      int
		}{
			{
				name:     "adds the offset to the DecodeError of a decode without the copy",
				giveSlab: lazy, wantAt: slabNestedAt + slabErrorAt,
			},
			{
				name:     "adds the offset to a DecodeError that another error wraps",
				giveSlab: lazy, giveWrapped: true, wantAt: slabNestedAt + slabErrorAt,
			},
			{
				name:     "adds the offset to the DecodeError of the zero Slab",
				giveSlab: func() *wire.Slab { return new(wire.Slab) }, wantAt: slabNestedAt + slabErrorAt,
			},
			{name: "keeps the offset of a decode after the copy", giveSlab: copied, wantAt: slabErrorAt},
			{name: "keeps the offset of a decode with a slab", giveSlab: caller, wantAt: slabErrorAt},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := wire.ReadError(0, "T.F", 1, slabErrorAt)
				if tt.giveWrapped {
					err = fmt.Errorf("decode: %w", err)
				}
				got := tt.giveSlab().Locate(err, slabNestedAt)
				assert.Equal(t, got, err, "Locate returns the error of the nested decode", assert.ByIdentity())
				d := assert.ErrorAs[*kanon.DecodeError](t, got, "the error is a *kanon.DecodeError")
				assert.Equal(t, d.Offset, tt.wantAt, "the offset of the error counts from the start of the slab")
			})
		}
		t.Run("returns an error without a DecodeError unchanged", func(t *testing.T) {
			t.Parallel()
			err := errors.New("decode: failed")
			assert.Equal(t, lazy().Locate(err, slabNestedAt), err, "Locate returns the error",
				assert.ByIdentity())
		})
		t.Run("returns nil for no error", func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, lazy().Locate(nil, slabNestedAt), "Locate returns nil")
		})
	})
}

// TestSlabAllocs checks the allocations of a Slab. It runs serially:
// testing.AllocsPerRun panics while a parallel test runs.
func TestSlabAllocs(t *testing.T) {
	data := []byte(slabInput)
	word := data[slabAt : slabAt+len(slabWord)]
	old := strings.Clone(slabWord)
	var lazy, caller, copied wire.Slab
	lazy.Reset(data, kanon.Options{})
	off := caller.Reset(data, callerOptions)
	copied.Reset(data, kanon.Options{})
	_ = copied.String("", data[:1], 0)
	t.Run("Reset/allocates nothing", func(t *testing.T) {
		var s wire.Slab
		assert.MaxAllocs(t, func() { sinkInt = s.Reset(data, callerOptions) }, 0, "Reset allocates nothing")
	})
	t.Run("String/allocates nothing for bytes that equal old", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkString = lazy.String(old, word, slabAt) }, 0,
			"String allocates nothing for a string that it keeps")
	})
	t.Run("String/allocates nothing with a slab", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkString = caller.String("", word, off+slabAt) }, 0,
			"String allocates nothing for a substring of the slab of the caller")
	})
	t.Run("String/allocates nothing once the copy exists", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkString = copied.String("", word, slabAt) }, 0,
			"String allocates nothing for a substring of the copy")
	})
	t.Run("String/allocates the copy for bytes that differ", func(t *testing.T) {
		setup := func() *wire.Slab {
			s := new(wire.Slab)
			s.Reset(data, kanon.Options{})
			return s
		}
		assert.MaxAllocsWithSetup(t, setup, func(s *wire.Slab) { sinkString = s.String("", word, slabAt) }, 1,
			"String allocates one copy of the input")
	})
	t.Run("Nested/allocates nothing", func(t *testing.T) {
		assert.MaxAllocs(t, func() { sinkOptions = copied.Nested(slabNestedAt, slabDepth) }, 0,
			"Nested allocates nothing")
	})
	t.Run("Locate/allocates nothing", func(t *testing.T) {
		err := wire.ReadError(0, "T.F", 1, slabErrorAt)
		assert.MaxAllocs(t, func() { errSink = lazy.Locate(err, 0) }, 0, "Locate allocates nothing")
	})
}

func BenchmarkSlab(b *testing.B) {
	data := []byte(slabInput)
	word := data[slabAt : slabAt+len(slabWord)]
	b.Run("Reset", func(b *testing.B) {
		var s wire.Slab
		var got int
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Reset(data, callerOptions)
		}
		assert.Equal(b, got, len(slabPrefix), "Reset returns the offset of the input in the slab")
	})
	b.Run("String", func(b *testing.B) {
		b.Run("kept", func(b *testing.B) {
			var s wire.Slab
			s.Reset(data, kanon.Options{})
			old := strings.Clone(slabWord)
			var got string
			c := bench.Start(b).MaxAllocs(0)
			defer c.End()
			for c.Loop() {
				got = s.String(old, word, slabAt)
			}
			assert.Equal(b, got, slabWord, "String returns the decoded string")
		})
		b.Run("slab", func(b *testing.B) {
			var s wire.Slab
			off := s.Reset(data, callerOptions)
			var got string
			c := bench.Start(b).MaxAllocs(0)
			defer c.End()
			for c.Loop() {
				got = s.String("", word, off+slabAt)
			}
			assert.Equal(b, got, slabWord, "String returns the decoded string")
		})
		b.Run("copied", func(b *testing.B) {
			var s wire.Slab
			s.Reset(data, kanon.Options{})
			_ = s.String("", data[:1], 0)
			var got string
			c := bench.Start(b).MaxAllocs(0)
			defer c.End()
			for c.Loop() {
				got = s.String("", word, slabAt)
			}
			assert.Equal(b, got, slabWord, "String returns the decoded string")
		})
	})
	b.Run("Nested", func(b *testing.B) {
		var s wire.Slab
		s.Reset(data, callerOptions)
		var got kanon.Options
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Nested(slabNestedAt, slabDepth)
		}
		assert.Equal(b, got.Offset, slabNestedAt, "Nested passes the offset of the nested struct")
	})
	b.Run("Locate", func(b *testing.B) {
		var s wire.Slab
		s.Reset(data, kanon.Options{})
		var got error
		c := bench.Start(b).MaxAllocs(0)
		defer c.End()
		for c.Loop() {
			got = s.Locate(nil, slabNestedAt)
		}
		assert.NoError(b, got, "Locate returns nil")
	})
}
