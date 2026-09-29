// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"go/types"
	"slices"
	"strings"
	"unicode"
)

// prefixMark begins and ends the prefix of the helpers of a code file.
const prefixMark = '_'

// File is a file that kanon generates.
type File struct {
	// Name is the base name of the file, which belongs in the directory of
	// its source file.
	Name string
	// Src is the content of the file, formatted by gofmt.
	Src []byte
}

// Generate returns the files that kanon generates for the types that
// opts.Types names, declared in the package in the directory dir, from the
// kanon directive of the Go source file named file, a base name:
//
//   - <base>.kanon.go declares the methods of kanon.Cloner on each struct
//     type, the functions of the struct types in the types of their fields
//     that have no kanon codec, which it encodes as their own codec would,
//     and a view type per struct type when opts.Views is set. It declares
//     the ValidateKanon method of kanon.Validator on each named type that is
//     not a struct, which calls the method that opts.Validate names.
//   - <base>.kanon_test.go runs the conformance suite of package kanontest
//     on each type.
//
// Field numbers follow declaration order unless a kanon tag sets them.
// Generate reads the numbers that the code files of the package record for
// the types and keeps them:
//
//   - A field keeps its number across edits of its struct.
//   - The number of a removed field goes to no other field.
//   - A struct that <base>.kanon.go does not record takes the numbers that
//     another code file of the package records for it.
//   - <base>.kanon.go keeps the numbers line of a struct of the package that
//     it no longer generates until another code file records the struct, so
//     that a struct that moves to another source file keeps its numbers.
//
// Generate fails when:
//
//   - opts.Types is empty;
//   - the package does not load;
//   - file is not a Go file of the package;
//   - a name of opts.Types is not a non-generic type of the package that is
//     a struct, a bool, a number, a string, a slice, an array or a map;
//   - a struct type of opts.Types has ValidateKanon;
//   - a named type of opts.Types that is not a struct declares ValidateKanon
//     itself or lacks the method that opts.Validate names;
//   - opts.Validate is set and opts.Types names only struct types;
//   - the kanon directive of another file names a type of opts.Types;
//   - a code file does not read or parse;
//   - two code files record different numbers for a struct that Generate
//     encodes;
//   - a field of a type or of an inline struct has a type or a tag that
//     kanon does not encode, a type that the generated code cannot name, or
//     a kanon.Validator whose ValidateKanon kanon rejects;
//   - a declaration of the package has a name that the generated code uses.
//
// Every error message starts with "kanon: ", except the error of gofmt for
// a generated file that it cannot parse, which marks a defect of kanon.
func Generate(dir, file string, opts Options) ([]File, error) {
	u, err := newUnit(dir, file, opts)
	if err != nil {
		return nil, err
	}
	u.views = opts.Views
	code, codeErr := codeFile(u)
	test, testErr := testFile(u)
	if err := errors.Join(codeErr, testErr); err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(file, goSuffix)
	return []File{{Name: base + CodeSuffix, Src: code}, {Name: base + testSuffix, Src: test}}, nil
}

// unit is one source file with a kanon directive and the types that the
// directive names. kanon generates one code file and one test file per
// unit.
type unit struct {
	pkg *pkg
	// file is the base name of the source file.
	file string
	// targets lists the struct types in the order in which the directive
	// names them.
	targets []*target
	// values lists the named types that are not structs, in the order in
	// which the directive names them.
	values []*valueType
	// inlines collects the inline structs of the types of the fields of the
	// targets.
	inlines *inlines
	// numbered lists the targets whose numbers lines the code file writes:
	// the targets, then the inline structs, as [inlines.number] returns
	// them.
	numbered []*target
	// lists are the lists of concrete types of the fields of the targets and
	// the inline structs, whose numbers lines follow those of numbered.
	lists []*typeList
	// prefix begins the name of every helper that the code file declares.
	prefix string
	// carried lists the numbers lines that the code file keeps for structs
	// that it no longer generates, as [carried] returns them.
	carried []string
	// keyStructs maps the type string of each struct type with a kanon codec
	// in the map keys of the targets to the numbers of its fields.
	keyStructs map[string]keyStruct
	// views reports that the code file declares a view type per target.
	views bool
}

// newUnit returns the unit of the kanon directive of the Go source file
// named file of the package in dir, for the types that opts.Types names. The
// unit contains the struct types, their inline structs and the lists of
// concrete types of their fields, numbered with the numbers that the code
// files of the package record, the numbers lines that the code file keeps
// for the structs that it no longer generates, and the named types that are
// not structs with the method that opts.Validate names. newUnit fails as
// [Generate] states, apart from the rendering of the files.
func newUnit(dir, file string, opts Options) (*unit, error) {
	names := opts.Types
	if len(names) == 0 {
		return nil, errors.New("kanon: -type is required: name the types to generate code for")
	}
	p, err := load(dir)
	if err != nil {
		return nil, err
	}
	if p.files[file] == nil {
		return nil, fmt.Errorf("kanon: %s is not a Go file of the package in %s", file, dir)
	}
	for _, t := range names {
		if other, ok := p.listed[t]; ok && other != file {
			return nil, fmt.Errorf("kanon: -type names %s, which the kanon directive in %s names too", t, other)
		}
		p.listed[t] = file
	}
	prefix, err := p.prefix(file)
	if err != nil {
		return nil, err
	}
	named, others, err := p.named(names)
	if err != nil {
		return nil, err
	}
	values, err := p.valueTypes(others, opts.Validate)
	if err != nil {
		return nil, err
	}
	own, recorded, err := p.codeRecords(codeName(file))
	if err != nil {
		return nil, err
	}
	c := classifier{
		nested:    p.nested,
		generated: p.generated,
		validated: p.validated,
		pkg:       p.types,
		fset:      p.fset,
		inlines:   newInlines(),
		lists:     newLists(),
		named:     make(map[string]*value),
	}
	u := &unit{pkg: p, file: file, prefix: prefix, inlines: c.inlines, values: values}
	if u.targets, err = c.targets(named, own, recorded); err != nil {
		return nil, err
	}
	inlined, err := c.inlines.number(own, recorded)
	if err != nil {
		return nil, err
	}
	err = numberLists(c.lists.list, own, recorded)
	if err != nil {
		return nil, err
	}
	u.numbered = slices.Concat(u.targets, inlined)
	u.lists = c.lists.list
	if u.keyStructs, err = u.orderNumbers(c, own, recorded); err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(u.numbered))
	for _, m := range u.numbered {
		keys = append(keys, m.key)
	}
	u.carried = carried(own, recorded, keys, p.declares)
	return u, nil
}

// header returns the comment lines that open both files of u.
func (u *unit) header() string {
	return generatedHeader + sourceHeader + u.file + "\n"
}

// printer returns a printer for a file of u. The names of the declarations
// of the package and of the locals of the generated code are blocked for
// imports.
func (u *unit) printer() *printer {
	blocked := append(localNames(), u.pkg.types.Scope().Names()...)
	return newPrinter(u.pkg.types.Path(), blocked)
}

// codeName returns the name of the code file of the source file named file.
func codeName(file string) string {
	return strings.TrimSuffix(file, goSuffix) + CodeSuffix
}

// helperPrefix returns the prefix of the helpers of the code file of the
// source file named file: the base name between underscores, with every
// character that cannot occur in an identifier replaced by an underscore.
func helperPrefix(file string) string {
	var sb strings.Builder
	sb.WriteRune(prefixMark)
	for _, r := range strings.TrimSuffix(file, goSuffix) {
		if r != prefixMark && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			r = prefixMark
		}
		sb.WriteRune(r)
	}
	sb.WriteRune(prefixMark)
	return sb.String()
}

// generable reports whether kanon generates code for the named type t: a
// struct type gets a codec, and a named bool, number, string, slice, array
// or map gets the ValidateKanon method of kanon.Validator. A pointer type
// takes no methods, kanon encodes an interface by its concrete type, and it
// encodes no function, channel or unsafe pointer.
func generable(t *types.Named) bool {
	switch u := t.Underlying().(type) {
	case *types.Struct, *types.Slice, *types.Array, *types.Map:
		return true
	case *types.Basic:
		return u.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) != 0
	default:
		return false
	}
}

// named returns the types that names names, in that order: the struct
// types, and the named types that are not structs. It fails for a name that
// the package does not declare at package level, an alias, a generic type, a
// struct type with ValidateKanon, and a named type of a kind that kanon
// generates no code for, as [generable] reports.
func (p *pkg) named(names []string) ([]*types.Named, []*types.Named, error) {
	var structs, others []*types.Named
	for _, name := range names {
		obj, _ := p.types.Scope().Lookup(name).(*types.TypeName)
		if obj == nil {
			return nil, nil, fmt.Errorf("kanon: -type names %s, which package %s does not declare", name,
				p.types.Name())
		}
		fail := func(reason string) error {
			return &sourceError{err: errors.New("kanon: " + reason), pos: p.fset.Position(obj.Pos()), subject: name}
		}
		if obj.IsAlias() {
			return nil, nil, fail("an alias cannot take methods: name the type it denotes")
		}
		t, _ := obj.Type().(*types.Named)
		if !generable(t) {
			return nil, nil, fail("not a struct, a bool, a number, a string, a slice, an array or a map")
		}
		if t.TypeParams().Len() > 0 {
			return nil, nil, fail("generic types are not supported")
		}
		if !isStruct(t) {
			others = append(others, t)
			continue
		}
		if _, err := validatorOf(t); err != nil {
			return nil, nil, &sourceError{err: err, pos: p.fset.Position(obj.Pos()), subject: name}
		}
		structs = append(structs, t)
	}
	return structs, others, nil
}

// directed reports whether a -type flag of a kanon directive of the package
// of the named type t names it, in p or in a dependency. A type that a
// directive names counts before its code file exists, so that the packages
// of a module generate in any order.
func (p *pkg) directed(t *types.Named) bool {
	obj := t.Obj()
	if obj.Pkg() == p.types {
		_, listed := p.listed[obj.Name()]
		return listed
	}
	return obj.Pkg() != nil && p.remoteListed(obj.Pkg().Path(), obj.Name())
}

// nested reports whether kanon encodes the named type t as a nested struct:
// a directive names t, a struct type, as [pkg.directed] reports, or the
// pointer to t has the methods that kanon generates, written by hand or
// generated before.
func (p *pkg) nested(t *types.Named) bool {
	return p.generated(t) || hasKanonMethods(t)
}

// generates reports whether a -type flag of a kanon directive of p names t,
// the named type of a struct with a kanon codec, so that its code file
// declares the unexported methods that the generated code of p calls.
func (p *pkg) generates(t types.Type) bool {
	n, _ := types.Unalias(t).(*types.Named)
	_, listed := p.listed[n.Obj().Name()]
	return listed && n.Obj().Pkg() == p.types && isStruct(n)
}

// generated reports whether kanon generates the codec of the struct type t:
// a directive names t, as [pkg.directed] reports.
func (p *pkg) generated(t *types.Named) bool {
	return isStruct(t) && p.directed(t)
}

// validated reports whether kanon generates the ValidateKanon method of the
// named type t: t is not a struct, [generable] reports t, and a directive
// names t, as [pkg.directed] reports. kanon encodes t as its underlying type
// before the code file that declares the method exists.
func (p *pkg) validated(t *types.Named) bool {
	return !isStruct(t) && generable(t) && p.directed(t)
}

// prefix returns the prefix of the helpers of the code file of the file
// named file. It fails when another file of p with a kanon directive maps
// to the same prefix.
func (p *pkg) prefix(file string) (string, error) {
	want := helperPrefix(file)
	for other := range p.directives {
		if other != file && helperPrefix(other) == want {
			return "", fmt.Errorf("kanon: %s and %s map to the same helper prefix %s: rename one of them",
				file, other, want)
		}
	}
	return want, nil
}

// targets returns the targets of the struct types named, in that order,
// with the field numbers that the records of the code files keep, own of
// the code file that kanon generates and others of the other code files,
// as [recordOf] picks them, and new numbers for the fields that they do not
// record. The inline structs in the types of their fields collect in the
// inlines of c.
func (c classifier) targets(named []*types.Named, own records, others map[string][]recordAt) ([]*target, error) {
	out := make([]*target, 0, len(named))
	for _, n := range named {
		m, err := c.target(n)
		if err != nil {
			return nil, err
		}
		rec, err := recordOf(m.key, own, others)
		if err != nil {
			return nil, err
		}
		if err := assign(m, rec); err != nil {
			return nil, err
		}
		m.tree = c.holdsString(n)
		m.fails = c.mayFail(n, make(map[*types.Named]bool))
		out = append(out, m)
	}
	return out, nil
}
