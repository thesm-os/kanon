// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

// Names of the index type of a view, which follows the name of its -type
// struct, and of the method of the view that returns the index.
const (
	indexSuffix = "Index"
	indexKanon  = "IndexKanon"
)

func TestView(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the view types that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				return strings.HasSuffix(d.key, viewSuffix) || strings.HasSuffix(d.recv, viewSuffix)
			})
		})
		t.Run("writes the index types that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				return strings.HasSuffix(d.key, indexSuffix) || strings.HasSuffix(d.recv, indexSuffix)
			})
		})
		t.Run("writes an index of no offsets for a view type that reads no field", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{source: structA("X []int32")})
			files, err := kanon.Generate(dir, source, kanon.Options{Types: []string{"A"}, Views: true})
			assert.NoError(t, err, "Generate writes the view type")
			code := string(files[0].Src)
			assert.Contains(t, code, "at [0]int", "the index records no offset")
			var index string
			for _, d := range declarations(t, source, code) {
				if d.key == "AView."+indexKanon {
					index = d.text
				}
			}
			assert.Contains(t, index, "return ix, nil", "the code file declares IndexKanon")
			assert.NotContains(t, index, "switch", "IndexKanon reads no field number")
		})
	})
}
