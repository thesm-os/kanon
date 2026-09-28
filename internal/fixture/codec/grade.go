// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import "errors"

// Letters of the text of a Grade: GradeA is 'A', and GradeF is 'F'.
const (
	firstGradeLetter = 'A'
	lastGradeLetter  = 'F'
)

// GradeF is the last Grade, the sixth, whose letter is F.
const GradeF Grade = 6

// Errors of the methods of Grade.
var (
	// ErrGradeRange is the error of AppendText for a Grade after GradeF.
	ErrGradeRange = errors.New("codec: grade is after F")
	// ErrGradeText is the error of UnmarshalText for text that is neither
	// empty nor one letter from A to F.
	ErrGradeText = errors.New("codec: grade text is not a letter from A to F")
)

// Grade is a mark from A to F that encodes itself as its letter, through
// AppendText and UnmarshalText. The zero Grade is ungraded and encodes to
// no bytes.
type Grade uint8

// AppendText appends the letter of g to b, and nothing for the zero Grade.
// It fails with ErrGradeRange for a Grade after GradeF.
func (g Grade) AppendText(b []byte) ([]byte, error) {
	if g > GradeF {
		return b, ErrGradeRange
	}
	if g == 0 {
		return b, nil
	}
	return append(b, byte(g)-1+firstGradeLetter), nil
}

// UnmarshalText sets g to the Grade whose letter text is, and to the zero
// Grade for empty text. It fails with ErrGradeText for any other text.
func (g *Grade) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*g = 0
		return nil
	}
	if len(text) != 1 || text[0] < firstGradeLetter || text[0] > lastGradeLetter {
		return ErrGradeText
	}
	*g = Grade(text[0]-firstGradeLetter) + 1
	return nil
}
