// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package field

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has a field of each type that encodes itself. == compares every bit
// of each of these types, so a field encodes when it is not the zero value
// of its type.
type Codecs struct {
	Token  codec.Token
	Ticket codec.Ticket
	Grade  codec.Grade
	Word   codec.Word
	Stamp  codec.Stamp
	Code   external.Code
}
