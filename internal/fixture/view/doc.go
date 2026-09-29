// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package view declares fixtures whose directive passes -views, so that
// their code files declare a view type per struct: the encoding of a
// struct, with a method per field of a bool, a number, a string, a byte
// slice, a byte array, a time, a struct or a type that encodes itself, or
// of a pointer to one, which reads the field by scanning the encoding.
//
// # Dependency position
//
// view imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package view
