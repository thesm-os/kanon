// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import "go/types"

// Steps of the path of a site: a field is fieldStep and the field's name,
// an element of a slice or an array is elemStep, and a key and a value of a
// map are keyStep and valueStep. The value that a pointer points at shares
// the site of the pointer.
const (
	fieldStep = "."
	elemStep  = "[]"
	keyStep   = "[key]"
	valueStep = "[value]"
)

// site is a place where a value occurs: the path to it from the struct
// whose numbers line records it. An anonymous struct type has no name, so
// its numbers line and its errors name it by the first site at which the
// analysis visits it.
type site struct {
	// key is the path from the key of the numbers line of the struct.
	key string
	// name is the path from the name of the struct in errors.
	name string
}

// field returns the site of the field name of the struct at s.
func (s site) field(name string) site {
	return site{key: s.key + fieldStep + name, name: s.name + fieldStep + name}
}

// step returns the site that step leads to from s: elemStep, keyStep,
// valueStep, or the step of [concreteStep].
func (s site) step(step string) site {
	return site{key: s.key + step, name: s.name + step}
}

// inlines collects the inline structs of a code file: the struct types in
// the types of the fields of its structs that have neither a kanon codec
// nor methods to encode themselves, which the file encodes with functions
// of its own.
type inlines struct {
	// byType maps the type string of a struct type to its target.
	byType map[string]*target
	// generics maps a generic struct type to the target that numbers the
	// fields of its instantiations.
	generics map[*types.Named]*target
	// list lists the targets in the order in which the analysis visits them.
	list []*target
}

// newInlines returns an empty collection.
func newInlines() *inlines {
	return &inlines{byType: make(map[string]*target), generics: make(map[*types.Named]*target)}
}

// inline returns the target of the inline struct type t, which occurs at the
// site at. The first visit of t records the target and then analyses the
// fields of t, so that a struct that contains itself finds its target. Each
// later site of an anonymous struct type adds to the sites of its target. A
// classifier in lookup mode returns the target it finds, or nil, and adds
// nothing.
func (c classifier) inline(t types.Type, at site) (*target, error) {
	id := types.TypeString(t, nil)
	m := c.inlines.byType[id]
	if c.lookup {
		return m, nil
	}
	if m != nil {
		if m.decl == nil {
			m.sites = append(m.sites, at)
		}
		return m, nil
	}
	m, base := c.newInline(t, at)
	c.inlines.byType[id] = m
	c.inlines.list = append(c.inlines.list, m)
	return m, c.analyze(m, base)
}

// newInline returns the target of the inline struct type t, which the
// analysis first visits at the site at, and the site of its fields, before
// the analysis of its fields. The target names the struct in errors and
// keys its numbers line as follows:
//
//   - a struct type of the file's package by its type name in both, as a
//     -type struct;
//   - a struct type of another package by its package name and type name
//     in errors, and by its import path and type name in the numbers line;
//   - an instantiation of a generic struct type by the instantiated type in
//     errors, and by the key of the generic type, which numbers the fields
//     of every instantiation alike;
//   - an anonymous struct type by the site at in both.
//
// The fields of a named type that is not an instantiation occur at the site
// of the type's own name. The fields of an instantiation and of an
// anonymous struct type occur at the site at, so that two instantiations
// that differ in an anonymous struct type do not share the site of its
// fields.
func (c classifier) newInline(t types.Type, at site) (*target, site) {
	m := &target{typ: t, fset: c.fset, inline: true, fails: c.mayFail(t, make(map[*types.Named]bool))}
	named, ok := t.(*types.Named)
	if !ok {
		m.name, m.key, m.sites = at.name, at.key, []site{at}
		return m, at
	}
	m.decl = named.Obj()
	m.name = types.TypeString(t, c.relative)
	if named.TypeArgs().Len() > 0 {
		m.generic = c.genericOf(named.Origin())
		m.key = m.generic.key
		return m, at
	}
	m.key = c.recordKey(named)
	return m, site{key: m.key, name: m.name}
}

// genericOf returns the target of the generic struct type origin, which
// numbers the fields of every instantiation of it alike, and creates it on
// the first call. [genericFields] sets its fields when the inline structs
// are numbered.
func (c classifier) genericOf(origin *types.Named) *target {
	g := c.inlines.generics[origin]
	if g == nil {
		g = &target{
			typ:    origin,
			decl:   origin.Obj(),
			fset:   c.fset,
			name:   c.typeName(origin),
			key:    c.recordKey(origin),
			inline: true,
		}
		c.inlines.generics[origin] = g
	}
	return g
}

// typeName returns the name of the named type t in errors: its type name,
// qualified by its package name when another package declares it.
func (c classifier) typeName(t *types.Named) string {
	obj := t.Obj()
	if q := c.relative(obj.Pkg()); q != "" {
		return q + fieldStep + obj.Name()
	}
	return obj.Name()
}

// recordKey returns the key of the numbers line of the named struct type t:
// its type name for a type of the file's package, and its import path and
// type name for a type of another package.
func (c classifier) recordKey(t *types.Named) string {
	obj := t.Obj()
	if obj.Pkg() == c.pkg {
		return obj.Name()
	}
	return obj.Pkg().Path() + fieldStep + obj.Name()
}

// relative is the [types.Qualifier] of the names of structs in errors: the
// file's package is unqualified, and any other package is qualified by its
// name.
func (c classifier) relative(pkg *types.Package) string {
	if pkg == c.pkg {
		return ""
	}
	return pkg.Name()
}
