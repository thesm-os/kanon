// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"strings"
)

// Parts of the go:generate directive that runs kanon.
const (
	// directiveTool and directiveName form "//go:generate".
	directiveTool = "go"
	directiveName = "generate"
	// commandName is the name of the kanon command.
	commandName = "kanon"
	// versionMark separates the import path of a command from its module
	// version, as in go run go.thesmos.sh/kanon/cmd/kanon@v1.0.0.
	versionMark = "@"
	// pathSeparators separate the elements of the path of a command: the
	// slash of every platform and the backslash of Windows.
	pathSeparators = `/\`
	// exeSuffix ends the name of an executable on Windows, in any case.
	exeSuffix = ".exe"
	// directiveSubject names a directive in errors.
	directiveSubject = "kanon directive"
)

// directiveArgs returns the arguments that the go:generate directive c
// passes to kanon, and whether c runs kanon: whether an argument of c names
// the kanon command, as [isCommand] states. The arguments after the command
// are kanon's.
func directiveArgs(c *ast.Comment) ([]string, bool) {
	d, ok := ast.ParseDirective(c.Slash, c.Text)
	if !ok || d.Tool != directiveTool || d.Name != directiveName {
		return nil, false
	}
	args, err := d.ParseArgs()
	if err != nil {
		return nil, false
	}
	for i, a := range args {
		if !isCommand(a.Arg) {
			continue
		}
		rest := make([]string, 0, len(args))
		for _, r := range args[i+1:] {
			rest = append(rest, r.Arg)
		}
		return rest, true
	}
	return nil, false
}

// isCommand reports whether the directive argument arg names the kanon
// command: its last path element, after the last slash or backslash and
// before any module version, is kanon, with or without the suffix of a
// Windows executable. go tool kanon, go run
// go.thesmos.sh/kanon/cmd/kanon@v1.0.0, bin/kanon and C:\bin\kanon.exe all
// name it. The result depends on the directive alone, and not on the
// platform that runs kanon.
func isCommand(arg string) bool {
	name := arg[strings.LastIndexAny(arg, pathSeparators)+1:]
	name, _, _ = strings.Cut(name, versionMark)
	rest, ok := strings.CutPrefix(name, commandName)
	return ok && (rest == "" || strings.EqualFold(rest, exeSuffix))
}

// scan records the kanon directives of f, the syntax tree of the file named
// name. It fails as [pkg.addDirective] fails.
func (p *pkg) scan(name string, f *ast.File) error {
	for _, group := range f.Comments {
		for _, c := range group.List {
			args, ok := directiveArgs(c)
			if !ok {
				continue
			}
			if err := p.addDirective(name, c, args); err != nil {
				return err
			}
		}
	}
	return nil
}

// addDirective records the kanon directive c, with the arguments args, of
// the file named name. It fails for a second directive in the file, for
// flags that do not parse, and for a type that the directive of another
// file names, with an error at the position of c.
func (p *pkg) addDirective(name string, c *ast.Comment, args []string) error {
	fail := func(err error) error {
		return &sourceError{err: err, pos: p.fset.Position(c.Pos()), subject: directiveSubject}
	}
	if _, dup := p.directives[name]; dup {
		return fail(errors.New("kanon: the file has a second kanon directive: name every type in one -type flag"))
	}
	opts, err := ParseOptions(args, io.Discard)
	if err != nil {
		return fail(fmt.Errorf("kanon: %w", err))
	}
	p.directives[name] = opts
	for _, t := range opts.Types {
		if other, ok := p.listed[t]; ok {
			return fail(fmt.Errorf("kanon: -type names %s, which the kanon directive in %s names too", t, other))
		}
		p.listed[t] = name
	}
	return nil
}

// remoteListed reports whether a kanon directive of the dependency with the
// import path importPath names the type name. It parses the Go files of the
// dependency on the first call and caches the names. A file that does not
// parse and a directive whose flags do not parse name nothing: the build and
// the generation of the dependency report them.
func (p *pkg) remoteListed(importPath, name string) bool {
	names, ok := p.remote[importPath]
	if !ok {
		names = make(map[string]bool)
		dep := p.deps[importPath]
		fset := token.NewFileSet()
		for _, file := range dep.GoFiles {
			f, err := parser.ParseFile(fset, filepath.Join(dep.Dir, file), nil, sourceMode)
			if err != nil {
				continue
			}
			for _, group := range f.Comments {
				for _, c := range group.List {
					args, ok := directiveArgs(c)
					if !ok {
						continue
					}
					opts, err := ParseOptions(args, io.Discard)
					if err != nil {
						continue
					}
					for _, t := range opts.Types {
						names[t] = true
					}
				}
			}
		}
		p.remote[importPath] = names
	}
	return names[name]
}
