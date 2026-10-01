// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"iter"
)

// holdsString reports whether decoding a value of type t can take a string
// from the slab: t is a string or an interface, whose concrete types can be
// strings, or contains one in its nested structs and composite types. The
// walk does not enter a type that encodes itself, which decodes without the
// slab.
func (c classifier) holdsString(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Basic:
			return t.Info()&types.IsString != 0, false
		case *types.Interface:
			return true, false
		case *types.Named:
			return false, !c.binaryType(t)
		default:
			return false, true
		}
	})
}

// holdsInterface reports whether a value of type t can contain an
// interface: t is an interface or contains one in its nested structs and
// composite types. The walk does not enter a type that encodes itself,
// which a map key orders by its encoding.
func (c classifier) holdsInterface(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Interface:
			return true, false
		case *types.Named:
			return false, !c.binaryType(t)
		default:
			return false, true
		}
	})
}

// mayFail reports whether encoding a value of type t can fail: t is or
// contains a type that encodes itself, whose encode method returns an
// error, a kanon.Validator whose ValidateKanon the code calls, as
// [classifier.checks] reports, which returns one, an interface,
// which can store a type that its list does not name, or a map whose keys
// can have a NaN component or share a projection, as [classifier.floats]
// and [classifier.ambiguous] report. A struct fails as
// [classifier.fieldsMayFail] reports. seen marks the named types visited.
func (c classifier) mayFail(t types.Type, seen map[*types.Named]bool) bool {
	var match func(types.Type) (bool, bool)
	match = func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Interface:
			return true, false
		case *types.Named:
			return c.binaryType(t) || c.checks(t), true
		case *types.Map:
			return c.floats(t.Key()) || c.ambiguous(t.Key()), true
		case *types.Struct:
			return c.fieldsMayFail(t, seen, match), false
		default:
			return false, true
		}
	}
	return c.reaches(t, seen, match)
}

// fieldsMayFail reports whether encoding a field of the struct type st can
// fail, as match reports it for the type of the field, for the fields that
// [walked] returns. A field that is not a union member and whose type
// declares kanon.Exact, as [exactType] reports, cannot fail, as
// [exactField] states.
func (c classifier) fieldsMayFail(st *types.Struct, seen map[*types.Named]bool,
	match func(types.Type) (bool, bool),
) bool {
	for fld, tg := range walked(st) {
		if tg.union == "" && exactType(fld.Type()) {
			continue
		}
		if c.reaches(fld.Type(), seen, match) {
			return true
		}
	}
	return false
}

// exactType reports whether t is a named type that declares ExactKanon(),
// the marker of kanon.Exact. The classification of a value of t fails for
// such a type that does not meet the requirements of kanon.Exact, as
// [classifier.exactable] states.
func exactType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && methodsOf(named).has(exactKanonName, nil, nil)
}

// exactField reports whether a field whose value is v and whose tag is t
// encodes without an error path: a field that is not a union member, of a
// type that declares kanon.Exact. The encoding leaves out the zero value of
// the field, and kanon.Exact guarantees the encode of every other value.
func exactField(v *value, t tag) bool {
	return t.union == "" && v.kind == kindBinary && v.self.exact
}

// indirect reports whether a value of the struct type t, named or
// anonymous, refers to memory outside itself: a pointer, a slice, a map or
// an interface among its encoded fields, at any depth of the structs it
// nests by value. The walk does not enter a type that encodes itself, which
// decodes into a zero value.
func (c classifier) indirect(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Pointer, *types.Slice, *types.Map, *types.Interface:
			return true, false
		case *types.Named:
			return false, !c.binaryType(t)
		default:
			return false, true
		}
	})
}

// hidden reports whether a value of type t can contain state that its
// encoding leaves out: t is a struct with such a field, as [leftOutFields]
// returns them, or contains one by value, in an encoded field or as an
// array element, at any depth. The walk does not enter a pointer, a slice, a
// map or an interface, whose values are elsewhere in memory, nor a time or a
// type that encodes itself, which a reset sets whole.
func (c classifier) hidden(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Named:
			return false, !isTime(t) && !c.binaryType(t)
		case *types.Array:
			return false, t.Len() > 0
		case *types.Struct:
			return len(leftOutFields(t)) > 0, true
		default:
			return false, false
		}
	})
}

// leftOutFields returns the fields of the struct type t that the encoding
// leaves out, as [target.leftOut] lists them for a struct that the code file
// analyses: the fields tagged "-", the unexported fields without a kanon
// tag, and the fields that [unencoded] reports, without blank fields.
// A field whose tag does not parse, which the analysis of its struct
// rejects, counts as encoded.
func leftOutFields(t types.Type) []*types.Var {
	st, _ := t.Underlying().(*types.Struct)
	var out []*types.Var
	for i := range st.NumFields() {
		fld := st.Field(i)
		tg, err := parseTag(st.Tag(i))
		if fld.Name() == blankName || err != nil {
			continue
		}
		if !reads(fld, tg) || unencoded(fld.Type()) || tg.skip {
			out = append(out, fld)
		}
	}
	return out
}

// floats reports whether the projection of a map key of type t has a float
// component: t is a float or a complex number, or contains one in an array
// element, the value that a pointer points at, or an encoded field of a
// struct. The walk does not enter an interface, whose concrete types the
// tag option types lists, a time or a type that encodes itself, nor an
// array of no elements, which has one value.
func (c classifier) floats(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Basic:
			return t.Info()&(types.IsFloat|types.IsComplex) != 0, false
		case *types.Named:
			return false, !isTime(t) && !c.binaryType(t)
		case *types.Interface:
			return false, false
		case *types.Array:
			return false, t.Len() > 0
		default:
			return false, true
		}
	})
}

// ambiguous reports whether two map keys of type t can differ under == and
// have one projection, or order as one: t is or contains, in an array
// element or an encoded field of a struct, a pointer, a time, a type that
// encodes itself, a struct with a field that the encoding leaves out, or an
// interface, which can store a type that its list does not name, which
// orders as nil. The walk does not enter an array of no elements, which has
// one value.
func (c classifier) ambiguous(t types.Type) bool {
	return c.reaches(t, make(map[*types.Named]bool), func(t types.Type) (bool, bool) {
		switch t := t.(type) {
		case *types.Pointer, *types.Interface:
			return true, false
		case *types.Named:
			return isTime(t) || c.binaryType(t), true
		case *types.Struct:
			return len(leftOutFields(t)) > 0, true
		case *types.Array:
			return false, t.Len() > 0
		default:
			return false, true
		}
	})
}

// hollow reports whether the zero value is the only value of the type t, as
// [value.zeroOnly] reports it for a value: t is an array of no elements or
// of elements of one value, or a struct without a union and without the
// field that keeps unknown fields, whose encoded fields have one value. A
// time, a type that encodes itself, and a struct with a kanon codec that
// kanon does not generate have more than one value, as every other type
// has.
func (c classifier) hollow(t types.Type) bool {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		if isTime(named) || c.binaryType(named) || c.nested(named) && !c.generated(named) {
			return false
		}
	}
	switch u := t.Underlying().(type) {
	case *types.Array:
		return u.Len() == 0 || c.hollow(u.Elem())
	case *types.Struct:
		for i := range u.NumFields() {
			fld := u.Field(i)
			tg, err := parseTag(u.Tag(i))
			if !reads(fld, tg) || unencoded(fld.Type()) || err == nil && tg.skip {
				continue
			}
			if err != nil || tg.unknown || tg.union != "" || !c.hollow(fld.Type()) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// binaryType reports whether t encodes itself through a family of methods:
// its pointer has them, and t is neither a nested struct, nor time.Time,
// which kanon encodes as seconds, nanoseconds and a zone offset, nor a
// kanon.Validator, which kanon encodes as its underlying type.
func (c classifier) binaryType(t *types.Named) bool {
	if isTime(t) || c.nested(t) || c.validates(t) {
		return false
	}
	_, ok := selfCodecOf(t)
	return ok
}

// reaches reports whether the type tree of t contains a type that match
// accepts. match returns whether it accepts a type and whether the walk
// enters it. The walk enters the underlying type of a named type, the
// element of a pointer, a slice or an array, the key and the value of a
// map, and the fields of a struct that [walked] returns. seen marks the
// named types visited, so that a type that contains itself ends the walk.
func (c classifier) reaches(t types.Type, seen map[*types.Named]bool, match func(types.Type) (bool, bool)) bool {
	t = types.Unalias(t)
	if named, ok := t.(*types.Named); ok {
		if seen[named] {
			return false
		}
		seen[named] = true
	}
	found, enter := match(t)
	if found || !enter {
		return found
	}
	switch t := t.(type) {
	case *types.Named:
		return c.reaches(t.Underlying(), seen, match)
	case *types.Pointer:
		return c.reaches(t.Elem(), seen, match)
	case *types.Slice:
		return c.reaches(t.Elem(), seen, match)
	case *types.Array:
		return c.reaches(t.Elem(), seen, match)
	case *types.Map:
		return c.reaches(t.Key(), seen, match) || c.reaches(t.Elem(), seen, match)
	case *types.Struct:
		for fld := range walked(t) {
			if c.reaches(fld.Type(), seen, match) {
				return true
			}
		}
	}
	return false
}

// walked returns the fields of the struct type st that the walks of a type
// tree enter, with their tags, in declaration order: the fields that kanon
// reads, as [reads] reports, whose tag parses and has no `kanon:"-"`. The
// analysis of the struct rejects a tag that does not parse.
func walked(st *types.Struct) iter.Seq2[*types.Var, tag] {
	return func(yield func(*types.Var, tag) bool) {
		for i := range st.NumFields() {
			fld := st.Field(i)
			tg, err := parseTag(st.Tag(i))
			if !reads(fld, tg) || err != nil || tg.skip {
				continue
			}
			if !yield(fld, tg) {
				return
			}
		}
	}
}

// within reports whether the values of type t that take the options of one
// field contain a type that match accepts. The walk enters the underlying
// type of a named type, the element of a pointer, a slice or an array, the
// value of a map, and the key of a map when keys is set. It stops at
// structs, times and types that encode themselves, whose values take
// options of their own, and at interfaces, whose concrete types the options
// list. seen marks the named types visited.
func (c classifier) within(t types.Type, keys bool, seen map[*types.Named]bool, match func(types.Type) bool) bool {
	t = types.Unalias(t)
	if match(t) {
		return true
	}
	switch t := t.(type) {
	case *types.Named:
		if seen[t] || isStruct(t) || c.nested(t) || c.binaryType(t) {
			return false
		}
		seen[t] = true
		return c.within(t.Underlying(), keys, seen, match)
	case *types.Pointer:
		return c.within(t.Elem(), keys, seen, match)
	case *types.Slice:
		return c.within(t.Elem(), keys, seen, match)
	case *types.Array:
		return c.within(t.Elem(), keys, seen, match)
	case *types.Map:
		return keys && c.within(t.Key(), keys, seen, match) || c.within(t.Elem(), keys, seen, match)
	default:
		return false
	}
}
