// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package validate declares the fixtures of kanon.Validator: named types
// that are not structs, which kanon encodes as their underlying types. The
// kanon directive of level.go generates the ValidateKanon method of each of
// [Level], a uint8 of four levels, [Amount], an int64 with binary methods
// of its own, [Code], a string with text methods, [Ratio], a float64 with
// gob methods, [Tags], a slice, [Scores], a map, [Hash], a byte array,
// [Blob], a byte slice, [Span], an array, and [Port], a uint16 whose zero
// value and dynamic range are not valid, each through its method valid.
// The directive of tick.go generates the method of [Tick], a uint64 with
// binary methods, and the directive of scalar.go the methods of [Flag], a
// bool, [Weight], a float32, [Wave], a complex64, and [Phase], a complex128,
// through their methods valid. Each of them rejects a value, so that the
// encode and the decode of every field of them fail for a value. [Grade]
// declares its ValidateKanon by hand.
// [Values] has fields of them in every position that a codec encodes:
// fields, a pointer, a slice, an array, the keys and values of a map, a
// union, an interface and the tag option fixed, with a view type. Its Flag
// is a union member, which the encode writes at false and at true.
//
// # Dependency position
//
// validate imports encoding/binary, errors, math, math/cmplx, slices,
// strings and unicode/utf8 from the standard library, and its generated
// code the kanon runtime.
package validate
