// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"strings"
)

// resetMethod writes the Reset method of the -type struct m, with the body
// that [emitter.resetBody] writes after the check of a nil receiver.
func (e *emitter) resetMethod(m *target) {
	e.doc("Reset clears every field of m, the fields that the encoding leaves out included. An encoded pointer " +
		"or interface becomes nil. An encoded slice or map becomes empty and keeps its storage for a decode to " +
		"reuse, with every element in the capacity of a slice reset, so that no decoded content remains in m " +
		"or in the storage that it keeps.")
	e.line("func (m *%s) Reset() {", e.p.typ(m.typ))
	e.line("if m == nil {")
	e.line("return")
	e.line("}")
	e.resetBody(m)
}

// resetBody writes the statements that reset the struct that m points at,
// and the end of the function: the fields that the encoding leaves out,
// every encoded field, the discriminators and the unknown fields, with the
// spare capacity of every slice.
func (e *emitter) resetBody(m *target) {
	e.clearLeftOut(m, func(f *field) bool {
		return f.val.kind != kindPointer && f.val.kind != kindInterface && f.val.holdsMemory()
	})
	for _, f := range m.fields {
		e.resetField(f, true)
	}
	for _, d := range m.discriminators {
		e.line("m.%s = 0", d.Name())
	}
	if u := m.unknown; u != nil {
		e.line("clear(m.%s[:cap(m.%s)])", u.Name(), u.Name())
		e.line("m.%s = m.%s[:0]", u.Name(), u.Name())
	}
	e.line("}")
	e.line("")
}

// clearLeftOut writes the statements that set the fields of the struct that
// m points at that the encoding leaves out, as [target.leftOut] lists them,
// to their zero values. When the code of the file can spell each such field
// and its type, it assigns each field its zero value. Otherwise, for a
// struct of another package with an unexported field, it assigns the
// struct a composite literal of its type that sets only the encoded fields
// that keep reports and the field that keeps unknown fields, each to its own
// value, so that their memory remains for the statements that follow.
func (e *emitter) clearLeftOut(m *target, keep func(*field) bool) {
	if !e.spells(m.leftOut) {
		var kept []string
		for _, f := range m.fields {
			if keep(f) {
				kept = append(kept, f.name+": m."+f.name)
			}
		}
		if u := m.unknown; u != nil {
			kept = append(kept, u.Name()+": m."+u.Name())
		}
		e.line("*m = %s{%s}", e.p.typ(m.typ), strings.Join(kept, ", "))
		return
	}
	for _, obj := range m.leftOut {
		e.line("m.%s = %s", obj.Name(), e.zero(obj.Type()))
	}
}

// spells reports whether the code of the file can spell each field of
// fields and its type: the field is exported or declared in the package of
// the file, and the file can name its type.
func (e *emitter) spells(fields []*types.Var) bool {
	for _, obj := range fields {
		if !visible(obj, e.cls.pkg) || !nameable(obj.Type(), e.cls.pkg) {
			return false
		}
	}
	return true
}

// resetField writes the statements that reset the field f: a pointer
// becomes nil, a slice and a byte slice become empty and keep their
// capacity, and any other field resets as [emitter.resetValue] states. With
// spare set, the statements also reset every element of the capacity of a
// slice and clear the spare bytes of a byte slice. Without it, a slice
// keeps its elements for the decode, which decodes into them and resets
// them in doing so.
func (e *emitter) resetField(f *field, spare bool) {
	x, v := "m."+f.name, f.val
	switch v.kind {
	case kindPointer:
		e.line("%s = nil", x)
	case kindSlice, kindBytes:
		if spare {
			e.resetValue(v, x)
			return
		}
		e.line("%s = %s[:0]", x, x)
	default:
		e.resetValue(v, x)
	}
}

// resetValue writes the statements that set x, an addressable value of v,
// to its zero value and keep its memory: a byte slice and a slice become
// empty and clear their capacity, a map is cleared, a struct resets in
// place, and a pointer keeps the value that it points at and resets it, or
// sets it to nil when it is a pointer in turn, which can point at itself. An
// array of values that keep no memory takes the zero value of its type, and
// so does any other value. An inert value, as [value.inert] reports, has
// its zero value always, and takes no statement.
func (e *emitter) resetValue(v *value, x string) {
	if v.inert() {
		return
	}
	switch v.kind {
	case kindStruct:
		if v.inline != nil {
			e.line("%s(%s)", e.fn(opReset, v), addr(x))
			return
		}
		e.line("%s()", method(x, resetName))
	case kindBytes:
		e.line("clear(%s[:cap(%s)])", primary(x), x)
		e.line("%s = %s[:0]", x, primary(x))
	case kindSlice:
		if !v.elem.holdsMemory() {
			e.line("clear(%s[:cap(%s)])", primary(x), x)
			e.line("%s = %s[:0]", x, primary(x))
			return
		}
		e.line("%s(%s)", e.fn(opReset, v), addr(x))
	case kindMap:
		e.line("clear(%s)", x)
	case kindArray:
		if !v.holdsMemory() {
			e.line("%s = %s", x, e.zero(v.typ))
			return
		}
		e.line("%s(%s)", e.fn(opReset, v), addr(x))
	case kindPointer:
		e.line("if %s != nil {", x)
		if v.elem.kind == kindPointer {
			e.line("%s = nil", deref(x))
		} else {
			e.resetValue(v.elem, deref(x))
		}
		e.line("}")
	default:
		e.line("%s = %s", x, e.zero(v.typ))
	}
}

// resetHelper writes the reset function of the values of v: of an inline
// struct, of a slice whose elements keep memory, and of an array whose
// elements keep memory.
func (e *emitter) resetHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	switch v.kind {
	case kindStruct:
		e.doc(name + " clears the " + v.inline.name + " that m points at and keeps the memory that a decode " +
			"reuses, as Reset does.")
		e.line("func %s(m *%s) {", name, typ)
		e.resetBody(v.inline)
		return
	case kindSlice:
		e.doc(name + " empties the " + typ + " that x points at and resets every element of its capacity, " +
			"which keeps its memory for the next decode.")
		e.line("func %s(x *%s) {", name, typ)
		e.line("s := (*x)[:cap(*x)]")
		e.line("for k := range s {")
		e.resetValue(v.elem, "s[k]")
		e.line("}")
		e.line("*x = (*x)[:0]")
	default:
		e.doc(name + " sets the " + typ + " that x points at to its zero value, which has nil pointers, and " +
			"keeps the memory of the slices and maps of its elements.")
		e.line("func %s(x *%s) {", name, typ)
		e.line("for k := range x {")
		if v.elem.kind == kindPointer {
			e.line("x[k] = nil")
		} else {
			e.resetValue(v.elem, "x[k]")
		}
		e.line("}")
	}
	e.line("}")
	e.line("")
}
