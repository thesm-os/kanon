// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package imports declares fixtures whose field types come from packages
// with the names of packages that the generated code imports, so that the
// generated code imports one of each pair under another name.
//
// # Dependency position
//
// The package depends on the fixture package math, and its generated code
// on the kanon runtime and the standard library's math.
package imports
