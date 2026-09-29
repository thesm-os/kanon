// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"errors"
	"unicode/utf8"
)

// ErrCodeUTF8 is the error of valid, AppendText and UnmarshalText for a Code
// that is not valid UTF-8.
var ErrCodeUTF8 = errors.New("validate: code is not valid UTF-8")

// Code is a string of valid UTF-8 with a text form of its own. Its
// ValidateKanon makes kanon encode it as a string instead.
type Code string

// valid returns ErrCodeUTF8 for a Code that is not valid UTF-8.
func (c Code) valid() error {
	if !utf8.ValidString(string(c)) {
		return ErrCodeUTF8
	}
	return nil
}

// AppendText appends the bytes of c to b. It fails with ErrCodeUTF8 for a
// Code that is not valid UTF-8.
func (c Code) AppendText(b []byte) ([]byte, error) {
	if err := c.valid(); err != nil {
		return b, err
	}
	return append(b, c...), nil
}

// UnmarshalText sets c to the text in data. It fails with ErrCodeUTF8 for
// text that is not valid UTF-8, and leaves c unchanged then.
func (c *Code) UnmarshalText(data []byte) error {
	v := Code(data)
	if err := v.valid(); err != nil {
		return err
	}
	*c = v
	return nil
}
