// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package array

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// Codecs has an array of each type that encodes itself. An array is
// present when the encoding of an element has bytes.
type Codecs struct {
	Token  [2]codec.Token
	Ticket [2]codec.Ticket
	Grade  [2]codec.Grade
	Word   [2]codec.Word
	Stamp  [2]codec.Stamp
	Code   [2]external.Code
}
