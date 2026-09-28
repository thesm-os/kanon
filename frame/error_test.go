// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/frame"
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
			{name: "states an unsupported version", err: frame.ErrVersion, want: "frame: unsupported version"},
			{name: "states an unknown flag", err: frame.ErrFlags, want: "frame: unknown flag"},
			{name: "states a frame above the limit", err: frame.ErrTooLarge, want: "frame: longer than the limit"},
			{name: "states a checksum mismatch", err: frame.ErrChecksum, want: "frame: checksum mismatch"},
			{name: "states an unregistered type", err: frame.ErrUnknown, want: "frame: type not registered"},
			{
				name: "states a malformed length or frame", err: frame.ErrMalformed,
				want: "frame: malformed length or frame",
			},
			{name: "states the absence of a current frame", err: frame.ErrNoFrame, want: "frame: no current frame"},
			{name: "states a nil message", err: frame.ErrNilMessage, want: "frame: nil message"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, tt.err.Error(), tt.want, "the error states its cause after the package name")
			})
		}
	})
}
