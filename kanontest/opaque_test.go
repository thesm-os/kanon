// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/kanontest"
)

// sumDrop is a codec.Codecs whose MarshalBinary leaves out its Sum, whose
// fields are unexported, so that only a sample with a Sum from its decode
// method tells its encoding from the reference encoding.
type sumDrop struct{ codec.Codecs }

// MarshalBinary returns the encoding of m with the zero Sum.
func (m *sumDrop) MarshalBinary() ([]byte, error) {
	c := m.Codecs
	c.Sum = codec.Sum{}
	return c.MarshalBinary()
}

func TestOpaque(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("fails for a codec that leaves out a field of a type whose fields are unexported", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[sumDrop]{Fields: codecsSpec.Fields},
				"MarshalBinary/returns the reference encoding", "MarshalBinary returns the reference encoding")
		})
	})
}
