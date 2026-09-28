// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package inline declares fixtures with inline structs, the struct types
// without a kanon codec that a code file encodes with functions of its
// own, as their own codec would: anonymous structs, structs of this
// package and of another, instantiations of generic structs, structs that
// contain themselves, structs with a union, structs that keep unknown
// fields, and structs with fields that a decode tracks. The code file
// records their field numbers beside those of the -type structs.
//
// # Dependency position
//
// inline imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package inline
