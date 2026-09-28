// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package external declares the types of another package that the fixtures
// use as field types: a struct with a kanon codec, a struct with a kanon
// codec whose encoding is empty, named basic types, an interface, a type
// that encodes itself through MarshalBinary alone, a type that appends its
// encoding, the discriminator of a union, and structs without a kanon
// codec, one of them with an unexported field, which the code files of the
// fixtures encode as inline structs.
//
// # Dependency position
//
// external imports encoding/binary, errors, fmt, math and strconv from the
// standard library, and its generated code the kanon runtime. The fixtures
// import it.
package external
