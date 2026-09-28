// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"strconv"
	"strings"
)

// mathPath is the import path of package math, whose bit conversions of
// floats the generated code calls.
const mathPath = "math"

// Names of the method of time.Time that reports whether a time is at the
// zero instant, and of the method that returns its zone, and of the zone
// UTC, which the presence check of a time field reads.
const (
	isZeroName   = "IsZero"
	locationName = "Location"
	utcName      = "UTC"
)

// sizeMethod writes the SizeKanon method of the -type struct m, with the
// body that [emitter.sizeBody] writes after the check of a nil receiver.
func (e *emitter) sizeMethod(m *target) {
	e.line("// SizeKanon returns the length of the encoding of m in bytes, and 0 for a")
	e.line("// nil m.")
	e.line("func (m *%s) SizeKanon() int {", e.p.typ(m.typ))
	e.line("if m == nil {")
	e.line("return 0")
	e.line("}")
	e.sizeBody(m)
}

// sizeBody writes the statements that return the length of the encoding of
// the struct that m points at, and the end of the function. They add each
// field that is not a union member when it is present, one switch per
// union on its discriminator, and the unknown fields that m keeps. They
// apply the presence rules of the encode, so that the two agree on every
// value.
func (e *emitter) sizeBody(m *target) {
	e.line("n := 0")
	done := make(map[*types.Var]bool)
	for _, f := range m.fields {
		if f.member == nil {
			if !f.val.zeroOnly() {
				e.sizeField(f)
			}
			continue
		}
		if done[f.member.disc] {
			continue
		}
		done[f.member.disc] = true
		e.line("switch m.%s {", f.member.disc.Name())
		for _, g := range f.member.union {
			e.line("case %s:", e.p.object(g.member.value))
			e.sizeMember(g)
		}
		e.line("}")
	}
	if m.unknown != nil {
		e.line("n += len(m.%s)", m.unknown.Name())
	}
	e.line("return n")
	e.line("}")
	e.line("")
}

// sizeField writes the statements that add the length of the field f, a
// field that is not a union member, to n when f is present: a struct and a
// type that encodes itself when their encoding has bytes, a pointer and an
// interface when they are not nil, and any other field as
// [emitter.present] states.
func (e *emitter) sizeField(f *field) {
	x, v, ts := "m."+f.name, f.val, tagSize(f.num)
	switch v.kind {
	case kindStruct, kindBinary:
		e.line("if s := %s; s > 0 {", e.content(v, x))
		e.line("n += %d + %sSizeBytes(s)", ts, e.wire())
		e.line("}")
	case kindPointer:
		e.line("if %s != nil {", x)
		e.line("n += %s", plus(ts, e.fieldSize(v.elem, deref(x))))
		e.line("}")
	default:
		e.line("if %s {", e.present(v, x))
		e.line("n += %s", plus(ts, e.fieldSize(v, x)))
		e.line("}")
	}
}

// sizeMember writes the statements that add the length of the union member
// f, which its discriminator selects, to n. A selected member is present
// whatever its value, and a nil pointer member encodes the zero value of
// the type that it points at, as [emitter.memberTarget] sets it.
func (e *emitter) sizeMember(f *field) {
	x, v, ts := "m."+f.name, f.val, tagSize(f.num)
	if v.kind != kindPointer {
		e.line("n += %s", plus(ts, e.fieldSize(v, x)))
		return
	}
	if c, ok := constSize(v.elem); ok {
		e.line("n += %d", ts+c)
		return
	}
	e.memberTarget(f)
	e.line("n += %s", plus(ts, e.fieldSize(v.elem, "*x")))
}

// memberTarget writes the statements that set x to the value that the
// pointer member f points at, or to a new zero value of its type when f is
// nil, which the member encodes when its discriminator selects it. The new
// value does not escape, so it takes no allocation.
func (e *emitter) memberTarget(f *field) {
	e.line("x := m.%s", f.name)
	e.line("if x == nil {")
	e.line("x = new(%s)", e.p.typ(f.val.elem.typ))
	e.line("}")
}

// fieldSize returns the expression of the length of x, the value of v that
// the tag of a field introduces: the length of the encoding of x, and of
// that encoding with its length for a pointer and an interface, which
// [value.unframed] reports.
func (e *emitter) fieldSize(v *value, x string) string {
	if v.unframed() {
		return e.wire() + "SizeBytes(" + e.sizeOf(v, x) + ")"
	}
	return e.sizeOf(v, x)
}

// sizeOf returns the expression of the length of the encoding of x, an
// addressable value of v: with its length, its presence byte or its type
// number. The length is a constant when every value of v has one, as
// [constSize] reports.
func (e *emitter) sizeOf(v *value, x string) string {
	if c, ok := constSize(v); ok {
		return strconv.Itoa(c)
	}
	w := e.wire()
	switch v.kind {
	case kindInt:
		return w + "SizeUvarint(" + w + "Zigzag(" + as(v, x, types.Int64) + "))"
	case kindUint:
		return w + "SizeUvarint(" + as(v, x, types.Uint64) + ")"
	case kindString, kindBytes:
		return w + "SizeBytes(len(" + x + "))"
	case kindTime:
		return w + "SizeBytes(" + w + "SizeTime(" + x + "))"
	case kindStruct, kindBinary:
		return w + "SizeBytes(" + e.content(v, x) + ")"
	case kindSlice:
		if c, ok := constSize(v.elem); ok {
			return w + "SizeBytes(" + times("len("+x+")", c) + ")"
		}
		return w + "SizeBytes(" + e.fn(opSize, v) + "(" + x + "))"
	case kindArray:
		return w + "SizeBytes(" + e.fn(e.keyOp(opSize, v), v) + "(" + addr(x) + "))"
	case kindMap:
		kc, kok := constSize(v.key)
		vc, vok := constSize(v.elem)
		if kok && vok {
			return w + "SizeBytes(" + times("len("+x+")", kc+vc) + ")"
		}
		return w + "SizeBytes(" + e.fn(opSize, v) + "(" + x + "))"
	default:
		return e.fn(e.keyOp(opSize, v), v) + "(" + x + ")"
	}
}

// content returns the expression of the length of the encoding of x, a
// struct or a value of a type that encodes itself, without its length: the
// SizeKanon method of a struct with a kanon codec, and the size function of
// the code file otherwise, which a struct with a kanon codec takes too in
// the projection of a map key, as [emitter.keyed] reports.
func (e *emitter) content(v *value, x string) string {
	if v.kind == kindStruct && v.inline == nil && !e.keyed(v) {
		return method(x, sizeKanonName) + "()"
	}
	return e.fn(e.keyOp(opSize, v), v) + "(" + addr(x) + ")"
}

// present returns the condition under which a field whose value is x, a
// value of v, is present: a bool when it is true, a number when it is not
// zero, a float and a complex number when a bit of them is set, so that
// negative zero is present, a string, a byte slice, a slice and a map when
// they are not empty, a time when it is not at the zero instant or not in
// UTC, so that the zone of a time at the zero instant survives, an array
// when an element is present, a struct and a type that encodes itself when
// their encoding has bytes, and a pointer and an interface when they are
// not nil. In the projection of a map key, which writes -0.0 as +0.0, a
// float and a complex number are present when they are not zero. The
// condition is an operand of || without parentheses.
func (e *emitter) present(v *value, x string) string {
	switch v.kind {
	case kindBool:
		return x
	case kindInt, kindUint, kindFixed32, kindFixed64:
		return x + " != 0"
	case kindFloat32, kindFloat64, kindComplex64, kindComplex128:
		if e.inKey {
			return x + " != 0"
		}
		return e.floatBits(v, x) + " != 0"
	case kindString:
		return x + ` != ""`
	case kindBytes, kindSlice, kindMap:
		return "len(" + x + ") > 0"
	case kindTime:
		return "!" + method(x, isZeroName) + "() || " + method(x, locationName) + "() != " +
			e.std(timePackage) + "." + utcName
	case kindByteArray:
		return x + " != " + e.composite(v.typ)
	case kindArray:
		if bitwise(v) {
			return x + " != " + e.composite(v.typ)
		}
		return e.fn(e.keyOp(opPresent, v), v) + "(" + addr(x) + ")"
	case kindStruct, kindBinary:
		return e.content(v, x) + " > 0"
	default:
		return x + " != nil"
	}
}

// floatBits returns the expression of the bits of x, a float or a complex
// number of v, which are zero exactly when x is positive zero: the bits of a
// float, and the bits of the real part or-ed with the bits of the imaginary
// part of a complex number.
func (e *emitter) floatBits(v *value, x string) string {
	m := e.std(mathPath)
	switch v.kind {
	case kindFloat32:
		return m + ".Float32bits(" + as(v, x, types.Float32) + ")"
	case kindFloat64:
		return m + ".Float64bits(" + as(v, x, types.Float64) + ")"
	case kindComplex64:
		return m + ".Float32bits(real(" + x + "))|" + m + ".Float32bits(imag(" + x + "))"
	default:
		return m + ".Float64bits(real(" + x + "))|" + m + ".Float64bits(imag(" + x + "))"
	}
}

// sizeHelper writes the size function of the values of v: the length of the
// encoding of an inline struct or of a value of a type that encodes itself,
// of the elements of a slice or an array, and of the keys and values of a
// map, without their length; and the length of the encoding of a pointer or
// an interface.
func (e *emitter) sizeHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	switch v.kind {
	case kindStruct:
		m := e.structTarget(v)
		e.doc(name + " returns the length of the encoding of the " + m.name + " that m points at" + e.inKeyDoc() +
			".")
		e.line("func %s(m *%s) int {", name, typ)
		e.sizeBody(m)
		return
	case kindBinary:
		e.doc(name + " returns the length of the encoding of the " + typ + " that x points at, and 1 when x" +
			" fails to encode itself, so that the value is present and its encode reports the failure.")
		e.line("func %s(x *%s) int {", name, typ)
		e.selfEncode(v, "x", "err")
		e.line("if err != nil {")
		e.line("return 1")
		e.line("}")
		e.line("return len(enc)")
	case kindSlice, kindArray:
		e.doc(name + " returns the length of the encoding of the elements of x, a " + typ + e.inKeyDoc() + ".")
		e.line("func %s(x %s) int {", name, param(v, typ))
		e.line("n := 0")
		e.line("for k := range x {")
		e.line("n += %s", e.sizeOf(v.elem, "x[k]"))
		e.line("}")
		e.line("return n")
	case kindMap:
		e.doc(name + " returns the length of the encoding of the keys and values of x, a " + typ + ".")
		e.line("func %s(x %s) int {", name, typ)
		e.sizeMap(v)
	case kindPointer:
		e.doc(name + " returns the length of the encoding of x, a " + typ + e.inKeyDoc() + ": its presence byte" +
			" and the value that it points at.")
		e.line("func %s(x %s) int {", name, typ)
		e.line("if x == nil {")
		e.line("return 1")
		e.line("}")
		e.line("return %s", plus(1, e.sizeOf(v.elem, "*x")))
	default:
		e.doc(name + " returns the length of the encoding of x, a " + typ + e.inKeyDoc() + ": the number of its" +
			" concrete type and its value. It returns 0 for a concrete type that the tag option types does not" +
			" list, whose encode fails.")
		e.line("func %s(x %s) int {", name, typ)
		e.sizeInterface(v)
	}
	e.line("}")
	e.line("")
}

// sizeMap writes the statements of the size function of the map type of v.
// The variables of the loop are declared before it: the compiler moves a
// variable of a loop body to the heap when the size method of a recursive
// struct takes its address.
func (e *emitter) sizeMap(v *value) {
	kc, kok := constSize(v.key)
	vc, vok := constSize(v.elem)
	if kok {
		e.line("n := %s", times("len(x)", kc))
		e.line("var mv %s", e.p.typ(v.elem.typ))
		e.line("for _, mv = range x {")
		e.line("n += %s", e.sizeOf(v.elem, "mv"))
	} else if vok {
		e.line("n := %s", times("len(x)", vc))
		e.line("var mk %s", e.p.typ(v.key.typ))
		e.line("for mk = range x {")
		e.line("n += %s", e.keySizeOf(v.key, "mk"))
	} else {
		e.line("n := 0")
		e.line("var mk %s", e.p.typ(v.key.typ))
		e.line("var mv %s", e.p.typ(v.elem.typ))
		e.line("for mk, mv = range x {")
		e.line("n += %s + %s", e.keySizeOf(v.key, "mk"), e.sizeOf(v.elem, "mv"))
	}
	e.line("}")
	e.line("return n")
}

// keySizeOf returns the expression of the length of the projection of x, a
// map key of v: [emitter.sizeOf] in the mode of a map key.
func (e *emitter) keySizeOf(v *value, x string) string {
	e.inKey = true
	s := e.sizeOf(v, x)
	e.inKey = false
	return s
}

// inKeyDoc returns the words that the doc comment of a helper of a map key
// adds after the value that it writes, and nothing for any other helper.
func (e *emitter) inKeyDoc() string {
	if e.inKey {
		return ", as the projection of a map key, which writes every float component of -0.0 as +0.0"
	}
	return ""
}

// sizeInterface writes the statements of the size function of the
// interface value of v: one type switch over its concrete types, in tag
// order, after nil when v is nilable. The switch binds the value when the
// length of a concrete type depends on it, since Go rejects a bound value
// that no case reads.
func (e *emitter) sizeInterface(v *value) {
	reads := false
	for _, w := range v.variants {
		if _, ok := constSize(w.val); !ok {
			reads = true
		}
	}
	if reads {
		e.line("switch x := x.(type) {")
	} else {
		e.line("switch x.(type) {")
	}
	if v.nilable {
		e.line("case nil:")
		e.line("return 1")
	}
	for _, w := range v.variants {
		e.line("case %s:", e.p.typ(w.c.typ))
		e.line("return %s", plus(varintSize(uint64(w.c.num)), e.sizeOf(w.val, "x")))
	}
	e.line("default:")
	e.line("return 0")
	e.line("}")
}

// presentHelper writes the function that reports whether a field of the
// array type of v is present: whether an element of it is. The loop tests
// the elements up to the first present one, and has no branch that the
// values of an element type cannot reach, such as the end of an array of a
// type whose every value is present.
func (e *emitter) presentHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	e.doc(name + " reports whether a field whose value is the " + typ + " that x points at is present" +
		e.inKeyDoc() + ": whether an element of it is.")
	e.line("func %s(x *%s) bool {", name, typ)
	e.line("present := false")
	e.line("for k := range x {")
	e.line("present = present || %s", e.present(v.elem, "x[k]"))
	e.line("}")
	e.line("return present")
	e.line("}")
	e.line("")
}

// constSize returns the length of the encoding of every value of v and
// true, when all values of v encode to one length, and false otherwise: a
// bool, a fixed-size number, a complex128, a byte array, an array of values
// of one length, and a value that [value.zeroOnly] reports.
func constSize(v *value) (int, bool) {
	if v.zeroOnly() {
		return len(v.constant()), true
	}
	switch v.kind {
	case kindBool:
		return 1, true
	case kindFixed32, kindFloat32, kindFixed64, kindFloat64, kindComplex64:
		return v.kind.width(), true
	case kindComplex128:
		return 1 + complex128Width, true
	case kindByteArray:
		return varintSize(uint64(v.size)) + int(v.size), true
	case kindArray:
		c, ok := constSize(v.elem)
		n := int(v.size) * c
		return varintSize(uint64(n)) + n, ok
	default:
		return 0, false
	}
}

// bitwise reports whether the values of v compare with == exactly as their
// presence decides: a value equals the zero value of its type exactly when
// a field of it is absent. It reports true for a bool, an integer, a string,
// a byte array, a pointer, and an array of such values. A float differs at
// negative zero, and a struct, an interface and a type that encodes itself
// at values that encode to no bytes or panic under ==.
func bitwise(v *value) bool {
	switch v.kind {
	case kindBool, kindInt, kindUint, kindFixed32, kindFixed64, kindString, kindByteArray, kindPointer:
		return true
	case kindArray:
		return bitwise(v.elem)
	default:
		return false
	}
}

// param returns the parameter type of the helpers of v, whose Go type is
// typ: a pointer to an array, which a call does not copy, and typ itself
// for any other type.
func param(v *value, typ string) string {
	if v.kind == kindArray {
		return "*" + typ
	}
	return typ
}

// method returns the expression of the method name of x, an addressable
// value: p.name for a dereference *p, and x.name otherwise. Go takes the
// address of x for a method of the pointer type.
func method(x, name string) string {
	if p, ok := strings.CutPrefix(x, "*"); ok {
		return primary(p) + "." + name
	}
	return x + "." + name
}

// plus returns the expression of c added to x: the sum when x is an
// integer constant.
func plus(c int, x string) string {
	if n, err := strconv.Atoi(x); err == nil {
		return strconv.Itoa(c + n)
	}
	return strconv.Itoa(c) + " + " + x
}

// times returns the expression of x multiplied by c: x itself for a c of 1.
func times(x string, c int) string {
	if c == 1 {
		return x
	}
	return x + " * " + strconv.Itoa(c)
}
