// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import "testing"

func TestSize(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the size methods and functions that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				return d.method == sizeKanon || d.op == opSize || d.op == opPresent
			})
		})
	})
}
