// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package number

//go:generate go tool kanon -type=Tags

// Tags pins field numbers at each boundary of the length of a tag, from one
// byte to five, and the largest number. The field without a tag takes the
// smallest free number.
type Tags struct {
	One      int32 `kanon:"1"`
	Auto     string
	Fifteen  int32 `kanon:"15"`
	Sixteen  int32 `kanon:"16"`
	Max2     int32 `kanon:"2047"`
	Min3     int32 `kanon:"2048"`
	Max3     int32 `kanon:"262143"`
	Min4     int32 `kanon:"262144"`
	Max4     int32 `kanon:"33554431"`
	Min5     int32 `kanon:"33554432"`
	Largest  int32 `kanon:"2147483647"`
	Pointer  *bool `kanon:"63"`
	Children []int32
}
