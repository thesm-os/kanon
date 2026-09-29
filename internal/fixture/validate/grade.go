// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import "errors"

// ErrGradeNegative is the error of ValidateKanon for a negative Grade.
var ErrGradeNegative = errors.New("validate: grade is negative")

// Grade is an int16 that is not negative, whose ValidateKanon is written
// by hand, so that kanon finds the method without a directive.
type Grade int16

// ValidateKanon returns ErrGradeNegative for a negative Grade.
func (g Grade) ValidateKanon() error {
	if g < 0 {
		return ErrGradeNegative
	}
	return nil
}
