// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/validate"
)

func TestTick(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns ErrTickLength for data that is not eight bytes long", func(t *testing.T) {
			t.Parallel()
			tick := validate.Tick(1)
			assert.ErrorIs(t, tick.UnmarshalBinary(make([]byte, 7)), validate.ErrTickLength,
				"UnmarshalBinary rejects the data")
			assert.Equal(t, tick, validate.Tick(1), "UnmarshalBinary leaves the tick unchanged")
		})
	})
}
