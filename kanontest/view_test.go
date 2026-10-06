// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/validate"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
	"go.thesmos.sh/kanon/wire"
)

// valuesView describes validate.Values with its view type, whose methods of
// a string and a byte slice of a kanon.Validator pass their ValidateKanon.
var valuesView = kanontest.Spec[validate.Values]{Fields: valuesSpec.Fields, View: validate.ValuesView(nil)}

// Names of the checks of a view type and of its index type.
const (
	viewCheck  = "View/returns the value of each field as the reference view"
	indexCheck = "IndexKanon/returns the value of each field as the reference view"
)

// doubledIndex is an index of view.Item whose Count returns twice the
// count.
type doubledIndex struct{ view.ItemIndex }

// Count returns twice the Count of the index.
func (x doubledIndex) Count() (uint32, error) {
	n, err := x.ItemIndex.Count()
	return 2 * n, err
}

// doubledView is a view of view.Item whose index returns twice the count.
type doubledView []byte

// Name returns the Name of the encoding in v.
func (v doubledView) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns the Count of the encoding in v.
func (v doubledView) Count() (uint32, error) { return view.ItemView(v).Count() }

// IndexKanon returns the index of the encoding in v, whose Count doubles.
func (v doubledView) IndexKanon() (doubledIndex, error) {
	x, err := view.ItemView(v).IndexKanon()
	return doubledIndex{x}, err
}

// strayIndex is an index of view.Item with a method that names no field.
type strayIndex struct{ view.ItemIndex }

// Size returns 0.
func (strayIndex) Size() (int, error) { return 0, nil }

// strayView is a view of view.Item whose index has a method that names no
// field.
type strayView []byte

// Name returns the Name of the encoding in v.
func (v strayView) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns the Count of the encoding in v.
func (v strayView) Count() (uint32, error) { return view.ItemView(v).Count() }

// IndexKanon returns the index of the encoding in v, with the method Size.
func (v strayView) IndexKanon() (strayIndex, error) {
	x, err := view.ItemView(v).IndexKanon()
	return strayIndex{x}, err
}

// lenientView is a view of view.Item whose IndexKanon returns no error, and
// the zero index for an encoding that its scan rejects.
type lenientView []byte

// Name returns the Name of the encoding in v.
func (v lenientView) Name() ([]byte, error) { return view.ItemView(v).Name() }

// Count returns the Count of the encoding in v.
func (v lenientView) Count() (uint32, error) { return view.ItemView(v).Count() }

// IndexKanon returns the index of the encoding in v, without its error.
func (v lenientView) IndexKanon() (view.ItemIndex, error) {
	x, _ := view.ItemView(v).IndexKanon()
	return x, nil
}

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
		t.Run("passes the view type of a struct of kanon.Validator values", func(t *testing.T) {
			t.Parallel()
			holds(t, valuesView, viewCheck)
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
				"stray.Size names a field of Item")
		})
	})
	t.Run("IndexKanon", func(t *testing.T) {
		t.Parallel()
		t.Run("passes the index of a struct of every type that a view reads", func(t *testing.T) {
			t.Parallel()
			holds(t, kanontest.Spec[view.Record]{Fields: recordSpec.Fields, View: view.RecordView(nil)}, indexCheck)
		})
		t.Run("passes the index of a struct of kanon.Validator values", func(t *testing.T) {
			t.Parallel()
			holds(t, valuesView, indexCheck)
		})
		t.Run("fails for an index method that returns another value", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: doubledView(nil)}, indexCheck,
				"doubledIndex.Count returns the value of the reference decode")
		})
		t.Run("fails for an IndexKanon that returns another error", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: lenientView(nil)}, indexCheck,
				"lenientView.IndexKanon returns the error of the reference index")
		})
		t.Run("fails for an index method that names no field", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[view.Item]{Fields: itemSpec.Fields, View: strayView(nil)}, indexCheck,
				"strayIndex.Size names a field of Item")
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
