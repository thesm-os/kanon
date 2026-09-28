// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package version reports the build metadata that the linker stamps into
// the kanon command. A release build sets three variables:
// -ldflags "-X go.thesmos.sh/kanon/internal/version.buildVersion=v1.2.3"
// sets the version, and buildCommit and buildDate take the commit and its
// date the same way. A build without them reports "dev".
//
// # Dependency position
//
// version imports no package. The kanon command imports it for its
// -version flag.
package version
