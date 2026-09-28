// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"errors"
	"testing"

	"go.dokimi.dev/assert"
)

func TestError(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("returns an error at a declaration that wraps the error of its cause", func(t *testing.T) {
			t.Parallel()
			_, err := generate(
				t,
				module(t, map[string]string{source: structA("X int32 `kanon:\"bogus\"`")}),
				source,
				"A",
			)
			assert.HasError(t, err, "Generate fails for the tag")
			cause := errors.Unwrap(err)
			assert.HasError(t, cause, "the error wraps its cause")
			assert.Equal(t, cause.Error(), "kanon: tag \"bogus\": unknown word \"bogus\"",
				"the cause has no position and no subject")
		})
	})
}
