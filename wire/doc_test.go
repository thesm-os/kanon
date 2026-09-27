// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"fmt"

	"go.thesmos.sh/kanon/wire"
)

// Example writes field 1 with the string "hi" backward, as generated code
// writes it, and reads it forward.
func Example() {
	buf := make([]byte, 4)
	i := wire.PutRaw(buf, len(buf), "hi")
	i = wire.PutUvarint(buf, i, 2)
	i = wire.PutTag(buf, i, 1<<3|wire.Bytes)
	fmt.Printf("% x\n", buf[i:])

	tag, n := wire.Uvarint(buf[i:])
	l, m := wire.Uvarint(buf[i+n:])
	start := i + n + m
	fmt.Println(tag>>3, tag&7 == wire.Bytes, string(buf[start:start+int(l)]))
	// Output:
	// 0a 02 68 69
	// 1 true hi
}
