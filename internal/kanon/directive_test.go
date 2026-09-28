// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// Directives of the directive tests. kanonA is on line 3 of a file, and
// kanonB on line 4 after it.
const (
	kanonA = "//go:generate go tool kanon -type=A\n"
	kanonB = "//go:generate go tool kanon -type=B\n"
	// structsAB declares the struct types A and B after the directives.
	structsAB = "\ntype A struct {\n\tX int32\n}\n\ntype B struct {\n\tY int32\n}\n"
)

// secondDirective is the error of a file whose kanon directive on line 4
// follows another on line 3.
const secondDirective = "kanon: a.go:4:1: kanon directive: the file has a second kanon directive: " +
	"name every type in one -type flag"

func TestDirective(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		commands := []struct {
			name    string
			command string
		}{
			{
				name:    "reads a directive that runs kanon with go run and a module version",
				command: "go run go.thesmos.sh/kanon/cmd/kanon@v1.0.0",
			},
			{name: "reads a directive that runs kanon by a path", command: "bin/kanon"},
			{name: "reads a directive that runs kanon by a Windows path", command: `C:\bin\kanon.exe`},
			{name: "reads a directive that runs kanon with the executable suffix in upper case", command: "kanon.EXE"},
		}
		for _, tt := range commands {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				src := "//go:generate " + tt.command + " -type=A\n" + kanonB + structsAB
				assert.Equal(t, generateError(t, map[string]string{source: src}), secondDirective,
					"Generate reads the first directive as a kanon directive")
			})
		}
		ignored := []struct {
			name      string
			directive string
		}{
			{name: "ignores a directive that runs another command", directive: "//go:generate stringer -type=A\n"},
			{
				name:      "ignores a directive of a command whose name only begins with kanon",
				directive: "//go:generate go tool kanonx -type=A\n",
			},
			{
				name:      "ignores a directive whose arguments do not parse",
				directive: "//go:generate go tool kanon \"-type=A\n",
			},
			{name: "ignores a directive of another tool", directive: "//go:noinline\n"},
		}
		for _, tt := range ignored {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := generate(
					t,
					module(t, map[string]string{source: tt.directive + kanonB + structsAB}),
					source,
					"B",
				)
				assert.NoError(t, err, "Generate reads the second directive as the only kanon directive")
			})
		}
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "returns an error for a second kanon directive in a file",
				files: map[string]string{source: kanonA + kanonB + structsAB},
				want:  secondDirective,
			},
			{
				name:  "returns an error for a directive whose flags do not parse",
				files: map[string]string{source: "//go:generate go tool kanon -bogus\n" + structsAB},
				want:  "kanon: a.go:3:1: kanon directive: flag provided but not defined: -bogus",
			},
			{
				name:  "returns an error for a type that the directives of two files name",
				files: map[string]string{source: kanonA + structsAB, "b.go": kanonA},
				want:  "kanon: b.go:3:1: kanon directive: -type names A, which the kanon directive in a.go names too",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the directive fails")
			})
		}
		t.Run("returns an error for a type that the directive of another file names", func(t *testing.T) {
			t.Parallel()
			_, err := generate(t, module(t, map[string]string{source: structsAB, "b.go": kanonA}), source, "A")
			assert.HasError(t, err, "Generate fails for the type")
			assert.Equal(t, err.Error(), "kanon: -type names A, which the kanon directive in b.go names too",
				"Generate states why the type fails")
		})
		t.Run("ignores a dependency directive whose flags do not parse", func(t *testing.T) {
			t.Parallel()
			files := map[string]string{
				"dep/dep.go": "//go:generate go tool kanon -bogus\n\n// T is a struct without a kanon codec.\n" +
					"type T struct {\n\tX int32\n}\n",
				source: "import \"example.com/m/dep\"\n\n" + structA("T dep.T"),
			}
			_, err := generate(t, module(t, files), source, "A")
			assert.NoError(t, err, "Generate encodes the struct of the dependency as an inline struct")
		})
	})
}
