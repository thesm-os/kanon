// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/kanontest"
)

func TestIdent(t *testing.T) {
	t.Parallel()
	kanontest.RunExact[codec.Ident](t)
}
