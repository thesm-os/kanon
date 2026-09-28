// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The slab and the offset of the struct that the Nested cases decode.
const (
	nestedSlab = "abc"
	nestedOff  = 2
)

func TestDepth(t *testing.T) {
	t.Parallel()
	t.Run("Nested", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name  string
			depth int
			want  int
		}{
			{name: "keeps a positive depth", depth: 3, want: 3},
			{name: "returns the limit 0 for depth 0", depth: 0, want: 0},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				opts := wire.Nested(nestedSlab, nestedOff, c.depth)
				assert.Equal(t, opts, kanon.Options{Slab: nestedSlab, Offset: nestedOff, Depth: opts.Depth},
					"Nested passes the slab and the offset")
				assert.Equal(t, opts.Limit(), c.want, "the nested decode enters depth levels below the struct")
			})
		}
	})
}
