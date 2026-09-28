// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package unknown declares fixtures that keep the fields that their decode
// does not know: a []byte field tagged unknown receives the bytes of every
// unknown field, tag included, and the encode writes them after the known
// fields, so that a struct of an older version rewrites a record of a newer
// one without losing fields.
//
// # Dependency position
//
// unknown imports its generated code's kanon runtime alone.
package unknown
