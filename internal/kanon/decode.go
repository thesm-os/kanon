// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"strconv"
	"strings"
)

// place locates a value that the code decodes: the field that its errors
// name, as Go expressions, and how deep it is.
type place struct {
	// loc and num are the expressions of the location and the field number
	// of the errors of the value.
	loc, num string
	// at is the expression of the offset of the value in the slab, and ""
	// for off+i, the offset of data[i].
	at string
	// k is the number of levels that the value is below the function that
	// decodes it: the value is at level depth-k, and its decode fails when
	// that is negative.
	k int
	// checked reports that the decode of a value before this one checked
	// the level depth-k, so that the decode of a struct at it makes no check
	// of its own: the value of a map whose key checks its level.
	checked bool
}

// offset returns the expression of the offset of the value at p in the
// slab.
func (p place) offset() string {
	if p.at == "" {
		return "off+i"
	}
	return p.at
}

// depth returns the expression of the level of a value at p: depth-k.
func (p place) depth() string {
	if p.k == 0 {
		return "depth"
	}
	return "depth-" + strconv.Itoa(p.k)
}

// below returns the place of a value one level below the value at p.
func (p place) below() place {
	p.k++
	return p
}

// deep returns the condition that the level of a value at p is below zero:
// depth < k.
func (p place) deep() string {
	return "depth < " + strconv.Itoa(p.k)
}

// tracked numbers the fields of m that the decode tracks in its seen
// bitmap, in declaration order:
//
//   - a pointer and an interface, which a decode that does not see them
//     sets to nil;
//   - a struct, whose first occurrence decodes into the memory of the field
//     and whose later occurrences merge, and which a decode that does not
//     see it resets;
//   - a map, whose first occurrence clears it and moves up to
//     freeListLength of its values that refer to memory into a free list,
//     and which a decode that does not see it clears;
//   - an array of values that refer to memory, which decodes into the
//     memory of its elements, and which a decode that does not see it
//     resets.
//
// An inert field, as [value.inert] reports, needs no reset and is not
// tracked.
func tracked(m *target) map[*field]int {
	bits := make(map[*field]int)
	for _, f := range m.fields {
		if f.val.inert() {
			continue
		}
		switch f.val.kind {
		case kindPointer, kindStruct, kindInterface, kindMap:
			bits[f] = len(bits)
		case kindArray:
			if f.val.holdsMemory() {
				bits[f] = len(bits)
			}
		default:
			// The decode of any other field replaces its value.
		}
	}
	return bits
}

// mergingValues returns the ids of the values of structs whose read
// functions take the merge flag: the value of an interface field outside a
// union that can store a concrete type that merges, as [value.variantMerges]
// reports, and each pointer to a struct among the concrete types of such a
// value. A repeated occurrence of such a field merges into the value that
// the interface stores when the concrete type repeats.
func mergingValues(structs []*target) map[string]bool {
	ids := make(map[string]bool)
	for _, m := range structs {
		for _, f := range m.fields {
			if f.member != nil || f.val.kind != kindInterface || !f.val.variantMerges() {
				continue
			}
			ids[f.val.id] = true
			for _, c := range f.val.variants {
				if c.val.kind == kindPointer && c.val.merges() {
					ids[c.val.id] = true
				}
			}
		}
	}
	return ids
}

// levelled reports whether a value of v is a level of the depth of a
// decode: a struct, a slice, a map, a pointer or an interface. The decode
// function of each such value checks its level.
func levelled(v *value) bool {
	switch v.kind {
	case kindStruct, kindSlice, kindMap, kindPointer, kindInterface:
		return true
	default:
		return false
	}
}

// checksLevel reports whether the decode of a value of v checks the level of
// v: v is a level, or an array of at least one value that checks its level,
// since the elements of an array are at the level of the array.
func checksLevel(v *value) bool {
	if v.kind == kindArray {
		return v.size > 0 && checksLevel(v.elem)
	}
	return levelled(v)
}

// decodeMethods writes the UnmarshalBinary, DecodeKanon and MergeKanon
// methods of the -type struct m, and its unexported decode, merge and
// fields methods, which the decode of the structs of the package that
// contain one call.
func (e *emitter) decodeMethods(m *target) {
	e.bits = tracked(m)
	typ := e.p.typ(m.typ)
	opts := e.rt() + optionsName
	copied := "which copy data once"
	if !m.tree {
		copied = "which do not copy data, since " + m.name + " contains no string"
	}
	canonical := ""
	if e.canonical {
		canonical = canonicalDoc
	}
	e.doc("UnmarshalBinary sets m to the value encoded in data, as DecodeKanon does with the zero Options, " +
		copied + ".")
	e.line("func (m *%s) UnmarshalBinary(data []byte) error {", typ)
	e.line("return m.DecodeKanon(data, %s{})", opts)
	e.line("}")
	e.line("")
	e.doc("DecodeKanon sets m to the value encoded in data and reuses the memory of m: the values that its " +
		"pointers point at, its nested structs, the capacity of its slices and maps, and up to " +
		strconv.Itoa(freeListLength) + " values of each map whose values refer to memory. It first clears the " +
		"fields that the encoding leaves out, as Reset does. Every decoded string is a substring of the slab of " +
		"opts." + canonical + " It returns a *kanon.DecodeError for malformed input, after which m contains the " +
		"fields decoded before the error.")
	e.line("func (m *%s) DecodeKanon(data []byte, opts %s) error {", typ, opts)
	e.source(m)
	e.line("return m.%s(data, slab, off, opts.Limit())", decodeMethod)
	e.line("}")
	e.line("")
	e.doc("MergeKanon decodes data into m without resetting m first, with the options and the errors of " +
		"DecodeKanon. A slice appends the elements in data, a map adds its entries, a nested struct merges, an " +
		"interface that stores the concrete type in data merges as that type merges, a union member replaces " +
		"the union, and any other field takes the value in data. The fields that the encoding leaves out keep " +
		"their values.")
	e.line("func (m *%s) MergeKanon(data []byte, opts %s) error {", typ, opts)
	e.source(m)
	e.line("return m.%s(data, slab, off, opts.Limit())", mergeMethod)
	e.line("}")
	e.line("")
	callers := " and the decode of the structs of the package that contain " + m.name + " call it."
	call := func(o op) string {
		if o == opMerge {
			return "m." + mergeMethod + "("
		}
		return "m." + fieldsMethod + "("
	}
	e.doc(decodeMethod + " sets m to the value encoded in data, which starts at offset off of slab, as " +
		"DecodeKanon does. The decode enters depth levels below m at most. DecodeKanon" + callers)
	e.line("func (m *%s) %s(data []byte, slab string, off, depth int) error {", typ, decodeMethod)
	e.decodeBody(m, call)
	e.doc(mergeMethod + " merges the encoding in data, which starts at offset off of slab, into m, as " +
		"MergeKanon does. The decode enters depth levels below m at most. MergeKanon" + callers)
	e.line("func (m *%s) %s(data []byte, slab string, off, depth int) error {", typ, mergeMethod)
	if e.seenWords() == 0 {
		e.fieldsBody(m)
		return
	}
	e.mergeBody(call(opFields))
	e.fieldsDoc(fieldsMethod, decodeMethod+" and "+mergeMethod)
	e.line("func (m *%s) %s(data []byte, slab string, off, depth int, seen %s) (%s, error) {",
		typ, fieldsMethod, e.bitmap(), e.bitmap())
	e.fieldsBody(m)
}

// source writes the statements of DecodeKanon and MergeKanon of m that set
// slab and off, the slab of the decode and the offset of data in it: those
// of opts, or a copy of data at offset 0 when opts has no slab, as
// kanon.Options.Source returns them. A struct that contains no string reads
// nothing from the slab, so its decode copies nothing and takes the offset
// of opts as the base of the offsets of its errors, also without a slab. The
// decode of a struct without strings passes the offset of a nested struct of
// another package that way, since it has no slab to pass.
func (e *emitter) source(m *target) {
	if m.tree {
		e.line("slab, off := opts.Source(data)")
		return
	}
	e.line(`slab, off := "", opts.Offset`)
}

// seenWords returns the length in 64-bit words of the seen bitmap of the
// struct whose decode the emitter writes.
func (e *emitter) seenWords() int {
	return (len(e.bits) + 63) / 64
}

// bitmap returns the type of the seen bitmap of the struct whose decode the
// emitter writes.
func (e *emitter) bitmap() string {
	return "[" + strconv.Itoa(e.seenWords()) + "]uint64"
}

// seen returns the condition that the bit of the tracked field f is clear
// in the seen bitmap, and the statement that sets the bit.
func (e *emitter) seen(f *field) (string, string) {
	w, s := e.bit(f)
	return "seen[" + w + "]&(1<<" + s + ") == 0", "seen[" + w + "] |= 1 << " + s
}

// bit returns the word and the shift of the bit of the tracked field f in
// the seen bitmap.
func (e *emitter) bit(f *field) (string, string) {
	b := e.bits[f]
	return strconv.Itoa(b / 64), strconv.Itoa(b % 64)
}

// decodeBody writes the statements that set the struct that m points at to
// the value encoded in data, and the end of the function: the statements of
// [emitter.decodePrologue], and then the decode of the fields of data.
// Without tracked fields they return the merge of data. With them they call
// the function of the loop with a clear seen bitmap, and then the
// statements of [emitter.decodeEpilogue]. call returns the beginning of the
// call of the merge, for opMerge, and of the function of the loop, for
// opFields, up to the argument data.
func (e *emitter) decodeBody(m *target, call func(op) string) {
	e.decodePrologue(m)
	if e.seenWords() == 0 {
		e.line("return %sdata, slab, off, depth)", call(opMerge))
		e.line("}")
		e.line("")
		return
	}
	e.line("seen, err := %sdata, slab, off, depth, %s{})", call(opFields), e.bitmap())
	e.decodeEpilogue(m)
	e.line("return err")
	e.line("}")
	e.line("")
}

// decodePrologue writes the statements of a decode that precede the decode
// of the fields of the struct that m points at: they clear the fields that
// the encoding leaves out, keeping the memory of every encoded field, and
// reset every field that the decode does not track, the discriminators and
// the unknown fields.
func (e *emitter) decodePrologue(m *target) {
	e.clearLeftOut(m, func(f *field) bool { return f.val.holdsMemory() })
	for _, f := range m.fields {
		if _, ok := e.bits[f]; !ok {
			e.resetField(f, false)
		}
	}
	for _, d := range m.discriminators {
		e.line("m.%s = 0", d.Name())
	}
	if m.unknown != nil {
		e.line("m.%s = m.%s[:0]", m.unknown.Name(), m.unknown.Name())
	}
}

// decodeEpilogue writes the statements of a decode that follow the decode of
// the fields of the struct that m points at, which records the tracked
// fields that the encoding contains in seen: they set each tracked pointer
// and interface that the encoding does not contain to nil, reset each such
// struct and array, and clear each such map.
func (e *emitter) decodeEpilogue(m *target) {
	for _, f := range m.fields {
		if _, ok := e.bits[f]; !ok {
			continue
		}
		unseen, _ := e.seen(f)
		e.line("if %s {", unseen)
		if f.val.kind == kindPointer || f.val.kind == kindInterface {
			e.line("m.%s = nil", f.name)
		} else {
			e.resetValue(f.val, "m."+f.name)
		}
		e.line("}")
	}
}

// mergeBody writes the statements that merge the encoding in data into the
// struct that m points at, and the end of the function: the call of the
// function of the loop, which fields begins up to the argument data, with a
// seen bitmap with every bit set, so that every nested struct merges.
func (e *emitter) mergeBody(fields string) {
	all := make([]string, e.seenWords())
	for k := range all {
		all[k] = "^uint64(0)"
	}
	e.line("_, err := %sdata, slab, off, depth, %s{%s})", fields, e.bitmap(), strings.Join(all, ", "))
	e.line("return err")
	e.line("}")
	e.line("")
}

// fieldsDoc writes the docblock of name, the function of the loop that
// decodes the fields of a struct with tracked fields for callers.
func (e *emitter) fieldsDoc(name, callers string) {
	e.doc(name + " decodes the fields in data into m for " + callers + ". The first occurrence of a tracked " +
		"field whose bit in seen is clear decodes into the memory of the field, and every other occurrence " +
		"merges. " + name + " returns seen with the bits of the tracked fields that data contains.")
}

// fieldsBody writes the statements of the loop that decodes the fields in
// data into the struct that m points at, and the end of the function. A
// field of m checks its wire format and decodes its value, and an unknown
// field is skipped, and appended to the field that keeps unknown fields
// when m has one. The caller of the decode of a nested struct checks its
// level, as [emitter.structLevel] writes it, and the top struct of a decode
// is at a level of 0 or more. A canonical code file decodes the fields in
// their order instead, as [emitter.decodeInOrder] writes it.
func (e *emitter) fieldsBody(m *target) {
	e.ret = retError
	if e.seenWords() > 0 {
		e.ret = retSeen
	}
	if e.canonical {
		e.decodeInOrder(m)
		return
	}
	w, loc := e.wire(), structLoc(m)
	e.line("for i := 0; i < len(data); {")
	e.line("at := i")
	e.line("tag, n := %sUvarint(data[i:])", w)
	e.line("if n <= 0 {")
	e.fail(w + "ReadError(n, " + loc + ", 0, off+i)")
	e.line("}")
	e.line("i += n")
	if len(m.fields) == 0 {
		e.skipField(m)
	} else {
		e.line("switch tag >> 3 {")
		for _, f := range m.fields {
			e.line("case %d:", f.num)
			e.readField(m, f)
		}
		e.line("default:")
		e.skipField(m)
		e.line("}")
	}
	e.line("}")
	e.fail(nilName)
	e.line("}")
	e.line("")
}

// skipField writes the statements that skip the unknown field of m whose
// tag starts at offset at, keep its bytes when m keeps unknown fields, and
// move i past it.
func (e *emitter) skipField(m *target) {
	e.line("n, err := %sSkip(data[i:], tag, %s, 0, off+at)", e.wire(), structLoc(m))
	e.check()
	if m.unknown != nil {
		e.line("m.%s = append(m.%s, data[at:i+n]...)", m.unknown.Name(), m.unknown.Name())
	}
	e.line("i += n")
}

// readField writes the body of the switch case that decodes one occurrence
// of the field f of m at data[i], after its tag, which starts at offset at:
//
//   - A struct decodes into the memory of the field on its first
//     occurrence, and any later occurrence merges.
//   - A slice appends the elements of each occurrence, and a map adds the
//     entries of each occurrence to the entries of the first.
//   - A pointer decodes into the value that it points at, or into a new one.
//   - An interface decodes into the value that it stores on its first
//     occurrence, and a later occurrence of the concrete type that it stores
//     merges into the value when the type merges, as [value.merges] reports.
//   - Any other field takes the value of the occurrence.
//
// An occurrence of a union member replaces the union: a member that the
// discriminator does not select yet zeroes the selected member and sets the
// discriminator, and every occurrence of a member decodes as a first
// occurrence. A union of one member has no other member to zero.
//
// The body checks the wire format of the tag first. The ordered decode of a
// canonical code file matches the tag, wire format included, before the
// body, and its body has no such check.
func (e *emitter) readField(m *target, f *field) {
	x, v := "m."+f.name, f.val
	p := place{loc: fieldLoc(m, f), num: strconv.Itoa(f.num), k: 1}
	if !e.canonical {
		format := wireName(fieldWire(f))
		e.line("if tag != %d<<3|%s%s {", f.num, e.wire(), format)
		e.fail(e.wire() + "FormatError(tag, " + e.wire() + format + ", " + p.loc + ", off+at)")
		e.line("}")
	}
	if f.member != nil {
		e.member(m, f, p)
		selector := e.p.object(f.member.value)
		if first, last := unionEnds(m, f); first == last {
			e.line("m.%s = %s", f.member.disc.Name(), selector)
		} else {
			e.line("if m.%s != %s {", f.member.disc.Name(), selector)
			e.line("%s(m)", e.deselector(m, f.member.disc))
			e.line("m.%s = %s", f.member.disc.Name(), selector)
			e.line("}")
		}
		if v.kind == kindSlice {
			e.line("%s = %s[:0]", x, x)
		}
	}
	if _, ok := e.bits[f]; ok && v.kind == kindStruct {
		e.readStructField(f, v, x, p)
		return
	}
	switch v.kind {
	case kindPointer:
		e.readPointerField(f, v, x, p)
		return
	case kindInterface:
		merge := ""
		if f.member == nil && v.variantMerges() {
			w, s := e.bit(f)
			merge = "seen[" + w + "]&(1<<" + s + ") != 0"
		}
		e.readFramed(v, x, p, merge)
		e.absent(f, x, p)
	case kindSlice:
		e.validFrom(v, p)
		e.readLength(p)
		e.line("if err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s, %s); err != nil {",
			e.fn(opRead, v), addr(x), p.depth(), p.loc, p.num)
		e.fail("err")
		e.line("}")
		e.line("i += int(l)")
		e.validRead(v, x, p)
		e.absent(f, x, p)
	case kindMap:
		collect := trueName
		if f.member == nil {
			collect, _ = e.seen(f)
		}
		e.validFrom(v, p)
		e.readLength(p)
		e.line("if err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s, %s, %s); err != nil {",
			e.fn(opRead, v), addr(x), p.depth(), collect, p.loc, p.num)
		e.fail("err")
		e.line("}")
		e.line("i += int(l)")
		e.validRead(v, x, p)
		e.absent(f, x, p)
	case kindBinary:
		if e.canonical && f.member == nil && v.self.exact {
			e.readExactField(v, x, p)
			break
		}
		if e.canonical && f.member == nil && zeroAbsent(v) {
			// The encoder never encodes the zero value of such a type, so the
			// decode checks the presence of the value before it encodes it.
			e.readSelf(v, x, p, func() { e.absent(f, x, p) })
			break
		}
		e.read(v, x, p, "")
		e.absent(f, x, p)
	default:
		e.read(v, x, p, "")
		e.absent(f, x, p)
	}
	if _, ok := e.bits[f]; ok {
		_, set := e.seen(f)
		e.line("%s", set)
	}
}

// readStructField writes the statements that decode an occurrence of the
// struct x of v, the value of the field f or the struct that f points at,
// at p: into the memory of x when the bit of f in the seen bitmap is clear
// or f is a union member, and merged into x otherwise. In a canonical code
// file a struct field of no bytes then fails, as [emitter.absent] writes it.
func (e *emitter) readStructField(f *field, v *value, x string, p place) {
	unseen, set := e.seen(f)
	e.readLength(p)
	e.structLevel(v, p)
	e.line("var err error")
	if f.member != nil {
		e.line("%s", set)
		e.line("err = %s", e.structCall(v, x, p, opRead))
	} else {
		e.line("if %s {", unseen)
		e.line("%s", set)
		e.line("err = %s", e.structCall(v, x, p, opRead))
		e.line("} else {")
		e.line("err = %s", e.structCall(v, x, p, opMerge))
		e.line("}")
	}
	e.check()
	e.line("i += int(l)")
	e.absent(f, x, p)
}

// readPointerField writes the statements that decode an occurrence of the
// pointer field x of v at p into the value that x points at, or into a new
// one. The pointer is one level, and the value that it points at is one
// below it. The pointer checks its own level when the value is no level,
// since the decode function of a level checks a level below the pointer's.
func (e *emitter) readPointerField(f *field, v *value, x string, p place) {
	if !levelled(v.elem) {
		e.line("if %s {", p.deep())
		e.fail(e.wire() + "DepthError(" + p.loc + ", " + p.num + ", off+i)")
		e.line("}")
	}
	e.line("if %s == nil {", x)
	e.line("%s = new(%s)", x, e.p.typ(v.elem.typ))
	e.line("}")
	target := p.below()
	if v.elem.kind == kindStruct {
		e.readStructField(f, v.elem, deref(x), target)
		return
	}
	if v.elem.unframed() {
		e.readFramed(v.elem, deref(x), target, "")
	} else {
		e.read(v.elem, deref(x), target, "")
	}
	_, set := e.seen(f)
	e.line("%s", set)
}

// readFramed writes the statements that decode dst, a value of v that
// [value.unframed] reports and that the tag of a field introduces, at p:
// the length that [emitter.putFramed] writes, then the value, which must
// take exactly that length. merge is empty, or the condition under which the
// read function merges the value, as [emitter.mergeArg] passes it.
func (e *emitter) readFramed(v *value, dst string, p place, merge string) {
	e.readLength(p)
	e.line("used, err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s%s, %s)", e.fn(opRead, v), addr(dst),
		p.depth(), e.mergeArg(v, merge), p.loc, p.num)
	e.check()
	e.line("if used != int(l) {")
	e.fail(e.wire() + "TrailingError(int(l)-used, " + p.loc + ", " + p.num + ", off+i+used)")
	e.line("}")
	e.line("i += int(l)")
}

// mergeArg returns the merge argument of a call of the read function of v
// and the comma after it: merge, or false when merge is empty, and nothing
// when the read function of v takes no merge flag, as e.merging marks it.
func (e *emitter) mergeArg(v *value, merge string) string {
	if !e.merging[v.id] {
		return ""
	}
	if merge == "" {
		merge = falseName
	}
	return merge + ", "
}

// readLength writes the statements that read the length l of the value at
// data[i], check that the value ends within data, and move i past the
// length. In a canonical code file they also check that the length has its
// shortest form.
func (e *emitter) readLength(p place) {
	w := e.wire()
	e.line("l, n := %sUvarint(data[i:])", w)
	e.line("if n <= 0 || uint64(len(data)-i-n) < l {")
	e.fail(w + "ReadError(n, " + p.loc + ", " + p.num + ", off+i)")
	e.line("}")
	e.shortest("n", "i+n", p.loc, p.num, "off+i")
	e.line("i += n")
}

// readExact writes the statements that read the length l of the value at
// data[i], check that it is size and that the value ends within data, and
// move i past the length. In a canonical code file they also check that the
// length has its shortest form.
func (e *emitter) readExact(p place, size int64) {
	w := e.wire()
	e.line("l, n := %sUvarint(data[i:])", w)
	e.line("if n > 0 && l != %d {", size)
	e.fail(w + "LengthError(l, " + strconv.FormatInt(size, 10) + ", " + p.loc + ", " + p.num + ", off+i)")
	e.line("}")
	e.line("if n <= 0 || uint64(len(data)-i-n) < l {")
	e.fail(w + "ReadError(n, " + p.loc + ", " + p.num + ", off+i)")
	e.line("}")
	e.shortest("n", "i+n", p.loc, p.num, "off+i")
	e.line("i += n")
}

// read writes the statements that decode the encoding of a value of v at
// data[i] into dst and move i past it. dst takes the decoded value whole
// and keeps its memory: the capacity of a byte slice and a slice, the
// entries of a map, the value that a pointer points at, the memory of a
// struct, and the value that an interface stores. merge is empty, or
// mergeParam in a read function that takes the merge flag, for a value that
// [value.merges] reports: with the flag set, dst merges the decoded value as
// a repeated field merges it. dst is addressable, or the value that a pointer
// points at, as (*p). The statements declare variables, and every group of
// them reads dst. A kanon.Validator then calls its ValidateKanon on dst, and
// its error names the offset of the value.
func (e *emitter) read(v *value, dst string, p place, merge string) {
	w := e.wire()
	e.validFrom(v, p)
	switch v.kind {
	case kindBool, kindInt, kindUint:
		e.line("u, n := %sUvarint(data[i:])", w)
		if v.native() {
			e.nativeVarint(v, dst, p)
		} else {
			e.line("if n <= 0 {")
			e.fail(w + "ReadError(n, " + p.loc + ", " + p.num + ", off+i)")
			e.line("}")
			x := e.varint(v, "u", p)
			e.shortest("n", "i+n", p.loc, p.num, p.offset())
			e.boolRange(v, p)
			e.line("%s = %s", dst, x)
		}
		e.line("i += n")
	case kindFixed32, kindFloat32, kindFixed64, kindFloat64, kindComplex64:
		reader := "Uint64"
		if v.kind.width() == fixed32Width {
			reader = "Uint32"
		}
		e.line("u, n := %s%s(data[i:])", w, reader)
		e.line("if n <= 0 {")
		e.fail(w + "ReadError(n, " + p.loc + ", " + p.num + ", off+i)")
		e.line("}")
		e.line("%s = %s", dst, e.fixed(v, "u"))
		e.line("i += n")
	case kindComplex128:
		m := e.std(mathPath)
		e.readExact(p, complex128Width)
		e.line("re, _ := %sUint64(data[i:])", w)
		e.line("im, _ := %sUint64(data[i+%d:])", w, fixed64Width)
		e.line("%s = %s", dst, e.cast(v, "complex("+m+".Float64frombits(re), "+m+".Float64frombits(im))",
			types.Complex128))
		e.line("i += %d", complex128Width)
	case kindString:
		e.readLength(p)
		e.line("%s = %s", dst, e.cast(v, "slab[off+i:off+i+int(l)]", types.String))
		e.line("i += int(l)")
	case kindBytes:
		e.readLength(p)
		e.line("%s = append(%s[:0], data[i:i+int(l)]...)", dst, primary(dst))
		e.line("i += int(l)")
	case kindByteArray:
		e.readExact(p, v.size)
		e.line("i += copy(%s[:], data[i:])", primary(dst))
	case kindTime:
		e.readLength(p)
		decodeTime := "Time"
		if e.canonical {
			decodeTime = "CanonicalTime"
		}
		e.line("t, err := %s%s(data[i:i+int(l)], %s, %s, off+i)", w, decodeTime, p.loc, p.num)
		e.check()
		e.line("%s = t", dst)
		e.line("i += int(l)")
	case kindStruct:
		e.readLength(p)
		e.structLevel(v, p)
		if merge == "" {
			e.line("if err := %s; err != nil {", e.structCall(v, dst, p, opRead))
			e.fail("err")
			e.line("}")
		} else {
			// The block scopes err, which the read function of a pointer
			// declares before it.
			e.line("{")
			e.line("var err error")
			e.line("if %s {", merge)
			e.line("err = %s", e.structCall(v, dst, p, opMerge))
			e.line("} else {")
			e.line("err = %s", e.structCall(v, dst, p, opRead))
			e.line("}")
			e.check()
			e.line("}")
		}
		e.line("i += int(l)")
	case kindBinary:
		e.readSelf(v, dst, p, nil)
	case kindArray:
		if v.size == 0 {
			e.readExact(p, 0)
			e.line("%s = %s", dst, e.zero(v.typ))
			break
		}
		e.readLength(p)
		e.line("if err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s, %s); err != nil {", e.fn(opRead, v),
			addr(dst), p.depth(), p.loc, p.num)
		e.fail("err")
		e.line("}")
		e.line("i += int(l)")
	case kindSlice:
		e.readLength(p)
		if merge == "" {
			e.line("%s = %s[:0]", dst, primary(dst))
		} else {
			e.line("if !%s {", merge)
			e.line("%s = %s[:0]", dst, primary(dst))
			e.line("}")
		}
		e.line("if err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s, %s); err != nil {", e.fn(opRead, v),
			addr(dst), p.depth(), p.loc, p.num)
		e.fail("err")
		e.line("}")
		e.line("i += int(l)")
	case kindMap:
		collect := trueName
		if merge != "" {
			collect = "!" + merge
		}
		e.readLength(p)
		e.line("if err := %s(%s, data[i:i+int(l)], slab, off+i, %s, %s, %s, %s); err != nil {",
			e.fn(opRead, v), addr(dst), p.depth(), collect, p.loc, p.num)
		e.fail("err")
		e.line("}")
		e.line("i += int(l)")
	default:
		e.line("used, err := %s(%s, data[i:], slab, off+i, %s, %s%s, %s)", e.fn(opRead, v), addr(dst), p.depth(),
			e.mergeArg(v, merge), p.loc, p.num)
		e.check()
		e.line("i += used")
	}
	e.validRead(v, dst, p)
}

// varint returns the expression of the value of v, a bool or an integer,
// that the varint u encodes. For an integer whose Go type is narrower than
// 64 bits, it first writes the statements that reject a value outside the
// range of the type, at the offset of the varint.
func (e *emitter) varint(v *value, u string, p place) string {
	if v.kind == kindBool {
		return e.cast(v, u+" != 0", types.Bool)
	}
	x, wide := u, types.Uint64
	if v.signed() {
		x, wide = e.wire()+"Unzigzag("+u+")", types.Int64
		if v.narrow() {
			e.line("s := %s", x)
			x = "s"
		}
	}
	if v.narrow() {
		basic := v.basicName()
		e.line("if %s != %s(%s(%s)) {", x, types.Typ[wide].Name(), basic, x)
		e.fail(e.wire() + "RangeError(" + x + ", " + quoted(basic) + ", " + p.loc + ", " + p.num + ", " +
			p.offset() + ")")
		e.line("}")
	}
	return e.cast(v, x, wide)
}

// readSelf writes the statements that decode the value dst of v, a type that
// encodes itself, at p and move i past it: its length, the decode method of
// its family into the zero value, the statements of check when it is not
// nil, and in a canonical code file the encode method of the value, as
// [emitter.reencode] writes it.
func (e *emitter) readSelf(v *value, dst string, p place, check func()) {
	e.readLength(p)
	e.line("%s = %s", dst, e.zero(v.typ))
	e.line("if err := %s(data[i:i+int(l)]); err != nil {", method(dst, v.self.unmarshaler))
	e.fail(e.wire() + "UnmarshalError(err, " + p.loc + ", " + p.num + ", off+i)")
	e.line("}")
	if check != nil {
		check()
	}
	e.reencode(v, dst, p)
	e.line("i += int(l)")
}

// readExactField writes, in a canonical code file, the statements that decode
// the field x of v, a type that declares kanon.Exact, at p and move i past
// it: its length, the decode method of its family into the zero value, and
// one statement that fails for an error of the method and for the zero value,
// which the encoding leaves out, as wire.ExactError tells them apart.
// kanon.Exact guarantees that the method accepts only the bytes that the
// append method writes for the value, so the decode does not encode the value
// again.
func (e *emitter) readExactField(v *value, x string, p place) {
	e.readLength(p)
	e.line("%s = %s", x, e.zero(v.typ))
	_, absent := e.presence(v, x)
	e.line("if err := %s(data[i:i+int(l)]); err != nil || %s {", method(x, v.self.unmarshaler), absent)
	e.fail(e.wire() + "ExactError(err, " + p.loc + ", " + p.num + ", off+at, off+i)")
	e.line("}")
	e.line("i += int(l)")
}

// nativeVarint writes the statements that set dst to the value of v, an
// int, a uint or a uintptr, that the varint u of length n encodes. One
// condition rejects both a varint that does not read and a value outside
// the range of the type, which only a platform with a 32-bit int has, so
// that the rejection is a statement that a 64-bit platform runs.
func (e *emitter) nativeVarint(v *value, dst string, p place) {
	w := e.wire()
	x, wide := "u", types.Uint64
	if v.signed() {
		e.line("s := %sUnzigzag(u)", w)
		x, wide = "s", types.Int64
	}
	basic := v.basicName()
	e.line("if n <= 0 || %s != %s(%s(%s)) {", x, types.Typ[wide].Name(), basic, x)
	e.fail(w + "VarintError(n, " + x + ", " + quoted(basic) + ", " + p.loc + ", " + p.num + ", " + p.offset() + ")")
	e.line("}")
	e.shortest("n", "i+n", p.loc, p.num, p.offset())
	e.line("%s = %s", dst, e.cast(v, x, wide))
}

// fixed returns the expression of the value of v, a fixed-size number, that
// the little-endian bits u encode.
func (e *emitter) fixed(v *value, u string) string {
	switch v.kind {
	case kindFloat32:
		return e.cast(v, e.std(mathPath)+".Float32frombits("+u+")", types.Float32)
	case kindFloat64:
		return e.cast(v, e.std(mathPath)+".Float64frombits("+u+")", types.Float64)
	case kindComplex64:
		m := e.std(mathPath)
		return e.cast(v, "complex("+m+".Float32frombits(uint32("+u+")), "+m+".Float32frombits(uint32("+u+">>32)))",
			types.Complex64)
	case kindFixed32:
		return e.cast(v, u, types.Uint32)
	default:
		return e.cast(v, u, types.Uint64)
	}
}

// structCall returns the call that decodes data[i:i+int(l)] into the struct
// x of v at p with the operation o, opRead or opMerge, and returns its
// error: the unexported methods of a -type struct of the package,
// DecodeKanon and MergeKanon of any other struct with a kanon codec, and
// the functions of the code file for an inline struct.
func (e *emitter) structCall(v *value, x string, p place, o op) string {
	args := "data[i:i+int(l)], slab, off+i, " + p.depth()
	if v.inline != nil {
		return e.fn(o, v) + "(" + addr(x) + ", " + args + ")"
	}
	if e.generates(v.typ) {
		name := decodeMethod
		if o == opMerge {
			name = mergeMethod
		}
		return method(x, name) + "(" + args + ")"
	}
	name := decodeKanonName
	if o == opMerge {
		name = mergeKanonName
	}
	return method(x, name) + "(data[i:i+int(l)], " + e.wire() + "Nested(slab, off+i, " + p.depth() + "))"
}

// structLevel writes the statements that reject the struct of v at p, whose
// encoding starts at off+i, when its level is below zero, before the call
// that decodes it, and nothing when the decode of a map key before it
// checked the level. The error names an inline struct by its name, and a
// struct with a kanon codec by its type, qualified by its package outside
// the package of the file. The decode of a struct does not check its own
// level, since the options of DecodeKanon and MergeKanon cannot state a
// level below zero.
func (e *emitter) structLevel(v *value, p place) {
	if p.checked {
		return
	}
	name := types.TypeString(v.typ, e.cls.relative)
	if v.inline != nil {
		name = v.inline.name
	}
	e.line("if %s {", p.deep())
	e.fail(e.wire() + "DepthError(" + quoted(name) + ", 0, off+i)")
	e.line("}")
}

// readHelper writes the read function of the values of v.
func (e *emitter) readHelper(name string, v *value) {
	typ := e.p.typ(v.typ)
	about := " data starts at offset off of slab, and the decode enters depth levels below the value at most."
	errs := " loc and num locate the field in errors."
	switch v.kind {
	case kindStruct:
		m := v.inline
		e.bits = tracked(m)
		e.doc(name + " sets the " + m.name + " that m points at to the value encoded in data and reuses its " +
			"memory, as DecodeKanon does." + about)
		e.line("func %s(m *%s, data []byte, slab string, off, depth int) error {", name, typ)
		e.decodeBody(m, func(o op) string { return e.fn(o, v) + "(m, " })
	case kindSlice:
		e.doc(name + " appends the elements that data encodes, a " + typ + " without its length, to *dst." +
			about + errs)
		e.line("func %s(dst *%s, data []byte, slab string, off, depth int, loc string, num int) error {", name, typ)
		e.readSlice(v)
	case kindArray:
		e.doc(name + " decodes the elements that data encodes, a " + typ + " without its length, into *dst. " +
			"It fails unless data encodes exactly " + strconv.FormatInt(v.size, 10) + " elements." + about + errs)
		e.line("func %s(dst *%s, data []byte, slab string, off, depth int, loc string, num int) error {", name, typ)
		e.readArray(v)
	case kindMap:
		e.doc(name + " adds the entries that data encodes, a " + typ + " without its length, to *dst. With " +
			"collect set, it clears *dst first, and the entries reuse the memory of up to " +
			strconv.Itoa(freeListLength) + " of its values." + about + errs)
		e.line("func %s(dst *%s, data []byte, slab string, off, depth int, collect bool, loc string, num int) error {",
			name, typ)
		e.readMap(v)
	case kindPointer:
		merging := ""
		if e.merging[v.id] {
			merging = " With merge set, the struct merges into the struct that *dst points at."
		}
		e.doc(name + " decodes the " + typ + " at the start of data into *dst and returns the length of its " +
			"encoding: nil for the presence byte 0, and for the presence byte 1 the value that follows, into the " +
			"value that *dst points at or into a new one." + merging + about + errs)
		e.line("func %s(dst *%s, data []byte, slab string, off, depth int, %sloc string, num int) (int, error) {",
			name, typ, e.mergeParams(v))
		e.readPointer(v)
	default:
		merging := ""
		if e.merging[v.id] {
			merging = " With merge set, a value of the concrete type that *dst stores merges into it when the type " +
				"merges, as a repeated field merges."
		}
		e.doc(name + " decodes the " + typ + " at the start of data into *dst and returns the length of its " +
			"encoding: the number of its concrete type and the value of that type, into the value that *dst " +
			"stores when it has that type. It fails for a type number that the list does not name." + merging +
			about + errs)
		e.line("func %s(dst *%s, data []byte, slab string, off, depth int, %sloc string, num int) (int, error) {",
			name, typ, e.mergeParams(v))
		e.readInterface(v)
	}
}

// readSlice writes the body of the read function of the slice type of v. A
// nil slice takes the capacity for the elements, which it counts in data,
// except for pointers and interfaces, whose count takes a walk of every
// value they contain. An element that refers to memory decodes into the
// element in the spare capacity, which a decode before left there.
func (e *emitter) readSlice(v *value) {
	e.ret = retError
	e.levelCheck()
	e.line("x := *dst")
	if count := e.count(v.elem); count != "" {
		e.line("if x == nil && len(data) > 0 {")
		e.line("x = make(%s, 0, %s)", e.p.typ(v.typ), count)
		e.line("}")
	}
	e.line("for i := 0; i < len(data); {")
	if v.elem.holdsMemory() {
		e.line("if len(x) < cap(x) {")
		e.line("x = x[:len(x)+1]")
		e.line("} else {")
		e.line("x = append(x, %s)", e.zero(v.elem.typ))
		e.line("}")
	} else {
		e.line("x = append(x, %s)", e.zero(v.elem.typ))
	}
	e.line("last := len(x) - 1")
	e.read(v.elem, "x[last]", place{loc: locParam, num: numParam, k: 1}, "")
	e.line("}")
	e.line("*dst = x")
	e.line("return nil")
	e.line("}")
	e.line("")
}

// count returns the expression of the number of the elements of v in data,
// for the capacity of a slice: the length of data over the length of an
// element of one length, the count of varints or of length-prefixed values,
// and "" for a pointer and an interface.
func (e *emitter) count(v *value) string {
	if c, ok := constSize(v); ok {
		if c == 1 {
			return "len(data)"
		}
		return "len(data) / " + strconv.Itoa(c)
	}
	switch v.kind {
	case kindInt, kindUint:
		return e.wire() + "CountVarints(data)"
	case kindPointer, kindInterface:
		return ""
	default:
		return e.wire() + "CountValues(data)"
	}
}

// readArray writes the body of the read function of the array type of v. An
// array is no level, so its elements are at its own level.
func (e *emitter) readArray(v *value) {
	e.ret = retError
	e.line("i := 0")
	e.line("for k := range dst {")
	e.read(v.elem, "dst[k]", place{loc: locParam, num: numParam}, "")
	e.line("}")
	e.line("if i != len(data) {")
	e.fail(e.wire() + "TrailingError(len(data)-i, loc, num, off+i)")
	e.line("}")
	e.line("return nil")
	e.line("}")
	e.line("")
}

// readMap writes the body of the read function of the map type of v. A map
// whose values refer to memory keeps a free list of up to freeListLength of
// its values when collect is set, and an entry takes the value of its key
// from the list, or the last value in it. The variables of the loop are
// declared before it: the compiler moves a variable of a loop body to the
// heap when the decode of a recursive struct takes its address.
//
// A key with a NaN component fails at its offset. Keys of one projection
// are one key: the map keeps the later value, and for a key type that
// [emitter.ambiguous] reports, which can decode two such keys to keys that
// differ under ==, the function collects the keys that its canon function
// rejects, and wire.KeepLastKeys deletes every entry of a key whose
// projection a later key repeats. Without collect, wire.MergeKeys first
// checks the keys that the map has, and its error, kanon.ErrAmbiguousKey for
// two of them with one projection, ends the function before it changes the
// map: the loop runs while err is nil, and wire.KeepLastKeys finds no key
// to delete. In a canonical code file the keys must ascend, as
// [emitter.keyOrder] writes the check, and a decode, which sets collect,
// collects no key.
func (e *emitter) readMap(v *value) {
	kt, vt := e.p.typ(v.key.typ), e.p.typ(v.elem.typ)
	reuse := v.elem.holdsMemory()
	nan, dedup := e.floats(v.key), e.ambiguous(v.key)
	w := e.wire()
	e.ret = retError
	e.levelCheck()
	e.line("x := *dst")
	if dedup {
		e.line("var arr [%d]%s", mapKeyBuffer, kt)
		e.line("keys := arr[:0]")
		e.line("var err error")
		e.line("if !collect {")
		e.line("keys, err = %s%s(x, keys, %s, %s, loc, num, off)", w, mergeKeysName, e.canonFunc(v.key),
			e.compareFunc(v.key))
		e.line("}")
		e.line("all := len(keys) != 0")
	}
	if reuse {
		e.line("var free [%d]%s", freeListLength, vt)
		e.line("var freeKeys [%d]%s", freeListLength, kt)
		e.line("held := 0")
		e.line("if collect {")
		e.line("for mk, mv := range x {")
		e.line("if held == len(free) {")
		e.line("break")
		e.line("}")
		e.line("freeKeys[held], free[held] = mk, mv")
		e.line("held++")
		e.line("}")
		e.line("clear(x)")
		e.line("}")
	} else {
		e.line("if collect {")
		e.line("clear(x)")
		e.line("}")
	}
	e.line("if x == nil && len(data) > 0 {")
	e.line("x = make(%s)", e.p.typ(v.typ))
	e.line("*dst = x")
	e.line("}")
	e.line("var mk %s", kt)
	e.line("var mv %s", vt)
	if e.keepsKey(v) {
		e.line("var %s %s", prevKeyName, kt)
	}
	e.line("for i := 0; %si < len(data); {", whileNoError(dedup))
	if v.key.holdsMemory() {
		e.line("mk = %s", e.zero(v.key.typ))
	}
	if nan || e.canonical {
		e.line("at := i")
	}
	e.readScoped(v.key, "mk", place{loc: locParam, num: numParam, k: 1})
	if nan {
		e.line("if %s {", e.componentExpr(opNaN, v.key, "mk"))
		e.fail(w + keyErrorName + "(loc, num, off+at)")
		e.line("}")
	}
	e.keyOrder(v)
	if reuse {
		e.line("mv = %s", e.zero(v.elem.typ))
		e.line("if held > 0 {")
		e.line("held = %sTake(freeKeys[:], free[:], held, mk)", w)
		e.line("mv = free[held]")
		e.line("}")
	}
	e.readScoped(v.elem, "mv", place{loc: locParam, num: numParam, k: 1, checked: checksLevel(v.key)})
	e.line("x[mk] = mv")
	if dedup {
		collected := "all || !" + e.canonFunc(v.key) + "(mk)"
		if e.canonical {
			// The order check leaves no two keys of one projection in data, so a
			// decode, which clears the map first, has no key to delete.
			collected = "!collect && (" + collected + ")"
		}
		e.line("if %s {", collected)
		e.line("keys = append(keys, mk)")
		e.line("}")
	}
	e.line("}")
	if dedup {
		e.line("%s%s(x, keys, %s)", w, keepLastName, e.compareFunc(v.key))
		e.line("return err")
	} else {
		e.line("return nil")
	}
	e.line("}")
	e.line("")
}

// readScoped writes the statements of [emitter.read] in a block of their
// own, so that two values in one block do not declare a variable twice.
func (e *emitter) readScoped(v *value, dst string, p place) {
	e.line("{")
	e.read(v, dst, p, "")
	e.line("}")
}

// readPointer writes the body of the read function of the pointer type of
// v. The function reads the presence byte, sets *dst to nil for 0, and for 1
// decodes the value that follows into the value that *dst points at, or into
// a new one, and merges it there when the function takes the merge flag and
// the flag is set.
func (e *emitter) readPointer(v *value) {
	e.ret = retLength
	e.levelCheck()
	// wire.Presence fails for empty data, which an array, a map and an
	// interface pass to the function, and the loop of a slice never does.
	e.line("present, err := %sPresence(data, loc, num, off)", e.wire())
	e.check()
	e.line("if !present {")
	e.line("*dst = nil")
	e.line("return 1, nil")
	e.line("}")
	e.line("x := *dst")
	e.line("if x == nil {")
	e.line("x = new(%s)", e.p.typ(v.elem.typ))
	e.line("*dst = x")
	e.line("}")
	e.line("i := 1")
	merge := ""
	if e.merging[v.id] {
		merge = mergeParam
	}
	e.read(v.elem, "*x", place{loc: locParam, num: numParam, k: 1}, merge)
	e.line("return i, nil")
	e.line("}")
	e.line("")
}

// mergeParams returns the merge flag among the parameters of the read
// function of v and the comma after it, and nothing when the function takes
// no merge flag, as e.merging marks it.
func (e *emitter) mergeParams(v *value) string {
	if !e.merging[v.id] {
		return ""
	}
	return mergeParam + " bool, "
}

// readInterface writes the body of the read function of the interface
// value of v: one switch over the type numbers of its concrete types. A
// concrete type whose values refer to memory decodes into the value that
// the interface stores when that value has the type. When the function
// takes the merge flag, a concrete type that merges, as [value.merges]
// reports, decodes into that value whatever its memory, and merges into it
// when the flag is set.
func (e *emitter) readInterface(v *value) {
	w := e.wire()
	e.ret = retLength
	e.levelCheck()
	e.line("t, i := %sUvarint(data)", w)
	e.line("if i <= 0 {")
	e.fail(w + "ReadError(i, loc, num, off)")
	e.line("}")
	e.shortest("i", "i", locParam, numParam, "off")
	e.line("switch t {")
	e.line("case 0:")
	e.line("*dst = nil")
	for _, c := range v.variants {
		typ := e.p.typ(c.c.typ)
		merge := ""
		if e.merging[v.id] && c.val.merges() {
			merge = mergeParam
		}
		e.line("case %d:", c.c.num)
		if c.val.holdsMemory() || merge != "" {
			e.line("x, _ := (*dst).(%s)", typ)
		} else {
			e.line("var x %s", typ)
		}
		e.read(c.val, "x", place{loc: locParam, num: numParam, k: 1}, merge)
		e.line("*dst = x")
	}
	e.line("default:")
	e.fail(w + "TypeError(t, loc, num, off)")
	e.line("}")
	e.line("return i, nil")
	e.line("}")
	e.line("")
}

// levelCheck writes the statements that fail the decode of a value at a
// level below zero.
func (e *emitter) levelCheck() {
	e.line("if depth < 0 {")
	e.fail(e.wire() + "DepthError(loc, num, off)")
	e.line("}")
}

// mergeHelper writes the merge function of the inline struct of v.
func (e *emitter) mergeHelper(name string, v *value) {
	m := v.inline
	e.bits = tracked(m)
	e.doc(name + " merges the encoding in data, which starts at offset off of slab, into the " + m.name +
		" that m points at, as MergeKanon does. The decode enters depth levels below m at most.")
	e.line("func %s(m *%s, data []byte, slab string, off, depth int) error {", name, e.p.typ(m.typ))
	if e.seenWords() == 0 {
		e.fieldsBody(m)
		return
	}
	e.mergeBody(e.fn(opFields, v) + "(m, ")
}

// fieldsHelper writes the function of the loop of the decode of the inline
// struct of v, which has tracked fields.
func (e *emitter) fieldsHelper(name string, v *value) {
	m := v.inline
	e.bits = tracked(m)
	e.fieldsDoc(name, "the decode and the merge of "+m.name)
	e.line("func %s(m *%s, data []byte, slab string, off, depth int, seen %s) (%s, error) {", name,
		e.p.typ(m.typ), e.bitmap(), e.bitmap())
	e.fieldsBody(m)
}

// deselectHelper writes the function that zeroes the member of the union of
// the discriminator disc of m that the discriminator selects, in one switch
// on the discriminator. A member that the input contains calls it when the
// discriminator selects another member, so that a decoded value has one
// member set at most.
func (e *emitter) deselectHelper(name string, m *target, disc *types.Var) {
	e.doc(name + " zeroes the member of the union of " + m.name + fieldStep + disc.Name() + " that the " +
		"discriminator selects.")
	e.line("func %s(m *%s) {", name, e.p.typ(m.typ))
	e.line("switch m.%s {", disc.Name())
	for _, f := range m.fields {
		if f.member == nil || f.member.disc != disc {
			continue
		}
		e.line("case %s:", e.p.object(f.member.value))
		e.resetField(f, false)
	}
	e.line("}")
	e.line("}")
	e.line("")
}
