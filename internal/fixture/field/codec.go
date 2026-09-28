// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has a field of each type that encodes itself. A field encodes when
// its encoding has bytes.
type Codecs struct {
	Token  codec.Token
	Ticket codec.Ticket
	Grade  codec.Grade
	Word   codec.Word
	Stamp  codec.Stamp
	Code   external.Code
}
