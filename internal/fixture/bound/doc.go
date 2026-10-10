// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package bound declares fixtures with slice and map fields that have the
// tag option max. The generated tests run the checks of kanontest over each
// of them:
//
//   - [Page] is the canonical type of the test vectors of the bound.
//   - [List] accepts every encoding of a value. An unbounded field and the
//     elements of a nested slice share its bounded read functions, and it
//     has a bounded union member, a bounded map with pointer keys and an
//     inline struct with a bounded field.
//   - [Shelf] has a slice of List, and its encode fails when an element
//     fails at a bound.
//   - [Feed] has a streamed slice with a bound.
//   - [Tallies] and [TalliesWide] have a slice of [Tally] with the bounds 2
//     and 64. The encode method of Tally counts its calls per value. A test
//     of kanontest compares the counts of the checks of the two types.
//
// # Dependency position
//
// bound imports the standard library, and its generated code the kanon
// runtime.
package bound
