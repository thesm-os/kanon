// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/kanontest"
)

func TestSpec(t *testing.T) {
	t.Parallel()
	t.Run("Keys", func(t *testing.T) {
		t.Parallel()
		t.Run("orders the map keys of a struct with a kanon codec by field number", func(t *testing.T) {
			t.Parallel()
			rejects(
				t,
				kanontest.Spec[mapkey.Arrays]{
					Fields: fields("Int32", "Point", "Float", "Time", "One", "Empty", "Void"),
				},
				"MarshalBinary/returns the reference encoding",
				"MarshalBinary returns the reference encoding",
			)
		})
	})
}
