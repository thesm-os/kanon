// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import "go/types"

// Import paths of the standard-library packages whose copy functions the
// clone of a struct calls.
const (
	stringsPath = "strings"
	mapsPath    = "maps"
)

// cloneMethods writes the CloneKanon method of the -type struct m and its
// unexported clone method, which the clone of the structs of the package
// that contain one calls.
func (e *emitter) cloneMethods(m *target) {
	typ := e.p.typ(m.typ)
	e.doc("CloneKanon returns a copy of m that shares no memory with m or with the slab that m decoded " +
		"from: its strings, slices, maps and the values of its pointers and interfaces are copies. The copy " +
		"has the fields of the encoding, the union discriminators and the unknown fields of m, and the zero " +
		"value in every other field, map keys included. A nil m returns nil.")
	e.line("func (m *%s) CloneKanon() *%s {", typ, typ)
	e.line("if m == nil {")
	e.line("return nil")
	e.line("}")
	e.line("c := new(%s)", typ)
	e.line("m.%s(c)", cloneMethod)
	e.line("return c")
	e.line("}")
	e.line("")
	e.doc(cloneMethod + " copies m into c, as CloneKanon does. The fields of c that the copy does not set keep " +
		"their values. CloneKanon and the clone of the structs of the package that contain " + m.name +
		" call it.")
	e.line("func (m *%s) %s(c *%s) {", typ, cloneMethod, typ)
	e.cloneBody(m, "c", "m")
}

// cloneBody writes the statements that copy the struct that src points at
// into the one that dst points at, and the end of the function: every field
// of the encoding, the discriminators and the unknown fields.
func (e *emitter) cloneBody(m *target, dst, src string) {
	for _, f := range m.fields {
		e.cloneInto(f.val, dst+"."+f.name, src+"."+f.name)
	}
	for _, d := range m.discriminators {
		e.line("%s.%s = %s.%s", dst, d.Name(), src, d.Name())
	}
	if u := m.unknown; u != nil {
		e.line("%s.%s = %s.Clone(%s.%s)", dst, u.Name(), e.std(slicesPath), src, u.Name())
	}
	e.line("}")
	e.line("")
}

// cloneInto writes the statements that copy src, an addressable value of v,
// into dst, an addressable value of v that has its zero value, without
// sharing memory: the assignment of the copy that [emitter.cloneExpr]
// returns, and otherwise the call that copies src into the address of dst.
// That call is the clone method of a -type struct of the package, and the
// clone function of the code file for an inline struct, a type that encodes
// itself and an array.
func (e *emitter) cloneInto(v *value, dst, src string) {
	if c, ok := e.cloneExpr(v, src); ok {
		e.line("%s = %s", dst, c)
		return
	}
	if v.kind == kindStruct && v.inline == nil {
		e.line("%s(%s)", method(src, cloneMethod), addr(dst))
		return
	}
	e.line("%s(%s, %s)", e.fn(opClone, v), addr(dst), addr(src))
}

// cloneExpr returns the expression of a copy of src, an addressable value of
// v, that shares no memory with src, and reports whether the copy is an
// expression:
//
//   - src itself for a value that [emitter.deep] does not report;
//   - strings.Clone for a string;
//   - slices.Clone for a byte slice and a slice of such values, and
//     maps.Clone for a map of them;
//   - CloneKanon for a struct with a kanon codec that kanon generates in
//     another package or that a hand-written codec provides;
//   - the clone function of the code file for any other slice and map, a
//     pointer and an interface.
//
// A copy of an inline struct, a -type struct of the package, a type that
// encodes itself and an array is no expression: [emitter.cloneInto] copies
// it into the address of its destination.
func (e *emitter) cloneExpr(v *value, src string) (string, bool) {
	if !e.deep(v) {
		return src, true
	}
	switch v.kind {
	case kindString:
		return e.cast(v, e.std(stringsPath)+".Clone("+as(v, src, types.String)+")", types.String), true
	case kindBytes:
		return e.std(slicesPath) + ".Clone(" + src + ")", true
	case kindStruct:
		if v.inline != nil || e.generates(v.typ) {
			return "", false
		}
		return "*" + method(src, cloneKanonName) + "()", true
	case kindBinary, kindArray:
		return "", false
	case kindSlice:
		if !e.deep(v.elem) {
			return e.std(slicesPath) + ".Clone(" + src + ")", true
		}
		return e.fn(opClone, v) + "(" + src + ")", true
	case kindMap:
		if !e.deep(v.key) && !e.deep(v.elem) {
			return e.std(mapsPath) + ".Clone(" + src + ")", true
		}
		return e.fn(opClone, v) + "(" + src + ")", true
	default:
		return e.fn(opClone, v) + "(" + src + ")", true
	}
}

// cloneValue returns the expression of a copy of src, a value of v: the
// expression that [emitter.cloneExpr] returns, or name, a variable that the
// statements it writes declare and copy src into.
func (e *emitter) cloneValue(v *value, name, src string) string {
	if c, ok := e.cloneExpr(v, src); ok {
		return c
	}
	e.line("var %s %s", name, e.p.typ(v.typ))
	e.cloneInto(v, name, src)
	return name
}

// deep reports whether a copy of a value of v needs more than an assignment:
// v is a string, a byte slice, a slice, a map, a pointer or an interface, a
// struct that contains one of them or state that its encoding leaves out,
// which the copy leaves zero, an array of values that need more, or a value
// of a type that encodes itself and refers to memory. An inert value, as
// [value.inert] reports, needs an assignment at most.
func (e *emitter) deep(v *value) bool {
	if v.inert() {
		return false
	}
	switch v.kind {
	case kindString, kindBytes, kindSlice, kindMap, kindPointer, kindInterface:
		return true
	case kindStruct:
		return v.indirect || v.hidden || e.cls.holdsString(v.typ)
	case kindArray:
		return e.deep(v.elem)
	case kindBinary:
		return refers(v.typ, make(map[*types.Named]bool))
	default:
		return false
	}
}

// cloneHelper writes the clone function of the values of v.
func (e *emitter) cloneHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	switch v.kind {
	case kindStruct:
		e.doc(name + " copies the " + v.inline.name + " that src points at into the one that dst points at, " +
			"as CloneKanon does.")
		e.line("func %s(dst, src *%s) {", name, typ)
		e.cloneBody(v.inline, "dst", "src")
		return
	case kindBinary:
		e.doc(name + " copies the " + typ + " that src points at into dst through its encode and decode " +
			"methods, and by assignment when they fail.")
		e.line("func %s(dst, src *%s) {", name, typ)
		e.line("*dst = *src")
		if v.self.appender != "" {
			e.line("enc, err := src.%s(nil)", v.self.appender)
		} else {
			e.line("enc, err := src.%s()", v.self.marshaler)
		}
		e.line("if err != nil {")
		e.line("return")
		e.line("}")
		e.line("var c %s", typ)
		e.line("if c.%s(enc) == nil {", v.self.unmarshaler)
		e.line("*dst = c")
		e.line("}")
	case kindArray:
		e.doc(name + " copies the " + typ + " that src points at into the one that dst points at, element by " +
			"element.")
		e.line("func %s(dst, src *%s) {", name, typ)
		e.line("for k := range src {")
		e.cloneInto(v.elem, "dst[k]", "src[k]")
		e.line("}")
	case kindSlice:
		e.doc(name + " returns a copy of x, a " + typ + ", and nil for a nil x.")
		e.line("func %s(x %s) %s {", name, typ, typ)
		e.line("if x == nil {")
		e.line("return nil")
		e.line("}")
		e.line("c := make(%s, len(x))", typ)
		e.line("for k := range x {")
		e.cloneInto(v.elem, "c[k]", "x[k]")
		e.line("}")
		e.line("return c")
	case kindMap:
		e.doc(name + " returns a copy of x, a " + typ + ", and nil for a nil x.")
		e.line("func %s(x %s) %s {", name, typ, typ)
		e.line("if x == nil {")
		e.line("return nil")
		e.line("}")
		e.line("c := make(%s, len(x))", typ)
		e.line("for mk, mv := range x {")
		ck := e.cloneValue(v.key, "ck", "mk")
		cv := e.cloneValue(v.elem, "cv", "mv")
		e.line("c[%s] = %s", ck, cv)
		e.line("}")
		e.line("return c")
	case kindPointer:
		e.doc(name + " returns a pointer to a copy of the value that x, a " + typ + ", points at, and nil for a " +
			"nil x.")
		e.line("func %s(x %s) %s {", name, typ, typ)
		e.line("if x == nil {")
		e.line("return nil")
		e.line("}")
		e.clonePointer(v.elem)
	default:
		e.doc(name + " returns a copy of the value that x, a " + typ + ", stores. A value of a type that the " +
			"tag option types does not list, whose encode fails, is not copied.")
		e.line("func %s(x %s) %s {", name, typ, typ)
		e.line("switch x := x.(type) {")
		for _, c := range v.variants {
			e.line("case %s:", e.p.typ(c.c.typ))
			e.line("return %s", e.cloneValue(c.val, "c", "x"))
		}
		e.line("default:")
		e.line("return x")
		e.line("}")
	}
	e.line("}")
	e.line("")
}

// clonePointer writes the statements of the clone function of a pointer to
// a value of v that return a pointer to a copy of *x: the result of
// CloneKanon for a struct with a kanon codec that [emitter.deep] reports, a
// new variable that holds the copy that [emitter.cloneExpr] returns, and
// otherwise a new variable that the copy writes into.
func (e *emitter) clonePointer(v *value) {
	if v.kind == kindStruct && v.inline == nil && e.deep(v) {
		e.line("return %s()", method("*x", cloneKanonName))
		return
	}
	if c, ok := e.cloneExpr(v, "*x"); ok {
		e.line("return new(%s)", c)
		return
	}
	e.line("c := new(%s)", e.p.typ(v.typ))
	e.cloneInto(v, "*c", "*x")
	e.line("return c")
}

// refers reports whether a value of type t refers to memory: t is or
// contains a string, a pointer, a slice, a map, a channel, a function or an
// interface, in any field of a struct, exported or not. seen marks the named
// types visited.
func refers(t types.Type, seen map[*types.Named]bool) bool {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		if seen[t] {
			return false
		}
		seen[t] = true
		return refers(t.Underlying(), seen)
	case *types.Basic:
		return t.Info()&types.IsString != 0
	case *types.Array:
		return refers(t.Elem(), seen)
	case *types.Struct:
		for f := range t.Fields() {
			if refers(f.Type(), seen) {
				return true
			}
		}
		return false
	default:
		return true
	}
}
