// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of kanontest.StrictChecks.
const (
	strictTypeCheck   = "StrictKanon/marks a type that encodes itself through the methods of a family"
	strictDecodeCheck = "StrictKanon/decodes only the bytes that the encode method writes for the decoded value"
)

// Names of the variants of a level, which its text methods write and parse.
const (
	lowName  = "low"
	highName = "high"
)

// Errors of the methods of the kanon.Strict types of these tests.
var (
	// errLevelValue is the error of the encode method of a level outside its
	// variants.
	errLevelValue = errors.New("kanontest_test: the level is not a variant")
	// errLevelText is the error of the decode method of a level for a text that
	// is not the name of a variant.
	errLevelText = errors.New("kanontest_test: the text is not the name of a level")
)

// level is an enumeration whose text methods write and parse the names of its
// variants, levelLow and levelHigh. It keeps the guarantee of kanon.Strict.
// Its encode method fails for a value outside its variants, and its zero
// value is not a variant.
type level uint8

// The variants of a level.
const (
	levelLow  level = 1
	levelHigh level = 2
)

// MarshalText returns the name of v. It fails with errLevelValue for a value
// outside the variants.
func (v level) MarshalText() ([]byte, error) {
	switch v {
	case levelLow:
		return []byte(lowName), nil
	case levelHigh:
		return []byte(highName), nil
	default:
		return nil, errLevelValue
	}
}

// UnmarshalText sets v to the variant that text names, as [parseLevel]
// parses it.
func (v *level) UnmarshalText(text []byte) error {
	x, err := parseLevel(string(text))
	*v = x
	return err
}

// StrictKanon marks level as a kanon.Strict type.
func (level) StrictKanon() {}

// loud is a level whose decode method also parses the names of the variants
// in capitals, which its encode method does not write, and which breaks the
// guarantee of kanon.Strict.
type loud uint8

// MarshalText returns the name of v, as a level writes it.
func (v loud) MarshalText() ([]byte, error) {
	return level(v).MarshalText()
}

// UnmarshalText sets v to the variant that text names in any case.
func (v *loud) UnmarshalText(text []byte) error {
	x, err := parseLevel(strings.ToLower(string(text)))
	*v = loud(x)
	return err
}

// StrictKanon marks loud as a kanon.Strict type.
func (loud) StrictKanon() {}

// bare is a type that declares kanon.Strict and has no methods of a family.
type bare uint8

// StrictKanon marks bare as a kanon.Strict type.
func (bare) StrictKanon() {}

// checked is a level that is a kanon.Validator, which kanon encodes as its
// underlying type and not through its text methods.
type checked uint8

// ValidateKanon returns nil.
func (checked) ValidateKanon() error { return nil }

// MarshalText returns the name of v, as a level writes it.
func (v checked) MarshalText() ([]byte, error) {
	return level(v).MarshalText()
}

// UnmarshalText sets v to the variant that text names, as [parseLevel]
// parses it.
func (v *checked) UnmarshalText(text []byte) error {
	x, err := parseLevel(string(text))
	*v = checked(x)
	return err
}

// StrictKanon marks checked as a kanon.Strict type.
func (checked) StrictKanon() {}

// hook is a function type that declares kanon.Strict, which kanon does not
// encode.
type hook func()

// StrictKanon marks hook as a kanon.Strict type.
func (hook) StrictKanon() {}

// rejectsStrict runs the check of kanontest.StrictChecks for T named name,
// and fails t unless the check fails with a first failure whose reason
// contains want.
func rejectsStrict[T kanon.Strict](t *testing.T, name, want string) {
	t.Helper()
	for _, c := range kanontest.StrictChecks[T]() {
		if c.Name == name {
			rejectedFor(t, name, c.Run, want)
			return
		}
	}
	t.Fatalf("no check is named %q", name)
}

func TestStrict(t *testing.T) {
	t.Parallel()
	t.Run("StrictChecks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the check of the guarantee for a type that encodes itself", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.StrictChecks[level]()
			assert.Length(t, checks, 1, "StrictChecks returns the check of the guarantee alone")
			assert.Equal(t, checks[0].Name, strictDecodeCheck, "StrictChecks names the check of the guarantee")
		})
		t.Run("returns one check for a type without the methods of a family", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.StrictChecks[bare]()
			assert.Length(t, checks, 1, "StrictChecks returns the check of the type alone")
			assert.Equal(t, checks[0].Name, strictTypeCheck, "StrictChecks names the check of the type")
		})
	})
	t.Run("StrictKanon", func(t *testing.T) {
		t.Parallel()
		rejections := []struct {
			name  string
			check string
			run   func(t *testing.T, name, want string)
			want  string
		}{
			{
				name:  "fails for a type without the methods of a family",
				check: strictTypeCheck,
				run:   rejectsStrict[bare],
				want: "kanontest_test.bare declares StrictKanon, and does not encode itself through binary, gob or " +
					"text methods",
			},
			{
				name:  "fails for a kanon.Validator",
				check: strictTypeCheck,
				run:   rejectsStrict[checked],
				want: "kanontest_test.checked declares StrictKanon, and does not encode itself through binary, gob " +
					"or text methods",
			},
			{
				name:  "fails for a type that kanon does not encode",
				check: strictTypeCheck,
				run:   rejectsStrict[hook],
				want:  "kanontest: kanontest_test.hook: ",
			},
			{
				name:  "fails for a decode method that accepts bytes that the encode method does not write",
				check: strictDecodeCheck,
				run:   rejectsStrict[loud],
				want: "input 4c6f77: the encode method writes the input for the value that the decode method " +
					"decodes from it",
			},
		}
		for _, tt := range rejections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tt.run(t, tt.check, tt.want)
			})
		}
	})
	t.Run("RunStrict", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a type that keeps the guarantee", func(t *testing.T) {
			t.Parallel()
			kanontest.RunStrict[level](t)
		})
	})
}

// parseLevel returns the variant of a level that name names, and
// errLevelText for any other name.
func parseLevel(name string) (level, error) {
	switch name {
	case lowName:
		return levelLow, nil
	case highName:
		return levelHigh, nil
	default:
		return 0, errLevelText
	}
}
