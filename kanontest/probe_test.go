// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"io"
	"testing"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of the families of probes that the cases run.
const (
	faultCheck  = "DecodeKanon/decodes a field between unknown fields with one value written wrong as the reference decode"
	changeCheck = "DecodeKanon/decodes a field between unknown fields with one changed byte as the reference decode"
	depthCheck  = "DecodeKanon/decodes a field between unknown fields under each depth limit as the reference decode"
	// errorCheck is the message of a check whose decode returns another
	// error than the reference decode.
	errorCheck = "DecodeKanon returns the error of the reference decode"
)

// eofOff is a view.Item whose DecodeKanon decodes truncated input without
// an error.
type eofOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps
// io.ErrUnexpectedEOF.
func (m *eofOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "eofOff"); !errors.Is(err, io.ErrUnexpectedEOF) {
		return err
	}
	return nil
}

// rangeOff is a view.Item whose DecodeKanon decodes a value outside the
// range of its field without an error.
type rangeOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps kanon.ErrRange.
func (m *rangeOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "rangeOff"); !errors.Is(err, kanon.ErrRange) {
		return err
	}
	return nil
}

// malformedOff is a view.Item whose DecodeKanon decodes malformed input
// without an error.
type malformedOff struct{ view.Item }

// DecodeKanon decodes data, and drops an error that wraps
// kanon.ErrMalformed.
func (m *malformedOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if err := renamed(m.Item.DecodeKanon(data, opts), "Item", "malformedOff"); !errors.Is(err, kanon.ErrMalformed) {
		return err
	}
	return nil
}

// causeOff is a view.Item whose DecodeKanon returns an error that states its
// cause in its detail with kanon.ErrRange as the cause, and the message of
// the error of the decode.
type causeOff struct{ view.Item }

// DecodeKanon decodes data, and sets the cause of an error with a detail to
// kanon.ErrRange.
func (m *causeOff) DecodeKanon(data []byte, opts kanon.Options) error {
	err := renamed(m.Item.DecodeKanon(data, opts), "Item", "causeOff")
	if e, ok := errors.AsType[*kanon.DecodeError](err); ok && e.Detail != "" {
		e.Err = kanon.ErrRange
	}
	return err
}

// depthOff is a view.Record whose DecodeKanon ignores the depth limit of its
// options.
type depthOff struct{ view.Record }

// DecodeKanon decodes data with the default depth limit.
func (m *depthOff) DecodeKanon(data []byte, opts kanon.Options) error {
	opts.Depth = 0
	return renamed(m.Record.DecodeKanon(data, opts), "Record", "depthOff")
}

// slabOff is a view.Item whose DecodeKanon reads its slab one byte after the
// offset of its options.
type slabOff struct{ view.Item }

// DecodeKanon decodes data with the offset of a slab one byte too far.
func (m *slabOff) DecodeKanon(data []byte, opts kanon.Options) error {
	if opts.Slab != "" {
		opts.Offset++
	}
	return m.Item.DecodeKanon(data, opts)
}

func TestProbe(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that decodes a cut encoding", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[eofOff]{Fields: itemSpec.Fields}, prefixCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a value outside its range", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[rangeOff]{Fields: itemSpec.Fields}, faultCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes a tag of another wire format", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[malformedOff]{Fields: itemSpec.Fields}, changeCheck, errorCheck)
		})
		t.Run("fails for a codec that returns an error with the same message and another cause", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[causeOff]{Fields: itemSpec.Fields}, changeCheck, errorCheck)
		})
		t.Run("fails for a codec that decodes deeper than the limit", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[depthOff]{Fields: recordSpec.Fields}, depthCheck, errorCheck)
		})
		t.Run("fails for a codec that reads its slab at another offset", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[slabOff]{Fields: itemSpec.Fields}, slabCheck,
				"DecodeKanon decodes the value of the reference decode")
		})
	})
}
