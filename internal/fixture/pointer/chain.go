// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

//go:generate go tool kanon -type=Chains

// Loop is a named pointer type that points at itself, so that a chain of
// Loops is as long as its value.
type Loop *Loop

// Chains has chains of pointers: a pointer to a pointer to a number, to a
// struct and to a slice, a pointer of three levels, and a Loop. The inner
// pointers have a presence byte, and the length of their encoding precedes
// it.
type Chains struct {
	Twice      **int32
	TwiceInner **Inner
	TwiceSlice **[]int32
	Thrice     ***string
	Loop       Loop
}
