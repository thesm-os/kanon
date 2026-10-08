// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package bound declares fixtures with slice and map fields that have the
// tag option max: the canonical type of the test vectors of the bound; a
// type that accepts every encoding of a value, whose bounded read functions
// an unbounded field and the elements of a nested slice share, with a
// bounded union member, a bounded map with pointer keys and an inline struct
// with a bounded field; a type with a slice of it, whose encode fails at the
// bound of an element; and a type with a streamed slice with a bound. The
// generated tests run the checks of kanontest over each of them.
//
// # Dependency position
//
// bound imports nothing, and its generated code the kanon runtime.
package bound
