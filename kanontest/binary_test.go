// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/array"
	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/kanontest"
)

func TestBinary(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a field of each family of methods that encode a type", func(t *testing.T) {
			t.Parallel()
			passes(t, codecsSpec)
		})
		t.Run("passes arrays of each family of methods that encode a type", func(t *testing.T) {
			t.Parallel()
			passes(t, kanontest.Spec[array.Codecs]{Fields: fields("Token", "Ticket", "Grade", "Word", "Stamp", "Code")})
		})
		t.Run("passes map keys of each family of methods that encode a type", func(t *testing.T) {
			t.Parallel()
			spec := kanontest.Spec[mapkey.Codecs]{
				Fields: fields("Token", "Ticket", "Grade", "Word", "Stamp", "Code", "Parity"),
			}
			passes(t, spec)
		})
	})
}
