// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inline

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Unions

// Choice is an inline struct of this package with a union, whose
// deselector the code file declares.
type Choice struct {
	Kind  ChoiceKind
	Text  string  `kanon:",union=Kind"`
	Items []int32 `kanon:",union=Kind"`
}

// ChoiceKind selects the member of the union of Choice.
type ChoiceKind uint8

// The members of the union of Choice.
const (
	ChoiceKindText  ChoiceKind = 1
	ChoiceKindItems ChoiceKind = 2
)

// Unions has inline structs with unions, of this package and of another,
// as a field and as elements.
type Unions struct {
	Choice   Choice
	Variant  external.Variant
	Variants []external.Variant
}
