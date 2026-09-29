// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package view

import (
	"time"

	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Item,Record -views

// Item is a struct with a view type, which the view of Record returns for
// its field Item.
type Item struct {
	Name  string
	Count uint32
}

// Level is a named int32.
type Level int32

// Record has a field of each type that a view reads: a bool, integers of
// every width and the fixed-size encoding, floats, complex numbers,
// strings, byte slices, byte arrays, a time, a struct with a view type and
// one without, types that encode themselves, one of which has no encoding of
// its zero value, and pointers to them. Its slice has no view method.
type Record struct {
	Flag     bool
	Int8     int8
	Level    Level
	Uint16   uint16
	Fixed    int64 `kanon:",fixed"`
	Float32  float32
	Float64  float64
	Complex  complex64
	Wave     complex128
	Text     string
	Blob     []byte
	Digest   [8]byte
	At       time.Time
	Item     Item
	Label    external.Label
	Ref      *int32
	ItemPtr  *Item
	Children []int32
	Token    codec.Token
	TokenPtr *codec.Token
	Seal     codec.Seal
}
