// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"errors"
	"unicode/utf8"
)

// clipLength is the length of the longest Clip whose SizeKanon is right.
const clipLength = 4

// ErrClipUTF8 is the error of UnmarshalText for text that is not valid
// UTF-8.
var ErrClipUTF8 = errors.New("codec: clip is not valid UTF-8")

// Clip is a string that encodes itself as its bytes, through AppendText and
// UnmarshalText. It is a kanon.Sizer whose SizeKanon counts 4 bytes at
// most, so that the encode of a longer Clip fails with kanon.ErrSize.
type Clip string

// AppendText appends the bytes of c to b.
func (c Clip) AppendText(b []byte) ([]byte, error) {
	return append(b, c...), nil
}

// UnmarshalText sets c to the text in data. It fails with ErrClipUTF8 for
// text that is not valid UTF-8.
func (c *Clip) UnmarshalText(data []byte) error {
	if !utf8.Valid(data) {
		return ErrClipUTF8
	}
	*c = Clip(data)
	return nil
}

// SizeKanon returns the length of c, and 4 for a longer Clip, whose
// encoding it does not count.
func (c Clip) SizeKanon() int {
	return min(len(c), clipLength)
}
