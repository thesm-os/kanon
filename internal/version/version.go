// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package version

// devVersion is the version of a build that the linker did not stamp.
const devVersion = "dev"

// buildVersion is the release tag that the linker stamps, such as "v1.2.3".
// It is empty in a build that the linker did not stamp.
var buildVersion string

// buildCommit is the git commit that the linker stamps. It may be empty.
var buildCommit string

// buildDate is the date of the commit that the linker stamps. It may be
// empty.
var buildDate string

// Full returns the version of the running build: the metadata that the
// linker stamped, as [Format] renders it.
func Full() string {
	return Format(buildVersion, buildCommit, buildDate)
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
