// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"cmp"
	"go/types"
	"slices"
	"strconv"
)

// compareExpr returns the expression that orders a and b, two addressable
// map keys of v, as -1, 0 or +1: false before true for a bool; the order of
// cmp.Compare for a number and a string; the real parts, then the imaginary
// parts for a complex number; the instants, then the zones for a time;
// bytewise for a byte array; and the compare function of v's type for any
// other key. v has more than one value, as [classifier.single] reports: the
// callers order a key of one value without it, and leave out the parts of
// a key that have one value.
func (e *emitter) compareExpr(v *value, a, b string) string {
	switch v.kind {
	case kindBool:
		return e.wire() + "CompareBool(" + as(v, a, types.Bool) + ", " + as(v, b, types.Bool) + ")"
	case kindInt, kindUint, kindFixed32, kindFixed64, kindFloat32, kindFloat64, kindString:
		return e.std(cmpPath) + ".Compare(" + a + ", " + b + ")"
	case kindComplex64, kindComplex128:
		return e.wire() + "CompareComplex(" + as(v, a, types.Complex128) + ", " + as(v, b, types.Complex128) + ")"
	case kindTime:
		return e.wire() + "CompareTime(" + a + ", " + b + ")"
	case kindByteArray:
		return e.std(bytesPath) + ".Compare(" + primary(a) + "[:], " + primary(b) + "[:])"
	default:
		return e.fn(opCompare, v) + "(" + a + ", " + b + ")"
	}
}

// compareHelper writes the function that orders two map keys of v: a struct
// field by field in ascending field number, leaving out the fields of one
// value, which tie; an array element by element; a pointer nil first, then
// by the value that it points at, which a target of one value leaves out;
// an interface by the numbers of the concrete types, then by the values;
// and a value of a type that encodes itself by its encoding. A key has ==,
// so the encoding of such a type contains no map and does not depend on
// the order of the range of a map. Two keys of a type of one value tie,
// which the decode of an ambiguous key type, as [emitter.ambiguous]
// reports, orders them by; [emitter.compareExpr] orders such keys without a
// function, so any other struct has a field and an array an element that
// the order reads.
func (e *emitter) compareHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	e.doc(name + " returns -1, 0 or +1 as the map key a sorts before, with or after b, two keys of type " +
		typ + ".")
	if e.cls.single(v) {
		e.line("func %s(%s, %s) int {", name, typ, typ)
		e.line("return 0")
		e.line("}")
		e.line("")
		return
	}
	e.line("func %s(a, b %s) int {", name, typ)
	switch v.kind {
	case kindStruct:
		fields := e.orderedFields(v)
		for _, f := range fields[:len(fields)-1] {
			e.line("if c := %s; c != 0 {", e.compareExpr(f.val, "a."+f.name, "b."+f.name))
			e.line("return c")
			e.line("}")
		}
		last := fields[len(fields)-1]
		e.line("return %s", e.compareExpr(last.val, "a."+last.name, "b."+last.name))
	case kindArray:
		last := strconv.FormatInt(v.size-1, 10)
		if v.size > 1 {
			e.line("for k := range %s {", last)
			e.line("if c := %s; c != 0 {", e.compareExpr(v.elem, "a[k]", "b[k]"))
			e.line("return c")
			e.line("}")
			e.line("}")
		}
		e.line("return %s", e.compareExpr(v.elem, "a["+last+"]", "b["+last+"]"))
	case kindPointer:
		if e.cls.single(v.elem) {
			e.line("return %sCompareBool(a != nil, b != nil)", e.wire())
			break
		}
		e.line("if a == nil || b == nil {")
		e.line("return %sCompareBool(a != nil, b != nil)", e.wire())
		e.line("}")
		e.line("return %s", e.compareExpr(v.elem, "*a", "*b"))
	case kindInterface:
		e.compareInterface(v)
	default:
		if v.self.appender != "" {
			e.line("var sa, sb [%d]byte", binaryScratch)
			e.line("ea, _ := a.%s(sa[:0])", v.self.appender)
			e.line("eb, _ := b.%s(sb[:0])", v.self.appender)
		} else {
			e.line("ea, _ := a.%s()", v.self.marshaler)
			e.line("eb, _ := b.%s()", v.self.marshaler)
		}
		e.line("return %s.Compare(ea, eb)", e.std(bytesPath))
	}
	e.line("}")
	e.line("")
}

// compareInterface writes the statements of the compare function of the
// interface value of v, a map key: by the numbers of the concrete types,
// with a type that the list does not name, whose encode fails, as 0 like
// nil, then by the values of the concrete type. Two keys of one listed
// type reach the switch on the type, whose last case is its default. A
// concrete type of one value has no case: two keys of it are equal, so a
// map has one of them at most.
func (e *emitter) compareInterface(v *value) {
	e.line("var na, nb int")
	for _, x := range []string{"a", "b"} {
		e.line("switch %s.(type) {", x)
		for _, w := range v.variants {
			e.line("case %s:", e.p.typ(w.c.typ))
			e.line("n%s = %d", x, w.c.num)
		}
		e.line("}")
	}
	var ordered []variant
	for _, w := range v.variants {
		if !e.cls.single(w.val) {
			ordered = append(ordered, w)
		}
	}
	if len(ordered) == 0 {
		e.line("return %s.Compare(na, nb)", e.std(cmpPath))
		return
	}
	e.line("if na != nb || na == 0 {")
	e.line("return %s.Compare(na, nb)", e.std(cmpPath))
	e.line("}")
	last := len(ordered) - 1
	if last == 0 {
		e.compareVariant(ordered[0])
		return
	}
	e.line("switch a.(type) {")
	for k, w := range ordered {
		if k == last {
			e.line("default:")
		} else {
			e.line("case %s:", e.p.typ(w.c.typ))
		}
		e.compareVariant(w)
	}
	e.line("}")
}

// compareVariant writes the statements that return the order of a and b,
// two map keys of an interface that store values of the concrete type of
// w, as av and bv.
func (e *emitter) compareVariant(w variant) {
	t := e.p.typ(w.c.typ)
	e.line("av, bv := a.(%s), b.(%s)", t, t)
	e.line("return %s", e.compareExpr(w.val, "av", "bv"))
}

// orderedField is a field of the struct type of a map key, with its number.
type orderedField struct {
	name string
	num  int
	val  *value
}

// orderedFields returns the fields of the struct key v that order it, in
// ascending field number, without the fields of one value: the fields of
// the inline struct of v, and the fields of a struct with a kanon codec, as
// [classifier.keyFields] returns them, with the numbers that its record
// gives them.
func (e *emitter) orderedFields(v *value) []orderedField {
	var out []orderedField
	if v.inline != nil {
		for _, f := range v.inline.fields {
			out = append(out, orderedField{name: f.name, num: f.num, val: f.val})
		}
	} else {
		// The analysis checked that kanon orders the key, as
		// classifier.orderable states, so the fields classify.
		fields, _ := e.cls.keyFields(v.typ)
		nums := e.keyStructs[types.TypeString(v.typ, nil)].nums
		for _, f := range fields {
			out = append(out, orderedField{name: f.obj.Name(), num: nums[f.obj.Name()], val: f.val})
		}
	}
	out = slices.DeleteFunc(out, func(f orderedField) bool { return e.cls.single(f.val) })
	slices.SortFunc(out, func(a, b orderedField) int { return cmp.Compare(a.num, b.num) })
	return out
}
