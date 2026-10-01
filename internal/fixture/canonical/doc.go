// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package canonical declares fixtures whose directives set the -canonical
// flag, so that their decode accepts only the canonical encoding of a value:
// the types of the test vectors of the wire format, and types with fields of
// every kind that a canonical decode checks. The generated tests run the
// suites of kanontest over them with canonical Specs, whose reference decode
// applies the same rules.
//
// Two types encode themselves as the vectors define them: [Digest], whose
// zero value has no encoding, and [Stamp], whose zero value encodes. [Count]
// encodes itself as its decimal text, and [Gauge] as the bits of a float,
// which == does not compare. The decode method of each also decodes bytes
// that its encode method does not write for the decoded value, which a
// canonical decode rejects, and each has a value without an encoding, so
// that an encode fails for it.
//
// # Dependency position
//
// canonical imports encoding/binary, errors, math, strconv and time from the
// standard library, the fixture packages codec, external and validate, and
// its generated code the kanon runtime.
package canonical
