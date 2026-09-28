// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

import "go.thesmos.sh/kanon/internal/fixture/codec"

//go:generate go tool kanon -type=Inner,Tracked

// Inner is a struct with a kanon codec of this package.
type Inner struct {
	Label string
}

// Holder is an inline struct with the fields that a decode tracks: a
// pointer, a struct, a map and an array of values that refer to memory,
// and fields of a struct with a kanon codec and of a type that encodes
// itself, so that its decode keeps a seen bitmap and its encode can fail.
type Holder struct {
	Ptr   *int64
	Inner Inner
	Index map[string]int32
	Lists [2][]int32
	Word  codec.Word
}

// Tracked has an inline struct with tracked fields, as a field and as the
// value of a pointer.
type Tracked struct {
	Holder  Holder
	Holders *Holder
}
