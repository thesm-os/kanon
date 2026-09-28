// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/frame"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// newItem is the constructor of view.Item that the registry tests register.
func newItem() kanon.Message { return new(view.Item) }

func TestRegistry(t *testing.T) {
	t.Parallel()
	t.Run("Register", func(t *testing.T) {
		t.Parallel()
		t.Run("returns an error for a registered type ID", func(t *testing.T) {
			t.Parallel()
			var reg frame.Registry
			assert.NoError(t, reg.Register(typeID, newItem), "Register maps a new type ID")
			err := reg.Register(typeID, newItem)
			assert.Equal(t, err.Error(), "frame: type 1 is registered", "Register states the registered ID")
		})
		t.Run("returns an error for a nil constructor", func(t *testing.T) {
			t.Parallel()
			var reg frame.Registry
			err := reg.Register(typeID, nil)
			assert.Equal(t, err.Error(), "frame: type 1 has a nil constructor", "Register states the nil constructor")
			_, ok := reg.New(typeID)
			assert.False(t, ok, "the failed Register maps nothing")
		})
	})
	t.Run("New", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a new message of the registered type", func(t *testing.T) {
			t.Parallel()
			var reg frame.Registry
			assert.NoError(t, reg.Register(typeID, newItem), "Register maps the type ID")
			m, ok := reg.New(typeID)
			assert.True(t, ok, "New finds the type ID")
			_, isItem := m.(*view.Item)
			assert.True(t, isItem, "New returns a message of the registered type")
		})
		t.Run("reports false for a type ID that is not registered", func(t *testing.T) {
			t.Parallel()
			var reg frame.Registry
			m, ok := reg.New(typeID)
			assert.False(t, ok, "New finds no constructor")
			assert.Nil(t, m, "New returns no message")
		})
	})
}
