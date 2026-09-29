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

// indexKanonName names the method of a view type that returns its index.
const indexKanonName = "IndexKanon"

// viewType writes the view type of the -type struct m, the encoding of an
// m, and a method per exported field of m that is not a union member and
// that [viewed] reports, which reads the field from the encoding, and then
// the index type of the view, as [emitter.indexType] writes it. An
// unexported field that a kanon tag opts into the encoding has no method,
// since a view exposes the fields of m to other packages.
func (e *emitter) viewType(m *target) {
	name := m.name + viewSuffix
	var fields []*field
	for _, f := range m.fields {
		if f.obj.Exported() && f.member == nil && viewed(f.val) {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		e.doc(name + " is the encoding of a " + m.name + ", which has no field that a view reads, so " + name +
			" has no method that reads a field. " + indexKanonName + " returns the index of the view.")
	} else {
		e.doc(name + " is the encoding of a " + m.name + ". Each method reads one field by scanning the " +
			"encoding, without decoding the rest, and returns the zero value when the encoding has no such " +
			"field, and the last value when it has several. A method of a struct fails with " +
			"kanon.ErrRepeatedView for a second occurrence instead, since a decode merges the occurrences. The " +
			"bytes that a method returns for a string, a byte slice and a struct alias the view. A value of a " +
			"type that encodes itself decodes with the method of its type, and the offsets of the errors of a " +
			"method are offsets in the view. " + indexKanonName + " reads the fields of the view with one scan.")
	}
	e.line("type %s []byte", name)
	e.line("")
	for _, f := range fields {
		e.viewMethod(m, name, f)
	}
	e.indexType(m, name, fields)
}

// viewMethod writes the method of the view type view that reads the field f
// of m, or the value that f points at for a pointer field. wire.Find skips
// the value of every occurrence of the field, so the value that it returns
// is complete, and the method reads it as [emitter.viewRead] states. The
// method of a struct finds the field with wire.FindOne, which fails for a
// second occurrence, since one byte slice cannot contain the merge of two
// encodings.
func (e *emitter) viewMethod(m *target, view string, f *field) {
	v, result, zero := e.viewParts(f)
	find := "Find"
	if v.kind == kindStruct {
		find = "FindOne"
	}
	doc := f.name + " returns the value of the field " + f.name + " of the encoding in v."
	if v.validate {
		doc += " It returns the error of " + validateKanonName + " for a value that the method rejects."
	}
	e.doc(doc)
	e.line("func (v %s) %s() (%s, error) {", view, f.name, result)
	e.line("i, err := %s%s(v, %s, %s)", e.wire(), find, e.tag(f.num, v.kind.wire()), fieldLoc(m, f))
	e.line("if err != nil || i < 0 {")
	e.line("return %s, err", zero)
	e.line("}")
	e.viewRead(m, f)
	e.line("}")
	e.line("")
}

// viewParts returns the value of the field f of a view, or the value that
// f points at for a pointer field, the result type of its view method, and
// the expression of the zero value of that type: nil for the bytes and the
// view type of a struct.
func (e *emitter) viewParts(f *field) (*value, string, string) {
	v := f.val
	if v.kind == kindPointer {
		v = v.elem
	}
	result := e.viewResult(v)
	if v.kind == kindStruct || result == byteSliceType {
		return v, result, nilName
	}
	return v, result, e.zero(v.typ)
}

// viewRead writes the statements that read the value of the field f of m
// at offset i of the view v and return it, the body of a method of the view
// type and of the index type after the offset. The offset is that of the
// value of a complete occurrence of the field, so the statements check only
// what the type of the field adds: the range of an integer, the length of
// an array and of a complex number, and the fields of a time. A type that
// encodes itself decodes the bytes of the field with the decode method of
// its family, whose error is the cause of the error of the method.
func (e *emitter) viewRead(m *target, f *field) {
	v, result, zero := e.viewParts(f)
	w := e.wire()
	loc, num := fieldLoc(m, f), strconv.Itoa(f.num)
	e.ret = zero + ", "
	switch v.kind {
	case kindBool, kindInt, kindUint:
		e.line("u, _ := %sUvarint(v[i:])", w)
		e.viewReturn(v, e.varint(v, "u", place{loc: loc, num: num, at: "i"}), loc, num)
	case kindFixed32, kindFloat32:
		e.line("u, _ := %sUint32(v[i:])", w)
		e.viewReturn(v, e.fixed(v, "u"), loc, num)
	case kindFixed64, kindFloat64, kindComplex64:
		e.line("u, _ := %sUint64(v[i:])", w)
		e.viewReturn(v, e.fixed(v, "u"), loc, num)
	case kindTime:
		e.line("l, n := %sUvarint(v[i:])", w)
		e.line("return %sTime(v[i+n:i+n+int(l)], %s, %s, i+n)", w, loc, num)
	case kindComplex128:
		math := e.std(mathPath)
		e.viewLength(complex128Width, loc, num)
		e.line("re, _ := %sUint64(v[i+n:])", w)
		e.line("im, _ := %sUint64(v[i+n+%d:])", w, fixed64Width)
		e.viewReturn(v, e.cast(v, "complex("+math+".Float64frombits(re), "+math+".Float64frombits(im))",
			types.Complex128), loc, num)
	case kindByteArray:
		e.viewLength(v.size, loc, num)
		e.line("var x %s", result)
		e.line("copy(x[:], v[i+n:])")
		e.viewReturn(v, "x", loc, num)
	case kindBinary:
		e.line("l, n := %sUvarint(v[i:])", w)
		e.line("var x %s", result)
		e.line("if err := %s(v[i+n : i+n+int(l)]); err != nil {", method("x", v.self.unmarshaler))
		e.fail(w + "UnmarshalError(err, " + loc + ", " + num + ", i+n)")
		e.line("}")
		e.line("return x, nil")
	default:
		e.line("l, n := %sUvarint(v[i:])", w)
		e.viewReturn(v, result+"(v[i+n : i+n+int(l)])", loc, num)
	}
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
// for and the bytes of any other struct, and the type of v otherwise. A
// string or a byte slice of a kanon.Validator takes its type, whose
// ValidateKanon the method calls.
func (e *emitter) viewResult(v *value) string {
	switch v.kind {
	case kindString, kindBytes:
		if v.validate {
			return e.p.typ(v.typ)
		}
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

// indexType writes the index type of the view type view of m, the method
// IndexKanon of the view, which scans the encoding once, and a method of
// the index per field of fields, the fields that the methods of the view
// read, in their order, which reads the field at the offset that the scan
// recorded. The index of a view without such fields is the view alone.
func (e *emitter) indexType(m *target, view string, fields []*field) {
	index := m.name + indexSuffix
	if len(fields) == 0 {
		e.doc(index + " is the index of a " + view + ", which has no method that reads a field: the view " +
			"alone.")
	} else {
		e.doc(index + " is the index of a " + view + ": the view and the offsets of the values of the fields " +
			"that its methods read, which " + indexKanonName + " records in one scan. Each method of " + index +
			" returns what the method of " + view + " of the same name returns, without a scan.")
	}
	e.line("type %s struct {", index)
	e.line("v %s", view)
	if len(fields) > 0 {
		e.line("// at records, per method, 1 + the offset of the value of the last")
		e.line("// occurrence of its field, and 0 when the encoding has none.")
		e.line("at [%d]int", len(fields))
	}
	e.line("}")
	e.line("")
	e.indexMethod(m, view, index, fields)
	for k, f := range fields {
		e.indexRead(m, view, index, k, f)
	}
}

// indexMethod writes the method IndexKanon of the view type view of m. It
// scans the encoding once, as wire.Find scans it, and records in the index
// type index the offset of the value of the last occurrence of each of
// fields. It fails where a method of the view fails before it reads a
// value: at a malformed tag or value, at an occurrence of one of fields
// with another wire format, and at the second occurrence of a struct field.
// A view without fields has no method that fails, so its IndexKanon
// returns the index without a scan.
func (e *emitter) indexMethod(m *target, view, index string, fields []*field) {
	if len(fields) == 0 {
		e.doc(indexKanonName + " returns the index of the encoding in v. " + view + " has no method that reads a " +
			"field, so " + indexKanonName + " reads nothing and returns no error.")
		e.line("func (v %s) %s() (%s, error) {", view, indexKanonName, index)
		e.line("return %s{v: v}, nil", index)
		e.line("}")
		e.line("")
		return
	}
	w, typ := e.wire(), structLoc(m)
	e.doc(indexKanonName + " returns the index of the encoding in v, from one scan of the encoding. It fails " +
		"where a method of " + view + " fails before it reads a value: at a malformed tag or value, at an " +
		"occurrence of a field that a method reads with another wire format, and at the second occurrence of a " +
		"struct field.")
	e.line("func (v %s) %s() (%s, error) {", view, indexKanonName, index)
	e.line("ix := %s{v: v}", index)
	e.line("for i := 0; i < len(v); {")
	e.line("at := i")
	e.line("tag, n := %sUvarint(v[i:])", w)
	e.line("if n <= 0 {")
	e.line("return %s{}, %sReadError(n, %s, 0, i)", index, w, typ)
	e.line("}")
	e.line("i += n")
	e.line("skipped, err := %sSkip(v[i:], tag, %s, 0, at)", w, typ)
	e.line("if err != nil {")
	e.line("return %s{}, err", index)
	e.line("}")
	e.line("switch tag >> 3 {")
	for k, f := range fields {
		v, _, _ := e.viewParts(f)
		loc := fieldLoc(m, f)
		e.line("case %d:", f.num)
		e.line("if tag != %s {", e.tag(f.num, v.kind.wire()))
		e.line("return %s{}, %sFormatError(tag, %s%s, %s, at)", index, w, w, wireName(v.kind.wire()), loc)
		e.line("}")
		if v.kind == kindStruct {
			e.line("if ix.at[%d] != 0 {", k)
			e.line("return %s{}, %sRepeatedError(%s, %d, at)", index, w, loc, f.num)
			e.line("}")
		}
		e.line("ix.at[%d] = i + 1", k)
	}
	e.line("}")
	e.line("i += skipped")
	e.line("}")
	e.line("return ix, nil")
	e.line("}")
	e.line("")
}

// indexRead writes the method of the index type index that reads the field
// f of m at the offset k of the index. Its result is that of the method of
// the view type view of the same name: the value that [emitter.viewRead]
// reads at the offset, and the zero value for a field that the encoding
// does not contain.
func (e *emitter) indexRead(m *target, view, index string, k int, f *field) {
	_, result, zero := e.viewParts(f)
	e.doc(f.name + " returns the value of the field " + f.name + ", as " + view + "." + f.name + " returns it, " +
		"at the offset that the index records.")
	e.line("func (ix %s) %s() (%s, error) {", index, f.name, result)
	e.line("v, i := ix.v, ix.at[%d]-1", k)
	e.line("if i < 0 {")
	e.line("return %s, nil", zero)
	e.line("}")
	e.viewRead(m, f)
	e.line("}")
	e.line("")
}

// viewed reports whether a view type has a method for a field whose value is
// v: a bool, a number, a string, a byte slice, a byte array, a time, a struct
// or a type that encodes itself, or a pointer to one of them.
func viewed(v *value) bool {
	if v.kind == kindPointer {
		v = v.elem
	}
	switch v.kind {
	case kindBool, kindInt, kindUint, kindFixed32, kindFixed64, kindFloat32, kindFloat64, kindComplex64,
		kindComplex128, kindString, kindBytes, kindByteArray, kindTime, kindStruct, kindBinary:
		return true
	default:
		return false
	}
}
