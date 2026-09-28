// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"slices"
	"strconv"
	"strings"
)

// Names of the functions of package wire that write, check and merge the
// projections of map keys.
const (
	keyFloatName    = "KeyFloat"
	invalidKeyName  = "InvalidKeyError"
	keyTiesName     = "KeyTies"
	pairTiesName    = "PairTies"
	oneKeyName      = "OneKey"
	keyErrorName    = "KeyError"
	mergeKeysName   = "MergeKeys"
	keepLastName    = "KeepLastKeys"
	decodedTimeName = "DecodedTime"
	compareTimeName = "CompareTime"
)

// floats reports whether the projection of v, a map key or a part of one,
// has a float component: v is a float or a complex number, or contains one
// in an array element, the value that a pointer points at, an encoded field
// of a struct or a concrete type of an interface. The projection writes such
// a component of -0.0 as +0.0, and a NaN component makes the key invalid.
func (e *emitter) floats(v *value) bool {
	return e.keyReaches(v, func(v *value) bool {
		switch v.kind {
		case kindFloat32, kindFloat64, kindComplex64, kindComplex128:
			return true
		default:
			return false
		}
	})
}

// ambiguous reports whether two map keys of v can differ under == and have
// one projection, or order as one: v is or contains, in an array element or
// an encoded field of a struct, a pointer, whose identity the projection
// leaves out, a time, whose zone name and monotonic clock reading it leaves
// out, a type that encodes itself, a struct with a field that the encoding
// leaves out, or an interface, which can store a type that its list does
// not name, which orders as nil.
func (e *emitter) ambiguous(v *value) bool {
	return e.keyReaches(v, func(v *value) bool {
		switch v.kind {
		case kindPointer, kindTime, kindBinary, kindInterface:
			return true
		case kindStruct:
			return v.hidden
		default:
			return false
		}
	})
}

// keyReaches reports whether match accepts v or a part of v that the
// projection of a map key contains: the element of an array, the value that
// a pointer points at, an encoded field of a struct and a concrete type of
// an interface. The walk visits each value once, so that a key type that
// contains itself behind a pointer ends it.
func (e *emitter) keyReaches(v *value, match func(*value) bool) bool {
	seen := make(map[string]bool)
	var walk func(v *value) bool
	walk = func(v *value) bool {
		if seen[v.id] {
			return false
		}
		seen[v.id] = true
		return match(v) || slices.ContainsFunc(e.keyParts(v), walk)
	}
	return walk(v)
}

// keyParts returns the parts of v, a map key or a part of one, that its
// projection contains, as [emitter.keyReaches] walks them. An array of no
// elements, which has one value, has none.
func (e *emitter) keyParts(v *value) []*value {
	switch v.kind {
	case kindArray:
		if v.size == 0 {
			return nil
		}
		return []*value{v.elem}
	case kindPointer:
		return []*value{v.elem}
	case kindInterface:
		out := make([]*value, 0, len(v.variants))
		for _, w := range v.variants {
			out = append(out, w.val)
		}
		return out
	case kindStruct:
		var out []*value
		for _, f := range e.orderedFields(v) {
			out = append(out, f.val)
		}
		return out
	default:
		return nil
	}
}

// keyed reports whether the emitter writes x, a value of v, as part of the
// projection of a map key that differs from its encoding as a value: it
// writes a map key, as inKey reports, and v contains a float.
func (e *emitter) keyed(v *value) bool {
	return e.inKey && e.floats(v)
}

// keyOp returns o, opSize, opPut or opPresent, or the op of a map key for
// it when the emitter writes v as part of the projection of a map key, as
// [emitter.keyed] reports.
func (e *emitter) keyOp(o op, v *value) op {
	if !e.keyed(v) {
		return o
	}
	switch o {
	case opSize:
		return opKeySize
	case opPut:
		return opKeyPut
	default:
		return opKeyPresent
	}
}

// structTarget returns the target whose fields the size and the put
// functions of v, a struct, write: the inline struct of v, and in a map key
// the target of a struct with a kanon codec that [emitter.keyTarget] builds.
func (e *emitter) structTarget(v *value) *target {
	if v.inline != nil {
		return v.inline
	}
	return e.keyTarget(v)
}

// keyTarget returns the target that writes the projection of v, a struct
// with a kanon codec in a map key that contains a float, field by field: its
// fields that [classifier.keyFields] returns, with the numbers of its record.
// The code of the struct writes a float field of -0.0 as present, so the
// projection does not come from its methods. The target names the struct by
// its type name, as the errors of its own code do.
func (e *emitter) keyTarget(v *value) *target {
	if m := e.keyTargets[v.id]; m != nil {
		return m
	}
	named, _ := types.Unalias(v.typ).(*types.Named)
	m := &target{typ: v.typ, fset: e.cls.fset, name: named.Obj().Name(), fails: v.fails}
	// The analysis checked that kanon orders the key, as
	// classifier.orderable states, so the fields classify.
	fields, _ := e.cls.keyFields(v.typ)
	nums := e.keyStructs[types.TypeString(v.typ, nil)].nums
	for _, f := range fields {
		m.fields = append(m.fields, &field{
			obj: f.obj, name: f.obj.Name(), val: f.val, tag: f.tag, num: nums[f.obj.Name()],
		})
	}
	e.keyTargets[v.id] = m
	return m
}

// floatWord returns the expression of the bits of x, a float32 or a float64
// operand of the given bit size: the bits of +0.0 for -0.0 in the
// projection of a map key, and the bits of x otherwise.
func (e *emitter) floatWord(size int, x string) string {
	if e.inKey {
		return e.wire() + keyFloatName + strconv.Itoa(size) + "(" + x + ")"
	}
	return e.std(mathPath) + ".Float" + strconv.Itoa(size) + "bits(" + x + ")"
}

// nanExpr returns the condition that x, a map key or a part of one of v, has
// a NaN component: x != x for a float and a complex number, and the call of
// the nan function of the code file for any other value that contains one.
func (e *emitter) nanExpr(v *value, x string) string {
	switch v.kind {
	case kindFloat32, kindFloat64, kindComplex64, kindComplex128:
		return x + " != " + x
	default:
		return e.fn(opNaN, v) + "(" + x + ")"
	}
}

// nanHelper writes the function that reports whether a map key of v has a
// NaN component, in the parts of v that contain a float: the fields of a
// struct, the elements of an array, the value that a pointer points at, and
// the value of each concrete type of an interface.
func (e *emitter) nanHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	e.doc(name + " reports whether x, a map key of type " + typ + ", has a NaN component, which makes the key " +
		"invalid.")
	e.line("func %s(x %s) bool {", name, typ)
	switch v.kind {
	case kindStruct:
		var conds []string
		for _, f := range e.orderedFields(v) {
			if e.floats(f.val) {
				conds = append(conds, e.nanExpr(f.val, "x."+f.name))
			}
		}
		e.line("return %s", strings.Join(conds, " || "))
	case kindArray:
		e.line("for k := range x {")
		e.line("if %s {", e.nanExpr(v.elem, "x[k]"))
		e.line("return true")
		e.line("}")
		e.line("}")
		e.line("return false")
	case kindPointer:
		e.line("return x != nil && %s", e.nanExpr(v.elem, "*x"))
	default:
		e.line("switch x := x.(type) {")
		for _, w := range v.variants {
			if e.floats(w.val) {
				e.line("case %s:", e.p.typ(w.c.typ))
				e.line("return %s", e.nanExpr(w.val, "x"))
			}
		}
		e.line("}")
		e.line("return false")
	}
	e.line("}")
	e.line("")
}

// canonFunc returns the function that reports whether a map key of v, which
// [emitter.ambiguous] reports, is the key that a decode of its projection
// yields: wire.DecodedTime for a time, and the canon function of the code
// file for any other key.
func (e *emitter) canonFunc(v *value) string {
	if v.kind == kindTime {
		return e.wire() + decodedTimeName
	}
	return e.fn(opCanon, v)
}

// canonExpr returns the condition that x, a map key or a part of one of v,
// is the value that a decode of its projection yields: true for a value
// whose keys of one projection are equal, false for a pointer and a type
// that encodes itself, whose decode yields a new value, and the call of
// [emitter.canonFunc] otherwise.
func (e *emitter) canonExpr(v *value, x string) string {
	if !e.ambiguous(v) {
		return trueName
	}
	if v.kind == kindPointer || v.kind == kindBinary {
		return falseName
	}
	return e.canonFunc(v) + "(" + x + ")"
}

// canonHelper writes the function that reports whether a map key of v is
// the key that a decode of its projection yields, so that two such keys of
// one projection are equal: false for a pointer and a type that encodes
// itself; for a struct, when its fields that the encoding leaves out have
// their zero values and its encoded fields are such values; for an array,
// when its elements are; and for an interface, when it is nil or its value
// is one.
func (e *emitter) canonHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	e.doc(name + " reports whether x, a map key of type " + typ + ", is the key that a decode of its projection " +
		"yields, so that the map of a decode needs no other check to keep one entry for the keys of one projection.")
	e.line("func %s(x %s) bool {", name, typ)
	switch v.kind {
	case kindStruct:
		e.line("return %s", e.canonStruct(v))
	case kindArray:
		c := e.canonExpr(v.elem, "x[k]")
		if c == falseName {
			e.line("return false")
			break
		}
		e.line("for k := range x {")
		e.line("if !%s {", c)
		e.line("return false")
		e.line("}")
		e.line("}")
		e.line("return true")
	case kindInterface:
		conds := make([]string, len(v.variants))
		binds := false
		for k, w := range v.variants {
			conds[k] = e.canonExpr(w.val, "x")
			binds = binds || conds[k] != trueName && conds[k] != falseName
		}
		if binds {
			e.line("switch x := x.(type) {")
		} else {
			e.line("switch x.(type) {")
		}
		e.line("case nil:")
		e.line("return true")
		for k, w := range v.variants {
			e.line("case %s:", e.p.typ(w.c.typ))
			e.line("return %s", conds[k])
		}
		e.line("}")
		e.line("return false")
	default:
		e.line("return false")
	}
	e.line("}")
	e.line("")
}

// canonStruct returns the condition of the canon function of v, a struct:
// every field that the encoding leaves out has its zero value, and every
// encoded field, of one value or more, is the value that a decode yields.
// It is false when the code of the file cannot spell a field that the
// encoding leaves out, or when an encoded field is never such a value.
func (e *emitter) canonStruct(v *value) string {
	leftOut := leftOutFields(v.typ)
	if !e.spells(leftOut) {
		return falseName
	}
	var conds []string
	for _, obj := range leftOut {
		conds = append(conds, "x."+obj.Name()+" == "+e.zeroOperand(obj.Type()))
	}
	for _, f := range e.structTarget(v).fields {
		switch c := e.canonExpr(f.val, "x."+f.name); c {
		case trueName:
		case falseName:
			return falseName
		default:
			conds = append(conds, c)
		}
	}
	return strings.Join(conds, " && ")
}

// zeroOperand returns the zero value of the type t as an operand of ==: the
// zero value that [emitter.zero] returns, in parentheses for a composite
// literal, which Go does not parse as an operand otherwise.
func (e *emitter) zeroOperand(t types.Type) string {
	z := e.zero(t)
	if strings.HasSuffix(z, "{}") {
		return "(" + z + ")"
	}
	return z
}

// compareFunc returns the function that orders two map keys of v, which
// [emitter.ambiguous] reports, by projection: wire.CompareTime for a time,
// and the compare function of the code file for any other key.
func (e *emitter) compareFunc(v *value) string {
	if v.kind == kindTime {
		return e.wire() + compareTimeName
	}
	return e.fn(opCompare, v)
}

// keyChecks writes the statements of a put function of the map type of v
// that check its keys after they sort, and reports whether they declare err,
// which the function returns and which ends the loop that writes the
// entries: a key with a NaN component, as [emitter.floats] finds it, returns
// kanon.ErrInvalidKey, and two keys of one projection, which an ambiguous
// key type, as [emitter.ambiguous] reports, can have and which sort next to
// each other, set err to the error of wire.KeyTies. sorted names the slice
// of the sorted keys, of pairs when pairs is set, and key returns the
// expression of its key k. The check of the projections is a call without a
// branch of its own, since no input of a type that encodes itself to
// distinct bytes reaches it.
func (e *emitter) keyChecks(v *value, sorted string, pairs bool, key func(k string) string) bool {
	w := e.wire()
	if e.floats(v.key) {
		e.line("for k := range len(%s) {", sorted)
		e.line("if %s {", e.nanExpr(v.key, key("k")))
		e.line("return 0, %s%s(%s, %s)", w, invalidKeyName, locParam, numParam)
		e.line("}")
		e.line("}")
	}
	if !e.ambiguous(v.key) {
		return false
	}
	ties := keyTiesName
	if pairs {
		ties = pairTiesName
	}
	e.line("err := %s%s(%s, %s, %s, %s)", w, ties, sorted, e.compareFunc(v.key), locParam, numParam)
	return true
}
