// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/canonical"
	"go.thesmos.sh/kanon/kanontest"
)

func TestMode(t *testing.T) {
	t.Parallel()
	kanontest.RunStrict[canonical.Mode](t)
}
