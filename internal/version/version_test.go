// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package version_test

import (
	"runtime/debug"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/version"
)

// Build metadata that the cases render, and the version of a build without it.
const (
	tag    = "v1.2.3"
	commit = "abc123"
	date   = "2026-01-01"
	dev    = "dev"
)

// The path of the main module of the cases, the keys of the build settings that the go command
// records for a build of a git checkout, and the version that it records for a main module without
// one.
const (
	modulePath  = "go.thesmos.sh/kanon"
	revisionKey = "vcs.revision"
	timeKey     = "vcs.time"
	devel       = "(devel)"
)

func TestVersion(t *testing.T) {
	t.Parallel()
	t.Run("Full", func(t *testing.T) {
		t.Parallel()
		t.Run("returns dev in a test binary, whose main module does not have a version", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, version.Full(), dev, "the version of the test binary")
		})
	})
	t.Run("From", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			version  string
			settings []debug.BuildSetting
			want     string
		}{
			{
				name:     "returns the tag with its commit and the time of the commit for a build at a tag",
				version:  tag,
				settings: []debug.BuildSetting{{Key: revisionKey, Value: commit}, {Key: timeKey, Value: date}},
				want:     tag + " (" + commit + ", built " + date + ")",
			},
			{name: "returns the version alone for go install of a version", version: tag, want: tag},
			{name: "returns dev for a main module whose version is (devel)", version: devel, want: dev},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				info := &debug.BuildInfo{
					Main:     debug.Module{Path: modulePath, Version: tt.version},
					Settings: tt.settings,
				}
				assert.Equal(t, version.From(info), tt.want, "the version of the build information")
			})
		}

		t.Run("returns dev for a binary without build information", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, version.From(nil), dev, "the version of no build information")
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
