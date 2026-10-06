// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// propertyCheck names the check of kanontest.Checks that runs the property
// of generated values.
const propertyCheck = "Message/encodes and decodes generated values as the reference"

// rejecting returns the case that fails t unless the property of spec fails
// with a message that contains want, as rejects checks it.
func rejecting[T any, P kanontest.Codec[T]](spec kanontest.Spec[T]) func(t *testing.T, want string) {
	return func(t *testing.T, want string) {
		t.Helper()
		rejects[T, P](t, spec, propertyCheck, want)
	}
}

func TestProperty(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes the codec of a struct with a view type and an index", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[view.Record]{Fields: recordSpec.Fields, View: view.RecordView(nil)}, propertyCheck)
		})
		// Each wrapper differs from the codec that it embeds in one method, and
		// the property meets that difference before any other one. Most differ
		// on the first case, whose values are all zero. A wrapper whose
		// difference needs a decode error names its type in that error as the
		// reference does.
		rejections := []struct {
			name string
			run  func(t *testing.T, want string)
			want string
		}{
			{
				name: "fails for a codec that counts one byte too many",
				run:  rejecting(kanontest.Spec[sizeOff]{Fields: itemSpec.Fields}),
				want: "SizeKanon returns the length of the reference encoding",
			},
			{
				name: "fails for a codec that appends a byte to the encoding",
				run:  rejecting(kanontest.Spec[marshalExtra]{Fields: itemSpec.Fields}),
				want: marshalCheck,
			},
			{
				name: "fails for a codec that drops the bytes of its buffer",
				run:  rejecting(kanontest.Spec[appendDrop]{Fields: itemSpec.Fields}),
				want: "AppendBinary appends the reference encoding to the buffer",
			},
			{
				name: "fails for a codec whose UnmarshalBinary decodes a field to another value",
				run:  rejecting(kanontest.Spec[unmarshalOff]{Fields: itemSpec.Fields}),
				want: "UnmarshalBinary decodes the value of the reference decode",
			},
			{
				name: "fails for a codec that decodes a field to another value",
				run:  rejecting(kanontest.Spec[countOff]{Fields: itemSpec.Fields}),
				want: "decodes the value of the reference decode",
			},
			{
				name: "fails for a codec that reports another offset for an error",
				run:  rejecting(kanontest.Spec[offsetOff]{Fields: itemSpec.Fields}),
				want: errorCheck,
			},
			{
				name: "fails for a codec that decodes a cut encoding",
				run:  rejecting(kanontest.Spec[eofOff]{Fields: itemSpec.Fields}),
				want: errorCheck,
			},
			{
				name: "fails for a codec whose DecodeKanon keeps the fields that the encoding leaves out",
				run:  rejecting(kanontest.Spec[mergeDecode]{Fields: itemSpec.Fields}),
				want: "DecodeKanon into a receiver that decoded",
			},
			{
				name: "fails for a codec whose DecodeKanon keeps a field that the encoding leaves out",
				run:  rejecting(kanontest.Spec[keepsMemo]{Fields: skippedFields}),
				want: "clears the fields that the encoding leaves out",
			},
			{
				name: "fails for a codec whose CloneKanon copies a field to another value",
				run:  rejecting(kanontest.Spec[cloneCount]{Fields: itemSpec.Fields}),
				want: "CloneKanon returns a copy of the receiver",
			},
			{
				name: "fails for a view method that returns another value",
				run:  rejecting(kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: doubled(nil)}),
				want: "doubled.Count returns the value of the reference decode",
			},
			{
				name: "fails for an IndexKanon that returns another error",
				run:  rejecting(kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: lenientView(nil)}),
				want: "lenientView.IndexKanon returns the error of the reference index",
			},
		}
		for _, tt := range rejections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tt.run(t, tt.want)
			})
		}
	})
}
