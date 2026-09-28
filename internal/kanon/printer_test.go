// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

func TestPrinter(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the imports that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, goSuffix, func(d declaration) bool { return d.imports })
		})
		t.Run("imports a package under the smallest numeric suffix that no declaration takes", func(t *testing.T) {
			t.Parallel()
			src := "type wire int32\n\ntype wire2 int32\n\n" + structA("X wire", "Y wire2")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A")
			assert.NoError(t, err, "Generate names the import")
			assert.Contains(t, files[codeName], "wire3 \"go.thesmos.sh/kanon/wire\"",
				"the code file imports package wire as wire3")
		})
	})
}
