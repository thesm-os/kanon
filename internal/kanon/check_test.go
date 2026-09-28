// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

// Sources of the struct type A in the check tests.
const (
	// checkXY declares the fields X and Y.
	checkXY = "type A struct {\n\tX int32\n\tY int32\n}\n"
	// checkTagged declares the fields X and Y, and tags Y with the number 2.
	checkTagged = "type A struct {\n\tX int32\n\tY int32 `kanon:\"2\"`\n}\n"
	// checkX declares the field X alone.
	checkX = "type A struct {\n\tX int32\n}\n"
	// checkList declares a field S whose interface lists two concrete types.
	checkList = "type A struct {\n\tS any `kanon:\",types=int32|string\"`\n}\n"
)

// check returns the error of Check for the struct type A of the module of
// source, against the code files of base, which maps each name to its
// content after the package clause.
func check(t *testing.T, source string, base map[string]string) error {
	t.Helper()
	dir := module(t, map[string]string{"a.go": source})
	files := make(map[string][]byte, len(base))
	for name, content := range base {
		files[name] = []byte(pkgClause + content)
	}
	return kanon.Check(dir, "a.go", kanon.Options{Types: []string{"A"}}, files)
}

func TestCheck(t *testing.T) {
	t.Parallel()
	t.Run("Check", func(t *testing.T) {
		t.Parallel()
		passes := []struct {
			name   string
			source string
			base   map[string]string
		}{
			{
				name:   "returns nil for the numbers that the base revision records",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 Y=2\n"},
			},
			{
				name:   "returns nil for a struct that the base revision does not record",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers B Z=1\n"},
			},
			{
				name:   "returns nil for a tag that takes a reserved number",
				source: checkTagged,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 ~2\n"},
			},
			{
				name:   "returns nil for a tag that takes the number of a removed field",
				source: checkTagged,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 W=2\n"},
			},
			{
				name:   "returns nil for the numbers that another code file of the base revision records",
				source: checkXY,
				base:   map[string]string{"b.kanon.go": "//kanon:numbers A X=1 Y=2\n"},
			},
			{
				name:   "returns nil for the type numbers that the base revision records",
				source: checkList,
				base:   map[string]string{codeName: "//kanon:numbers A S=1\n//kanon:numbers A.S int32=1 string=2\n"},
			},
		}
		for _, tt := range passes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.NoError(t, check(t, tt.source, tt.base), "Check passes the numbers")
			})
		}
		failures := []struct {
			name   string
			source string
			base   map[string]string
			want   string
		}{
			{
				name:   "returns an error for each field with another number",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers A Y=1 X=2\n"},
				want: "kanon: field Y of A has number 2, and the base revision gives it number 1\n" +
					"kanon: field X of A has number 1, and the base revision gives it number 2",
			},
			{
				name:   "returns an error for a field that takes the number of a removed field",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 W=2\n"},
				want:   "kanon: field Y of A takes number 2, which the base revision gives the removed field W",
			},
			{
				name:   "returns an error for a field that takes a reserved number",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 ~2\n"},
				want:   "kanon: field Y of A takes number 2, which the base revision reserves",
			},
			{
				name:   "returns an error for a reserved number that is reserved no longer",
				source: checkX,
				base:   map[string]string{codeName: "//kanon:numbers A X=1 ~5\n"},
				want:   "kanon: A does not reserve number 5, which the base revision reserves",
			},
			{
				name:   "returns an error for a concrete type with another number",
				source: checkList,
				base:   map[string]string{codeName: "//kanon:numbers A S=1\n//kanon:numbers A.S int32=2 string=1\n"},
				want: "kanon: type int32 of A.S has number 1, and the base revision gives it number 2\n" +
					"kanon: type string of A.S has number 2, and the base revision gives it number 1",
			},
			{
				name:   "returns an error for a code file of the base revision that does not parse",
				source: checkXY,
				base:   map[string]string{codeName: "//kanon:numbers\n"},
				want:   "kanon: a numbers line names no struct",
			},
			{
				name:   "returns an error for two code files of the base revision that record different numbers",
				source: checkXY,
				base: map[string]string{
					"b.kanon.go": "//kanon:numbers A X=1 Y=2\n",
					"c.kanon.go": "//kanon:numbers A X=2 Y=1\n",
				},
				want: "kanon: b.kanon.go and c.kanon.go record different numbers for A: " +
					"regenerate the one that does not generate A",
			},
			{
				name:   "returns an error for two code files of the base revision that record different type numbers",
				source: checkList,
				base: map[string]string{
					"b.kanon.go": "//kanon:numbers A.S int32=1 string=2\n",
					"c.kanon.go": "//kanon:numbers A.S int32=2 string=1\n",
				},
				want: "kanon: b.kanon.go and c.kanon.go record different numbers for A.S: " +
					"regenerate the one that does not generate A.S",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := check(t, tt.source, tt.base)
				assert.HasError(t, err, "Check fails for the numbers")
				assert.Equal(t, err.Error(), tt.want, "Check states each broken rule")
			})
		}
		t.Run("returns nil for a number that the code file and the base revision reserve", func(t *testing.T) {
			t.Parallel()
			reserved := "//kanon:numbers A X=1 ~2\n"
			dir := module(t, map[string]string{"a.go": checkX, codeName: reserved})
			base := map[string][]byte{codeName: []byte(pkgClause + reserved)}
			assert.NoError(t, kanon.Check(dir, "a.go", kanon.Options{Types: []string{"A"}}, base),
				"Check passes the reserved number")
		})
		t.Run("returns the error of Generate for options without a struct type", func(t *testing.T) {
			t.Parallel()
			err := kanon.Check(module(t, map[string]string{"a.go": checkXY}), "a.go", kanon.Options{}, nil)
			assert.HasError(t, err, "Check fails without -type")
			assert.Equal(t, err.Error(), "kanon: -type is required: name the struct types to generate codecs for",
				"Check states why the generation fails")
		})
	})
}
