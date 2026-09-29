// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
	"slices"
)

// Import path and type name of time.Time, the one standard-library type
// with a kind of its own.
const (
	timePackage = "time"
	timeName    = "Time"
)

// anyName is the predeclared alias of the empty interface. The generated
// code spells it where the source spells it.
const anyName = "any"

// Parts of the id of a value, as [classifier.id] joins them.
const (
	// idSeparator precedes each option in an id.
	idSeparator = "|"
	// idFixed marks a value that the tag option fixed changes.
	idFixed = "fixed"
	// idKey marks a value inside a map key whose interfaces store the
	// comparable concrete types of their list alone.
	idKey = "key"
)

// classifier maps the Go types of the fields of one code file to their
// encodings. Its methods share the collections of the code file, so a copy
// of a classifier classifies into the same file.
type classifier struct {
	// nested reports whether kanon encodes a named type as a nested struct:
	// a -type flag of its package names it, or its pointer has the methods
	// kanon generates.
	nested func(*types.Named) bool
	// generated reports whether kanon generates the codec of a nested
	// struct: a -type flag of its package names it.
	generated func(*types.Named) bool
	// validated reports whether kanon generates the ValidateKanon method of
	// a named type that is not a struct: a -type flag of its package names
	// it. Such a type is a kanon.Validator before its code file exists.
	validated func(*types.Named) bool
	// pkg is the package of the code file, which names every type the file
	// encodes.
	pkg *types.Package
	// fset positions the declarations of the classified types.
	fset *token.FileSet
	// inlines collects the inline structs of the classified types.
	inlines *inlines
	// lists collects the lists of concrete types of the classified fields.
	lists *lists
	// named maps the id of each named type that encodes as its underlying
	// slice, array, map or pointer to its value, so that a type that
	// contains itself, such as type Tree []Tree, ends the classification.
	named map[string]*value
	// codecs collects, by the key of its numbers line as an inline struct,
	// each struct type of another package that the classified types encode
	// through the struct's own kanon codec. [Check] compares its numbers with
	// those that a base revision records for it.
	codecs map[string]*types.Named
	// lookup makes the classifier find the inline struct of a struct type in
	// inlines without adding one. The compare functions of map keys classify
	// the fields of struct keys in this mode, and a struct that the
	// classifier does not find has no target.
	lookup bool
}

// classify returns the encoding of a value of type t, the type of a field
// at the site at. With fixed set, every 32- and 64-bit integer in the tree
// of t takes the fixed-size encoding, except in map keys and in the fields
// of inline structs, which take their own tags. The interfaces in the tree
// store the concrete types of list, as [classifier.iface] states.
//
// A named type resolves in this order: a nested struct; time.Time; a
// kanon.Validator, as [classifier.validator] finds it, which resolves to its
// underlying type ahead of its methods and validates its values; a type
// that encodes itself through a family of methods, as [selfCodecOf] finds
// it; and last, its underlying type, so that a time.Duration encodes as the
// int64 it is. A struct type that resolves to its underlying type is an
// inline struct. Slices, arrays, maps, pointers and interfaces classify
// their elements, keys, values and concrete types in turn, at any depth,
// and a named type that contains itself refers to its own value.
//
// classify fails for a type that kanon does not encode: a function, a
// channel and an unsafe pointer; for a type that the generated code cannot
// name, as [nameable] states; for a ValidateKanon that [validatorOf]
// rejects; for a kanon.Validator whose only value is its zero value, as
// [value.zeroOnly] reports, since its encoding is a constant that the
// generated code writes and reads without the value; for an interface
// without a list; and for an inline struct whose
// fields fail the analysis. It also fails when fixed applies to no integer
// of the tree, when the tree has no interface for list, and when a type of
// list fits no interface of the tree.
func (c classifier) classify(t types.Type, fixed bool, list *typeList, at site) (*value, error) {
	o := treeOpts{fixed: fixed, used: new(bool), list: list, placed: make(map[*concrete]bool)}
	v, err := c.tree(t, o, at)
	if err != nil {
		return nil, err
	}
	if fixed && !*o.used {
		return nil, fmt.Errorf("kanon: tag option %s does not apply to type %s", tagFixed, t)
	}
	if list == nil {
		return v, nil
	}
	if len(o.placed) == 0 {
		return nil, fmt.Errorf("kanon: tag option %s does not apply to type %s", optionTypes, t)
	}
	for _, ct := range list.types {
		if !o.placed[ct] {
			return nil, fmt.Errorf(
				"kanon: the tag option types lists %s, which fits no interface of type %s",
				ct.typ,
				t,
			)
		}
	}
	return v, nil
}

// tree is [classifier.classify] without the checks of the options. It sets
// o.used when an integer of the tree takes the fixed-size encoding, and
// records in o.placed the concrete types that the interfaces of the tree
// store.
func (c classifier) tree(t types.Type, o treeOpts, at site) (*value, error) {
	declared := t
	t = types.Unalias(t)
	if b, ok := t.(*types.Basic); ok && b.Kind() == types.Invalid {
		return nil, errors.New("kanon: the type does not type-check: go build reports why")
	}
	if part := unnamed(t, c.pkg); part != "" {
		return nil, fmt.Errorf("kanon: the generated code cannot name type %s: %s", t, part)
	}
	v := &value{typ: t, id: c.id(t, o)}
	if named, ok := t.(*types.Named); ok {
		validates, err := c.validator(named)
		if err != nil {
			return nil, err
		}
		if c.nested(named) {
			if named.Obj().Pkg() != c.pkg {
				c.codecs[c.recordKey(named)] = named
			}
			v.kind = kindStruct
			v.fails = c.mayFail(named, make(map[*types.Named]bool))
			v.indirect = c.indirect(named)
			v.hollow = c.generated(named) && c.hollow(named)
			v.hidden = c.hidden(named)
			return v, nil
		}
		if isTime(named) {
			v.kind = kindTime
			return v, nil
		}
		if validates {
			v.validate, v.fails = true, true
		} else if self, ok := selfCodecOf(named); ok {
			v.kind, v.self, v.fails = kindBinary, self, true
			return v, nil
		}
		if !isStruct(named) {
			if found := c.named[v.id]; found != nil {
				return found, nil
			}
			c.named[v.id] = v
		}
	}
	var err error
	switch u := t.Underlying().(type) {
	case *types.Basic:
		return basic(v, u, o)
	case *types.Struct:
		v.kind = kindStruct
		v.fails = c.mayFail(t, make(map[*types.Named]bool))
		v.indirect = c.indirect(t)
		v.hidden = c.hidden(t)
		v.inline, err = c.inline(t, at)
	case *types.Slice:
		if isByte(u.Elem()) {
			v.kind = kindBytes
			return v, nil
		}
		v.kind = kindSlice
		v.elem, err = c.tree(u.Elem(), o, at.step(elemStep))
	case *types.Array:
		v.size = u.Len()
		if isByte(u.Elem()) {
			v.kind = kindByteArray
			break
		}
		v.kind = kindArray
		v.elem, err = c.tree(u.Elem(), o, at.step(elemStep))
	case *types.Map:
		v.kind = kindMap
		v.fails = v.fails || c.floats(u.Key()) || c.ambiguous(u.Key())
		keyOpts := o
		keyOpts.fixed, keyOpts.used, keyOpts.key = false, new(bool), true
		if v.key, err = c.tree(u.Key(), keyOpts, at.step(keyStep)); err == nil {
			err = c.orderable(v.key, make(map[string]bool))
		}
		if err != nil {
			return nil, err
		}
		v.elem, err = c.tree(u.Elem(), o, at.step(valueStep))
	case *types.Pointer:
		v.kind = kindPointer
		v.elem, err = c.tree(u.Elem(), o, at)
	case *types.Interface:
		if declared == types.Universe.Lookup(anyName).Type() {
			v.typ = declared
		}
		return c.iface(v, o, at)
	default:
		return nil, fmt.Errorf("kanon: type %s is not supported", t)
	}
	if err != nil {
		return nil, err
	}
	if v.elem != nil {
		v.fails = v.fails || v.elem.fails || v.key != nil && v.key.fails
	}
	if v.validate && v.zeroOnly() {
		return nil, fmt.Errorf("kanon: ValidateKanon of type %s has nothing to check: the zero value is the only "+
			"value of the type", t)
	}
	return v, nil
}

// validator reports whether the named type t is a kanon.Validator: a
// directive names t, a type that is not a struct, as c.validated reports, or
// t has the method, as [validatorOf] reports. It fails as validatorOf
// fails.
func (c classifier) validator(t *types.Named) (bool, error) {
	if c.validated(t) {
		return true, nil
	}
	return validatorOf(t)
}

// validates reports whether the named type t is a kanon.Validator, as
// [classifier.validator] reports it, without its error, which the
// classification of a value of t returns.
func (c classifier) validates(t *types.Named) bool {
	ok, _ := c.validator(t)
	return ok
}

// id returns the id of a value of type t in a tree with the options o: the
// type string of t, followed by each option that changes the encoding of t.
// fixed changes it when t contains a 32- or 64-bit integer outside map keys
// or an interface, whose concrete types can contain one. A list and a map
// key change it when t contains an interface. Two values with one id encode
// alike, so a []string shares its functions across fields with and without
// options.
func (c classifier) id(t types.Type, o treeOpts) string {
	id := types.TypeString(t, nil)
	iface := c.within(t, true, make(map[*types.Named]bool), types.IsInterface)
	if o.fixed && (iface || c.within(t, false, make(map[*types.Named]bool), fixable)) {
		id += idSeparator + idFixed
	}
	if iface && o.list != nil {
		id += idSeparator + o.list.key()
	}
	if iface && o.key {
		id += idSeparator + idKey
	}
	return id
}

// treeOpts are the options that apply to the values of the type tree of one
// field.
type treeOpts struct {
	// used records that an integer of the tree takes the fixed-size
	// encoding that fixed selects.
	used *bool
	// list is the list of concrete types of the field's interfaces, and nil
	// for a field without the tag option types.
	list *typeList
	// placed records the concrete types of list that an interface of the tree
	// stores.
	placed map[*concrete]bool
	// fixed selects the fixed-size encoding for the 32- and 64-bit integers.
	fixed bool
	// key reports a value inside a map key, whose interfaces store the
	// comparable concrete types of list alone.
	key bool
}

// basic returns v, a value of the basic type u, with its kind set, and sets
// o.used when the fixed option of o applies to it. It fails for a basic type
// that kanon does not encode: an unsafe pointer.
func basic(v *value, u *types.Basic, o treeOpts) (*value, error) {
	switch u.Kind() {
	case types.Bool:
		v.kind = kindBool
	case types.String:
		v.kind = kindString
	case types.Float32:
		v.kind = kindFloat32
	case types.Float64:
		v.kind = kindFloat64
	case types.Complex64:
		v.kind = kindComplex64
	case types.Complex128:
		v.kind = kindComplex128
	case types.Int8, types.Int16, types.Int32, types.Int, types.Int64:
		v.kind = kindInt
	case types.Uint8, types.Uint16, types.Uint32, types.Uint, types.Uint64, types.Uintptr:
		v.kind = kindUint
	default:
		return nil, fmt.Errorf("kanon: type %s is not supported", v.typ)
	}
	if !o.fixed {
		return v, nil
	}
	switch u.Kind() {
	case types.Int32, types.Uint32:
		v.kind, *o.used = kindFixed32, true
	case types.Int64, types.Uint64:
		v.kind, *o.used = kindFixed64, true
	default:
		// The option applies to the 32- and 64-bit integers alone.
	}
	return v, nil
}

// fixable reports whether the tag option fixed changes the encoding of the
// basic type t: an int32, an int64, a uint32 or a uint64.
func fixable(t types.Type) bool {
	b, ok := t.(*types.Basic)
	if !ok {
		return false
	}
	switch b.Kind() {
	case types.Int32, types.Int64, types.Uint32, types.Uint64:
		return true
	default:
		return false
	}
}

// nameable reports whether the generated code of package pkg can name the
// type t, as [unnamed] reports it.
func nameable(t types.Type, pkg *types.Package) bool {
	return unnamed(t, pkg) == ""
}

// unnamed returns why the generated code of package pkg cannot name the type
// t, and "" when it can: the first named type in t that another package
// declares unexported, or the first field of a struct type or method of an
// interface type in t that another package declares unexported, with that
// package. A field or a method that another package declares unexported
// belongs to that package, so a type literal in pkg that spells it denotes
// another type.
func unnamed(t types.Type, pkg *types.Package) string {
	switch t := t.(type) {
	case *types.Named:
		if obj := t.Obj(); obj.Pkg() != nil && obj.Pkg() != pkg && !obj.Exported() {
			return unexported(obj, "type "+types.TypeString(t, nil))
		}
		return unnamedIn(slices.Collect(t.TypeArgs().Types()), pkg)
	case *types.Pointer:
		return unnamed(t.Elem(), pkg)
	case *types.Slice:
		return unnamed(t.Elem(), pkg)
	case *types.Array:
		return unnamed(t.Elem(), pkg)
	case *types.Chan:
		return unnamed(t.Elem(), pkg)
	case *types.Map:
		return unnamedIn([]types.Type{t.Key(), t.Elem()}, pkg)
	case *types.Struct:
		for f := range t.Fields() {
			if !visible(f, pkg) {
				return unexported(f, "field "+f.Name()+" of "+types.TypeString(t, nil))
			}
			if part := unnamed(f.Type(), pkg); part != "" {
				return part
			}
		}
		return ""
	case *types.Signature:
		return unnamedIn(slices.Concat(tupleTypes(t.Params()), tupleTypes(t.Results())), pkg)
	case *types.Interface:
		for m := range t.ExplicitMethods() {
			if !visible(m, pkg) {
				return unexported(m, "method "+m.Name()+" of "+types.TypeString(t, nil))
			}
			if part := unnamed(m.Type(), pkg); part != "" {
				return part
			}
		}
		return unnamedIn(slices.Collect(t.EmbeddedTypes()), pkg)
	case *types.Alias:
		return unnamed(types.Unalias(t), pkg)
	default:
		return ""
	}
}

// unnamedIn returns the first reason of [unnamed] for the types ts, and ""
// when pkg can name them all.
func unnamedIn(ts []types.Type, pkg *types.Package) string {
	for _, t := range ts {
		if part := unnamed(t, pkg); part != "" {
			return part
		}
	}
	return ""
}

// tupleTypes returns the types of the variables of the parameters or
// results tup.
func tupleTypes(tup *types.Tuple) []types.Type {
	var out []types.Type
	for v := range tup.Variables() {
		out = append(out, v.Type())
	}
	return out
}

// unexported returns the reason that the generated code cannot name what,
// a type, a field or a method that obj declares: its package does not
// export it.
func unexported(obj types.Object, what string) string {
	return "package " + obj.Pkg().Name() + " does not export " + what
}

// visible reports whether the code of package pkg can spell obj, a field or
// a method: obj is exported or declared in pkg.
func visible(obj types.Object, pkg *types.Package) bool {
	return obj.Exported() || obj.Pkg() == pkg
}

// isTime reports whether t is time.Time.
func isTime(t *types.Named) bool {
	obj := t.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == timePackage && obj.Name() == timeName
}

// isStruct reports whether the underlying type of t is a struct.
func isStruct(t types.Type) bool {
	_, ok := t.Underlying().(*types.Struct)
	return ok
}

// isByte reports whether t is byte itself. A slice or an array of a named
// type whose underlying type is byte is a sequence of integers, not bytes.
func isByte(t types.Type) bool {
	b, ok := types.Unalias(t).(*types.Basic)
	return ok && b.Kind() == types.Uint8
}

// isByteSlice reports whether the underlying type of t is a slice of byte
// itself, the type of the field that keeps unknown fields.
func isByteSlice(t types.Type) bool {
	s, ok := t.Underlying().(*types.Slice)
	return ok && isByte(s.Elem())
}
