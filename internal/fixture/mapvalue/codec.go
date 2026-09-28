// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapvalue

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has a map with values of each type that encodes itself.
type Codecs struct {
	Token  map[string]codec.Token
	Ticket map[string]codec.Ticket
	Grade  map[string]codec.Grade
	Word   map[string]codec.Word
	Stamp  map[string]codec.Stamp
	Code   map[string]external.Code
}
