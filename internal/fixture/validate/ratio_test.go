// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate_test

import (
	"encoding/binary"
	"math"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/fixture/validate"
)

func TestRatio(t *testing.T) {
	t.Parallel()
	t.Run("GobDecode", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			give []byte
			want error
		}{
			{
				name: "returns ErrRatioLength for data that is not eight bytes long",
				give: make([]byte, 9),
				want: validate.ErrRatioLength,
			},
			{
				name: "returns ErrRatioNaN for a NaN",
				give: binary.LittleEndian.AppendUint64(nil, math.Float64bits(math.NaN())),
				want: validate.ErrRatioNaN,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				r := validate.Ratio(0.5)
				assert.ErrorIs(t, r.GobDecode(tt.give), tt.want, "GobDecode rejects the data")
				assert.Equal(t, r, validate.Ratio(0.5), "GobDecode leaves the ratio unchanged")
			})
		}
	})
}
