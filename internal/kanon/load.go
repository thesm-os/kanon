// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Suffixes of the two files that kanon writes beside a source file.
const (
	// CodeSuffix ends the name of the code file, <base>.kanon.go, which
	// declares the codecs and records the field numbers.
	CodeSuffix = ".kanon.go"
	// testSuffix ends the name of the test file, <base>.kanon_test.go.
	testSuffix = ".kanon_test.go"
)

// goSuffix ends the name of a Go source file.
const goSuffix = ".go"

// The go list command that [load] runs.
const (
	// goTool and listCommand run go list.
	goTool      = "go"
	listCommand = "list"
	// tolerantFlag makes go list report a package that fails to load in its
	// output instead of failing.
	tolerantFlag = "-e"
	// jsonFlag selects the fields of the JSON output after its "=".
	jsonFlag = "-json="
	// exportFlag and depsFlag list the dependencies with the paths of their
	// compiled export data.
	exportFlag = "-export"
	depsFlag   = "-deps"
	// thisPackage is the pattern of the package in the working directory.
	thisPackage = "."
)

// Fields of the output of go list that [load] decodes, for the package that
// it loads and for the dependencies of the package.
const (
	targetFields = "ImportPath,Name,Dir,GoFiles,Imports,Error"
	depFields    = "ImportPath,Dir,GoFiles,Export"
)

// compiler is the toolchain whose export data [load] imports.
const compiler = "gc"

// cgoImport is the import path of the pseudo-package of cgo, which has no
// export data.
const cgoImport = "C"

// sourceMode parses a Go file with its comments, which contain the kanon
// directives and the numbers lines, and without the resolution of
// identifiers, which the type checker makes.
const sourceMode = parser.ParseComments | parser.SkipObjectResolution

// listed is one package of the output of go list.
type listed struct {
	// Error is set when go list could not load the package completely.
	Error *struct {
		Err string
	}
	ImportPath string
	Name       string
	Dir        string
	// Export is the path of the compiled export data of the package. It is
	// empty when the package failed to compile.
	Export  string
	GoFiles []string
	Imports []string
}

// pkg is the type-checked package of a directory, with the kanon directives
// of its files and the go list entries of its dependencies.
type pkg struct {
	types *types.Package
	fset  *token.FileSet
	// files maps the base name of each Go file of the package, except the
	// code files, to its syntax tree with its comments.
	files map[string]*ast.File
	// codes lists the base names of the code files of the package, in the
	// order of go list.
	codes []string
	// directives maps the base name of each file with a kanon directive to
	// the options of the directive.
	directives map[string]Options
	// listed maps each type name that the -type flag of a kanon directive
	// names to the base name of the file of the directive.
	listed map[string]string
	// deps maps the import path of each dependency to its go list entry.
	deps map[string]listed
	// remote caches, per import path of a dependency, the type names that
	// its kanon directives name.
	remote map[string]map[string]bool
	// dir is the directory of the package, as go list reports it.
	dir string
}

// load lists the package in dir with go list, parses its Go files other than
// its code files, and type-checks them against the export data of its
// dependencies. It records the kanon directives of every file and the names
// of the code files.
//
// load never compiles the package itself, so that stale or broken
// generated code in it does not block a regeneration. Its dependencies
// compile, with their generated code, to provide export data.
//
// Type errors do not fail the load. The code files are left out, so code of
// the package that calls their methods does not type-check, and the struct
// declarations that kanon reads do not depend on that code. The analysis
// reports a field whose type did not resolve.
//
// load fails when go list fails, when the package has no Go files, when a
// file does not parse, and when the kanon directives conflict: a file with
// two of them, a directive whose flags do not parse, and a type that two
// directives name.
func load(dir string) (*pkg, error) {
	targets, err := goList(dir, jsonFlag+targetFields, thisPackage)
	if err != nil {
		return nil, err
	}
	target := targets[0]
	if len(target.GoFiles) == 0 {
		reason := "no Go files"
		if target.Error != nil {
			reason = target.Error.Err
		}
		return nil, fmt.Errorf("kanon: package in %s: %s", dir, reason)
	}
	imports := slices.DeleteFunc(slices.Clone(target.Imports), func(path string) bool { return path == cgoImport })
	exports := make(map[string]string)
	deps := make(map[string]listed)
	if len(imports) != 0 {
		list, err := goList(dir, append([]string{exportFlag, jsonFlag + depFields, depsFlag}, imports...)...)
		if err != nil {
			return nil, err
		}
		for _, d := range list {
			exports[d.ImportPath] = d.Export
			deps[d.ImportPath] = d
		}
	}
	p := &pkg{
		fset:       token.NewFileSet(),
		files:      make(map[string]*ast.File),
		directives: make(map[string]Options),
		listed:     make(map[string]string),
		deps:       deps,
		remote:     make(map[string]map[string]bool),
		dir:        target.Dir,
	}
	syntax := make([]*ast.File, 0, len(target.GoFiles))
	for _, name := range target.GoFiles {
		if strings.HasSuffix(name, CodeSuffix) {
			p.codes = append(p.codes, name)
			continue
		}
		f, err := parser.ParseFile(p.fset, filepath.Join(target.Dir, name), nil, sourceMode)
		if err != nil {
			return nil, fmt.Errorf("kanon: %w", err)
		}
		p.files[name] = f
		syntax = append(syntax, f)
		if err := p.scan(name, f); err != nil {
			return nil, err
		}
	}
	conf := types.Config{
		Importer:    importer.ForCompiler(p.fset, compiler, exportLookup(exports)),
		Error:       func(error) {},
		FakeImportC: true,
	}
	// With Error set, Check type-checks every file and returns the package
	// whatever it finds. The docblock states why load ignores the first
	// error, which Check returns too.
	p.types, _ = conf.Check(target.ImportPath, p.fset, syntax, nil)
	return p, nil
}

// goList runs go list -e with the arguments args in dir and decodes the
// packages it lists. The error of a failed run ends with what go list wrote
// to its standard error.
func goList(dir string, args ...string) ([]listed, error) {
	cmd := exec.Command(goTool, append([]string{listCommand, tolerantFlag}, args...)...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("kanon: go list in %s: %w", dir, err)
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			err = fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}
	var pkgs []listed
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var p listed
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("kanon: decode go list output: %w", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// exportLookup returns the [importer.Lookup] that opens the export data that
// go list reported for an import path. A path without export data, such as
// a dependency that failed to compile, fails the import, and the fields that
// use it resolve to invalid types. The type checker records a failed import
// as a type error, which [load] ignores.
func exportLookup(exports map[string]string) importer.Lookup {
	return func(importPath string) (io.ReadCloser, error) {
		export := exports[importPath]
		if export == "" {
			return nil, fmt.Errorf("kanon: no export data for %s", importPath)
		}
		return os.Open(export)
	}
}
