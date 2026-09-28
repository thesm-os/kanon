// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"cmp"

	"go.thesmos.sh/kanon"
)

// Nested returns the options of the decode of a struct of another package,
// nested in a decode: slab, the offset off of the struct in it, and depth,
// the number of levels that the decode of the struct may enter below it,
// which is 0 or more. The decode of a struct without strings has an empty
// slab and nests only structs without strings, which count the offsets of
// their errors from off. A depth of 0 becomes a Depth of -1, which lets the
// decode of the struct enter no level below it, since a Depth of 0 selects
// kanon.DefaultDepth. The caller rejects a struct at a negative depth
// itself, with [DepthError].
func Nested(slab string, off, depth int) kanon.Options {
	return kanon.Options{Slab: slab, Offset: off, Depth: cmp.Or(depth, -1)}
}
