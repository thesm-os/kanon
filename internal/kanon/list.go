// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"maps"
	"slices"
)

// typeList is the list of concrete types that the tag option types of one
// field gives the interfaces of the field's type tree: the interfaces of the
// field itself, of its elements, keys, values and pointers, and of the
// concrete types of the list that are not structs. Each concrete type has a
// number on the wire, which the numbers line of the list records under the
// site of the field.
type typeList struct {
	// owner is the struct that declares the field, and field is the name of
	// the field.
	owner *target
	field string
	// types lists the concrete types in tag order.
	types []*concrete
	// values maps the id of each interface value of the tree to its
	// encoding, so that a concrete type that contains the interface again
	// refers to it.
	values map[string]*value
	// reserved lists the numbers of the concrete types that the list no
	// longer names, ascending. The numbers line keeps recording them.
	reserved []int
}

// key returns the key of the numbers line of l: the site of its field, from
// the key of the struct that declares the field.
func (l *typeList) key() string {
	return l.owner.key + fieldStep + l.field
}

// recordKeys returns the keys under which a code file can record the
// numbers of l: the site of its field from each key of the struct that
// declares it, as [target.recordKeys] returns them.
func (l *typeList) recordKeys() []string {
	keys := l.owner.recordKeys()
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+fieldStep+l.field)
	}
	return out
}

// assign numbers the concrete types of l. A type takes the number that rec
// records for it, and otherwise the smallest number that is neither taken
// nor reserved, in tag order. assign sets l.reserved to the reservations of
// rec and the numbers of the recorded types that l no longer names. It fails
// when rec gives two types one number.
func (l *typeList) assign(rec *record) error {
	named := make(map[string]bool, len(l.types))
	for _, c := range l.types {
		named[c.name] = true
	}
	reserved := make(map[int]bool, len(rec.reserved))
	for _, n := range rec.reserved {
		reserved[n] = true
	}
	for name, n := range rec.nums {
		if !named[name] {
			reserved[n] = true
		}
	}
	taken := make(map[int]string, len(l.types))
	for _, c := range l.types {
		n, ok := rec.nums[c.name]
		if !ok {
			continue
		}
		if other, dup := taken[n]; dup {
			return fmt.Errorf(
				"kanon: the numbers line for %s gives %s and %s type number %d",
				l.key(),
				other,
				c.name,
				n,
			)
		}
		taken[n] = c.name
		c.num = n
	}
	next := 1
	for _, c := range l.types {
		if c.num != 0 {
			continue
		}
		next = nextFree(next, taken, reserved)
		taken[next] = c.name
		c.num = next
	}
	l.reserved = slices.Sorted(maps.Keys(reserved))
	return nil
}

// numbersLine returns the numbers line of l: its key, the type numbers in
// tag order, and the reserved numbers.
func (l *typeList) numbersLine() string {
	names := make([]string, 0, len(l.types))
	nums := make(map[string]int, len(l.types))
	for _, c := range l.types {
		names = append(names, c.name)
		nums[c.name] = c.num
	}
	return renderNumbers(l.key(), names, nums, l.reserved)
}

// concrete is one concrete type of a typeList.
type concrete struct {
	typ types.Type
	// name names the type in the numbers line: its type string, with the
	// import path of each package other than the file's.
	name string
	// num is the type number on the wire.
	num int
}

// variant is a concrete type that an interface value can store: a type of
// the value's list that implements the interface, and its encoding.
type variant struct {
	c   *concrete
	val *value
}

// lists collects the lists of concrete types of a code file, one per field
// with the tag option types. The instantiations of a generic struct share
// the list of a field.
type lists struct {
	// byKey maps the key of each list, as [typeList.key] returns it, to the
	// list.
	byKey map[string]*typeList
	// list lists the lists in the order in which the analysis visits them.
	list []*typeList
}

// newLists returns an empty collection.
func newLists() *lists {
	return &lists{byKey: make(map[string]*typeList)}
}

// numberLists numbers the concrete types of every list in ls, in that
// order, with the record that [recordFor] picks for the keys that
// [typeList.recordKeys] returns.
func numberLists(ls []*typeList, own records, others map[string][]recordAt) error {
	for _, l := range ls {
		rec, err := recordFor(l.recordKeys(), own, others)
		if err != nil {
			return err
		}
		if err := l.assign(rec); err != nil {
			return err
		}
	}
	return nil
}

// concreteStep returns the step of a site to the value of the concrete type
// named name that an interface stores, in the syntax of a type assertion.
func concreteStep(name string) string {
	return ".(" + name + ")"
}

// markNilable sets nilable on every interface value of the fields of
// structs that the code can size and encode while it is nil: every one but
// the value of an interface field that is not a union member, which the
// presence check of the field passes only when it is not nil. That value is
// nilable too when its tree contains it again, as the elements of a []any
// that an any stores do.
func markNilable(structs []*target) {
	seen := make(map[*value]bool)
	var walk func(v *value)
	walk = func(v *value) {
		if seen[v] {
			return
		}
		seen[v] = true
		v.nilable = v.kind == kindInterface
		for _, w := range v.variants {
			walk(w.val)
		}
		if v.key != nil {
			walk(v.key)
		}
		if v.elem != nil {
			walk(v.elem)
		}
	}
	for _, m := range structs {
		for _, f := range m.fields {
			if f.val.kind != kindInterface || f.member != nil {
				walk(f.val)
				continue
			}
			for _, w := range f.val.variants {
				walk(w.val)
			}
		}
	}
}

// listOf returns the list of concrete types that the tag option types of the
// tag t of the field obj of the struct of m gives. Each expression of the
// option evaluates as a type in the scope of the file that declares the
// struct, or in the scope of its package for a struct of another package.
// The instantiations of a generic struct share the list of a field, under
// the key of the generic struct.
//
// listOf fails for an expression that is not a type, for an interface type,
// and for a type that the option lists twice.
func (c classifier) listOf(m *target, obj *types.Var, t tag) (*typeList, error) {
	l := &typeList{owner: m, field: obj.Name(), values: make(map[string]*value)}
	if found := c.lists.byKey[l.key()]; found != nil {
		return found, nil
	}
	pos := obj.Pos()
	if obj.Pkg() != c.pkg {
		pos = token.NoPos
	}
	for _, expr := range t.concreteTypes() {
		tv, err := types.Eval(c.fset, obj.Pkg(), pos, expr)
		if te, ok := errors.AsType[types.Error](err); ok {
			return nil, fmt.Errorf("kanon: the tag option types lists %s: %s", expr, te.Msg)
		}
		if err != nil {
			return nil, fmt.Errorf("kanon: the tag option types lists %s, which does not parse as a type", expr)
		}
		if !tv.IsType() {
			return nil, fmt.Errorf("kanon: the tag option types lists %s, which is not a type", expr)
		}
		if types.IsInterface(tv.Type) {
			return nil, fmt.Errorf("kanon: the tag option types lists interface type %s, which no value has", expr)
		}
		for _, other := range l.types {
			if types.Identical(other.typ, tv.Type) {
				return nil, fmt.Errorf("kanon: the tag option types lists %s twice", expr)
			}
		}
		l.types = append(l.types, &concrete{typ: tv.Type, name: types.TypeString(tv.Type, c.recordQualifier)})
	}
	c.lists.byKey[l.key()] = l
	c.lists.list = append(c.lists.list, l)
	return l, nil
}

// recordQualifier is the [types.Qualifier] of the names of concrete types in
// the numbers line: the file's package is unqualified, and any other package
// is qualified by its import path.
func (c classifier) recordQualifier(pkg *types.Package) string {
	if pkg == c.pkg {
		return ""
	}
	return pkg.Path()
}

// iface classifies v, a value of an interface type at the site at. Its
// variants are the concrete types of o.list that implement the interface,
// and inside a map key the comparable ones alone. A value of the interface
// type met again in the tree, as in a list that names []any for an any,
// refers to the encoding of the first, which iface records before it
// classifies the variants.
//
// iface fails when the field has no list, and when no concrete type of the
// list fits the interface.
func (c classifier) iface(v *value, o treeOpts, at site) (*value, error) {
	if o.list == nil {
		return nil, fmt.Errorf(
			"kanon: interface type %s needs the tag option types, which lists its concrete types",
			v.typ,
		)
	}
	if found := o.list.values[v.id]; found != nil {
		for _, w := range found.variants {
			o.placed[w.c] = true
		}
		return found, nil
	}
	v.kind = kindInterface
	v.fails = true
	v.list = o.list
	o.list.values[v.id] = v
	for _, ct := range o.list.types {
		if !types.AssignableTo(ct.typ, v.typ) || o.key && !types.Comparable(ct.typ) {
			continue
		}
		o.placed[ct] = true
		w, err := c.tree(ct.typ, o, at.step(concreteStep(ct.name)))
		if err != nil {
			return nil, err
		}
		v.variants = append(v.variants, variant{c: ct, val: w})
	}
	if len(v.variants) == 0 && o.key {
		return nil, fmt.Errorf(
			"kanon: the tag option types lists no comparable concrete type of interface type %s inside a map key",
			v.typ,
		)
	}
	if len(v.variants) == 0 {
		return nil, fmt.Errorf("kanon: the tag option types lists no concrete type of interface type %s", v.typ)
	}
	return v, nil
}
