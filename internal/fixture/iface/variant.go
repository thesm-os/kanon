// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

import (
	"time"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Variants

// Named types of this package, which the list of Variants names.
type (
	// Level is a named int32.
	Level int32
	// Name is a named string.
	Name string
	// Grid is a named array.
	Grid [2][2]int8
)

// Variants has interfaces whose concrete types are of every Go type that
// kanon encodes, one list per type family, and an interface of pointers
// alone, which stores a nil pointer of each of its types. A concrete type of
// a list encodes as it encodes as a value. The fields after the interfaces
// use the packages whose types the lists name, which a file imports for its
// tags only when its code uses them too.
type Variants struct {
	Numbers  any `kanon:",types=bool|int8|int|uint16|uint64|uintptr|float32|float64|complex64|complex128|Level"`
	Fixed    any `kanon:",fixed,types=int32|uint64"`
	Text     any `kanon:",types=string|[]byte|Name|[4]byte|[0]byte"`
	Times    any `kanon:",types=time.Time|time.Duration"`
	Remote   any `kanon:",types=external.Rank|external.Ident|external.Label"`
	Structs  any `kanon:",types=Circle|*Square|Patch|struct{}|external.Pair[string, int8]"`
	Codecs   any `kanon:",types=codec.Token|codec.Word|*codec.Stamp|external.Code"`
	Nested   any `kanon:",types=[]int32|map[string]int32|Grid|*int32|**int32|[]*Circle|[]*Square|map[string]*Square|[]Square|map[string]Square"`
	Pointers any `kanon:",types=*int32|**int32"`
	Deadline time.Time
	Token    codec.Token
	Rank     external.Rank
}
