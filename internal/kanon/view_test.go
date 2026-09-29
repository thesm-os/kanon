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
		t.Run("writes an IndexKanon without a scan for a view type that reads no field", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{source: structA("X []int32")})
			files, err := kanon.Generate(dir, source, kanon.Options{Types: []string{"A"}, Views: true})
			assert.NoError(t, err, "Generate writes the view type")
			var index, indexType string
			for _, d := range declarations(t, source, string(files[0].Src)) {
				switch d.key {
				case "AView." + indexKanon:
					index = d.text
				case "A" + indexSuffix:
					indexType = d.text
				}
			}
			assert.Contains(t, index, "return AIndex{v: v}, nil", "IndexKanon returns the index of the view")
			assert.NotContains(t, index, "Skip", "IndexKanon scans no field")
			assert.Contains(t, indexType, "v AView", "the code file declares the index type")
			assert.NotContains(t, indexType, "at [", "the index records no offset")
		})
	})
}
