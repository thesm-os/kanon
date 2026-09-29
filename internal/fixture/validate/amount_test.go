// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"encoding/binary"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/validate"
)

// minAmountBits is the bit pattern of math.MinInt64 as a uint64.
const minAmountBits = 1 << 63

func TestAmount(t *testing.T) {
	t.Parallel()
	t.Run("UnmarshalBinary", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give []byte
			want error
		}{
			{
				name: "returns ErrAmountLength for data shorter than eight bytes",
				give: make([]byte, 7),
				want: validate.ErrAmountLength,
			},
			{
				name: "returns ErrAmountLength for data longer than eight bytes",
				give: make([]byte, 9),
				want: validate.ErrAmountLength,
			},
			{
				name: "returns ErrAmountRange for math.MinInt64",
				give: binary.BigEndian.AppendUint64(nil, minAmountBits),
				want: validate.ErrAmountRange,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				a := validate.Amount(1)
				assert.ErrorIs(t, a.UnmarshalBinary(tt.give), tt.want, "UnmarshalBinary rejects the data")
				assert.Equal(t, a, validate.Amount(1), "UnmarshalBinary leaves the amount unchanged")
			})
		}
	})
}
