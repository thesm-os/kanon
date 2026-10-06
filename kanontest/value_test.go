// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of kanontest.ValueChecks.
const (
	valueTypeCheck     = "ValidateKanon/validates a named type that is not a struct"
	valueAllocsCheck   = "ValidateKanon/allocates nothing for a value that it accepts"
	valueDomainCheck   = "ValidateKanon/accepts a value exactly when the encode method of the type accepts it"
	valuePropertyCheck = "ValidateKanon/accepts a generated value exactly when the encode method of the type " +
		"accepts it"
	valueGoldenCheck = "ValidateKanon/matches the golden file of the values"
)

// errMinInt64 is the error of the encode methods of lax and strict, and of
// the ValidateKanon of strict, for math.MinInt64.
var errMinInt64 = errors.New("kanontest_test: the value is math.MinInt64")

// strict is an int64 whose ValidateKanon rejects math.MinInt64, the one value
// that its encode method rejects.
type strict int64

// ValidateKanon returns errMinInt64 for math.MinInt64, and nil otherwise.
func (x strict) ValidateKanon() error {
	if x == math.MinInt64 {
		return errMinInt64
	}
	return nil
}

// AppendBinary appends the eight big-endian bytes of x to b. It fails with
// the error of ValidateKanon for a value that ValidateKanon rejects.
func (x strict) AppendBinary(b []byte) ([]byte, error) {
	if err := x.ValidateKanon(); err != nil {
		return b, err
	}
	return binary.BigEndian.AppendUint64(b, uint64(x)), nil
}

// UnmarshalBinary sets x to the eight big-endian bytes in data.
func (x *strict) UnmarshalBinary(data []byte) error {
	*x = strict(binary.BigEndian.Uint64(data))
	return nil
}

// errUndecodable is the error of the decode method of undecodable.
var errUndecodable = errors.New("kanontest_test: the type decodes no encoding")

// lax is an int64 whose encode method rejects math.MinInt64, and whose
// ValidateKanon accepts every value.
type lax int64

// ValidateKanon returns nil.
func (lax) ValidateKanon() error { return nil }

// AppendBinary appends the eight big-endian bytes of x to b. It fails with
// errMinInt64 for math.MinInt64.
func (x lax) AppendBinary(b []byte) ([]byte, error) {
	if x == math.MinInt64 {
		return b, errMinInt64
	}
	return binary.BigEndian.AppendUint64(b, uint64(x)), nil
}

// UnmarshalBinary sets x to the eight big-endian bytes in data.
func (x *lax) UnmarshalBinary(data []byte) error {
	*x = lax(binary.BigEndian.Uint64(data))
	return nil
}

// undecodable is an int64 whose decode method rejects every encoding.
type undecodable int64

// ValidateKanon returns nil.
func (undecodable) ValidateKanon() error { return nil }

// AppendBinary appends the eight big-endian bytes of x to b.
func (x undecodable) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, uint64(x)), nil
}

// UnmarshalBinary returns errUndecodable.
func (*undecodable) UnmarshalBinary([]byte) error { return errUndecodable }

// lossy is an int64 whose decode method decodes every encoding to 0.
type lossy int64

// ValidateKanon returns nil.
func (lossy) ValidateKanon() error { return nil }

// AppendBinary appends the eight big-endian bytes of x to b.
func (x lossy) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, uint64(x)), nil
}

// UnmarshalBinary sets x to 0.
func (x *lossy) UnmarshalBinary([]byte) error {
	*x = 0
	return nil
}

// allocatingValue is a uint8 whose ValidateKanon allocates.
type allocatingValue uint8

// ValidateKanon returns nil, and allocates.
func (allocatingValue) ValidateKanon() error {
	escaped = make([]byte, 1)
	return nil
}

// unpinnedValue is a uint8 without a golden file.
type unpinnedValue uint8

// ValidateKanon returns nil.
func (unpinnedValue) ValidateKanon() error { return nil }

// structValue is a struct type with ValidateKanon, which kanon does not
// validate.
type structValue struct{ X int32 }

// ValidateKanon returns nil.
func (structValue) ValidateKanon() error { return nil }

// funcValue is a function type with ValidateKanon, which kanon does not
// encode.
type funcValue func()

// ValidateKanon returns nil.
func (funcValue) ValidateKanon() error { return nil }

// rejectsValue runs the check of kanontest.ValueChecks for T named name, and
// fails t unless the check fails with a first failure whose reason contains
// want.
func rejectsValue[T kanon.Validator](t *testing.T, name, want string) {
	t.Helper()
	rejectedFor(t, name, valueCheck[T](t, name).Run, want)
}

// valueCheck returns the check of kanontest.ValueChecks for T named name,
// and fails t when no check has that name.
func valueCheck[T kanon.Validator](t *testing.T, name string) kanontest.Check {
	t.Helper()
	for _, c := range kanontest.ValueChecks[T]() {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no check is named %q", name)
	return kanontest.Check{}
}

func TestValue(t *testing.T) {
	t.Parallel()
	t.Run("ValueChecks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns no domain check for a type without binary, gob or text methods", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.ValueChecks[unpinnedValue]()
			names := make([]string, 0, len(checks))
			for _, c := range checks {
				names = append(names, c.Name)
			}
			assert.Equal(t, names, []string{valueAllocsCheck, valueGoldenCheck},
				"ValueChecks returns the allocation check and the golden check")
		})
		t.Run("returns the domain check and the property for a type with binary methods", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.ValueChecks[strict]()
			names := make([]string, 0, len(checks))
			for _, c := range checks {
				names = append(names, c.Name)
			}
			assert.Equal(t, names, []string{valueAllocsCheck, valueDomainCheck, valuePropertyCheck, valueGoldenCheck},
				"ValueChecks returns the domain check and the property between the allocation and golden checks")
		})
		t.Run("returns one check for a struct type", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.ValueChecks[structValue]()
			assert.Length(t, checks, 1, "ValueChecks returns the check of the type alone")
			assert.Equal(t, checks[0].Name, valueTypeCheck, "ValueChecks names the check of the type")
		})
	})
	t.Run("ValidateKanon", func(t *testing.T) {
		t.Parallel()
		rejections := []struct {
			name  string
			check string
			run   func(t *testing.T, name, want string)
			want  string
		}{
			{
				name:  "fails for a struct type",
				check: valueTypeCheck,
				run:   rejectsValue[structValue],
				want:  "kanontest_test.structValue is a struct or an interface type, which kanon does not validate",
			},
			{
				name:  "fails for an interface type",
				check: valueTypeCheck,
				run:   rejectsValue[kanon.Validator],
				want:  "kanon.Validator is a struct or an interface type, which kanon does not validate",
			},
			{
				name:  "fails for a type that kanon does not encode",
				check: valueTypeCheck,
				run:   rejectsValue[funcValue],
				want:  "kanontest: kanontest_test.funcValue: ",
			},
			{
				name:  "fails for a method that accepts a value that the encode method of the type rejects",
				check: valueDomainCheck,
				run:   rejectsValue[lax],
				want:  "ValidateKanon accepts it exactly when the encode method of the type does",
			},
			{
				name:  "fails for a type whose decode method rejects the encoding of a value",
				check: valueDomainCheck,
				run:   rejectsValue[undecodable],
				want:  "the decode method of the type decodes its encoding",
			},
			{
				name:  "fails for a type whose decode method returns another value",
				check: valueDomainCheck,
				run:   rejectsValue[lossy],
				want:  "the decode method of the type returns the value",
			},
			{
				name:  "fails for a method that accepts a generated value that the encode method of the type rejects",
				check: valuePropertyCheck,
				run:   rejectsValue[lax],
				want:  "ValidateKanon accepts it exactly when the encode method of the type does",
			},
			{
				name:  "fails for a type whose decode method rejects the encoding of a generated value",
				check: valuePropertyCheck,
				run:   rejectsValue[undecodable],
				want:  "the decode method of the type decodes its encoding",
			},
			{
				name:  "fails for a type whose decode method returns another generated value",
				check: valuePropertyCheck,
				run:   rejectsValue[lossy],
				want:  "the decode method of the type returns the value",
			},
		}
		for _, tt := range rejections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tt.run(t, tt.check, tt.want)
			})
		}
		t.Run("fails for a type without a golden file", func(t *testing.T) {
			t.Parallel()
			if golden.ShouldUpdate() {
				t.Skip("the -update flag writes the golden file that the check compares")
			}
			missesGolden(t, valueGoldenCheck, valueCheck[unpinnedValue](t, valueGoldenCheck).Run)
		})
	})
}

// TestValueAllocs runs the check of ValueChecks that counts allocations, and
// RunValue, which runs it, while no parallel test runs, so that it and its
// subtests do not call t.Parallel.
func TestValueAllocs(t *testing.T) {
	t.Run("ValidateKanon", func(t *testing.T) {
		t.Run("fails for a method that allocates", func(t *testing.T) {
			countsAllocations(t)
			rejectsValue[allocatingValue](t, valueAllocsCheck, "ValidateKanon allocates nothing")
		})
	})
	t.Run("RunValue", func(t *testing.T) {
		t.Run("passes a type whose ValidateKanon rejects the value that its encode method rejects", func(t *testing.T) {
			kanontest.RunValue[strict](t)
		})
	})
}
