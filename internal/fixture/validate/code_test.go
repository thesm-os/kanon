// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/validate"
)

// invalidUTF8 is a byte that begins no UTF-8 sequence.
const invalidUTF8 = 0xff

func TestCode(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalText", func(t *testing.T) {
		t.Parallel()
		t.Run("returns ErrCodeUTF8 for text that is not valid UTF-8", func(t *testing.T) {
			t.Parallel()
			c := validate.Code("a")
			assert.ErrorIs(t, c.UnmarshalText([]byte{invalidUTF8}), validate.ErrCodeUTF8,
				"UnmarshalText rejects the text")
			assert.Equal(t, c, validate.Code("a"), "UnmarshalText leaves the code unchanged")
		})
	})
}
