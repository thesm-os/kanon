// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

//go:generate go tool kanon -type=Places

// Places has an interface in every place that a value can be: a field, the
// element of a slice and of an array, the value of a map, the value that a
// pointer points at, a pointer to a pointer, and the field of an anonymous
// struct. A value of a type that the list does not name fails to encode,
// and a type number that the list does not name fails to decode.
type Places struct {
	Field  Shape            `kanon:",types=Circle|*Square|Patch"`
	Slice  []Shape          `kanon:",types=Circle|*Square"`
	Array  [2]Shape         `kanon:",types=Circle|*Patch"`
	Map    map[string]Shape `kanon:",types=Circle|*Square"`
	Ptr    *Shape           `kanon:",types=*Square"`
	Twice  **Shape          `kanon:",types=Circle"`
	Nested struct {
		Shape Shape `kanon:",types=Circle"`
	}
}
