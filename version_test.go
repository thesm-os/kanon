// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// The oldest and the newest generator version that the runtime supports.
const (
	oldestVersion = 1
	newestVersion = 4
)

// The constants that generated files of both versions declare compile.
const (
	_ = kanon.EnforceVersion(oldestVersion - kanon.MinVersion)
	_ = kanon.EnforceVersion(kanon.MaxVersion - oldestVersion)
	_ = kanon.EnforceVersion(newestVersion - kanon.MinVersion)
	_ = kanon.EnforceVersion(kanon.MaxVersion - newestVersion)
)

func TestEnforceVersion(t *testing.T) {
	t.Parallel()
	t.Run("MinVersion", func(t *testing.T) {
		t.Parallel()
		t.Run("is the oldest generator version", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.MinVersion, oldestVersion, "the runtime supports generator version 1")
		})
	})
	t.Run("MaxVersion", func(t *testing.T) {
		t.Parallel()
		t.Run("is the newest generator version", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.MaxVersion, newestVersion, "the runtime supports generator version 4")
		})
	})
}
