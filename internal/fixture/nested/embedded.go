// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package nested

import (
	"go.thesmos.sh/kanon/internal/fixture/codec"
	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Embedded

// Embedded embeds a struct by value, a struct of another package behind a
// pointer, and a type that encodes itself. An embedded field encodes under
// the name of its type.
type Embedded struct {
	Inner
	*external.Label
	codec.Token
	Note string
}
