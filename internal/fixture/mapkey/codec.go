// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has a map with keys of each type that encodes itself. The keys
// sort by their encodings, and two keys of Parity can have one encoding,
// which fails the encode of their map.
type Codecs struct {
	Token  map[codec.Token]string
	Ticket map[codec.Ticket]string
	Grade  map[codec.Grade]string
	Word   map[codec.Word]string
	Stamp  map[codec.Stamp]string
	Code   map[external.Code]string
	Parity map[codec.Parity]string
}
