// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

import "time"

//go:generate go tool kanon -type=Keys

// Keys has maps with interface keys. The interface of a key stores the
// comparable types of its list alone, so that the slice of the list is a
// value of Keys alone, and keys sort by the number of their concrete type
// and then by value. A key whose types have one value each sorts by the
// number alone, and a key of one type of more values by that type. A time
// key in a zone that time.FixedZone allocates is no key that a decode
// yields, so that the decode of Moments collects such keys. The field Epoch
// uses package time, which the list of Moments names, and which a file
// imports for its tags only when its code uses it too.
type Keys struct {
	Keys    map[any]any     `kanon:",types=string|int32|[2]any|Circle|[]int32"`
	Shapes  map[Shape]int32 `kanon:",types=Circle|*Square"`
	Units   map[any]int32   `kanon:",types=struct{}|[0]int32"`
	Ints    map[any]int32   `kanon:",types=int32"`
	Moments map[any]int32   `kanon:",types=time.Time|int32"`
	Epoch   time.Time
}
