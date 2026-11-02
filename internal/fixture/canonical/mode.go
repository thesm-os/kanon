// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import "errors"

// Names of the variants of Mode, which its text methods write and parse.
const (
	modeReadName  = "read"
	modeWriteName = "write"
)

// Errors of the methods of Mode.
var (
	// ErrModeValue is the error of MarshalText for a value outside the
	// variants.
	ErrModeValue = errors.New("canonical: the mode is not a variant")
	// ErrModeText is the error of UnmarshalText for a text that is not the
	// name of a variant.
	ErrModeText = errors.New("canonical: the text is not the name of a mode")
)

// Mode is an enumeration whose text methods write and parse the names of its
// variants. It declares kanon.Strict, since its decode method accepts only
// the names that its encode method writes. Its encode method fails for a
// value outside the variants, so Mode is not a kanon.Exact type. The zero
// Mode is not a variant, and UnmarshalText never sets it.
type Mode uint8

// The variants of Mode.
const (
	// ModeRead is the variant named read.
	ModeRead Mode = 1
	// ModeWrite is the variant named write.
	ModeWrite Mode = 2
)

// MarshalText returns the name of m. It fails with ErrModeValue for a value
// outside the variants.
func (m Mode) MarshalText() ([]byte, error) {
	switch m {
	case ModeRead:
		return []byte(modeReadName), nil
	case ModeWrite:
		return []byte(modeWriteName), nil
	default:
		return nil, ErrModeValue
	}
}

// UnmarshalText sets m to the variant that text names. It fails with
// ErrModeText for any other text.
func (m *Mode) UnmarshalText(text []byte) error {
	switch string(text) {
	case modeReadName:
		*m = ModeRead
	case modeWriteName:
		*m = ModeWrite
	default:
		return ErrModeText
	}
	return nil
}

// StrictKanon marks Mode as a kanon.Strict type.
func (Mode) StrictKanon() {}
