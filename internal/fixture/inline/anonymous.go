// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

//go:generate go tool kanon -type=Anonymous

// Anonymous has anonymous structs in every place: two fields of one type,
// which share their functions and the numbers of the first field, the
// elements of a slice, the keys and values of maps, and a set, a map of
// empty structs.
type Anonymous struct {
	Meta struct {
		Host string
		Port uint16 `kanon:"4"`
	}
	Twin struct {
		Host string
		Port uint16 `kanon:"4"`
	}
	Rows  []struct{ A, B int32 }
	Cells map[struct{ Row, Col int8 }]struct{ Value string }
	Tags  map[string]struct{}
}
