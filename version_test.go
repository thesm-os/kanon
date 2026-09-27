// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// generatorVersion is the version of the generator that the runtime
// supports.
const generatorVersion = 1

// The constants a generated file of generatorVersion declares compile.
const (
	_ = kanon.EnforceVersion(generatorVersion - kanon.MinVersion)
	_ = kanon.EnforceVersion(kanon.MaxVersion - generatorVersion)
)

func TestEnforceVersion(t *testing.T) {
	t.Parallel()
	t.Run("MinVersion", func(t *testing.T) {
		t.Parallel()
		t.Run("is the generator version", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.MinVersion, generatorVersion, "the runtime supports generator version 1")
		})
	})
	t.Run("MaxVersion", func(t *testing.T) {
		t.Parallel()
		t.Run("is the generator version", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, kanon.MaxVersion, generatorVersion, "no newer generator version exists")
		})
	})
}
