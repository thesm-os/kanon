// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package version

import "runtime/debug"

// devVersion is the version of a build whose main module does not have a version.
const devVersion = "dev"

// develVersion is the version that the go command records for a main module that it builds
// without a version, such as a test binary.
const develVersion = "(devel)"

// The keys of the build settings that the go command records for a build of a git checkout.
const (
	revisionKey = "vcs.revision"
	timeKey     = "vcs.time"
)

// Full returns the version of the running build, as [From] reads the build information of
// [debug.ReadBuildInfo].
func Full() string {
	info, _ := debug.ReadBuildInfo()
	return From(info)
}

// From returns the version string of info, as [Format] renders it. It passes Format the version of
// the main module, the commit and the time of the commit. The go command writes the tag of a build
// at a tag, such as v1.2.3, with the commit and its time, and the version alone for go install of a
// version. From returns dev for a nil info and for a main module whose version is (devel).
func From(info *debug.BuildInfo) string {
	if info == nil || info.Main.Version == develVersion {
		return devVersion
	}
	var commit, date string
	for _, s := range info.Settings {
		switch s.Key {
		case revisionKey:
			commit = s.Value
		case timeKey:
			date = s.Value
		}
	}
	return Format(info.Main.Version, commit, date)
}

// Format renders build metadata as the kanon command reports it:
//
//   - "dev" when version is empty;
//   - version alone when commit is empty;
//   - "version (commit)" when date is empty;
//   - "version (commit, built date)" otherwise.
func Format(version, commit, date string) string {
	if version == "" {
		return devVersion
	}
	if commit == "" {
		return version
	}
	suffix := commit
	if date != "" {
		suffix += ", built " + date
	}
	return version + " (" + suffix + ")"
}
