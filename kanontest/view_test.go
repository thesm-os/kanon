// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
	"go.thesmos.sh/kanon/wire"
)

// viewCheck is the name of the check of a view type.
const viewCheck = "View/returns the value of each field as the reference view"

// nameless is a view of view.Item whose Name returns no bytes.
type nameless []byte

// Name returns no bytes and no error.
func (nameless) Name() ([]byte, error) { return nil, nil }

// Count returns the Count of the encoding in v.
func (v nameless) Count() (uint32, error) { return view.ItemView(v).Count() }

// doubled is a view of view.Item whose Count returns twice the count, which
// differs from the count for every count but 0.
type doubled []byte

// Name returns the Name of the encoding in v.
func (v doubled) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns twice the Count of the encoding in v.
func (v doubled) Count() (uint32, error) {
	n, err := view.ItemView(v).Count()
	return 2 * n, err
}

// unchecked is a view of view.Item whose Count returns the low 32 bits of a
// varint of any range.
type unchecked []byte

// Name returns the Name of the encoding in v.
func (v unchecked) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns the low 32 bits of the varint of the last Count of the
// encoding in v, without a range check.
func (v unchecked) Count() (uint32, error) {
	i, err := wire.Find(v, 2<<3|wire.Varint, "Item.Count")
	if err != nil || i < 0 {
		return 0, err
	}
	u, _ := wire.Uvarint(v[i:])
	return uint32(u), nil
}

// lastItem is a view of view.Record whose Item returns the last occurrence
// of the field, where the view of a struct fails for a second occurrence.
type lastItem []byte

// Item returns the bytes of the last Item of the encoding in v.
func (v lastItem) Item() ([]byte, error) {
	i, err := wire.Find(v, 14<<3|wire.Bytes, "Record.Item")
	if err != nil || i < 0 {
		return nil, err
	}
	l, n := wire.Uvarint(v[i:])
	return v[i+n : i+n+int(l)], nil
}

// stray is a view of view.Item with a method that names no field.
type stray []byte

// Name returns the Name of the encoding in v.
func (v stray) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns the Count of the encoding in v.
func (v stray) Count() (uint32, error) { return view.ItemView(v).Count() }

// Size returns the length of the encoding in v.
func (v stray) Size() (int, error) { return len(v), nil }

func TestView(t *testing.T) {
	t.Parallel()
	t.Run("View", func(t *testing.T) {
		t.Parallel()
		t.Run("passes the view type of a struct of every type that a view reads", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[view.Record]{Fields: recordSpec.Fields, View: view.RecordView(nil)}, viewCheck)
		})
		t.Run("passes the view type of a nested struct", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: view.ItemView(nil)}, viewCheck)
		})
		t.Run("fails for a view method that returns another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: nameless(nil)}, viewCheck,
				"nameless.Name returns the value of the reference decode")
		})
		t.Run("fails for a view method of a number that returns another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: doubled(nil)}, viewCheck,
				"doubled.Count returns the value of the reference decode")
		})
		t.Run("fails for a view method that returns another error", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: unchecked(nil)}, viewCheck,
				"unchecked.Count returns the error of the reference decode")
		})
		t.Run("fails for a view method of a struct that returns the last of two occurrences", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Record]{Fields: recordSpec.Fields, View: lastItem(nil)}, viewCheck,
				"lastItem.Item returns the error of the reference decode")
		})
		t.Run("fails for a view method that names no field", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: stray(nil)}, viewCheck,
				"has the method Size, which names no field of Item")
		})
	})
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the check of the Spec for a view type that is not a byte slice", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: 0},
				"Spec/describes the fields of the struct type", "view type int, which is not a byte slice")
		})
	})
}
