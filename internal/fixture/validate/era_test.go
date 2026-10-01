// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/validate"
)

func TestEra(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("returns ErrEraLength for data that is not eight bytes long", func(t *testing.T) {
			t.Parallel()
			era := validate.Era(1)
			assert.ErrorIs(t, era.UnmarshalBinary(make([]byte, 7)), validate.ErrEraLength,
				"UnmarshalBinary rejects the data")
			assert.Equal(t, era, validate.Era(1), "UnmarshalBinary leaves the era unchanged")
		})
	})
}
