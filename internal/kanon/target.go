// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"go/constant"
	"go/token"
	"go/types"
	"unicode"
	"unicode/utf8"
)

// errUnknownType is the error for a field tagged unknown that is unexported
// or whose type is not a []byte.
var errUnknownType = errors.New("kanon: the field tagged unknown must be an exported []byte")

// target is a struct type whose encoding the code file writes: a -type
// struct, which gets the methods of kanon.Cloner, or an inline struct, a
// struct type without a kanon codec in the types of the fields of a -type
// struct, which gets functions that write the same encoding.
type target struct {
	// typ is the struct type: a named type for a -type struct, and a named,
	// instantiated or anonymous struct type for an inline struct.
	typ types.Type
	// decl is the declaration of a named struct type, and nil for an
	// anonymous one.
	decl types.Object
	// generic is the target of the generic struct type of an instantiation,
	// which numbers the fields of every instantiation alike, and nil for any
	// other struct.
	generic *target
	fset    *token.FileSet
	// name names the struct in errors: its type name for a struct of the
	// file's package, and as [classifier.newInline] names the others.
	name string
	// key names the struct in its numbers line, as [classifier.newInline]
	// keys it. It is the type name for a -type struct.
	key string
	// sites lists the sites at which an anonymous struct type occurs, in the
	// order in which the analysis visits them. The first names the struct.
	sites []site
	// fields lists the encoded fields in declaration order.
	fields []*field
	// unknown is the field tagged unknown, which keeps the unknown fields of
	// a decode, or nil.
	unknown *types.Var
	// discriminators lists the discriminator fields of the unions in the
	// order of their first members. The encoding leaves them out, and Reset
	// zeroes them.
	discriminators []*types.Var
	// leftOut lists the other fields that the encoding leaves out, in
	// declaration order: the fields tagged "-", the unexported fields
	// without a kanon tag, and the fields that [unencoded] reports, without
	// blank fields, which no code can name. Reset and the decode of
	// the struct set them to their zero values, and a merge keeps them.
	leftOut []*types.Var
	// reserved lists the numbers of removed fields, ascending. The numbers
	// line keeps recording them.
	reserved []int
	// inline reports an inline struct.
	inline bool
	// tree reports that the struct, or a struct it nests at any depth, has a
	// string, so that a decode without a slab copies its input.
	tree bool
	// fails reports that the encoding can fail, as [classifier.mayFail]
	// reports: the struct contains a type that encodes itself outside a field
	// of a kanon.Exact type and is no kanon.Appender, a kanon.Validator, an
	// interface, or a map whose keys can have a NaN component or share a
	// projection.
	fails bool
}

// fail returns err tied to obj, a field or a method of the struct of m, at
// the position of obj.
func (m *target) fail(obj types.Object, err error) error {
	return &sourceError{err: err, pos: m.fset.Position(obj.Pos()), subject: m.name + fieldStep + obj.Name()}
}

// keepUnknown makes obj the field of m that keeps the unknown fields of a
// decode. It fails when obj is unexported or not a []byte, and when m has
// such a field already.
func (m *target) keepUnknown(obj *types.Var) error {
	if !obj.Exported() || !isByteSlice(obj.Type()) {
		return m.fail(obj, errUnknownType)
	}
	if m.unknown != nil {
		return m.fail(obj, fmt.Errorf("kanon: field %s keeps the unknown fields already", m.unknown.Name()))
	}
	m.unknown = obj
	return nil
}

// leaveOut adds obj, a field of m that the encoding leaves out, to the
// fields that Reset and the decode of m clear, unless obj is a blank field.
func (m *target) leaveOut(obj *types.Var) {
	if obj.Name() != blankName {
		m.leftOut = append(m.leftOut, obj)
	}
}

// field is one encoded field of a target.
type field struct {
	obj *types.Var
	// member ties a union member to its discriminator, and is nil for any
	// other field.
	member *member
	name   string
	// val is the encoding of the field's value.
	val *value
	// list is the list of the concrete types of the field's interfaces, and
	// nil for a field without the tag option types.
	list *typeList
	tag  tag
	num  int
}

// member records the union of a member field: the discriminator field, the
// constant of its type that selects the field, and every member.
type member struct {
	// disc is the discriminator field, which the encoding leaves out.
	disc *types.Var
	// value is the constant of the discriminator's type, named
	// <DiscriminatorType><Member> with the first letter of the member's
	// name in upper case, that selects the member.
	value *types.Const
	// union lists every member of the discriminator in declaration order.
	union []*field
}

// unencoded reports whether a field of type t is left out of the encoding,
// as encoding/gob leaves it out: t is a function or a channel, or a pointer
// to one at any depth. A named pointer type that points at itself is not
// left out.
func unencoded(t types.Type) bool {
	seen := make(map[types.Type]bool)
	for !seen[t] {
		seen[t] = true
		switch u := t.Underlying().(type) {
		case *types.Signature, *types.Chan:
			return true
		case *types.Pointer:
			t = u.Elem()
		default:
			return false
		}
	}
	return false
}

// reads reports whether kanon reads the struct field obj, whose kanon tag is
// t: obj is exported, or it has a kanon tag, which opts an unexported field
// into the encoding. The code of the package of obj alone can read an
// unexported field.
func reads(obj *types.Var, t tag) bool {
	return obj.Exported() || t.tagged
}

// unions resolves the members of every union of m. A member names its
// discriminator with the tag option union=<Field>. The discriminator is an
// exported field of a named integer type, is not encoded, and has a
// distinct non-zero constant <Type><Member> for each member, which its
// type's package declares, with the first letter of the member's name in
// upper case. A member has any type that kanon encodes.
func unions(m *target, byName map[string]*types.Var) error {
	unions := make(map[*types.Var][]*field)
	var order []*types.Var
	for _, f := range m.fields {
		if f.tag.union == "" {
			continue
		}
		disc := byName[f.tag.union]
		if disc == nil || !disc.Exported() {
			return m.fail(
				f.obj,
				fmt.Errorf("kanon: union discriminator %s is not an exported field of %s", f.tag.union, m.name),
			)
		}
		if unions[disc] == nil {
			order = append(order, disc)
		}
		unions[disc] = append(unions[disc], f)
	}
	for _, disc := range order {
		if err := resolveMembers(m, disc, unions[disc]); err != nil {
			return err
		}
		m.discriminators = append(m.discriminators, disc)
	}
	return dropDiscriminators(m, unions)
}

// resolveMembers ties each member of the union of disc to the constant of
// the discriminator's type that selects it, and to the members of its
// union.
func resolveMembers(m *target, disc *types.Var, members []*field) error {
	named, ok := types.Unalias(disc.Type()).(*types.Named)
	if !ok || !isInteger(named) {
		return m.fail(disc, fmt.Errorf("kanon: union discriminator type %s is not a named integer type", disc.Type()))
	}
	seen := make(map[string]*field, len(members))
	for _, f := range members {
		name := named.Obj().Name() + upperFirst(f.name)
		c, _ := named.Obj().Pkg().Scope().Lookup(name).(*types.Const)
		if c == nil || !types.Identical(c.Type(), named) {
			return m.fail(
				f.obj,
				fmt.Errorf("kanon: the union member needs the constant %s of type %s", name, named.Obj().Name()),
			)
		}
		if constant.Sign(c.Val()) == 0 {
			return m.fail(f.obj, fmt.Errorf("kanon: union constant %s is zero, which selects no member", name))
		}
		key := c.Val().ExactString()
		if prev := seen[key]; prev != nil {
			return m.fail(
				f.obj,
				fmt.Errorf("kanon: union constant %s equals the constant of member %s", name, prev.name),
			)
		}
		seen[key] = f
		f.member = &member{disc: disc, value: c, union: members}
	}
	return nil
}

// dropDiscriminators removes the discriminator fields from the encoded
// fields of m. A discriminator must not have a kanon tag.
func dropDiscriminators(m *target, unions map[*types.Var][]*field) error {
	kept := m.fields[:0]
	for _, f := range m.fields {
		if unions[f.obj] == nil {
			kept = append(kept, f)
			continue
		}
		if f.tag.tagged {
			return m.fail(f.obj, errors.New("kanon: a union discriminator is not encoded: remove its kanon tag"))
		}
	}
	m.fields = kept
	return nil
}

// isInteger reports whether the underlying type of t is an integer.
func isInteger(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

// upperFirst returns the identifier name with its first letter in upper
// case. The constant that selects the unexported union member text of a
// discriminator of type Kind is KindText, as it is for a member Text.
func upperFirst(name string) string {
	r, n := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[n:]
}

// target analyses the -type struct type named, as [classifier.analyze]
// states. [assign] numbers its fields afterwards.
func (c classifier) target(named *types.Named) (*target, error) {
	name := named.Obj().Name()
	m := &target{typ: named, decl: named.Obj(), fset: c.fset, name: name, key: name}
	if err := c.analyze(m, site{key: name, name: name}); err != nil {
		return nil, err
	}
	return m, nil
}

// analyze analyses the fields of the struct of m, which occur at the site
// at. It encodes the fields that kanon reads, as [reads] reports, without a
// `kanon:"-"` tag, in declaration order: the exported fields, and the
// unexported fields that a kanon tag opts in. It resolves the union members
// against their discriminators. A field that [unencoded] reports is left
// out, and so is a struct without fields to encode, which encodes to no
// bytes. The fields left out are recorded in leftOut. The field tagged
// unknown keeps the unknown fields of a decode and has no number.
//
// analyze fails for a blank field or a field that [unencoded] reports with
// a kanon tag, for an unexported field with a kanon tag in a
// struct of another package, which the code of the file cannot reach, for
// a types option that [classifier.listOf] rejects, for a field type that
// kanon does not encode, for an invalid union, and for a field tagged
// unknown that is unexported, is not a []byte or follows another.
func (c classifier) analyze(m *target, at site) error {
	st, _ := m.typ.Underlying().(*types.Struct)
	byName := make(map[string]*types.Var, st.NumFields())
	for i := range st.NumFields() {
		obj := st.Field(i)
		byName[obj.Name()] = obj
		t, err := parseTag(st.Tag(i))
		if err != nil {
			return m.fail(obj, err)
		}
		if obj.Name() == blankName || unencoded(obj.Type()) {
			if t.tagged {
				return m.fail(
					obj,
					errors.New("kanon: the field has a kanon tag, and kanon does not encode "+
						"blank fields, functions, channels or pointers to them"),
				)
			}
			m.leaveOut(obj)
			continue
		}
		if t.skip || !reads(obj, t) {
			m.leaveOut(obj)
			continue
		}
		if !obj.Exported() && obj.Pkg() != c.pkg {
			return m.fail(obj, fmt.Errorf("kanon: the field is unexported, and code outside package %s "+
				"cannot reach it: generate a kanon codec for %s in its package", obj.Pkg().Name(), m.name))
		}
		if t.unknown {
			err = m.keepUnknown(obj)
			if err != nil {
				return err
			}
			continue
		}
		var list *typeList
		if t.types != "" {
			if list, err = c.listOf(m, obj, t); err != nil {
				return m.fail(obj, err)
			}
		}
		f := &field{obj: obj, name: obj.Name(), tag: t, list: list}
		if f.val, err = c.classify(obj.Type(), t.fixed, list, at.field(obj.Name())); err != nil {
			return m.fail(obj, err)
		}
		if exactField(f.val, t) {
			f.val.fails = false
		}
		m.fields = append(m.fields, f)
	}
	return unions(m, byName)
}
