// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import "errors"

//go:generate go tool kanon -type=Level,Amount,Code,Ratio,Tags,Scores,Hash,Blob,Span -validate=valid

// LevelMax is the highest valid Level.
const LevelMax Level = 3

// ErrLevel is the error of valid for a Level above LevelMax.
var ErrLevel = errors.New("validate: level above 3")

// Level is one of four levels, from 0 to LevelMax, without methods of its
// own besides valid.
type Level uint8

// valid returns ErrLevel for a level above LevelMax.
func (l Level) valid() error {
	if l > LevelMax {
		return ErrLevel
	}
	return nil
}
