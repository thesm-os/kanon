// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strings"
	"testing"
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
	})
}
