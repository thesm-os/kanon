// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/bound"
)

// Values of Tally that no other test encodes, so that their counts change by
// the calls of one case alone.
const (
	countedTally bound.Tally = 0xc0ffee
	unusedTally  bound.Tally = 0xbadc0de
)

func TestTally(t *testing.T) {
	t.Parallel()
	t.Run("AppendBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("appends the four little-endian bytes of the number", func(t *testing.T) {
			t.Parallel()
			got, err := bound.Tally(0x04030201).AppendBinary([]byte{0xff})
			assert.NoError(t, err, "AppendBinary encodes every Tally")
			assert.Equal(t, got, []byte{0xff, 0x01, 0x02, 0x03, 0x04}, "AppendBinary appends the bytes to the buffer")
		})
		t.Run("returns ErrTallyVoid for TallyVoid", func(t *testing.T) {
			t.Parallel()
			got, err := bound.TallyVoid.AppendBinary([]byte{0xff})
			assert.ErrorIs(t, err, bound.ErrTallyVoid, "AppendBinary fails for TallyVoid")
			assert.Equal(t, got, []byte{0xff}, "AppendBinary returns the buffer unchanged")
		})
		t.Run("counts each call for its value", func(t *testing.T) {
			t.Parallel()
			before := bound.Tallied()[countedTally]
			_, _ = countedTally.AppendBinary(nil)
			_, _ = countedTally.AppendBinary(nil)
			assert.Equal(t, bound.Tallied()[countedTally], before+2, "Tallied counts both calls")
		})
	})
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		t.Run("decodes the number of four little-endian bytes", func(t *testing.T) {
			t.Parallel()
			var got bound.Tally
			assert.NoError(t, got.UnmarshalBinary([]byte{0x01, 0x02, 0x03, 0x04}), "UnmarshalBinary decodes four bytes")
			assert.Equal(t, got, bound.Tally(0x04030201), "UnmarshalBinary decodes the number of the bytes")
		})
		t.Run("returns ErrTallyLength for three bytes", func(t *testing.T) {
			t.Parallel()
			var got bound.Tally
			assert.ErrorIs(t, got.UnmarshalBinary([]byte{0x01, 0x02, 0x03}), bound.ErrTallyLength,
				"UnmarshalBinary returns ErrTallyLength for three bytes")
		})
	})
	t.Run("Tallied", func(t *testing.T) {
		t.Parallel()
		t.Run("returns a copy of the counts", func(t *testing.T) {
			t.Parallel()
			bound.Tallied()[unusedTally] = 1
			assert.Equal(t, bound.Tallied()[unusedTally], 0, "a change of the returned map leaves the counts unchanged")
		})
	})
}
