// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// defaultBuffer pins the buffer limit of the zero StreamOptions: 16 MiB.
const defaultBuffer = 16777216

func TestStreamOptions(t *testing.T) {
	t.Parallel()
	t.Run("DefaultBuffer", func(t *testing.T) {
		t.Parallel()
		t.Run("is 16 MiB", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.DefaultBuffer, defaultBuffer, "the default buffer limit is 16 MiB")
		})
	})
}
