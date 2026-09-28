// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"fmt"
	"go/types"
	"slices"
)

// keyField is an encoded field of the struct type of a map key, which the
// compare function of the key's type orders keys by.
type keyField struct {
	obj *types.Var
	tag tag
	val *value
}

// visit is a value that a walk of the map keys of a code file visits, and
// whether it is part of a map key.
type visit struct {
	v   *value
	key bool
}

// visitID identifies a visit by the id of its value. The fields of a struct
// with a kanon codec in a map key classify anew on every call of
// [classifier.keyFields], so a walk that marks its visits by id ends at a
// key type that contains itself.
type visitID struct {
	id  string
	key bool
}

// keyStruct is a struct type with a kanon codec in a map key of a code
// file, with the numbers of its fields by field name, which order its keys.
type keyStruct struct {
	typ  types.Type
	nums map[string]int
}

// numsOf returns the numbers of the fields of m, by field name.
func numsOf(m *target) map[string]int {
	out := make(map[string]int, len(m.fields))
	for _, f := range m.fields {
		out[f.name] = f.num
	}
	return out
}

// keyFields returns the fields of the struct type t that kanon encodes, in
// declaration order, with their tags and encodings: the fields that kanon
// reads, as [reads] reports, without a `kanon:"-"` tag, other than
// functions, channels and the field that keeps unknown fields. A field
// whose tag does not parse counts as not encoded, as [classifier.reaches]
// counts it. The fields classify in lookup mode, so that ordering a key adds
// no inline struct.
//
// keyFields fails when t is not a struct, when a field is a union member,
// whose encoding depends on its discriminator while the order of the keys
// reads every member, when a field can contain an interface, whose concrete
// types the tag of a field numbers in the code file of t, when a field is
// unexported and t belongs to another package, whose fields the compare
// function of the file cannot read, and when kanon does not encode the type
// of a field.
func (c classifier) keyFields(t types.Type) ([]keyField, error) {
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("kanon: map key type %s has kanon methods and is not a struct: kanon cannot order it", t)
	}
	finder := c
	finder.lookup = true
	var out []keyField
	for i := range st.NumFields() {
		obj := st.Field(i)
		tg, err := parseTag(st.Tag(i))
		if !reads(obj, tg) || unencoded(obj.Type()) || err != nil || tg.skip || tg.unknown {
			continue
		}
		if !obj.Exported() && obj.Pkg() != c.pkg {
			return nil, fmt.Errorf(
				"kanon: map key type %s has the unexported field %s, which code outside package %s cannot read: "+
					"kanon cannot order it",
				t,
				obj.Name(),
				obj.Pkg().Name(),
			)
		}
		if tg.union != "" {
			return nil, fmt.Errorf(
				"kanon: map key type %s has the union member %s, whose encoding depends on its discriminator: "+
					"kanon cannot order it",
				t,
				obj.Name(),
			)
		}
		if c.holdsInterface(obj.Type()) {
			return nil, fmt.Errorf(
				"kanon: map key type %s has field %s, which can contain an interface: kanon cannot order it",
				t,
				obj.Name(),
			)
		}
		v, err := finder.tree(obj.Type(), treeOpts{used: new(bool), key: true}, site{})
		if err != nil {
			return nil, err
		}
		out = append(out, keyField{obj: obj, tag: tg, val: v})
	}
	return out, nil
}

// orderable checks that kanon can order the map keys of v, the key of a
// map: every struct in v, through arrays, pointers, the concrete types of
// interfaces and the fields of structs, is a struct whose fields
// [classifier.keyFields] accepts. seen marks the values checked by their
// ids, so that a key type that contains itself ends the walk.
func (c classifier) orderable(v *value, seen map[string]bool) error {
	if seen[v.id] {
		return nil
	}
	seen[v.id] = true
	switch v.kind {
	case kindArray, kindPointer:
		return c.orderable(v.elem, seen)
	case kindInterface:
		for _, w := range v.variants {
			if err := c.orderable(w.val, seen); err != nil {
				return err
			}
		}
		return nil
	case kindStruct:
		fields, err := c.keyFields(v.typ)
		if err != nil {
			return err
		}
		for _, f := range fields {
			if err := c.orderable(f.val, seen); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

// single reports whether v, a map key or a part of one, has one value, so
// that any two values of v are equal: an array of no elements, an array
// whose elements have one value, and a struct none of whose encoded fields
// has more than one value. A map with such a key has one entry at most. The
// fields of every struct in v classify, as [classifier.orderable] checked.
func (c classifier) single(v *value) bool {
	if v.zeroOnly() {
		return true
	}
	switch v.kind {
	case kindArray:
		return c.single(v.elem)
	case kindStruct:
		fields, _ := c.keyFields(v.typ)
		return !slices.ContainsFunc(fields, func(f keyField) bool { return !c.single(f.val) })
	default:
		return false
	}
}

// orderNumbers returns the struct types with a kanon codec in the map keys
// of the fields of u, with the numbers of their fields as
// [unit.structNumbers] finds them, by the type string of the struct. A map
// orders such keys field by field in ascending field number. It fails as
// structNumbers fails.
func (u *unit) orderNumbers(c classifier, own records, others map[string][]recordAt) (map[string]keyStruct, error) {
	out := make(map[string]keyStruct)
	seen := make(map[visitID]bool)
	var walk func(v *value, key bool) error
	walk = func(v *value, key bool) error {
		if v == nil || seen[visitID{v.id, key}] {
			return nil
		}
		seen[visitID{v.id, key}] = true
		var next []visit
		switch v.kind {
		case kindMap:
			next = []visit{{v.key, true}, {v.elem, key}}
		case kindStruct:
			if !key {
				return nil
			}
			if v.inline != nil {
				for _, f := range v.inline.fields {
					next = append(next, visit{f.val, true})
				}
				break
			}
			nums, err := u.structNumbers(c, v.typ, own, others)
			if err != nil {
				return err
			}
			out[types.TypeString(v.typ, nil)] = keyStruct{typ: v.typ, nums: nums}
			fields, _ := c.keyFields(v.typ)
			for _, f := range fields {
				next = append(next, visit{f.val, true})
			}
		default:
			next = append(next, visit{v.elem, key})
			for _, w := range v.variants {
				next = append(next, visit{w.val, key})
			}
		}
		for _, n := range next {
			if err := walk(n.v, n.key); err != nil {
				return err
			}
		}
		return nil
	}
	for _, m := range slices.Concat(u.targets, u.inlines.list) {
		for _, f := range m.fields {
			if err := walk(f.val, false); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// structNumbers returns the numbers of the fields of t, the struct type of a
// map key that the code file does not encode with functions of its own: the
// numbers of the target of u for a -type struct of u, and otherwise the
// numbers that [assign] gives the fields that [classifier.keyFields]
// returns, with the record that the code files of the package of t keep
// for it. The record of a struct of the package of u comes from own and
// others, and that of a struct of another package from the code files of
// that package. An anonymous struct type has no record. structNumbers fails
// for a record that does not read, and as assign fails. [classifier.orderable]
// checked the fields of t.
func (u *unit) structNumbers(
	c classifier,
	t types.Type,
	own records,
	others map[string][]recordAt,
) (map[string]int, error) {
	for _, m := range u.targets {
		if types.Identical(m.typ, t) {
			return numsOf(m), nil
		}
	}
	// The classification of the key checked its fields with orderable.
	fields, _ := c.keyFields(t)
	m := &target{typ: t, fset: c.fset, name: types.TypeString(t, c.relative)}
	for _, f := range fields {
		m.fields = append(m.fields, &field{obj: f.obj, name: f.obj.Name(), tag: f.tag})
	}
	rec := &record{}
	var err error
	if named, ok := types.Unalias(t).(*types.Named); ok {
		m.decl, m.key = named.Obj(), named.Obj().Name()
		if pkg := named.Obj().Pkg(); pkg == u.pkg.types {
			rec, err = recordFor([]string{m.key}, own, others)
		} else {
			var recs records
			if recs, err = u.pkg.depRecords(pkg.Path()); err == nil {
				rec, err = recordOf(m.key, recs, nil)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if err := assign(m, rec); err != nil {
		return nil, err
	}
	return numsOf(m), nil
}
