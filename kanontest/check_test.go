// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.dokimi.dev/assert/golden"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// codecsSpec describes codec.Codecs, a struct with a field of each type of
// the codec package that encodes itself, each of which fails to encode for
// one value.
var codecsSpec = kanontest.Spec[codec.Codecs]{
	Fields: []kanontest.Field{
		{Name: "Token", Number: 1},
		{Name: "Ticket", Number: 2},
		{Name: "Grade", Number: 3},
		{Name: "Word", Number: 4},
		{Name: "Stamp", Number: 5},
		{Name: "Note", Number: 6},
		{Name: "Parity", Number: 7},
		{Name: "Seal", Number: 8},
		{Name: "Reading", Number: 9},
		{Name: "Blob", Number: 10},
	},
}

// sizeOff is a view.Item whose SizeKanon counts one byte too many.
type sizeOff struct{ view.Item }

// SizeKanon returns one more than the length of the encoding.
func (m *sizeOff) SizeKanon() int { return m.Item.SizeKanon() + 1 }

// marshalExtra is a view.Item whose MarshalBinary appends a byte to the
// encoding.
type marshalExtra struct{ view.Item }

// MarshalBinary returns the encoding and a zero byte.
func (m *marshalExtra) MarshalBinary() ([]byte, error) {
	b, err := m.Item.MarshalBinary()
	return append(b, 0), err
}

// swallow is a codec.Codecs whose MarshalBinary drops the error of a value
// that fails to encode.
type swallow struct{ codec.Codecs }

// MarshalBinary returns the encoding without its error.
func (m *swallow) MarshalBinary() ([]byte, error) {
	b, _ := m.Codecs.MarshalBinary()
	return b, nil
}

// appendDrop is a view.Item whose AppendBinary drops the bytes of its
// buffer.
type appendDrop struct{ view.Item }

// AppendBinary returns the encoding alone.
func (m *appendDrop) AppendBinary([]byte) ([]byte, error) { return m.Item.AppendBinary(nil) }

// encodeFront is a view.Item whose EncodeKanon writes the encoding into the
// start of its buffer.
type encodeFront struct{ view.Item }

// EncodeKanon writes the encoding into the first SizeKanon bytes of buf.
func (m *encodeFront) EncodeKanon(buf []byte) (int, error) {
	return m.Item.EncodeKanon(buf[:m.SizeKanon()])
}

// shortIgnore is a view.Item whose EncodeKanon returns no error for a short
// buffer.
type shortIgnore struct{ view.Item }

// EncodeKanon writes nothing into a short buffer, and returns 0 and nil.
func (m *shortIgnore) EncodeKanon(buf []byte) (int, error) {
	if len(buf) < m.SizeKanon() {
		return 0, nil
	}
	return m.Item.EncodeKanon(buf)
}

// nilSizeOne is a view.Item whose SizeKanon returns 1 for a nil receiver.
type nilSizeOne struct{ view.Item }

// SizeKanon returns 1 for a nil receiver.
func (m *nilSizeOne) SizeKanon() int {
	if m == nil {
		return 1
	}
	return m.Item.SizeKanon()
}

// nilEncodeByte is a view.Item whose EncodeKanon writes a byte for a nil
// receiver.
type nilEncodeByte struct{ view.Item }

// EncodeKanon writes a zero byte for a nil receiver.
func (m *nilEncodeByte) EncodeKanon(buf []byte) (int, error) {
	if m == nil {
		buf[len(buf)-1] = 0
		return 1, nil
	}
	return m.Item.EncodeKanon(buf)
}

// nilAppendByte is a view.Item whose AppendBinary appends a byte for a nil
// receiver.
type nilAppendByte struct{ view.Item }

// AppendBinary appends a zero byte for a nil receiver.
func (m *nilAppendByte) AppendBinary(b []byte) ([]byte, error) {
	if m == nil {
		return append(b, 0), nil
	}
	return m.Item.AppendBinary(b)
}

// unpinned is a view.Item without a golden file.
type unpinned struct{ view.Item }

func TestCheck(t *testing.T) {
	t.Parallel()
	t.Run("SizeKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that counts one byte too many", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[sizeOff]{Fields: itemSpec.Fields},
				"SizeKanon/returns the length of the reference encoding",
				"SizeKanon returns the length of the reference encoding")
		})
		t.Run("fails for a codec that counts one byte for a nil receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[nilSizeOne]{Fields: itemSpec.Fields},
				"SizeKanon/returns 0 for a nil receiver", "SizeKanon returns 0 for a nil receiver")
		})
	})
	t.Run("MarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that appends a byte to the encoding", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[marshalExtra]{Fields: itemSpec.Fields},
				"MarshalBinary/returns the reference encoding", "MarshalBinary returns the reference encoding")
		})
		t.Run("fails for a codec that drops the error of a value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[swallow]{Fields: codecsSpec.Fields},
				"MarshalBinary/returns the error of a value that fails to encode",
				"MarshalBinary returns the error of the value that fails to encode")
		})
		t.Run("fails for a codec without a golden file", func(t *testing.T) {
			t.Parallel()
			if golden.ShouldUpdate() {
				t.Skip("the -update flag writes the golden file that the check compares")
			}
			rejects(t, kanontest.Spec[unpinned]{Fields: itemSpec.Fields},
				"MarshalBinary/matches the golden file of the samples and the probes", "the golden file does not exist")
		})
	})
	t.Run("AppendBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that drops the bytes of its buffer", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[appendDrop]{Fields: itemSpec.Fields},
				"AppendBinary/appends the reference encoding to the buffer",
				"AppendBinary appends the reference encoding to the buffer")
		})
		t.Run("fails for a codec that appends a byte for a nil receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[nilAppendByte]{Fields: itemSpec.Fields},
				"AppendBinary/returns the buffer for a nil receiver",
				"AppendBinary returns the buffer for a nil receiver")
		})
	})
	t.Run("EncodeKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that writes into the start of the buffer", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[encodeFront]{Fields: itemSpec.Fields},
				"EncodeKanon/writes the reference encoding into the end of the buffer",
				"EncodeKanon writes the reference encoding into the end of the buffer")
		})
		t.Run("fails for a codec that returns no error for a short buffer", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[shortIgnore]{Fields: itemSpec.Fields},
				"EncodeKanon/returns io.ErrShortBuffer for a buffer shorter than the encoding",
				"EncodeKanon returns io.ErrShortBuffer for a short buffer")
		})
		t.Run("fails for a codec that writes a byte for a nil receiver", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[nilEncodeByte]{Fields: itemSpec.Fields},
				"EncodeKanon/writes nothing for a nil receiver", "EncodeKanon returns length 0 for a nil receiver")
		})
	})
}
