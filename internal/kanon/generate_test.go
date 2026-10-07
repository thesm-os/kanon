// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"

	"go.thesmos.sh/kanon/internal/kanon"
)

// Locations of the fixture packages, whose generated files are the golden
// files of the generator.
const (
	// fixtureDir is the directory of the fixture packages.
	fixtureDir = "../fixture"
	// testdataName names the directory of a fixture package that the walk
	// of the fixture files skips.
	testdataName = "testdata"
	// directive begins the kanon directive of a fixture file.
	directive = "//go:generate go tool kanon "
	// goSuffix ends the name of a Go file.
	goSuffix = ".go"
)

// generation is one run of the generator: the output of Generate for the
// kanon directive of one fixture file.
type generation struct {
	// dir is the directory of the package of the file, and file its base
	// name.
	dir, file string
	opts      kanon.Options
	files     []kanon.File
	err       error
}

// fixtureRuns runs the generator on the kanon directive of every fixture
// file, once per test binary, in the order of a walk of fixtureDir. It
// fails when a file does not read or a directive does not parse. The runs
// of Generate record their errors.
var fixtureRuns = sync.OnceValues(func() ([]generation, error) {
	var out []generation
	err := filepath.WalkDir(fixtureDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == testdataName {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, goSuffix) {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for line := range strings.Lines(string(src)) {
			args, ok := strings.CutPrefix(strings.TrimSpace(line), directive)
			if !ok {
				continue
			}
			opts, err := kanon.ParseOptions(strings.Fields(args), io.Discard)
			if err != nil {
				return err
			}
			dir, file := filepath.Split(path)
			files, err := kanon.Generate(dir, file, opts)
			out = append(out, generation{dir: dir, file: file, opts: opts, files: files, err: err})
		}
		return nil
	})
	return out, err
})

// generations returns the runs of the generator on the fixture files, and
// fails t when they do not run or one of them fails.
func generations(t *testing.T) []generation {
	t.Helper()
	runs, err := fixtureRuns()
	assert.NoError(t, err, "the kanon directives of the fixture files parse")
	for _, g := range runs {
		assert.NoError(t, g.err, filepath.Join(g.dir, g.file)+": Generate generates the fixture")
	}
	return runs
}

// Files of the modules that the tests write: the go.mod of a module of one
// package, the source file of its struct types, and its code file.
const (
	goMod    = "module example.com/m\n\ngo 1.27\n"
	modName  = "go.mod"
	source   = "a.go"
	codeName = "a.kanon.go"
	// pkgClause begins every Go file of the package.
	pkgClause = "package m\n\n"
)

// Sources of a named type N that the kanon directive of its file names: with
// -validate, which names the method valid, and without -validate, so that the
// ValidateKanon that kanon generates for N returns nil.
const (
	checkedN = "//go:generate go tool kanon -type=N -validate=valid\n\n// N is a number.\ntype N int32\n\n" +
		"// valid returns nil.\nfunc (n N) valid() error { return nil }\n"
	plainN = "//go:generate go tool kanon -type=N\n\n// N is a number.\ntype N int32\n"
)

// module writes a module into a new directory of the test's own with
// files.Workspace, and returns the directory. sources maps the
// slash-separated path of each file to its content. A Go file gets the
// package clause of its directory before its content: package m at the root
// of the module, and the name of its directory below the root. Every other
// file keeps its content, and a go.mod in sources takes the place of goMod.
func module(t *testing.T, sources map[string]string) string {
	t.Helper()
	tree := files.Tree{modName: files.Text(goMod)}
	for name, content := range sources {
		if strings.HasSuffix(name, goSuffix) {
			clause := pkgClause
			if pkg := path.Dir(name); pkg != "." {
				clause = "package " + path.Base(pkg) + "\n\n"
			}
			content = clause + content
		}
		tree[name] = files.Text(content)
	}
	return files.Workspace(t, tree)
}

// generate returns the files that Generate returns for the struct types
// that types names, from file in dir, by name, and the error of Generate.
func generate(t *testing.T, dir, file, types string) (map[string]string, error) {
	t.Helper()
	generated, err := kanon.Generate(dir, file, kanon.Options{Types: strings.Split(types, ",")})
	out := make(map[string]string, len(generated))
	for _, f := range generated {
		out[f.Name] = string(f.Src)
	}
	return out, err
}

// generateError returns the message of the error of Generate for the struct
// type A of a.go in the module of files, and fails t when Generate
// succeeds.
func generateError(t *testing.T, files map[string]string) string {
	t.Helper()
	_, err := generate(t, module(t, files), source, "A")
	assert.HasError(t, err, "Generate fails for the struct type")
	return err.Error()
}

// structA returns the source of the struct type A with the field lines
// fields, the first of which is on line 4 of a.go.
func structA(fields ...string) string {
	return "type A struct {\n\t" + strings.Join(fields, "\n\t") + "\n}\n"
}

func TestGenerate(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		runs := generations(t)
		assert.NotEmpty(t, runs, "the fixture files have kanon directives")
		for _, g := range runs {
			rel, _ := filepath.Rel(fixtureDir, filepath.Join(g.dir, g.file))
			t.Run("writes the files that go generate wrote for "+filepath.ToSlash(rel), func(t *testing.T) {
				t.Parallel()
				for _, f := range g.files {
					assert.Equal(t, string(f.Src), files.Read(t, filepath.Join(g.dir, f.Name)),
						f.Name+": Generate writes the file that go generate wrote")
				}
			})
		}
		t.Run("names the helpers after the file name, with an underscore for each other character", func(t *testing.T) {
			t.Parallel()
			got, err := generate(t, module(t, map[string]string{"a-b.go": sliceA}), "a-b.go", "A")
			assert.NoError(t, err, "Generate generates the struct type")
			assert.Contains(t, got["a-b.kanon.go"], "func _a_b_sizeSliceInt32(", "the helper has the prefix _a_b_")
		})
		failures := []struct {
			name     string
			files    map[string]string
			file     string
			types    string
			validate string
			want     string
		}{
			{
				name: "returns an error for no type", files: map[string]string{source: structXY}, file: source,
				want: "kanon: -type is required: name the types to generate code for",
			},
			{
				name:  "returns an error for a type that the package does not declare",
				files: map[string]string{source: structXY}, file: source, types: "Missing",
				want: "kanon: -type names Missing, which package m does not declare",
			},
			{
				name:  "returns an error for an alias",
				files: map[string]string{source: structXY + "\ntype B = A\n"},
				file:  source,
				types: "B",
				want:  "kanon: a.go:8:6: B: an alias cannot take methods: name the type it denotes",
			},
			{
				name:  "returns an error for a function type",
				files: map[string]string{source: "type N func()\n"},
				file:  source,
				types: "N",
				want:  "kanon: a.go:3:6: N: not a struct, a bool, a number, a string, a slice, an array or a map",
			},
			{
				name:  "returns an error for an interface type",
				files: map[string]string{source: "type N interface {\n\tM()\n}\n"},
				file:  source,
				types: "N",
				want:  "kanon: a.go:3:6: N: not a struct, a bool, a number, a string, a slice, an array or a map",
			},
			{
				name:  "returns an error for a pointer type",
				files: map[string]string{source: "type N *int32\n"},
				file:  source,
				types: "N",
				want:  "kanon: a.go:3:6: N: not a struct, a bool, a number, a string, a slice, an array or a map",
			},
			{
				name:  "returns an error for an unsafe pointer type",
				files: map[string]string{source: "import \"unsafe\"\n\ntype N unsafe.Pointer\n"},
				file:  source,
				types: "N",
				want:  "kanon: a.go:5:6: N: not a struct, a bool, a number, a string, a slice, an array or a map",
			},
			{
				name:  "returns an error for a generic struct type",
				files: map[string]string{source: "type G[T any] struct {\n\tX T\n}\n"}, file: source, types: "G",
				want: "kanon: a.go:3:6: G: generic types are not supported",
			},
			{
				name:  "returns an error for a generic slice type",
				files: map[string]string{source: "type G[T any] []T\n"}, file: source, types: "G",
				want: "kanon: a.go:3:6: G: generic types are not supported",
			},
			{
				name:  "returns an error for a struct type with ValidateKanon",
				files: map[string]string{source: structXY + "\nfunc (A) ValidateKanon() error { return nil }\n"},
				file:  source,
				types: "A",
				want: "kanon: a.go:3:6: A: struct type example.com/m.A declares ValidateKanon, which applies to " +
					"types that are not structs",
			},
			{
				name:  "returns an error for two files that map to one helper prefix",
				files: map[string]string{"a-b.go": kanonA + structsAB, "a_b.go": kanonB}, file: "a-b.go", types: "A",
				want: "kanon: a-b.go and a_b.go map to the same helper prefix _a_b_: rename one of them",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var types []string
				if tt.types != "" {
					types = []string{tt.types}
				}
				opts := kanon.Options{Types: types, Validate: tt.validate}
				_, err := kanon.Generate(module(t, tt.files), tt.file, opts)
				assert.HasError(t, err, "Generate fails")
				assert.Equal(t, err.Error(), tt.want, "Generate states why it fails")
			})
		}
		directed := []struct {
			name  string
			files map[string]string
			field string
			want  bool
		}{
			{
				name: "writes a ValidateKanon call for a field of a type that a directive of the package names " +
					"with -validate before its code file exists",
				files: map[string]string{
					"b.go": checkedN,
					source: structA("F N"),
				},
				field: "F",
				want:  true,
			},
			{
				name: "writes a ValidateKanon call for a field of a type that a directive of another package names " +
					"with -validate before its code file exists",
				files: map[string]string{
					"dep/dep.go": checkedN,
					source:       "import \"example.com/m/dep\"\n\n" + structA("F dep.N"),
				},
				field: "F",
				want:  true,
			},
			{
				name: "writes no ValidateKanon call for a field of a type that a directive of the package names " +
					"without -validate",
				files: map[string]string{
					"b.go": plainN,
					source: structA("F N"),
				},
				field: "F",
			},
			{
				name: "writes no ValidateKanon call for a field of a type that a directive of another package names " +
					"without -validate",
				files: map[string]string{
					"dep/dep.go": plainN,
					source:       "import \"example.com/m/dep\"\n\n" + structA("F dep.N"),
				},
				field: "F",
			},
			{
				name: "writes a ValidateKanon call for a field of a type of another package that declares the method " +
					"itself",
				files: map[string]string{
					"dep/dep.go": "// N is a number.\ntype N int32\n\n// ValidateKanon returns nil.\n" +
						"func (N) ValidateKanon() error { return nil }\n",
					source: "import \"example.com/m/dep\"\n\n" + structA("F dep.N"),
				},
				field: "F",
				want:  true,
			},
			{
				name: "writes no ValidateKanon call for a field of an interface type that a directive of the package names",
				files: map[string]string{
					"b.go": "//go:generate go tool kanon -type=V\n\n// V is an interface.\ntype V interface {\n\tM()\n}\n",
					source: "type N int32\n\nfunc (N) M() {}\n\n" + structA("F V `kanon:\",types=N\"`"),
				},
				field: "F",
			},
		}
		for _, tt := range directed {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				got, err := generate(t, module(t, tt.files), source, "A")
				assert.NoError(t, err, "Generate encodes the field")
				assert.Equal(t, strings.Contains(got[codeName], "m."+tt.field+".ValidateKanon()"), tt.want,
					"the encode calls ValidateKanon on the value of the field")
			})
		}
		t.Run("returns an error for a file that the package does not contain", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{source: structXY})
			_, err := generate(t, dir, "missing.go", "A")
			assert.HasError(t, err, "Generate fails for the file")
			assert.Equal(t, err.Error(), "kanon: missing.go is not a Go file of the package in "+dir,
				"Generate states why the file fails")
		})
	})
}
