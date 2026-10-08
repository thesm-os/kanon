// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"cmp"
	"go/types"
	"slices"
	"strconv"
)

// Import paths of the standard-library packages that the encoding methods
// call.
const (
	ioPath     = "io"
	slicesPath = "slices"
	cmpPath    = "cmp"
	bytesPath  = "bytes"
)

// encodeFailure states the failure of the encoding methods and functions of
// a struct whose encoding can fail. [emitter.doc] wraps its words.
const encodeFailure = `It fails when the value of a field fails to encode: a type that encodes itself returns an
error, ValidateKanon of a type rejects a value, an interface stores a type that the tag option types of its field
does not list, a map has a key with a NaN component or two keys of one projection, or a slice or a map has more
elements than the tag option max of its field allows.`

// neverFails states the error of the encoding methods of a struct whose
// encoding cannot fail.
const neverFails = "The error is always nil."

// group is a field that the encode of a struct writes on its own, or a run
// of members of one union with consecutive field numbers, which one switch
// on the discriminator writes.
type group struct {
	// fields lists the field, or the members of the run in ascending field
	// number.
	fields []*field
	// disc is the discriminator of a run, and nil for a field of its own.
	disc *types.Var
}

// layout returns the fields of m in ascending field number, the order in
// which an encoding contains them, as groups. A field that is never
// present, whose value has one value as [value.zeroOnly] reports, is left
// out.
func layout(m *target) []group {
	fields := slices.SortedFunc(slices.Values(m.fields), func(a, b *field) int { return cmp.Compare(a.num, b.num) })
	var out []group
	for _, f := range fields {
		if f.member == nil {
			if !f.val.zeroOnly() {
				out = append(out, group{fields: []*field{f}})
			}
			continue
		}
		if n := len(out); n > 0 && out[n-1].disc == f.member.disc {
			out[n-1].fields = append(out[n-1].fields, f)
			continue
		}
		out = append(out, group{fields: []*field{f}, disc: f.member.disc})
	}
	return out
}

// encodeMethods writes the EncodeKanon, encodeKanon, AppendBinary and
// MarshalBinary methods of the -type struct m. EncodeKanon of a struct
// whose encoding is always empty, without a field that [layout] keeps and
// without unknown fields, fits every buffer and checks no length.
func (e *emitter) encodeMethods(m *target) {
	typ := e.p.typ(m.typ)
	failure := neverFails
	if m.fails {
		failure = encodeFailure
	}
	e.doc("EncodeKanon writes the encoding of m into the last SizeKanon bytes of buf and returns their count. " +
		"A nil m writes nothing. When buf is shorter than SizeKanon bytes, EncodeKanon writes nothing and " +
		"returns io.ErrShortBuffer. " + failure)
	e.line("func (m *%s) EncodeKanon(buf []byte) (int, error) {", typ)
	e.line("if m == nil {")
	e.line("return 0, nil")
	e.line("}")
	if len(layout(m)) > 0 || m.unknown != nil {
		e.line("if len(buf) < m.SizeKanon() {")
		e.line("return 0, %s.ErrShortBuffer", e.std(ioPath))
		e.line("}")
	}
	e.line("return m.%s(buf)", encodeMethod)
	e.line("}")
	e.line("")
	e.doc(encodeMethod + " writes the encoding of m into the end of buf, which has room for it, and returns its " +
		"length. The encoding methods of m and of the structs of the package that contain one call it. " + failure)
	e.line("func (m *%s) %s(buf []byte) (int, error) {", typ, encodeMethod)
	e.encodeBody(m, true)
	e.doc("AppendBinary appends the encoding of m to b and returns the extended slice. It does not allocate " +
		"when b has SizeKanon bytes of spare capacity. " + failure)
	e.line("func (m *%s) AppendBinary(b []byte) ([]byte, error) {", typ)
	e.line("if m == nil {")
	e.line("return b, nil")
	e.line("}")
	e.line("size := m.SizeKanon()")
	e.line("out := %s.Grow(b, size)[:len(b)+size]", e.std(slicesPath))
	if m.fails {
		e.line("if _, err := m.%s(out); err != nil {", encodeMethod)
		e.line("return b, err")
		e.line("}")
	} else {
		e.line("m.%s(out)", encodeMethod)
	}
	e.line("return out, nil")
	e.line("}")
	e.line("")
	e.doc("MarshalBinary returns the encoding of m in a new slice, and nil for an empty encoding. " + failure)
	e.line("func (m *%s) MarshalBinary() ([]byte, error) {", typ)
	e.line("return m.AppendBinary(nil)")
	e.line("}")
	e.line("")
}

// encodeBody writes the statements that write the encoding of the struct
// that m points at into the end of buf and return its length, and the end
// of the function. They write backward: the unknown fields that m keeps,
// then the groups of [layout] from the last to the first, so that the
// encoding reads forward in ascending field number with the unknown fields
// last. A value that follows a length writes before it, so that the length
// is the count of the bytes that it wrote. The statements return an error
// when withErr is set.
func (e *emitter) encodeBody(m *target, withErr bool) {
	e.ret = retLength
	e.line("i := len(buf)")
	if m.unknown != nil {
		e.line("i = %sPutRaw(buf, i, m.%s)", e.wire(), m.unknown.Name())
	}
	for _, g := range slices.Backward(layout(m)) {
		if g.disc == nil {
			e.putField(m, g.fields[0])
			continue
		}
		if len(g.fields) == 1 {
			f := g.fields[0]
			e.line("if m.%s == %s {", g.disc.Name(), e.p.object(f.member.value))
			e.putMember(m, f)
			e.line("}")
			continue
		}
		e.line("switch m.%s {", g.disc.Name())
		for _, f := range g.fields {
			e.line("case %s:", e.p.object(f.member.value))
			e.putMember(m, f)
		}
		e.line("}")
	}
	if withErr {
		e.line("return len(buf) - i, nil")
	} else {
		e.line("return len(buf) - i")
	}
	e.line("}")
	e.line("")
}

// putField writes the statements that write the field f of m, a field that
// is not a union member, before buf[i] when it is present, as
// [emitter.sizeField] states, and move i to its first byte. A field of a
// kanon.Exact type writes through the put function of a field, and a field
// of a kanon.Appender through the put function of its type, which writes
// every position of it without an error. A field with the tag option max
// checks its length first, as [emitter.boundCheck] writes it.
func (e *emitter) putField(m *target, f *field) {
	x, v, loc, num := "m."+f.name, f.val, fieldLoc(m, f), strconv.Itoa(f.num)
	switch v.kind {
	case kindStruct, kindBinary:
		if zeroAbsent(v) {
			e.line("if %s {", e.present(v, x))
			switch {
			case v.fails:
				e.line("w, err := %s", e.contentCall(v, x, loc, num))
				e.line("if err != nil {")
				e.line("return 0, err")
				e.line("}")
			case v.self.appendsKanon:
				e.line("w := %s", e.contentCall(v, x, loc, num))
			default:
				e.line("w := %s(buf[:i], %s, %s, %s)", e.fn(opExactPut, v), addr(x), loc, num)
			}
			e.line("i -= w")
			e.line("i = %sPutUvarint(buf, i, uint64(w))", e.wire())
			e.putTag(f.num, wireBytes)
			e.line("}")
			return
		}
		call := e.contentCall(v, x, loc, num)
		if v.fails {
			e.line("if w, err := %s; err != nil {", call)
			e.line("return 0, err")
			e.line("} else if w > 0 {")
		} else if e.encodesItself(v) {
			e.line("if w, _ := %s; w > 0 {", call)
		} else {
			e.line("if w := %s; w > 0 {", call)
		}
		e.line("i -= w")
		e.line("i = %sPutUvarint(buf, i, uint64(w))", e.wire())
		e.putTag(f.num, wireBytes)
		e.line("}")
	case kindPointer:
		e.line("if %s != nil {", x)
		e.putFramed(v.elem, deref(x), loc, num)
		e.putTag(f.num, v.elem.kind.wire())
		e.line("}")
	default:
		e.line("if %s {", e.present(v, x))
		e.boundCheck(m, f, x)
		e.putFramed(v, x, loc, num)
		e.putTag(f.num, v.kind.wire())
		e.line("}")
	}
}

// putMember writes the statements that write the union member f of m, which
// its discriminator selects, before buf[i]: its value whatever it is, and
// the zero value of the type that it points at for a nil pointer, as
// [emitter.memberTarget] sets it. The one value of a type that
// [value.zeroOnly] reports is a constant, so its statements do not read a
// target. A member with the tag option max checks its length first, as
// [emitter.boundCheck] writes it.
func (e *emitter) putMember(m *target, f *field) {
	x, v, loc, num := "m."+f.name, f.val, fieldLoc(m, f), strconv.Itoa(f.num)
	if v.kind != kindPointer {
		e.boundCheck(m, f, x)
		e.putFramed(v, x, loc, num)
		e.putTag(f.num, v.kind.wire())
		return
	}
	if !v.elem.zeroOnly() {
		e.memberTarget(f)
	}
	e.putFramed(v.elem, "*x", loc, num)
	e.putTag(f.num, v.elem.kind.wire())
}

// putFramed writes the statements that write x, the value of v that the tag
// of a field introduces, before buf[i]: its encoding, and for a value that
// [value.unframed] reports its encoding after its length.
func (e *emitter) putFramed(v *value, x, loc, num string) {
	if !v.unframed() {
		e.put(v, x, loc, num)
		return
	}
	e.line("end := i")
	e.put(v, x, loc, num)
	e.line("i = %sPutUvarint(buf, i, uint64(end-i))", e.wire())
}

// put writes the statements that write the encoding of x, an addressable
// value of v, before buf[i] and move i to its first byte. The statements of
// a value whose encoding can fail return the error for the field that loc
// and num locate: a kanon.Validator first calls its ValidateKanon, before
// the values that it contains. The statements of a time, a struct, a type
// that encodes itself and a value whose encoding can fail declare
// variables, which [emitter.putScoped] scopes. The statements of a value
// that [value.zeroOnly] reports write its constant encoding and do not read
// x, and the statements of any other value do.
func (e *emitter) put(v *value, x, loc, num string) {
	w := e.wire()
	if v.zeroOnly() {
		if b := v.constant(); len(b) > 1 {
			e.line("i = %sPutRaw(buf, i, %s)", w, quoted(string(b)))
		} else {
			e.line("i = %sPutUvarint(buf, i, 0)", w)
		}
		return
	}
	e.validPut(v, x, loc, num)
	switch v.kind {
	case kindBool:
		e.line("i = %sPutBool(buf, i, %s)", w, as(v, x, types.Bool))
	case kindInt:
		e.line("i = %sPutUvarint(buf, i, %sZigzag(%s))", w, w, as(v, x, types.Int64))
	case kindUint:
		e.line("i = %sPutUvarint(buf, i, %s)", w, as(v, x, types.Uint64))
	case kindFixed32:
		e.line("i = %sPutUint32(buf, i, %s)", w, as(v, x, types.Uint32))
	case kindFixed64:
		e.line("i = %sPutUint64(buf, i, %s)", w, as(v, x, types.Uint64))
	case kindFloat32:
		e.line("i = %sPutUint32(buf, i, %s)", w, e.floatWord(fixed32Width*8, as(v, x, types.Float32)))
	case kindFloat64:
		e.line("i = %sPutUint64(buf, i, %s)", w, e.floatWord(fixed64Width*8, as(v, x, types.Float64)))
	case kindComplex64:
		e.line("i = %sPutUint32(buf, i, %s)", w, e.floatWord(fixed32Width*8, "imag("+x+")"))
		e.line("i = %sPutUint32(buf, i, %s)", w, e.floatWord(fixed32Width*8, "real("+x+")"))
	case kindComplex128:
		e.line("i = %sPutUint64(buf, i, %s)", w, e.floatWord(fixed64Width*8, "imag("+x+")"))
		e.line("i = %sPutUint64(buf, i, %s)", w, e.floatWord(fixed64Width*8, "real("+x+")"))
		e.line("i = %sPutUvarint(buf, i, %d)", w, complex128Width)
	case kindString, kindBytes:
		e.line("i = %sPutRaw(buf, i, %s)", w, x)
		e.line("i = %sPutUvarint(buf, i, uint64(len(%s)))", w, x)
	case kindByteArray:
		e.line("i = %sPutRaw(buf, i, %s[:])", w, primary(x))
		e.line("i = %sPutUvarint(buf, i, uint64(len(%s)))", w, x)
	case kindTime:
		e.line("end := i")
		e.line("i = %sPutTime(buf, i, %s)", w, x)
		e.line("i = %sPutUvarint(buf, i, uint64(end-i))", w)
	case kindStruct, kindBinary:
		call := e.contentCall(v, x, loc, num)
		if v.fails {
			e.line("w, err := %s", call)
			e.check()
		} else if e.encodesItself(v) {
			e.line("w, _ := %s", call)
		} else {
			e.line("w := %s", call)
		}
		e.line("i -= w")
		e.line("i = %sPutUvarint(buf, i, uint64(w))", w)
	case kindArray:
		e.putCall(v, addr(x), loc, num)
	default:
		e.putCall(v, x, loc, num)
	}
}

// putScoped writes the statements of [emitter.put] for x, a key or a value
// of a map that a put function writes, in a block of their own when they
// declare variables, so that two values in one block do not declare a
// variable twice. The parameters of the put function locate its errors.
func (e *emitter) putScoped(v *value, x string) {
	if v.zeroOnly() || v.kind != kindTime && v.kind != kindStruct && v.kind != kindBinary && !v.fails {
		e.put(v, x, locParam, numParam)
		return
	}
	e.line("{")
	e.put(v, x, locParam, numParam)
	e.line("}")
}

// putCall writes the statements that write arg, a slice, an array, a map, a
// pointer or an interface of v, with the put function of v, before buf[i],
// and return the error of a value whose encoding fails. The call passes loc
// and num to a put function that takes them, as [emitter.located] reports.
func (e *emitter) putCall(v *value, arg, loc, num string) {
	name := e.fn(e.keyOp(opPut, v), v)
	if !e.putFails(v) {
		if e.located(v) {
			e.line("i -= %s(buf[:i], %s, %s, %s)", name, arg, loc, num)
		} else {
			e.line("i -= %s(buf[:i], %s)", name, arg)
		}
		return
	}
	e.line("w, err := %s(buf[:i], %s, %s, %s)", name, arg, loc, num)
	e.check()
	e.line("i -= w")
}

// located reports whether the put function of v, a slice, an array, a map, a
// pointer or an interface, takes loc and num, the location of the field that
// it writes: it can fail, as [emitter.putFails] reports, or it writes a
// kanon.Appender, as [appenderIn] reports, whose put function names the
// location in the panic of an encoding of another length.
func (e *emitter) located(v *value) bool {
	return e.putFails(v) || appenderIn(v, make(map[*value]bool))
}

// appenderIn reports whether the put function of v writes a value of a
// kanon.Appender: v is one, or a slice, an array, a pointer or a map whose
// elements, targets, keys or values contain one. The walk does not enter a
// struct, whose encode passes literal locations to the put functions of its
// fields, nor an interface, whose put function takes the location for a
// type that its list does not name. seen marks the values visited, so that a
// type that contains itself ends the walk.
func appenderIn(v *value, seen map[*value]bool) bool {
	if seen[v] {
		return false
	}
	seen[v] = true
	switch v.kind {
	case kindBinary:
		return v.self.appendsKanon
	case kindSlice, kindArray, kindPointer:
		return appenderIn(v.elem, seen)
	case kindMap:
		return appenderIn(v.key, seen) || appenderIn(v.elem, seen)
	default:
		return false
	}
}

// putFails reports whether the put function of v, a slice, an array, a map,
// a pointer or an interface, can fail: a value that v contains can fail to
// encode, v is an interface, whose concrete type can be missing from its
// list, or the keys of v need the checks of [emitter.keyChecks]. The
// ValidateKanon of v itself does not count, since [emitter.put] calls it
// before the put function.
func (e *emitter) putFails(v *value) bool {
	switch v.kind {
	case kindSlice, kindArray, kindPointer:
		return v.elem.fails
	case kindMap:
		return v.elem.fails || v.key.fails || e.floats(v.key) || e.ambiguous(v.key)
	default:
		return v.fails
	}
}

// contentCall returns the call that writes the encoding of x, a struct or a
// value of a type that encodes itself, without its length into the end of
// buf[:i], and returns its length: the unexported encode method of a
// -type struct of the package, EncodeKanon of any other struct with a kanon
// codec, and the put function of the code file otherwise, which a struct
// with a kanon codec takes too in the projection of a map key, as
// [emitter.keyed] reports. The call returns an error when it calls a method
// of a struct with a kanon codec or writes a value whose encoding can fail.
func (e *emitter) contentCall(v *value, x, loc, num string) string {
	if v.kind == kindBinary {
		return e.fn(opPut, v) + "(buf[:i], " + addr(x) + ", " + loc + ", " + num + ")"
	}
	if !e.encodesItself(v) {
		return e.fn(e.keyOp(opPut, v), v) + "(buf[:i], " + addr(x) + ")"
	}
	if e.generates(v.typ) {
		return method(x, encodeMethod) + "(buf[:i])"
	}
	return method(x, encodeKanonName) + "(buf[:i])"
}

// encodesItself reports whether the code writes a struct of v through the
// methods of its kanon codec: v is a struct with a kanon codec, and the code
// does not write it as part of the projection of a map key that contains a
// float, as [emitter.keyed] reports, which its methods do not write.
func (e *emitter) encodesItself(v *value) bool {
	return v.kind == kindStruct && v.inline == nil && !e.keyed(v)
}

// selfEncode writes the statement that sets enc to the encoding of the value
// of v that the pointer x points at, through its append method into a stack
// array when it has one, and through its encode method otherwise. The error
// of the method goes to errVar.
func (e *emitter) selfEncode(v *value, x, errVar string) {
	if v.self.appender != "" {
		e.line("var scratch [%d]byte", binaryScratch)
		e.line("enc, %s := %s.%s(scratch[:0])", errVar, x, v.self.appender)
		return
	}
	e.line("enc, %s := %s.%s()", errVar, x, v.self.marshaler)
}

// putSized writes the statements of the put function of v, a kanon.Sizer,
// which writes the value that x points at into the end of buf: they size
// the value with SizeKanon, encode it into that room in place through the
// append method of its family, or through its encode method and a copy,
// and return the error of wire.SizeError for a length that SizeKanon did
// not return, and for a length that the room cannot take.
func (e *emitter) putSized(v *value) {
	w := e.wire()
	e.line("n := x.%s()", sizeKanonName)
	e.line("if uint(n) > uint(len(buf)) {")
	e.line("return 0, %sSizeError(loc, num)", w)
	e.line("}")
	e.line("i := len(buf) - n")
	if v.self.appender != "" {
		e.line("enc, err := x.%s(buf[i:i:len(buf)])", v.self.appender)
	} else {
		e.line("enc, err := x.%s()", v.self.marshaler)
	}
	e.line("if err != nil {")
	e.line("return 0, %sMarshalError(err, loc, num)", w)
	e.line("}")
	e.line("if len(enc) != n {")
	e.line("return 0, %sSizeError(loc, num)", w)
	e.line("}")
	if v.self.appender == "" {
		e.line("copy(buf[i:], enc)")
	}
	e.line("return n, nil")
}

// exactPutHelper writes the put function of a field of v, a type that
// declares kanon.Exact, which the encoding writes only at a value other than
// the zero value: it appends the value that x points at, as
// [emitter.exactAppend] appends it, and returns its length. kanon.Exact
// guarantees that the append method does not fail and appends SizeKanon bytes
// for such a value, so the function has no error path, and wire.MustExact
// panics for a type that breaks the guarantee.
func (e *emitter) exactPutHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	e.doc(name + " writes the encoding of the " + typ + " that x points at, which is not its zero value, into " +
		"the end of buf, which has room for its SizeKanon, and returns its length. " + typ + " declares " +
		"kanon.Exact, so the encoding does not fail, and a value that breaks the guarantee panics.")
	e.line("func %s(buf []byte, x *%s, loc string, num int) int {", name, typ)
	e.exactAppend(v)
	e.line("%sMustExact(enc, err, n, loc, num)", e.wire())
	e.line("return n")
	e.line("}")
	e.line("")
}

// putExact writes the statements of the put function of v, a type that
// declares kanon.Exact, in a position that writes the zero value: an element,
// a map key or value, the target of a pointer, a union member or the value of
// an interface. They append the value that x points at, as
// [emitter.exactAppend] appends it, return the error of the append method,
// which kanon.Exact allows for the zero value alone, and pass the encoding to
// wire.MustExact, which panics for another length than SizeKanon. kanon.Exact
// rules out a SizeKanon below 0 and an encoding of another length, so the
// statements check neither the room nor the length.
func (e *emitter) putExact(v *value) {
	e.exactAppend(v)
	e.line("if err != nil {")
	e.line("return 0, %sMarshalError(err, loc, num)", e.wire())
	e.line("}")
	e.line("%sMustExact(enc, nil, n, loc, num)", e.wire())
	e.line("return n, nil")
}

// putAppender writes the statements of the put function of a kanon.Appender
// in every position: they size the value that x points at with SizeKanon as
// n, append its encoding through AppendKanon into that room at the end of
// buf, in place, and pass the encoding to wire.MustExact, which panics for
// another length than SizeKanon. AppendKanon returns no error, and
// kanon.Exact rules out a SizeKanon below 0, so the statements have no error
// path and check no room.
func (e *emitter) putAppender() {
	e.line("n := x.%s()", sizeKanonName)
	e.line("i := len(buf) - n")
	e.line("enc := x.%s(buf[i:i:len(buf)])", appendKanonName)
	e.line("%sMustExact(enc, nil, n, loc, num)", e.wire())
	e.line("return n")
}

// exactAppend writes the statements of the put functions of v, a type that
// declares kanon.Exact, that size the value that x points at with SizeKanon
// as n, and append its encoding through the append method into that room at
// the end of buf, in place, as enc with the error err.
func (e *emitter) exactAppend(v *value) {
	e.line("n := x.%s()", sizeKanonName)
	e.line("i := len(buf) - n")
	e.line("enc, err := x.%s(buf[i:i:len(buf)])", v.self.appender)
}

// putHelper writes the put function of the values of v: the function that
// writes the encoding of an inline struct or of a value of a type that
// encodes itself without its length, and of a slice, an array, a map, a
// pointer or an interface with it, into the end of buf, and returns the
// length that it wrote. The put function of a kanon.Appender writes the
// statements of [emitter.putAppender], of any other type that declares
// kanon.Exact those of [emitter.putExact], and of any other kanon.Sizer
// those of [emitter.putSized]. The put function of a slice, an array, a map,
// a pointer or an interface also returns an error when [emitter.putFails]
// reports that it can fail.
func (e *emitter) putHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	text := name + " writes the encoding of "
	room := " into the end of buf, which has room for it, and returns its length."
	keyed := e.inKeyDoc()
	if keyed != "" {
		keyed += ","
	}
	if v.kind == kindStruct {
		m := e.structTarget(v)
		if m.fails {
			e.doc(text + "the " + m.name + " that m points at" + keyed + room + " " + encodeFailure)
			e.line("func %s(buf []byte, m *%s) (int, error) {", name, typ)
		} else {
			e.doc(text + "the " + m.name + " that m points at" + keyed + room)
			e.line("func %s(buf []byte, m *%s) int {", name, typ)
		}
		e.encodeBody(m, m.fails)
		return
	}
	if v.kind == kindBinary && v.self.appendsKanon {
		e.doc(text + "the " + typ + " that x points at" + room + " The room is the length that the SizeKanon " +
			"of x returns. " + typ + " is a kanon.Appender, whose AppendKanon does not fail, and an encoding of " +
			"another length panics.")
		e.line("func %s(buf []byte, x *%s, loc string, num int) int {", name, typ)
		e.putAppender()
		e.line("}")
		e.line("")
		return
	}
	if v.kind == kindBinary && v.self.exact {
		e.doc(text + "the " + typ + " that x points at" + room + " The room is the length that the SizeKanon " +
			"of x returns. It fails when x fails to encode itself, which kanon.Exact allows for the zero " + typ +
			" alone, and panics for an encoding of another length.")
		e.line("func %s(buf []byte, x *%s, loc string, num int) (int, error) {", name, typ)
		e.putExact(v)
		e.line("}")
		e.line("")
		return
	}
	if v.kind == kindBinary && v.self.sizer {
		e.doc(text + "the " + typ + " that x points at" + room + " The room is the length that the SizeKanon " +
			"of x returns. It fails when x fails to encode itself, and when its encoding has another length.")
		e.line("func %s(buf []byte, x *%s, loc string, num int) (int, error) {", name, typ)
		e.putSized(v)
		e.line("}")
		e.line("")
		return
	}
	if v.kind == kindBinary {
		e.doc(text + "the " + typ + " that x points at" + room + " It fails when x fails to encode itself.")
		e.line("func %s(buf []byte, x *%s, loc string, num int) (int, error) {", name, typ)
		e.selfEncode(v, "x", "err")
		e.line("if err != nil {")
		e.line("return 0, %sMarshalError(err, loc, num)", e.wire())
		e.line("}")
		e.line("return copy(buf[len(buf)-len(enc):], enc), nil")
		e.line("}")
		e.line("")
		return
	}
	if keyed == "" {
		keyed = ","
	}
	fails := e.putFails(v)
	if fails {
		failure := " It fails for a value that fails to encode"
		if v.kind == kindMap && (e.floats(v.key) || e.ambiguous(v.key)) {
			failure += ", a key with a NaN component and two keys of one projection"
		}
		e.doc(text + "x, a " + typ + keyed + room + failure + ", which loc and num locate.")
		e.line("func %s(buf []byte, x %s, loc string, num int) (int, error) {", name, param(v, typ))
	} else if e.located(v) {
		e.doc(text + "x, a " + typ + keyed + room + " loc and num locate the panic of a kanon.Appender in x " +
			"that appends another length than its SizeKanon.")
		e.line("func %s(buf []byte, x %s, loc string, num int) int {", name, param(v, typ))
	} else {
		e.doc(text + "x, a " + typ + keyed + room)
		e.line("func %s(buf []byte, x %s) int {", name, param(v, typ))
	}
	e.ret = retLength
	e.line("i := len(buf)")
	result := nilName
	switch v.kind {
	case kindSlice, kindArray:
		e.line("for k := len(x) - 1; k >= 0; k-- {")
		e.put(v.elem, "x[k]", locParam, numParam)
		e.line("}")
		e.line("i = %sPutUvarint(buf, i, uint64(len(buf)-i))", e.wire())
	case kindMap:
		if e.putMap(v) {
			result = "err"
		}
		e.line("i = %sPutUvarint(buf, i, uint64(len(buf)-i))", e.wire())
	case kindPointer:
		e.line("if x != nil {")
		e.put(v.elem, "*x", locParam, numParam)
		e.line("}")
		e.line("i = %sPutBool(buf, i, x != nil)", e.wire())
	default:
		e.putInterface(v)
	}
	if fails {
		e.line("return len(buf) - i, %s", result)
	} else {
		e.line("return len(buf) - i")
	}
	e.line("}")
	e.line("")
}

// putMap writes the statements that write the keys and values of the map x
// of v backward, the last key first and each value before its key, so that
// the keys ascend, and reports whether they declare err, the error of the
// checks of the keys, which the put function returns. Each key writes its
// projection, as [emitter.keyPutScoped] states. A key of one value, as
// [classifier.single] reports it, allows one entry at most, which needs no
// order, and wire.OneKey fails a map with two such keys of an ambiguous key
// type, as [emitter.ambiguous] reports. Bool keys write true, then false.
// A key that [lookupable] does not report sorts with its value, as a
// [wire.Pair], in a stack array of mapKeyBuffer pairs. An integer or string
// key with a value that [small] reports writes as [emitter.putOrderedMap]
// states. Any other key sorts in a stack array of mapKeyBuffer keys, and a
// lookup finds the value of each. The sorted keys pass the checks of
// [emitter.keyChecks] before the first one writes, and the loop writes the
// entries while err is nil.
func (e *emitter) putMap(v *value) bool {
	mk, mv := loopVar(v.key, "mk"), loopVar(v.elem, "mv")
	if e.cls.single(v.key) {
		checked := e.ambiguous(v.key)
		if checked {
			e.line("err := %s%s(len(x), %s, %s)", e.wire(), oneKeyName, locParam, numParam)
			e.line("if err == nil {")
		}
		if mk == blankName && mv == blankName {
			e.line("for range x {")
		} else {
			e.line("for %s, %s := range x {", mk, mv)
		}
		e.putScoped(v.elem, "mv")
		e.keyPutScoped(v.key, "mk")
		e.line("}")
		if checked {
			e.line("}")
		}
		return checked
	}
	if v.key.kind == kindBool {
		for _, key := range []string{trueName, falseName} {
			e.line("if %s, ok := x[%s]; ok {", mv, key)
			e.putScoped(v.elem, "mv")
			e.keyPutScoped(v.key, key)
			e.line("}")
		}
		return false
	}
	if !lookupable(v.key) {
		e.line("type pair = %sPair[%s, %s]", e.wire(), e.p.typ(v.key.typ), e.p.typ(v.elem.typ))
		e.line("var arr [%d]pair", mapKeyBuffer)
		e.line("pairs := arr[:0]")
		e.line("for mk, mv := range x {")
		e.line("pairs = append(pairs, pair{Key: mk, Value: mv})")
		e.line("}")
		e.line("%s.SortFunc(pairs, func(a, b pair) int {", e.std(slicesPath))
		e.line("return %s", e.compareExpr(v.key, "a.Key", "b.Key"))
		e.line("})")
		checked := e.keyChecks(v, "pairs", true, func(k string) string { return "pairs[" + k + "].Key" })
		e.line("for k := len(pairs) - 1; %sk >= 0; k-- {", whileNoError(checked))
		e.putScoped(v.elem, "pairs[k].Value")
		e.keyPutScoped(v.key, "pairs[k].Key")
		e.line("}")
		return checked
	}
	if ordered(v.key) && small(v.elem) && mv != blankName {
		e.putOrderedMap(v)
		return false
	}
	e.line("var arr [%d]%s", mapKeyBuffer, e.p.typ(v.key.typ))
	e.line("keys := arr[:0]")
	e.line("for mk := range x {")
	e.line("keys = append(keys, mk)")
	e.line("}")
	if ordered(v.key) {
		e.line("%s.Sort(keys)", e.std(slicesPath))
	} else {
		e.line("%s.SortFunc(keys, func(a, b %s) int {", e.std(slicesPath), e.p.typ(v.key.typ))
		e.line("return %s", e.compareExpr(v.key, "a", "b"))
		e.line("})")
	}
	checked := e.keyChecks(v, "keys", false, func(k string) string { return "keys[" + k + "]" })
	e.line("for k := len(keys) - 1; %sk >= 0; k-- {", whileNoError(checked))
	e.line("mk := keys[k]")
	if mv != blankName {
		e.line("mv := x[mk]")
	}
	e.putScoped(v.elem, "mv")
	e.keyPutScoped(v.key, "mk")
	e.line("}")
	return checked
}

// putOrderedMap writes the statements that write the map x of v, whose keys
// are integers or strings and whose values [small] reports. Such keys are
// neither floats nor ambiguous, so no check precedes the first one. A map of
// at most mapKeyBuffer entries sorts its entries as [wire.Pair] values in a
// stack array with wire.SortPairs, so that no lookup finds a value after the
// sort. A larger map sorts its keys with slices.Sort in a slice of its
// length, which the encode allocates, and a lookup finds the value of each
// key, since a lookup of an integer or string key costs less than a
// comparison through a function value, which the sort of pairs would call.
// The put function of a map that e.short marks takes no larger map, so its
// statements sort in the stack array alone.
func (e *emitter) putOrderedMap(v *value) {
	w := e.wire()
	key := e.p.typ(v.key.typ)
	short := e.short[v.id]
	if !short {
		e.line("if len(x) <= %d {", mapKeyBuffer)
	}
	e.line("type pair = %sPair[%s, %s]", w, key, e.p.typ(v.elem.typ))
	e.line("var arr [%d]pair", mapKeyBuffer)
	e.line("pairs := arr[:0]")
	e.line("for mk, mv := range x {")
	e.line("pairs = append(pairs, pair{Key: mk, Value: mv})")
	e.line("}")
	e.line("%sSortPairs(pairs)", w)
	e.line("for k := len(pairs) - 1; k >= 0; k-- {")
	e.putScoped(v.elem, "pairs[k].Value")
	e.keyPutScoped(v.key, "pairs[k].Key")
	e.line("}")
	if short {
		return
	}
	e.line("} else {")
	e.line("keys := make([]%s, 0, len(x))", key)
	e.line("for mk := range x {")
	e.line("keys = append(keys, mk)")
	e.line("}")
	e.line("%s.Sort(keys)", e.std(slicesPath))
	e.line("for k := len(keys) - 1; k >= 0; k-- {")
	e.line("mk := keys[k]")
	e.line("mv := x[mk]")
	e.putScoped(v.elem, "mv")
	e.keyPutScoped(v.key, "mk")
	e.line("}")
	e.line("}")
}

// small reports whether a value of v is small enough that its copy into a
// [wire.Pair] costs less than the lookup of the value after a sort of the
// keys: a value of any kind but a struct, an array, a byte array and a type
// that encodes itself, whose sizes have no bound.
func small(v *value) bool {
	switch v.kind {
	case kindStruct, kindArray, kindByteArray, kindBinary:
		return false
	default:
		return true
	}
}

// whileNoError returns the operand of && that the condition of a loop
// starts with when checked reports that the statements before the loop
// declare err: the condition that err is nil, so that the loop ends at the
// error. It returns nothing otherwise.
func whileNoError(checked bool) string {
	if checked {
		return "err == nil && "
	}
	return ""
}

// keyPutScoped writes the statements of [emitter.putScoped] for x, a map key
// of v, in the mode of a map key: x writes its projection, whose float
// components of -0.0 write as +0.0.
func (e *emitter) keyPutScoped(v *value, x string) {
	e.inKey = true
	e.putScoped(v, x)
	e.inKey = false
}

// loopVar returns name, the name of the variable of a loop that takes a
// value of v, or the blank identifier when the statements of [emitter.put]
// for v do not read the value, which Go rejects for a declared variable.
func loopVar(v *value, name string) string {
	if v.zeroOnly() {
		return blankName
	}
	return name
}

// putInterface writes the statements that write the interface value x of v
// backward: one type switch over its concrete types, in tag order, after nil
// when v is nilable, each writing the value of the concrete type before its
// type number. A concrete type that the list does not name fails.
func (e *emitter) putInterface(v *value) {
	e.line("switch x := x.(type) {")
	if v.nilable {
		e.line("case nil:")
		e.line("i = %sPutUvarint(buf, i, 0)", e.wire())
	}
	for _, w := range v.variants {
		e.line("case %s:", e.p.typ(w.c.typ))
		e.put(w.val, "x", locParam, numParam)
		e.line("i = %sPutUvarint(buf, i, %d)", e.wire(), w.c.num)
	}
	e.line("default:")
	e.line("return 0, %sUnlistedError(x, loc, num)", e.wire())
	e.line("}")
}

// lookupable reports whether a map lookup finds every key of v: a key of v
// equals itself under ==, as a bool, an integer, a string, a byte array, a
// time and a pointer do. A float, a complex number, and an array, a
// struct, a type that encodes itself or an interface, which can contain
// one, can be a NaN, which equals nothing.
func lookupable(v *value) bool {
	switch v.kind {
	case kindBool, kindInt, kindUint, kindFixed32, kindFixed64, kindString, kindByteArray, kindTime, kindPointer:
		return true
	default:
		return false
	}
}

// ordered reports whether the order of cmp.Compare, which slices.Sort
// applies, is the order of the map keys of v: integers, floats with NaN
// first, and strings.
func ordered(v *value) bool {
	switch v.kind {
	case kindInt, kindUint, kindFixed32, kindFixed64, kindFloat32, kindFloat64, kindString:
		return true
	default:
		return false
	}
}
