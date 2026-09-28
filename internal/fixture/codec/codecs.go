// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

//go:generate go tool kanon -type=Codecs,Appenders

// Codecs has a field of each type of the package, so that its generated
// tests call every method of those types.
type Codecs struct {
	Token  Token
	Ticket Ticket
	Grade  Grade
	Word   Word
	Stamp  Stamp
	Note   Note
	Parity Parity
}

// Appenders has a field of each type of the package with an append method,
// which a codec calls with a stack array, so that the allocation checks of
// its generated tests measure the encode of every such type.
type Appenders struct {
	Token Token
	Grade Grade
	Stamp Stamp
	Note  Note
}
