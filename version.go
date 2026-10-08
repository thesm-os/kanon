// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

// EnforceVersion checks at compile time that a generated file and the
// runtime it imports agree. A file that generator version 3 wrote declares
//
//	const (
//		_ = kanon.EnforceVersion(3 - kanon.MinVersion)
//		_ = kanon.EnforceVersion(kanon.MaxVersion - 3)
//	)
//
// A constant below zero does not convert to the unsigned EnforceVersion,
// so a runtime that no longer supports the file's version, or a file newer
// than the runtime, fails to compile.
type EnforceVersion uint

// The generator versions that the runtime supports, from MinVersion to
// MaxVersion.
const (
	// MinVersion is the oldest generator version whose files compile
	// against the runtime.
	MinVersion = 1
	// MaxVersion is the newest generator version whose files compile
	// against the runtime.
	MaxVersion = 3
)
