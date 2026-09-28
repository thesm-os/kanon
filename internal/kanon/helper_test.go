// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
)

// ops lists the operations of the helpers of a code file.
var ops = []string{
	opSize, opPresent, opPut, opRead, opMerge, opFields, opDeselect, opReset, opClone, opCompare, opKeySize, opKeyPut,
	opKeyPresent, opNaN, opCanon,
}

func TestHelper(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("names every function of every code file after its file and its operation", func(t *testing.T) {
			t.Parallel()
			for _, g := range generations(t) {
				for _, f := range g.files {
					if !strings.HasSuffix(f.Name, codeSuffix) {
						continue
					}
					for _, d := range declarations(t, g.file, string(f.Src)) {
						if d.method != "" || !strings.HasPrefix(d.key, "_") {
							continue
						}
						assert.True(t, slices.Contains(ops, d.op), f.Name+": "+d.key+" has the prefix "+
							prefixOf(g.file)+" and an operation")
					}
				}
			}
		})
	})
}
