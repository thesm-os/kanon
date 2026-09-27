// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// defaultDepth pins the nesting limit of the zero Options.
const defaultDepth = 100

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("Source", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a copy of data at offset 0 without a slab", func(t *testing.T) {
			t.Parallel()
			data := []byte("abc")
			slab, off := kanon.Options{Offset: 7}.Source(data)
			data[0] = 'x'
			assert.Equal(t, slab, "abc", "the slab is a copy that a later write to data does not change")
			assert.Equal(t, off, 0, "data starts the copy")
		})
		t.Run("returns Slab and Offset", func(t *testing.T) {
			t.Parallel()
			slab, off := kanon.Options{Slab: "xxabc", Offset: 2}.Source([]byte("abc"))
			assert.Equal(t, slab, "xxabc", "the slab is Slab")
			assert.Equal(t, off, 2, "the offset is Offset")
		})
	})
	t.Run("Limit", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			depth int
			want  int
		}{
			{name: "returns DefaultDepth for 0", depth: 0, want: defaultDepth},
			{name: "returns a positive Depth", depth: 3, want: 3},
			{name: "returns a negative Depth", depth: -1, want: -1},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, kanon.Options{Depth: c.depth}.Limit(), c.want, "Limit returns the nesting limit")
			})
		}
	})
	t.Run("DefaultDepth", func(t *testing.T) {
		t.Parallel()
		t.Run("is 100", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.DefaultDepth, defaultDepth, "the default nesting limit is 100 levels")
		})
	})
}
