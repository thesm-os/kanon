// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package pointer

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has a pointer field to each type that encodes itself. A pointer
// to a value whose encoding is empty is present.
type Codecs struct {
	Token  *codec.Token
	Ticket *codec.Ticket
	Grade  *codec.Grade
	Word   *codec.Word
	Stamp  *codec.Stamp
	Code   *external.Code
}
