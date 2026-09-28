// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks that count allocations.
const (
	sizeAllocsCheck   = "SizeKanon/allocates nothing"
	encodeAllocsCheck = "EncodeKanon/allocates nothing into a buffer of the length of the encoding"
	appendAllocsCheck = "AppendBinary/allocates nothing into a buffer with room for the encoding"
	decodeAllocsCheck = "DecodeKanon/allocates nothing into a receiver that decoded the encoding before"
)

// escaped keeps the memory that the codecs that allocate allocate, so that
// the memory escapes to the heap.
var escaped []byte

// allocating is a view.Item whose methods that count their allocations
// allocate.
type allocating struct{ view.Item }

// SizeKanon returns the length of the encoding, and allocates.
func (m *allocating) SizeKanon() int {
	escaped = make([]byte, 1)
	return m.Item.SizeKanon()
}

// EncodeKanon writes the encoding into the end of buf, and allocates.
func (m *allocating) EncodeKanon(buf []byte) (int, error) {
	escaped = make([]byte, 1)
	return m.Item.EncodeKanon(buf)
}

// AppendBinary appends the encoding to b, and allocates.
func (m *allocating) AppendBinary(b []byte) ([]byte, error) {
	escaped = make([]byte, 1)
	return m.Item.AppendBinary(b)
}

// DecodeKanon decodes data into m, and allocates.
func (m *allocating) DecodeKanon(data []byte, opts kanon.Options) error {
	escaped = make([]byte, 1)
	return m.Item.DecodeKanon(data, opts)
}

// counts runs the checks of spec that count allocations on t, each as a
// subtest, one after the other.
func counts[T any, P kanontest.Codec[T]](t *testing.T, spec kanontest.Spec[T]) {
	t.Helper()
	for _, c := range kanontest.Checks[T, P](spec) {
		if c.Serial {
			t.Run(c.Name, func(t *testing.T) { c.Run(t) })
		}
	}
}

// countsAllocations skips t unless assert.MaxAllocs fails a function that
// allocates on every call. A build with the race detector, msan or asan,
// and one whose -gcflags turn off optimisation or inlining, checks no
// ceiling, so a check that counts allocations passes every codec there.
func countsAllocations(t *testing.T) {
	t.Helper()
	r := assert.NewRecorder()
	assert.MaxAllocs(r, func() { escaped = make([]byte, 1) }, 0, "the probe allocates on every call")
	if !r.Failed() {
		t.Skip("assert.MaxAllocs checks no ceiling in this build")
	}
}

// TestAllocs runs the checks that count allocations, which count them while
// no parallel test runs, so that it and its subtests do not call
// t.Parallel.
func TestAllocs(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		t.Run("runs the checks that a generated codec passes", func(t *testing.T) {
			kanontest.Run(t, itemSpec)
		})
	})
	t.Run("Checks", func(t *testing.T) {
		t.Run("passes a struct of every scalar type, a time, nested structs and pointers", func(t *testing.T) {
			counts(t, recordSpec)
		})
		t.Run("passes a field of each family of methods that encode a type", func(t *testing.T) {
			counts(t, codecsSpec)
		})
		t.Run("passes interfaces in every place of a value", func(t *testing.T) {
			counts(t, placesSpec)
		})
		t.Run("passes interfaces that store a type of every family", func(t *testing.T) {
			counts(t, variantsSpec)
		})
		t.Run("passes map keys of pointers", func(t *testing.T) {
			counts(t, kanontest.Spec[mapkey.Pointers]{
				Fields: fields("Int32", "Point", "Empty", "Array", "Loop"),
				Keys:   []kanontest.Struct{pointKey},
			})
		})
		t.Run("fails for a codec whose SizeKanon allocates", func(t *testing.T) {
			countsAllocations(t)
			rejects(t, kanontest.Spec[allocating]{Fields: itemSpec.Fields}, sizeAllocsCheck,
				"SizeKanon allocates nothing")
		})
		t.Run("fails for a codec whose EncodeKanon allocates", func(t *testing.T) {
			countsAllocations(t)
			rejects(t, kanontest.Spec[allocating]{Fields: itemSpec.Fields}, encodeAllocsCheck,
				"EncodeKanon allocates nothing")
		})
		t.Run("fails for a codec whose AppendBinary allocates", func(t *testing.T) {
			countsAllocations(t)
			rejects(t, kanontest.Spec[allocating]{Fields: itemSpec.Fields}, appendAllocsCheck,
				"AppendBinary allocates nothing")
		})
		t.Run("fails for a codec whose DecodeKanon allocates", func(t *testing.T) {
			countsAllocations(t)
			rejects(t, kanontest.Spec[allocating]{Fields: itemSpec.Fields}, decodeAllocsCheck,
				"DecodeKanon allocates nothing")
		})
	})
}
