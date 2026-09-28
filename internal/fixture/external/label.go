// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

//go:generate go tool kanon -type=Label

// Rank is a named int32 of another package.
type Rank int32

// Ident is a named string of another package.
type Ident string

// Label is a struct of another package with a kanon codec, which the
// fixtures nest through its exported methods. Its Mark can fail to encode,
// so that the encoding of a Label can fail, and appends its encoding, so
// that the encode of a Label allocates nothing.
type Label struct {
	Key   Ident
	Value string
	Rank  Rank
	Mark  Mark
}

// Tagged is an interface of another package, which Label implements.
type Tagged interface {
	tagged()
}

// tagged marks Label as a Tagged.
func (Label) tagged() {}
