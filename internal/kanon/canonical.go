// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"cmp"
	"slices"
	"strconv"
)

// Names of the locals of a canonical decode: of the decode of the fields of
// a struct, and of the read function of a map.
const (
	// priorName names the number of the last field that the decode of the
	// fields of a struct read.
	priorName = "prior"
	// membersName names the bitmap of the unions of a struct whose member the
	// input contains, one bit per discriminator.
	membersName = "members"
	// prevKeyName names the map key before the current one, which the
	// current key must be above.
	prevKeyName = "pk"
)

// canonicalDoc is the sentence that the docblocks of the decode methods of a
// struct of a canonical code file add.
const canonicalDoc = " It accepts only the canonical encoding of a value, the input that EncodeKanon writes for " +
	"it, and returns a *kanon.DecodeError that wraps kanon.ErrNotCanonical for any other input that the wire " +
	"format lets a decoder accept."

// shortest writes, in a canonical code file, the statement that fails for
// the varint whose length is the variable n and whose last byte is
// data[end-1], when that byte is 0 after another: the varint then has a
// shorter form. The error names loc and num at the offset off of the varint.
// A varint of one byte has no shorter form.
func (e *emitter) shortest(n, end, loc, num, off string) {
	if !e.canonical {
		return
	}
	e.line("if %s > 1 && data[%s-1] == 0 {", n, end)
	e.fail(e.wire() + "LongFormError(" + n + ", " + loc + ", " + num + ", " + off + ")")
	e.line("}")
}

// decodeInOrder writes the statements of a canonical code file that decode
// the fields in data into the struct that m points at, and the end of the
// function. The canonical encoding writes the fields of m in ascending field
// number, so the statements match the tag of each field in that order at
// data[i], as encoding/asn1 matches the elements of a SEQUENCE, and decode
// the value of each field whose tag matches, as [emitter.readField] writes
// it. A tag that they match is in its shortest form, above the field before
// it, in the schema and of the wire format of its field, so no statement
// reads a tag or checks one. A byte that remains after the last match
// begins no field after the fields before it, and the helper of
// [emitter.tagger] returns its error: prior is the number of the last field
// that matched.
func (e *emitter) decodeInOrder(m *target) {
	fields := byNumber(m.fields)
	if len(fields) > 0 {
		e.line("var %s uint64", priorName)
	}
	if unionBits(m) {
		e.line("var %s [%d]uint64", membersName, (len(m.discriminators)+63)/64)
	}
	e.line("i := 0")
	for _, f := range fields {
		tag := tagBytes(f.num, fieldWire(f))
		if len(tag) == 1 {
			e.line("if i < len(data) && data[i] == %d<<3|%s%s {", f.num, e.wire(), wireName(fieldWire(f)))
		} else {
			e.line("if len(data)-i >= %d && string(data[i:i+%d]) == %s {", len(tag), len(tag),
				strconv.Quote(string(tag)))
		}
		if readsTagOffset(m, f) {
			e.line("at := i")
		}
		if len(tag) == 1 {
			e.line("i++")
		} else {
			e.line("i += %d", len(tag))
		}
		e.line("%s = %d", priorName, f.num)
		e.readField(m, f)
		e.line("}")
	}
	e.line("if i != len(data) {")
	args := "data, i, off"
	if len(fields) > 0 {
		args = "data, i, " + priorName + ", off"
	}
	e.fail(e.tagger(m) + "(" + args + ")")
	e.line("}")
	e.fail(nilName)
	e.line("}")
	e.line("")
}

// byNumber returns fields in ascending field number, the order of the
// canonical encoding.
func byNumber(fields []*field) []*field {
	return slices.SortedFunc(slices.Values(fields), func(a, b *field) int { return cmp.Compare(a.num, b.num) })
}

// fieldWire returns the wire format of the tag of f: the wire format of its
// value, and of the value that it points at for a pointer.
func fieldWire(f *field) int {
	if f.val.kind == kindPointer {
		return f.val.elem.kind.wire()
	}
	return f.val.kind.wire()
}

// readsTagOffset reports whether the canonical decode of the field f of m
// reads at, the offset of its tag: for the error of a second member of a
// union, as [emitter.member] writes it, and for the error of a value that
// the encoding leaves out, as [emitter.absent] writes it.
func readsTagOffset(m *target, f *field) bool {
	if f.member != nil {
		first, _ := unionEnds(m, f)
		return f.num != first
	}
	return f.val.kind != kindPointer
}

// unionEnds returns the smallest and the largest field number of the
// members of the union of the member f of m.
func unionEnds(m *target, f *field) (int, int) {
	first, last := f.num, f.num
	for _, g := range m.fields {
		if g.member != nil && g.member.disc == f.member.disc {
			first, last = min(first, g.num), max(last, g.num)
		}
	}
	return first, last
}

// unionBits reports whether the canonical decode of m records the members of
// a union that the input contains: for a union of two members or more.
func unionBits(m *target) bool {
	return slices.ContainsFunc(m.fields, func(f *field) bool {
		if f.member == nil {
			return false
		}
		first, last := unionEnds(m, f)
		return first != last
	})
}

// member writes, in a canonical code file, the statements that fail for the
// union member f of m, whose tag is at the offset at, when the input contains
// a member of its union with a smaller number, and that record the member
// when the union has a member with a larger number. The member with the
// smallest number of a union follows no member of it, since the fields
// ascend, and the member with the largest number precedes none.
func (e *emitter) member(m *target, f *field, p place) {
	if !e.canonical {
		return
	}
	first, last := unionEnds(m, f)
	k := slices.Index(m.discriminators, f.member.disc)
	w, s := strconv.Itoa(k/64), strconv.Itoa(k%64)
	if f.num != first {
		e.line("if %s[%s]&(1<<%s) != 0 {", membersName, w, s)
		e.fail(e.wire() + "MemberError(" + quoted(f.member.disc.Name()) + ", " + p.loc + ", " + p.num + ", off+at)")
		e.line("}")
	}
	if f.num != last {
		e.line("%s[%s] |= 1 << %s", membersName, w, s)
	}
}

// tagger returns the name of the helper of the canonical decode of m that
// returns the error of a byte that begins no field of m after the fields
// before it, as [emitter.tagHelper] writes it, and records that the code
// calls it.
func (e *emitter) tagger(m *target) string {
	return e.helper(helper{op: opTag, m: m}, helperKey{op: opTag, id: m.key}, camel(m.name))
}

// tagHelper writes the helper of the canonical decode of m that returns the
// error for the tag at data[i], which is not the tag of a field of m after
// prior, the last field that the decode read. It checks the tag in the order
// of the rules: that it reads, that it is in its shortest form, that its
// field number is not 0, that the number is above prior, and last its wire
// format for a field of m, and an unknown field otherwise. A struct without
// fields reads no field before the tag, so its helper has no prior.
func (e *emitter) tagHelper(name string, m *target) {
	e.ret = retError
	w, loc := e.wire(), structLoc(m)
	fields := byNumber(m.fields)
	e.doc(name + " returns the error of the canonical decode of " + m.name + " for the tag at data[i], which " +
		"begins no field after the fields that the decode read before it. data starts at offset off of the slab.")
	if len(fields) == 0 {
		e.line("func %s(data []byte, i, off int) error {", name)
	} else {
		e.line("func %s(data []byte, i int, %s uint64, off int) error {", name, priorName)
	}
	e.line("tag, n := %sUvarint(data[i:])", w)
	e.line("if n <= 0 {")
	e.fail(w + "ReadError(n, " + loc + ", 0, off+i)")
	e.line("}")
	e.shortest("n", "i+n", loc, "0", "off+i")
	e.line("if tag>>3 == 0 {")
	e.fail(w + "TagError(tag, " + loc + ", 0, off+i)")
	e.line("}")
	if len(fields) > 0 {
		e.line("if tag>>3 <= %s {", priorName)
		e.fail(w + "OrderError(tag>>3, " + priorName + ", " + loc + ", 0, off+i)")
		e.line("}")
		e.line("switch tag >> 3 {")
		for _, f := range fields {
			e.line("case %d:", f.num)
			e.fail(w + "FormatError(tag, " + w + wireName(fieldWire(f)) + ", " + fieldLoc(m, f) + ", off+i)")
		}
		e.line("}")
	}
	e.fail(w + "UnknownFieldError(tag>>3, " + loc + ", 0, off+i)")
	e.line("}")
	e.line("")
}

// absent writes, in a canonical code file, the statements that fail for the
// field f, whose value the loop read into x, when the value is one that the
// encoding leaves out, at the offset of the tag. A union member and a pointer
// are present whatever their value, and get no statements. A string, a byte
// slice, a slice, a map, a struct and a value of a type that encodes itself
// and that is not absent at its zero value are present when their encoding
// has bytes after its length, l. An interface is present when it is not nil,
// and any other value as [emitter.present] states.
func (e *emitter) absent(f *field, x string, p place) {
	if !e.canonical || f.member != nil || f.val.kind == kindPointer {
		return
	}
	v := f.val
	switch {
	case byLength(v):
		e.line("if l == 0 {")
	case v.kind == kindInterface:
		e.line("if %s == nil {", x)
	default:
		e.line("if !(%s) {", e.present(v, x))
	}
	e.fail(e.wire() + "AbsentError(" + p.loc + ", " + p.num + ", off+at)")
	e.line("}")
}

// byLength reports whether a field of v is present exactly when its
// encoding has bytes after its length: a string, a byte slice, a slice, a
// map, a struct, and a value of a type that encodes itself and that is not
// absent at its zero value, as [zeroAbsent] reports.
func byLength(v *value) bool {
	switch v.kind {
	case kindString, kindBytes, kindSlice, kindMap, kindStruct:
		return true
	case kindBinary:
		return !zeroAbsent(v)
	default:
		return false
	}
}

// reencode writes, in a canonical code file, the statements that fail for
// dst, the value of v, a type that encodes itself, which its decode method
// set from data[i:i+int(l)], when the encode method of its family does not
// write those bytes for it: through the append method into a stack array
// when the type has one, and through the encode method otherwise. The error
// names the offset of the value.
func (e *emitter) reencode(v *value, dst string, p place) {
	if !e.canonical {
		return
	}
	e.line("{")
	if v.self.appender != "" {
		e.line("var scratch [%d]byte", binaryScratch)
		e.line("enc, err := %s(scratch[:0])", method(dst, v.self.appender))
	} else {
		e.line("enc, err := %s()", method(dst, v.self.marshaler))
	}
	e.line("if err != nil || !%s.Equal(enc, data[i:i+int(l)]) {", e.std(bytesPath))
	e.fail(e.wire() + "EncodingError(" + p.loc + ", " + p.num + ", off+i)")
	e.line("}")
	e.line("}")
}

// boolRange writes, in a canonical code file, the statement that fails for
// the varint u of a bool at p when it is above 1.
func (e *emitter) boolRange(v *value, p place) {
	if !e.canonical || v.kind != kindBool {
		return
	}
	e.line("if u > 1 {")
	e.fail(e.wire() + "BoolError(u, " + p.loc + ", " + p.num + ", " + p.offset() + ")")
	e.line("}")
}

// keyOrder writes, in a canonical code file, the statements of the read
// function of the map type of v that fail for the key mk, which starts at the
// offset at of the entries, when it has a float component of -0.0, which its
// projection writes as +0.0, and when it is not above the key before it, pk,
// and that record the key. A key type of one value has no key above another,
// so its second key fails, and its function keeps no key before it.
func (e *emitter) keyOrder(v *value) {
	if !e.canonical {
		return
	}
	w := e.wire()
	if e.floats(v.key) {
		e.line("if %s {", e.componentExpr(opNegZero, v.key, "mk"))
		e.fail(w + "NegativeZeroError(loc, num, off+at)")
		e.line("}")
	}
	cond := "at > 0"
	if e.keepsKey(v) {
		cond += " && " + e.compareExpr(v.key, prevKeyName, "mk") + " >= 0"
	}
	e.line("if %s {", cond)
	e.fail(w + "KeyOrderError(loc, num, off+at)")
	e.line("}")
	if e.keepsKey(v) {
		e.line("%s = mk", prevKeyName)
	}
}

// keepsKey reports whether the read function of the map type of v keeps the
// key before the current one, pk, which [emitter.keyOrder] compares with the
// current key: in a canonical code file, for a key type of more than one
// value.
func (e *emitter) keepsKey(v *value) bool {
	return e.canonical && !e.cls.single(v.key)
}
