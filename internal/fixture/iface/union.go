// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

//go:generate go tool kanon -type=Unions

// UnionKind selects the member of the union of Unions.
type UnionKind uint8

// The members of the union of Unions.
const (
	UnionKindShape UnionKind = 1
	UnionKindNote  UnionKind = 2
)

// Unions has a union with an interface member. A selected nil interface
// encodes as the type number 0.
type Unions struct {
	Kind  UnionKind
	Shape Shape  `kanon:",union=Kind,types=Circle|*Square"`
	Note  string `kanon:",union=Kind"`
}
