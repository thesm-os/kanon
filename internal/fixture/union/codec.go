// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Codecs

// CodecKind selects the member of the union of Codecs.
type CodecKind uint8

// The members of the union of Codecs.
const (
	CodecKindToken CodecKind = 1
	CodecKindWord  CodecKind = 2
	CodecKindStamp CodecKind = 3
	CodecKindCode  CodecKind = 4
)

// Codecs has a union with members of types that encode themselves. A
// selected member encodes even when its encoding is empty.
type Codecs struct {
	Kind  CodecKind
	Token codec.Token   `kanon:",union=Kind"`
	Word  codec.Word    `kanon:",union=Kind"`
	Stamp codec.Stamp   `kanon:",union=Kind"`
	Code  external.Code `kanon:",union=Kind"`
}
