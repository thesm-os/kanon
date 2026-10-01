// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

// validateKanon is the name of the method of kanon.Validator that a code file
// declares on each named type that is not a struct.
const validateKanon = "ValidateKanon"

func TestValidator(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the ValidateKanon methods that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool { return d.method == validateKanon })
		})
		t.Run("writes a ValidateKanon that returns nil for a type without -validate", func(t *testing.T) {
			t.Parallel()
			files, err := generate(t, module(t, map[string]string{source: "type N int32\n"}), source, "N")
			assert.NoError(t, err, "Generate generates a named type that is not a struct")
			assert.Contains(t, files[codeName], "func (N) ValidateKanon() error {\n\treturn nil\n}\n",
				"the method accepts every value of N")
		})
		t.Run("encodes a struct with a field of a type without -validate without an error", func(t *testing.T) {
			t.Parallel()
			src := "type N int32\n\n" + structA("F N")
			files, err := generate(t, module(t, map[string]string{source: src}), source, "A,N")
			assert.NoError(t, err, "Generate generates the struct and the named type")
			appends := pickDeclarations(t, source, files[codeName], func(d declaration) bool {
				return d.key == "A."+appendBinary
			})
			assert.Contains(t, appends["A."+appendBinary], "\tm.encodeKanon(out)\n",
				"AppendBinary of A calls the encode of the struct without an error check")
		})
		t.Run("generates a field of a type without -validate whose only value is its zero value", func(t *testing.T) {
			t.Parallel()
			src := "type E [0]byte\n\n" + structA("E []E")
			_, err := generate(t, module(t, map[string]string{source: src}), source, "A,E")
			assert.NoError(t, err, "Generate generates the struct and the named type")
		})
		failures := []struct {
			name     string
			src      string
			types    string
			validate string
			want     string
		}{
			{
				name: "returns an error for -validate without a type that is not a struct",
				src:  structXY, types: "A", validate: "valid",
				want: "kanon: -validate names valid, and -type names no type that is not a struct",
			},
			{
				name: "returns an error for a type that declares ValidateKanon",
				src:  "type N int32\n\nfunc (N) ValidateKanon() error { return nil }\n", types: "N",
				want: "kanon: a.go:3:6: N: N declares ValidateKanon, which kanon generates",
			},
			{
				name: "returns an error for a type that declares ValidateKanon on a pointer receiver",
				src:  "type N int32\n\nfunc (*N) ValidateKanon() error { return nil }\n", types: "N",
				want: "kanon: a.go:3:6: N: N declares ValidateKanon, which kanon generates",
			},
			{
				name: "returns an error for a type without the method that -validate names",
				src:  "type N int32\n", types: "N", validate: "valid",
				want: "kanon: a.go:3:6: N: N has no method valid() error on a value receiver, which -validate names",
			},
			{
				name: "returns an error for a method that -validate names on a pointer receiver",
				src:  "type N int32\n\nfunc (*N) valid() error { return nil }\n", types: "N", validate: "valid",
				want: "kanon: a.go:3:6: N: N has no method valid() error on a value receiver, which -validate names",
			},
			{
				name: "returns an error for a method that -validate names with another signature",
				src:  "type N int32\n\nfunc (N) valid() bool { return true }\n", types: "N", validate: "valid",
				want: "kanon: a.go:3:6: N: N has no method valid() error on a value receiver, which -validate names",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := kanon.Generate(module(t, map[string]string{source: tt.src}), source,
					kanon.Options{Types: strings.Split(tt.types, ","), Validate: tt.validate})
				assert.HasError(t, err, "Generate fails")
				assert.Equal(t, err.Error(), tt.want, "Generate states why it fails")
			})
		}
	})
}
