// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package batch_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/batch"
)

func TestError(t *testing.T) {
	t.Parallel()
	t.Run("Error", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name string
			err  error
			want string
		}{
			{name: "states an unsupported version", err: batch.ErrVersion, want: "batch: unsupported version"},
			{name: "states an unknown flag", err: batch.ErrFlags, want: "batch: unknown flag"},
			{
				name: "states an index that does not match the data", err: batch.ErrLayout,
				want: "batch: index does not match the data",
			},
			{
				name: "states a batch that does not fit its offsets", err: batch.ErrTooBig,
				want: "batch: the batch does not fit its offsets or an int",
			},
			{name: "states an append after Bytes", err: batch.ErrFinalized, want: "batch: append after Bytes"},
			{name: "states a nil message", err: batch.ErrNilMessage, want: "batch: nil message"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.err.Error(), tt.want, "the error states its cause after the package name")
			})
		}
	})
}
