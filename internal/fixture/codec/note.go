// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"errors"
	"unicode/utf8"
)

// ErrNoteText is the error of AppendText and UnmarshalText for text that is
// not valid UTF-8.
var ErrNoteText = errors.New("codec: note text is not valid UTF-8")

// Note is a string that encodes itself as its bytes, through AppendText and
// UnmarshalText, so that its encoding is as long as the string. A codec
// appends the encoding to a stack array, and a Note longer than the array
// allocates.
type Note string

// AppendText appends the bytes of n to b. It fails with ErrNoteText when n
// is not valid UTF-8.
func (n Note) AppendText(b []byte) ([]byte, error) {
	if !utf8.ValidString(string(n)) {
		return b, ErrNoteText
	}
	return append(b, n...), nil
}

// UnmarshalText sets n to text. It fails with ErrNoteText when text is not
// valid UTF-8.
func (n *Note) UnmarshalText(text []byte) error {
	if !utf8.Valid(text) {
		return ErrNoteText
	}
	*n = Note(text)
	return nil
}
