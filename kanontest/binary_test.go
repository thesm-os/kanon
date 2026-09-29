// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"testing"

	"go.thesmos.sh/kanon/internal/fixture/array"
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/mapkey"
	"go.thesmos.sh/kanon/kanontest"
)

// sizerOff is a codec.Codecs whose MarshalBinary drops the error of the
// append method of codec.Hash, a kanon.Sizer, which fails for a Hash whose
// SizeKanon is 0 or above 8.
type sizerOff struct{ codec.Codecs }

// MarshalBinary returns the encoding, and no error for an error that wraps
// codec.ErrHashSize.
func (m *sizerOff) MarshalBinary() ([]byte, error) {
	b, err := m.Codecs.MarshalBinary()
	if errors.Is(err, codec.ErrHashSize) {
		return b, nil
	}
	return b, renamed(err, "Codecs", "sizerOff")
}

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
		t.Run("fails for a codec that drops the error of the append method of a kanon.Sizer", func(t *testing.T) {
			t.Parallel()
			rejects(t, kanontest.Spec[sizerOff]{Fields: codecsSpec.Fields}, marshalErrorCheck,
				"MarshalBinary returns the error of the value that fails to encode")
		})
	})
}
