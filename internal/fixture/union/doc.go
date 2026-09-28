// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package union declares fixtures with unions whose members have every Go
// type that kanon encodes, one member type family per file. A union member
// names its discriminator with the tag option union, and the constant
// <Type><Member> of the discriminator's type selects it. The encoding
// contains the selected member whatever its value, and no other member,
// and leaves the discriminator out. A nil pointer member encodes the zero
// value that it would point at.
//
// # Dependency position
//
// union imports math and time from the standard library and the fixture
// packages codec and external, and its generated code the kanon runtime.
package union
