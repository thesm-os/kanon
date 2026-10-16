// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"strconv"
)

// boundName is the name of the parameter of a read function that takes the
// bound of the tag option max, the most elements of its slice or its map.
const boundName = "bound"

// boundedKinds lists the kinds of the values whose length the tag option max
// bounds: a string and a byte slice by their bytes, and a slice and a map by
// their elements.
var boundedKinds = []kind{kindString, kindBytes, kindSlice, kindMap}

// errBoundKind is the error of the tag option max on a field whose length the
// generated code does not check: a field that is not a string, a byte slice,
// a slice or a map, and a type that encodes itself, whose decode method
// decodes the whole value.
var errBoundKind = errors.New("kanon: the tag option max applies to a string, a byte slice, a slice or a map, " +
	"other than a type that encodes itself")

// boundedValues returns the ids of the values of the fields of structs with
// the tag option max. The read function of a slice or a map of such an id
// takes a bound. The functions of the values of one id are shared, so a call
// for a value without the option, a field or a slice or a map in another
// value, passes math.MaxInt, and a read function that no such field calls
// takes no bound. A string and a byte slice have no read function, and the
// decode of their field checks the bound, as [emitter.lengthCheck] writes it.
func boundedValues(structs []*target) map[string]bool {
	ids := make(map[string]bool)
	for _, m := range structs {
		for _, f := range m.fields {
			if f.tag.max != 0 {
				ids[f.val.id] = true
			}
		}
	}
	return ids
}

// shortMaps returns the ids of the map values whose put functions take at
// most mapKeyBuffer entries: every place of such a value is a field of
// structs with the tag option max at mapKeyBuffer or less, whose encode fails
// for a larger map before it calls the function. A map in any other place,
// such as a field without the option or the element of a slice, takes any
// number of entries. The walk of the places does not enter an inline struct,
// whose fields structs lists.
func shortMaps(structs []*target) map[string]bool {
	short, long := make(map[string]bool), make(map[string]bool)
	seen := make(map[*value]bool)
	var inner func(v *value)
	inner = func(v *value) {
		if v == nil || seen[v] {
			return
		}
		seen[v] = true
		if v.kind == kindMap {
			long[v.id] = true
		}
		inner(v.elem)
		inner(v.key)
		for _, w := range v.variants {
			inner(w.val)
		}
	}
	for _, m := range structs {
		for _, f := range m.fields {
			v := f.val
			if v.kind != kindMap || f.tag.max == 0 || f.tag.max > mapKeyBuffer {
				inner(v)
				continue
			}
			short[v.id] = true
			inner(v.elem)
			inner(v.key)
		}
	}
	for id := range long {
		delete(short, id)
	}
	return short
}

// boundParam returns the bound among the parameters of the read function of
// v, before the comma and the space that follow depth: ", bound" when the
// function takes one, as e.bounded marks it, and nothing otherwise.
func (e *emitter) boundParam(v *value) string {
	if !e.bounded[v.id] {
		return ""
	}
	return ", " + boundName
}

// boundDoc returns the sentence of the docblock of the read function of v
// about its bound, after a space: that the function fails at item, an
// element or an entry, that would take whole, the slice or the map, past
// bound elements. It returns nothing when the function takes no bound.
func (e *emitter) boundDoc(v *value, item, whole string) string {
	if !e.bounded[v.id] {
		return ""
	}
	return " It fails at " + item + " that would take " + whole + " past " + boundName + " elements."
}

// boundArg returns the bound argument of a call of the read function of v
// and the comma after it: bound, the bound of the field that the call
// decodes, or math.MaxInt for a bound of 0, and nothing when the read
// function of v takes no bound, as e.bounded marks it.
func (e *emitter) boundArg(v *value, bound int) string {
	if !e.bounded[v.id] {
		return ""
	}
	if bound == 0 {
		return e.std(mathPath) + ".MaxInt, "
	}
	return strconv.Itoa(bound) + ", "
}

// boundCheck writes the statements of the encode of the field f of m, whose
// value is x, that fail for a value longer than the bound of the tag option
// max of f, before the statements that write the value. It writes nothing for
// a field without the option.
func (e *emitter) boundCheck(m *target, f *field, x string) {
	if f.tag.max == 0 {
		return
	}
	e.line("if len(%s) > %d {", x, f.tag.max)
	e.fail(e.wire() + "MarshalError(" + e.rt() + "ErrMax, " + fieldLoc(m, f) + ", " + strconv.Itoa(f.num) + ")")
	e.line("}")
}

// lengthCheck writes the statements that fail the read of the string or the
// byte slice at p when its length, the variable l, is above the bound of p.
// They fail with the error of wire.MaxError before the read copies a byte.
// The offset of the error is start plus the bound, where start is the
// expression of the offset of the first byte of the value. lengthCheck does
// not write a check for a place without a bound, or for a place whose value
// a scan checked before, as p.scanned reports.
func (e *emitter) lengthCheck(p place, start string) {
	if p.bound == 0 || p.scanned {
		return
	}
	bound := strconv.Itoa(p.bound)
	e.line("if l > %s {", bound)
	e.fail(e.wire() + "MaxError(" + p.loc + ", " + p.num + ", " + start + "+" + bound + ", " + bound + ")")
	e.line("}")
}
