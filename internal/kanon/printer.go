// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
)

// printer accumulates the body of one generated Go file and the imports
// its code refers to.
type printer struct {
	// names maps an import path to the name the file refers to it by.
	names map[string]string
	// taken maps a name to the import path that uses or reserves it. A
	// name reserved for anything other than an import, such as a local
	// variable or a package-level identifier, maps to "".
	taken map[string]string
	// refs contains the names of the package-level objects of the file's
	// own package that the code refers to.
	refs map[string]bool
	// self is the import path of the package the file belongs to. Its
	// declarations print unqualified.
	self string
	body bytes.Buffer
}

// newPrinter returns a printer for a file of the package self. No import
// takes a name in blocked.
func newPrinter(self string, blocked []string) *printer {
	p := &printer{
		names: make(map[string]string),
		taken: make(map[string]string),
		refs:  make(map[string]bool),
		self:  self,
	}
	for _, n := range blocked {
		p.taken[n] = ""
	}
	return p
}

// line writes format, expanded with args, and a newline.
func (p *printer) line(format string, args ...any) {
	fmt.Fprintf(&p.body, format, args...)
	p.body.WriteByte('\n')
}

// use records an import of importPath and returns the name the file
// refers to it by: name, or name with the smallest numeric suffix from 2
// that nothing else uses or reserves.
func (p *printer) use(importPath, name string) string {
	if n, ok := p.names[importPath]; ok {
		return n
	}
	n := name
	for i := 2; ; i++ {
		if owner, ok := p.taken[n]; !ok || owner == importPath {
			break
		}
		n = name + strconv.Itoa(i)
	}
	p.names[importPath] = n
	p.taken[n] = importPath
	return n
}

// std records an import of the standard-library package importPath and
// returns its name.
func (p *printer) std(importPath string) string {
	return p.use(importPath, path.Base(importPath))
}

// qualify is the file's [types.Qualifier]: the package the file belongs
// to prints unqualified, and any other package by its import name.
func (p *printer) qualify(pkg *types.Package) string {
	if pkg.Path() == p.self {
		return ""
	}
	return p.use(pkg.Path(), pkg.Name())
}

// typ returns the source text of t for the file.
func (p *printer) typ(t types.Type) string {
	p.refer(t)
	return types.TypeString(t, p.qualify)
}

// object returns the source text naming the package-level object obj.
func (p *printer) object(obj types.Object) string {
	if q := p.qualify(obj.Pkg()); q != "" {
		return q + "." + obj.Name()
	}
	p.refs[obj.Name()] = true
	return obj.Name()
}

// refer records the named types of the file's own package that t
// mentions.
func (p *printer) refer(t types.Type) {
	switch t := t.(type) {
	case *types.Named:
		if obj := t.Obj(); obj.Pkg() != nil && obj.Pkg().Path() == p.self {
			p.refs[obj.Name()] = true
		}
	case *types.Pointer:
		p.refer(t.Elem())
	case *types.Slice:
		p.refer(t.Elem())
	case *types.Array:
		p.refer(t.Elem())
	case *types.Map:
		p.refer(t.Key())
		p.refer(t.Elem())
	}
}

// source assembles the file from the header comment, the package clause,
// the imports and the body, and formats it with gofmt. The imports come in
// path order, the standard library first and then the other packages
// after a blank line, as goimports groups them. gofmt drops the blank line
// when the file imports no other package. Every generated file imports a
// package: the code file imports kanon, and the test file imports testing.
// An import is renamed only when its name differs from its path's last
// element. Source that gofmt rejects is a defect of the emitter, and
// source returns gofmt's error for it unchanged.
func (p *printer) source(header, pkgName string) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(header)
	fmt.Fprintf(&out, "\npackage %s\n", pkgName)
	var std, other []string
	for _, ip := range slices.Sorted(maps.Keys(p.names)) {
		if first, _, _ := strings.Cut(ip, "/"); strings.Contains(first, ".") {
			other = append(other, ip)
		} else {
			std = append(std, ip)
		}
	}
	out.WriteString("\nimport (\n")
	p.imports(&out, std)
	out.WriteByte('\n')
	p.imports(&out, other)
	out.WriteString(")\n")
	out.Write(p.body.Bytes())
	return format.Source(out.Bytes())
}

// imports writes one import spec per path in paths to out.
func (p *printer) imports(out *bytes.Buffer, paths []string) {
	for _, ip := range paths {
		if n := p.names[ip]; n != path.Base(ip) {
			fmt.Fprintf(out, "\t%s %q\n", n, ip)
		} else {
			fmt.Fprintf(out, "\t%q\n", ip)
		}
	}
}
