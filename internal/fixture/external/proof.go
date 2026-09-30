// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

//go:generate go tool kanon -type=Proof -canonical

// Proof is a struct with a kanon codec whose directive sets -canonical, which
// a canonical struct type of another package can contain. Code is a type of
// this package that encodes itself, whose decode method also decodes a text
// with leading zeros.
type Proof struct {
	Signer string
	Sig    []byte
	Code   Code
}
