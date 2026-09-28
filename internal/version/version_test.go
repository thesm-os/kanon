// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package version_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/version"
)

// Build metadata that the cases render, and the version of a build without
// it.
const (
	tag    = "v1.2.3"
	commit = "abc123"
	date   = "2026-01-01"
	dev    = "dev"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	t.Run("Full", func(t *testing.T) {
		t.Parallel()
		t.Run("returns dev in a build that the linker did not stamp", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, version.Full(), dev, "a test binary has no stamped version")
		})
	})
	t.Run("Format", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			version string
			commit  string
			date    string
			want    string
		}{
			{name: "returns dev without a version", commit: commit, date: date, want: dev},
			{name: "returns the version alone without a commit", version: tag, date: date, want: tag},
			{
				name:    "returns the commit in parentheses without a date",
				version: tag,
				commit:  commit,
				want:    tag + " (" + commit + ")",
			},
			{
				name:    "returns the commit and the build date in parentheses",
				version: tag,
				commit:  commit,
				date:    date,
				want:    tag + " (" + commit + ", built " + date + ")",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, version.Format(tt.version, tt.commit, tt.date), tt.want,
					"Format renders the metadata that it gets")
			})
		}
	})
}
