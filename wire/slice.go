// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import "unsafe"

// firstAllocation is the most bytes that the first allocation of a slice of
// a decode takes: 10 MiB, the most that encoding/gob allocates for the first
// allocation of a slice and of a map.
const firstAllocation = 10485760

// SliceCap returns the capacity of the first allocation of a slice of n
// elements of T, for an n that is not negative: n when n elements take 10
// MiB at most, and otherwise the number of elements of T that fit in 10 MiB,
// and 1 at least. The decode of a slice makes the slice with this capacity
// for the elements that it counts in its input, and appends the elements past
// it, so that an input that counts many elements and then fails to decode
// them allocates 10 MiB at most.
//
// SliceCap allocates nothing. unsafe.Sizeof does not evaluate the value of
// T that it sizes, so a T larger than the stack allows takes no memory.
func SliceCap[T any](n int) int {
	size := int(unsafe.Sizeof(*new(T)))
	if size == 0 || n <= firstAllocation/size {
		return n
	}
	return max(firstAllocation/size, 1)
}
