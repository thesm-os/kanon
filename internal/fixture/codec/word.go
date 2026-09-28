// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// wordNul is the byte that the text of a Word cannot contain.
const wordNul = 0

// Errors of the methods of Word.
var (
	// ErrWordNul is the error of MarshalText for a text that contains a NUL
	// byte.
	ErrWordNul = errors.New("codec: word text contains a NUL byte")
	// ErrWordText is the error of UnmarshalText for text that is not valid
	// UTF-8.
	ErrWordText = errors.New("codec: word text is not valid UTF-8")
)

// Word is a struct that encodes itself as the bytes of its Text, through
// MarshalText and UnmarshalText, which a codec calls instead of encoding
// its field. A Word refers to memory, so that a copy of a Word goes through
// its methods. The text of a Word that is not valid UTF-8 encodes and does
// not decode.
type Word struct {
	Text string
}

// MarshalText returns the bytes of w.Text. It fails with ErrWordNul when
// the text contains a NUL byte.
func (w Word) MarshalText() ([]byte, error) {
	if strings.IndexByte(w.Text, wordNul) >= 0 {
		return nil, ErrWordNul
	}
	return []byte(w.Text), nil
}

// UnmarshalText sets w.Text to text. It fails with ErrWordText when text
// is not valid UTF-8.
func (w *Word) UnmarshalText(text []byte) error {
	if !utf8.Valid(text) {
		return ErrWordText
	}
	w.Text = string(text)
	return nil
}
