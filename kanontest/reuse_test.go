// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/bound"
	"go.thesmos.sh/kanon/internal/fixture/number"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of the methods that keep the memory of a receiver.
const (
	reuseCheck = "DecodeKanon/decodes into a receiver that decoded before as into a zero one"
	mergeCheck = "MergeKanon/merges an encoding into a decoded value as the reference decode"
	resetCheck = "Reset/sets the receiver to the zero value"
	cloneCheck = "CloneKanon/returns a copy that shares no memory with the receiver"
)

// mergeDecode is a view.Item whose DecodeKanon merges into the receiver
// without resetting it.
type mergeDecode struct{ view.Item }

// DecodeKanon merges data into m.
func (m *mergeDecode) DecodeKanon(data []byte, opts kanon.Options) error {
	return m.MergeKanon(data, opts)
}

// decodeMerge is a view.Item whose MergeKanon resets the receiver before it
// decodes.
type decodeMerge struct{ view.Item }

// MergeKanon decodes data into m as DecodeKanon does.
func (m *decodeMerge) MergeKanon(data []byte, opts kanon.Options) error {
	return m.DecodeKanon(data, opts)
}

// mergePast is a bound.Page whose MergeKanon drops the error of an element
// past a bound.
type mergePast struct{ bound.Page }

// MergeKanon merges data into m, and drops an error that wraps kanon.ErrMax.
func (m *mergePast) MergeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Page.MergeKanon(data, opts), "Page", "mergePast"); !errors.Is(err, kanon.ErrMax) {
		return err
	}
	return nil
}

// resetNothing is a view.Item whose Reset keeps the fields of the receiver.
type resetNothing struct{ view.Item }

// Reset does nothing.
func (*resetNothing) Reset() {}

// cloneShallow is a view.Record whose CloneKanon copies the receiver without
// copying the memory that it refers to.
type cloneShallow struct{ view.Record }

// CloneKanon returns a copy of the struct that m points at.
func (m *cloneShallow) CloneKanon() *cloneShallow {
	if m == nil {
		return nil
	}
	c := *m
	return &c
}

// cloneNil is a view.Item whose CloneKanon returns a zero Item for a nil
// receiver.
type cloneNil struct{ view.Item }

// CloneKanon returns a copy of the Item that m points at, and a new zero
// Item for a nil m.
func (m *cloneNil) CloneKanon() *cloneNil {
	if m == nil {
		return new(cloneNil)
	}
	return &cloneNil{*m.Item.CloneKanon()}
}

// skippedFields describes the encoded fields of number.Skipped, which the
// wrappers of number.Skipped promote.
var skippedFields = []kanontest.Field{{Name: "First", Number: 1}, {Name: "Second", Number: 2}}

// keepsMemo is a number.Skipped with a field that the encoding leaves out,
// which the Reset and the DecodeKanon of number.Skipped keep.
type keepsMemo struct {
	number.Skipped
	// Memo is left out of the encoding.
	Memo string `kanon:"-"`
}

// Reset resets the number.Skipped of m and keeps Memo. A nil m returns.
func (m *keepsMemo) Reset() {
	if m != nil {
		m.Skipped.Reset()
	}
}

// mergeClearsMemo is a number.Skipped whose MergeKanon clears Memo, a field
// that the encoding leaves out.
type mergeClearsMemo struct {
	number.Skipped
	// Memo is left out of the encoding.
	Memo string `kanon:"-"`
}

// MergeKanon clears Memo and merges data into m.
func (m *mergeClearsMemo) MergeKanon(data []byte, opts kanon.Options) error {
	m.Memo = ""
	return m.Skipped.MergeKanon(data, opts)
}

// cloneKeepsMemo is a number.Skipped whose CloneKanon copies Memo, a field
// that the encoding leaves out.
type cloneKeepsMemo struct {
	number.Skipped
	// Memo is left out of the encoding.
	Memo string `kanon:"-"`
}

// CloneKanon returns a copy of the struct that m points at, Memo included.
func (m *cloneKeepsMemo) CloneKanon() *cloneKeepsMemo {
	if m == nil {
		return nil
	}
	return &cloneKeepsMemo{Skipped: *m.Skipped.CloneKanon(), Memo: m.Memo}
}

// cloneCount is a view.Item whose CloneKanon copies Count one too high.
type cloneCount struct{ view.Item }

// CloneKanon returns a copy of the Item that m points at, whose Count is one
// more.
func (m *cloneCount) CloneKanon() *cloneCount {
	if m == nil {
		return nil
	}
	c := &cloneCount{*m.Item.CloneKanon()}
	c.Count++
	return c
}

func TestReuse(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec whose DecodeKanon keeps the fields that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[mergeDecode]{Fields: itemSpec.Fields}, reuseCheck,
				"DecodeKanon into a receiver that decoded")
		})
		t.Run("fails for a codec whose MergeKanon resets the receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[decodeMerge]{Fields: itemSpec.Fields}, mergeCheck,
				"MergeKanon decodes the value of the reference decode")
		})
		t.Run("fails for a codec whose MergeKanon merges past the bound of a field", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[mergePast]{Fields: boundSpec.Fields, Canonical: true}, mergeCheck,
				"MergeKanon returns the error of the reference decode")
		})
		t.Run("fails for a codec whose Reset keeps the fields of the receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[resetNothing]{Fields: itemSpec.Fields}, resetCheck,
				"Reset sets the receiver to the zero value")
		})
		t.Run("fails for a codec without CloneKanon", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[resetNothing]{Fields: itemSpec.Fields}, cloneCheck,
				"the codec implements kanon.Cloner")
		})
		t.Run("fails for a codec whose CloneKanon returns a value for a nil receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[cloneNil]{Fields: itemSpec.Fields}, cloneCheck,
				"CloneKanon returns nil for a nil receiver")
		})
		t.Run("fails for a codec whose CloneKanon copies a field to another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[cloneCount]{Fields: itemSpec.Fields}, cloneCheck,
				"CloneKanon returns a copy of the receiver")
		})
		t.Run("fails for a codec whose CloneKanon shares memory with the receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[cloneShallow]{Fields: recordSpec.Fields}, cloneCheck,
				"the copy that CloneKanon returns shares no memory with the receiver")
		})
		t.Run("fails for a codec whose DecodeKanon keeps a field that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[keepsMemo]{Fields: skippedFields}, reuseCheck,
				"DecodeKanon into a receiver that decoded wide sample 0 clears the fields that the encoding leaves out")
		})
		t.Run("fails for a codec whose MergeKanon clears a field that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[mergeClearsMemo]{Fields: skippedFields}, mergeCheck,
				"keeps the fields that the encoding leaves out as the reference decode keeps them")
		})
		t.Run("fails for a codec whose Reset keeps a field that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[keepsMemo]{Fields: skippedFields}, resetCheck,
				"Reset clears the fields that the encoding leaves out")
		})
		t.Run("fails for a codec whose CloneKanon copies a field that the encoding leaves out", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[cloneKeepsMemo]{Fields: skippedFields}, cloneCheck,
				"CloneKanon clears the fields that the encoding leaves out")
		})
	})
}
