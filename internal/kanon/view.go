// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"strconv"
)

// byteSliceType is the result type of the view method of a string, a byte
// slice and a struct without a view type of its own: the bytes of the
// field, which alias the view.
const byteSliceType = "[]byte"

// viewType writes the view type of the -type struct m, the encoding of an
// m, and a method per exported field of m that is not a union member and
// that [viewed] reports, which reads the field from the encoding. An
// unexported field that a kanon tag opts into the encoding has no method,
// since a view exposes the fields of m to other packages.
func (e *emitter) viewType(m *target) {
	name := m.name + viewSuffix
	e.doc(name + " is the encoding of a " + m.name + ". Each method reads one field by scanning the " +
		"encoding, without decoding the rest, and returns the zero value when the encoding has no such " +
		"field, and the last value when it has several. A method of a struct fails with " +
		"kanon.ErrRepeatedView for a second occurrence instead, since a decode merges the occurrences. The " +
		"bytes that a method returns for a string, a byte slice and a struct alias the view, and the offsets " +
		"of its errors are offsets in the view.")
	e.line("type %s []byte", name)
	e.line("")
	for _, f := range m.fields {
		if f.obj.Exported() && f.member == nil && viewed(f.val) {
			e.viewMethod(m, name, f)
		}
	}
}

// viewMethod writes the method of the view type view that reads the field f
// of m, or the value that f points at for a pointer field. wire.Find skips
// the value of every occurrence of the field, so the value that it returns
// is complete, and the method checks only what the type of the field adds:
// the range of an integer, the length of an array and of a complex number,
// and the fields of a time. The method of a struct finds the field with
// wire.FindOne, which fails for a second occurrence, since one byte slice
// cannot hold the merge of two encodings.
func (e *emitter) viewMethod(m *target, view string, f *field) {
	v := f.val
	if v.kind == kindPointer {
		v = v.elem
	}
	result := e.viewResult(v)
	zero := nilName
	find := "Find"
	if v.kind == kindStruct {
		find = "FindOne"
	} else if result != byteSliceType {
		zero = e.zero(v.typ)
	}
	w := e.wire()
	loc, num := fieldLoc(m, f), strconv.Itoa(f.num)
	e.doc(f.name + " returns the value of the field " + f.name + " of the encoding in v.")
	e.line("func (v %s) %s() (%s, error) {", view, f.name, result)
	e.line("i, err := %s%s(v, %s, %s)", w, find, e.tag(f.num, v.kind.wire()), loc)
	e.line("if err != nil || i < 0 {")
	e.line("return %s, err", zero)
	e.line("}")
	e.ret = zero + ", "
	switch v.kind {
	case kindBool, kindInt, kindUint:
		e.line("u, _ := %sUvarint(v[i:])", w)
		e.line("return %s, nil", e.varint(v, "u", place{loc: loc, num: num, at: "i"}))
	case kindFixed32, kindFloat32:
		e.line("u, _ := %sUint32(v[i:])", w)
		e.line("return %s, nil", e.fixed(v, "u"))
	case kindFixed64, kindFloat64, kindComplex64:
		e.line("u, _ := %sUint64(v[i:])", w)
		e.line("return %s, nil", e.fixed(v, "u"))
	case kindTime:
		e.line("l, n := %sUvarint(v[i:])", w)
		e.line("return %sTime(v[i+n:i+n+int(l)], %s, %s, i+n)", w, loc, num)
	case kindComplex128:
		m := e.std(mathPath)
		e.viewLength(complex128Width, loc, num)
		e.line("re, _ := %sUint64(v[i+n:])", w)
		e.line("im, _ := %sUint64(v[i+n+%d:])", w, fixed64Width)
		e.line("return %s, nil", e.cast(v, "complex("+m+".Float64frombits(re), "+m+".Float64frombits(im))",
			types.Complex128))
	case kindByteArray:
		e.viewLength(v.size, loc, num)
		e.line("var x %s", result)
		e.line("copy(x[:], v[i+n:])")
		e.line("return x, nil")
	default:
		e.line("l, n := %sUvarint(v[i:])", w)
		e.line("return %s(v[i+n : i+n+int(l)]), nil", result)
	}
	e.line("}")
	e.line("")
}

// viewLength writes the statements of a view method that read the length l
// of the value at v[i], of n bytes, and reject a length other than size.
func (e *emitter) viewLength(size int64, loc, num string) {
	w := e.wire()
	e.line("l, n := %sUvarint(v[i:])", w)
	e.line("if l != %d {", size)
	e.fail(w + "LengthError(l, " + strconv.FormatInt(size, 10) + ", " + loc + ", " + num + ", i)")
	e.line("}")
}

// viewResult returns the result type of the view method of a field whose
// value, or the value that it points at, is v: the bytes of a string and a
// byte slice, the view type of a struct that the code file declares one
// for and the bytes of any other struct, and the type of v otherwise.
func (e *emitter) viewResult(v *value) string {
	switch v.kind {
	case kindString, kindBytes:
		return byteSliceType
	case kindStruct:
		if named, ok := types.Unalias(v.typ).(*types.Named); ok && v.inline == nil && e.views[named.Obj()] {
			return named.Obj().Name() + viewSuffix
		}
		return byteSliceType
	default:
		return e.p.typ(v.typ)
	}
}

// viewed reports whether a view type has a method for a field whose value is
// v: a bool, a number, a string, a byte slice, a byte array, a time or a
// struct, or a pointer to one of them.
func viewed(v *value) bool {
	if v.kind == kindPointer {
		v = v.elem
	}
	switch v.kind {
	case kindBool, kindInt, kindUint, kindFixed32, kindFixed64, kindFloat32, kindFloat64, kindComplex64,
		kindComplex128, kindString, kindBytes, kindByteArray, kindTime, kindStruct:
		return true
	default:
		return false
	}
}
