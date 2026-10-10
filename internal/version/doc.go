// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package version reports the version of the kanon command from the build information that the go
// command records in it. A build at a tag records the tag, such as v1.2.3, with the commit and the
// time of the commit, and go install of a version records the version. A build without a version,
// such as a test binary, reports "dev". The command needs no -X flag of the linker.
//
// # Dependency position
//
// version imports runtime/debug. The kanon command imports it for its -version flag.
package version
