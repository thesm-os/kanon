// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package union

import (
	"math"

	"go.thesmos.sh/kanon/internal/fixture/external"
)

//go:generate go tool kanon -type=Signed,Remote

// SignedKind selects the member of the union of Signed. Its constants are
// the most negative and the largest value of its type.
type SignedKind int16

// The members of the union of Signed.
const (
	SignedKindLow  SignedKind = math.MinInt16
	SignedKindHigh SignedKind = math.MaxInt16
)

// Signed has a union whose discriminator is signed, and whose constants
// are the extreme values of its type.
type Signed struct {
	Kind SignedKind
	Low  string `kanon:",union=Kind"`
	High int64  `kanon:",union=Kind"`
}

// Remote has a union whose discriminator type and constants another
// package declares.
type Remote struct {
	Choice external.Choice
	Text   string `kanon:",union=Choice"`
	Number int64  `kanon:",union=Choice"`
}
