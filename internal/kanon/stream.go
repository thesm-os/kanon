// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"errors"
	"go/types"
	"slices"
	"strconv"
	"strings"
)

// Parts of the names of the declarations of the stream decoder of a struct
// type T: the type TField of its streamed fields, whose constants append the
// field names, the type TStream, its constructor NewTStream, the decode
// method DecodeF of each streamed slice F, and the schema of T, which the
// prefix of the file and T complete.
const (
	fieldSuffix  = "Field"
	streamSuffix = "Stream"
	newPrefix    = "New"
	decodePrefix = "Decode"
	schemaWord   = "stream"
	// skipPrefix begins the name of the field of TStream into which Next
	// decodes the elements of a streamed slice that the caller skips.
	skipPrefix = "skip"
	// runName names the method of TStream that decodes a run of the fields of
	// a canonical type with a union, as [emitter.streamRun] writes it.
	runName = "run"
)

// Errors of the tag option stream on a field that does not stream.
var (
	errStreamInline = errors.New("kanon: the tag option stream applies to the fields of a struct type that a " +
		"-type flag names, and an inline struct has no stream decoder")
	errStreamUnexported = errors.New("kanon: the tag option stream applies to an exported field")
	errStreamMember     = errors.New("kanon: a union member decodes whole: remove the tag option stream")
	errStreamKind       = errors.New("kanon: the tag option stream applies to a byte slice, a string and a slice " +
		"of a struct type with a kanon codec, each of a type whose ValidateKanon the generated code does not call")
)

// streamable returns the error for the tag option stream on the field f of
// m when the stream decoder of m cannot return the value of f: m is an
// inline struct, which has no stream decoder; f is unexported, and the
// stream decoder exposes its streamed fields to other packages; f is a union
// member, whose occurrence replaces its union; or the value of f does not
// stream, as [streams] reports. It returns nil otherwise.
func streamable(m *target, f *field) error {
	switch {
	case m.inline:
		return errStreamInline
	case !f.obj.Exported():
		return errStreamUnexported
	case f.tag.union != "":
		return errStreamMember
	case !streams(f.val):
		return errStreamKind
	default:
		return nil
	}
}

// streams reports whether a field of v streams: a byte slice and a string,
// whose bytes the caller reads, and a slice of a struct type with a kanon
// codec, whose elements the caller decodes, each of a type whose
// ValidateKanon the generated code does not call, since that method reads the
// whole value.
func streams(v *value) bool {
	if v.validate {
		return false
	}
	switch v.kind {
	case kindString, kindBytes:
		return true
	case kindSlice:
		return v.elem.kind == kindStruct && v.elem.inline == nil
	default:
		return false
	}
}

// streamed returns the fields of m with the tag option stream, in
// declaration order.
func streamed(m *target) []*field {
	var out []*field
	for _, f := range m.fields {
		if f.tag.stream {
			out = append(out, f)
		}
	}
	return out
}

// streamNames returns the names of the package-level declarations of the
// stream decoder of m, whose streamed fields are fields, in the code file
// whose helpers take prefix: TField, a constant per field, the schema,
// TStream and NewTStream.
func streamNames(m *target, fields []*field, prefix string) []string {
	names := []string{m.name + fieldSuffix}
	for _, f := range fields {
		names = append(names, m.name+fieldSuffix+f.name)
	}
	return append(names, prefix+schemaWord+camel(m.name), m.name+streamSuffix, newPrefix+m.name+streamSuffix)
}

// streamType writes the stream decoder of the -type struct m, whose streamed
// fields are fields: the type TField with a constant per streamed field, the
// schema of m, the type TStream with its constructor and its methods, and for
// a canonical m with a union the method that decodes a run, as
// [emitter.streamRun] writes it.
func (e *emitter) streamType(m *target, fields []*field) {
	e.bits = tracked(m)
	field, stream := m.name+fieldSuffix, m.name+streamSuffix
	schema := e.prefix + schemaWord + camel(m.name)
	e.doc(field + " names a streamed field of " + m.name + ", which " + stream + ".Next returns. Its value is " +
		"the field number.")
	e.line("type %s int", field)
	e.line("")
	e.doc("The streamed fields of " + m.name + ".")
	e.line("const (")
	for _, f := range fields {
		e.line("%s%s %s = %d", field, f.name, field, f.num)
	}
	e.line(")")
	e.line("")
	e.streamSchema(m, fields, schema)
	e.streamStruct(m, fields, stream)
	e.streamOpen(m, stream, schema)
	e.streamNext(m, fields, stream)
	e.streamReaders(fields, stream)
	if e.canonical && unionBits(m) {
		e.streamRun(m, stream)
	}
}

// streamSchema writes the variable schema, the wire.StreamSchema of m and its
// streamed fields, fields: the location of m in errors, whether its decode
// is canonical, and per field its tag, its location and, for a slice, the
// name of the struct of its elements in errors and the bound of the tag
// option max.
func (e *emitter) streamSchema(m *target, fields []*field, schema string) {
	w := e.wire()
	e.doc(schema + " describes " + m.name + " to the wire.Stream of its stream decoder.")
	e.line("var %s = %sStreamSchema{", schema, w)
	e.line("Loc: %s,", structLoc(m))
	if e.canonical {
		e.line("Canonical: %s,", trueName)
	}
	e.line("Fields: []%sStreamField{", w)
	for _, f := range fields {
		elem := ""
		if f.val.kind == kindSlice {
			elem = ", Elem: " + quoted(types.TypeString(f.val.elem.typ, e.cls.relative))
		}
		if f.tag.max != 0 {
			elem += ", Max: " + strconv.Itoa(f.tag.max)
		}
		e.line("{Tag: %s, Loc: %s%s},", e.tag(f.num, wireBytes), fieldLoc(m, f), elem)
	}
	e.line("},")
	e.line("}")
	e.line("")
}

// streamStruct writes the type stream, the stream decoder of m, whose
// streamed fields are fields, with its docblock.
func (e *emitter) streamStruct(m *target, fields []*field, stream string) {
	names := make([]string, 0, len(fields))
	var bytes, bounded, clauses []string
	for _, f := range fields {
		names = append(names, f.name)
		if f.val.kind != kindSlice {
			bytes = append(bytes, f.name)
		}
		if f.tag.max != 0 {
			bounded = append(bounded, f.name)
		}
	}
	noAlloc := []string{"Reset", "Len"}
	if len(bytes) > 0 {
		clauses = append(clauses,
			"Read returns the bytes of "+series(bytes, "and")+" and WriteTo writes them to a writer")
		noAlloc = append(noAlloc, "Read")
	}
	if len(bytes) < len(fields) {
		noAlloc = append(noAlloc, "Element")
	}
	allocs := series(noAlloc, "and") + " do not allocate"
	if len(bytes) > 0 {
		allocs += ", and WriteTo allocates only what its writer allocates"
	}
	for _, f := range fields {
		if f.val.kind == kindSlice {
			clauses = append(clauses, decodePrefix+f.name+" decodes the elements of "+f.name)
		}
	}
	list := clauses[0]
	if len(clauses) > 1 {
		list = strings.Join(clauses[:len(clauses)-1], ", ") + ", and " + clauses[len(clauses)-1]
	}
	e.doc(stream + " decodes the encoding of a " + m.name + " from a reader, one field at a time, in input " +
		"order. It decodes each field without the tag option stream into the receiver that " + newPrefix + stream +
		" takes, with the code of DecodeKanon, and stops at each field with the option: " + list + ". Once Next " +
		"returns io.EOF, the receiver contains the value of the encoding, with " + series(names, "and") + " empty.")
	e.line("//")
	e.line("// # Checks")
	e.line("//")
	e.doc("The stream applies the checks of DecodeKanon to every byte, in the order of DecodeKanon, and returns " +
		"the error that DecodeKanon returns for the same encoding: a *kanon.DecodeError whose Offset counts from " +
		"the first byte of the encoding. It passes a streamed value to the caller before it has read the rest of " +
		"the encoding, so an error can follow the bytes and the elements that the caller has read. A caller uses " +
		"them only after Next returns io.EOF.")
	e.line("//")
	e.line("// # Limits")
	e.line("//")
	limits := "The stream has in memory the decoded fields of the encoding together, one element, and a read " +
		"buffer of 4096 bytes. A decoded field that would take the decoded fields past the Buffer of its " +
		"kanon.StreamOptions, and an element longer than that Buffer, fail with a *kanon.DecodeError that wraps " +
		"kanon.ErrLimit."
	allocated := "the stream and its read buffer of 4096 bytes"
	if len(bounded) > 0 {
		limits += " An element of " + series(bounded, "or") + " past the bound of the tag option max of its field " +
			"fails with a *kanon.DecodeError that wraps kanon.ErrMax."
		allocated = "the stream, its read buffer of 4096 bytes and the counts of the elements of " +
			series(bounded, "and")
	}
	e.doc(limits + " A reader that returns fewer than size bytes fails the call that needs the next byte with a " +
		"*kanon.DecodeError that wraps io.ErrUnexpectedEOF, at the offset of that byte. The methods return any " +
		"other error of the reader unchanged, and io.ErrNoProgress when the reader returns no byte and no error " +
		"100 times before the bytes that a call needs. After an error, every method returns that error.")
	e.line("//")
	e.line("// # Aliasing")
	e.line("//")
	e.doc("The strings of the receiver are substrings of the buffer of the decoded fields until the next call of " +
		"Reset. The strings of an element are substrings of the buffers of the stream until the next call of " +
		"Next, Element or a decode method. A caller that needs a value after that copies it with CloneKanon.")
	e.line("//")
	e.line("// # Allocation contract")
	e.line("//")
	e.doc(newPrefix + stream + " allocates " + allocated + ". Next and the decode methods allocate nothing once " +
		"the buffers of the stream have grown to the decoded fields and to the longest element of more than 4096 " +
		"bytes, except where DecodeKanon allocates for the same values. " + allocs + ".")
	e.line("//")
	e.line("// # Concurrency")
	e.line("//")
	e.doc("A " + stream + " is not safe for concurrent use. It writes the receiver until Next returns io.EOF or " +
		"an error, so no other goroutine reads the receiver before then.")
	e.line("type %s struct {", stream)
	e.doc("s reads the encoding, and m is the receiver of the decoded fields.")
	e.line("s %sStream", e.wire())
	e.line("m *%s", e.p.typ(m.typ))
	if e.seenWords() > 0 {
		e.doc("seen records the tracked fields that the encoding contains, as the decode of " + m.name +
			" records them.")
		e.line("seen %s", e.bitmap())
	}
	if e.canonical && unionBits(m) {
		e.doc(membersName + " records the unions whose member the runs before contain.")
		e.line("%s [%d]uint64", membersName, (len(m.discriminators)+63)/64)
	}
	for _, f := range fields {
		if f.val.kind == kindSlice {
			e.doc(skipPrefix + f.name + " is the element into which Next decodes the elements of " + f.name +
				" that the caller skips.")
			e.line("%s%s %s", skipPrefix, f.name, e.p.typ(f.val.elem.typ))
		}
	}
	e.line("}")
	e.line("")
}

// streamOpen writes the constructor of stream, the stream decoder of m,
// which initializes it with schema, and its Reset method, which runs the
// statements of [emitter.decodePrologue].
func (e *emitter) streamOpen(m *target, stream, schema string) {
	typ, reader := e.p.typ(m.typ), e.std(ioPath)+".Reader"
	e.doc(newPrefix + stream + " returns a " + stream + " that reads the encoding of a " + m.name + ", size " +
		"bytes, from r and decodes it into m, which Reset prepares as it states. opts sets the nesting limit and " +
		"the buffer limit of the stream.")
	e.line("func %s%s(r %s, size int64, m *%s, opts %sStreamOptions) *%s {", newPrefix, stream, reader, typ, e.rt(),
		stream)
	e.line("d := new(%s)", stream)
	e.line("d.s.Init(&%s, opts)", schema)
	e.line("d.Reset(r, size, m)")
	e.line("return d")
	e.line("}")
	e.line("")
	e.doc("Reset makes d read the encoding of a " + m.name + ", size bytes, from r into m, and reuses the buffers " +
		"of d. It clears the fields of m that the encoding leaves out, and the fields that the decode does not " +
		"track, the streamed fields among them, as DecodeKanon clears them. It reads at most size bytes from r. A " +
		"size below 0, or above math.MaxInt, makes the first call of Next fail with kanon.ErrStreamSize. The " +
		"strings that d decoded before are substrings of the buffers that it reuses.")
	e.line("func (d *%s) Reset(r %s, size int64, m *%s) {", stream, reader, typ)
	e.line("d.s.Reset(r, size)")
	e.line("d.m = m")
	if e.seenWords() > 0 {
		e.line("d.seen = %s{}", e.bitmap())
	}
	if e.canonical && unionBits(m) {
		e.line("d.%s = [%d]uint64{}", membersName, (len(m.discriminators)+63)/64)
	}
	e.decodePrologue(m)
	e.line("}")
	e.line("")
}

// streamNext writes the Next method of stream, the stream decoder of m,
// whose streamed fields are fields. It decodes the elements that the caller
// skips of each streamed slice, calls the Next of the wire.Stream, decodes
// the run that it returns, and opens the streamed field after the run. The
// run decodes with the method of [emitter.streamRun] for a canonical m with a
// union, and otherwise with the unexported method of m that DecodeKanon calls
// for the fields: fieldsKanon with tracked fields, and mergeKanon without. A
// struct with tracked fields keeps its seen bitmap across the runs, and runs
// the statements of [emitter.decodeEpilogue] when Open returns io.EOF after
// the last run.
func (e *emitter) streamNext(m *target, fields []*field, stream string) {
	twice := ""
	if !e.canonical {
		twice = " A streamed field that occurs twice in the encoding returns twice."
	}
	e.doc("Next moves to the next streamed field of the encoding and returns it. It first moves past the rest " +
		"of the current field: it discards the unread bytes of a byte slice or a string, and decodes the unread " +
		"elements of a slice into an element of its own, so that their checks run. It then decodes the fields " +
		"before the next streamed field into the receiver, and checks the tag and the length of the streamed " +
		"field as DecodeKanon checks them. At the end of the encoding it finishes the decode of the receiver as " +
		"DecodeKanon finishes it, and returns io.EOF." + twice)
	field := m.name + fieldSuffix
	e.line("func (d *%s) Next() (%s, error) {", stream, field)
	for _, f := range fields {
		if f.val.kind != kindSlice {
			continue
		}
		e.line("for d.s.More(%d) {", f.num)
		e.line("if err := d.%s%s(&d.%s%s); err != nil {", decodePrefix, f.name, skipPrefix, f.name)
		e.line("return 0, err")
		e.line("}")
		e.line("}")
	}
	e.line("r, err := d.s.Next()")
	e.line("if err != nil {")
	e.line("return 0, err")
	e.line("}")
	args := "r.Data, r.Slab(), 0, r.Depth"
	run, own := "d."+runName, e.canonical && unionBits(m)
	if e.seenWords() == 0 {
		if !own {
			run = "d.m." + mergeMethod
		}
		e.line("if err := %s(%s); err != nil {", run, args)
		e.line("return 0, d.s.Fail(err)")
		e.line("}")
		e.line("n, err := d.s.Open()")
	} else {
		if !own {
			run = "m." + fieldsMethod
		}
		e.line("m := d.m")
		e.line("seen, err := %s(%s, d.seen)", run, args)
		e.line("if err != nil {")
		e.line("return 0, d.s.Fail(err)")
		e.line("}")
		e.line("d.seen = seen")
		e.line("n, err := d.s.Open()")
		e.line("if err == %s.EOF {", e.std(ioPath))
		e.decodeEpilogue(m)
		e.line("}")
	}
	e.line("return %s(n), err", field)
	e.line("}")
	e.line("")
}

// streamReaders writes the methods of stream, a stream decoder whose
// streamed fields are fields, that read the value of the current field. It
// writes Len, Read and WriteTo when a byte slice or a string streams, and
// Element and the decode method of each streamed slice when a slice streams.
func (e *emitter) streamReaders(fields []*field, stream string) {
	e.doc("Len returns the declared length in bytes of the value of the current field, and 0 before the first " +
		"call of Next and after Next returns io.EOF or an error.")
	e.line("func (d *%s) Len() int64 {", stream)
	e.line("return d.s.Len()")
	e.line("}")
	e.line("")
	var bytes, slices, bounded []string
	for _, f := range fields {
		if f.val.kind == kindSlice {
			slices = append(slices, f.name)
		} else {
			bytes = append(bytes, f.name)
		}
		if f.tag.max != 0 {
			bounded = append(bounded, f.name)
		}
	}
	if len(bytes) > 0 {
		e.doc("Read reads the next bytes of the value of the current field into p, when the current field is " +
			series(bytes, "or") + ": at most len(p) bytes, and at most the bytes that remain of the value. It " +
			"returns io.EOF when no byte of the value remains, and when the current field is another field.")
		e.line("func (d *%s) Read(p []byte) (int, error) {", stream)
		e.line("return d.s.Read(p)")
		e.line("}")
		e.line("")
		e.doc("WriteTo writes the rest of the value of the current field to w, when the current field is " +
			series(bytes, "or") + ", and returns the number of bytes that w took. io.Copy from d calls it, so it " +
			"allocates nothing. It writes nothing, and returns 0 and nil, when no byte of the value remains and " +
			"when the current field is another field. It returns the error of w, io.ErrShortWrite when w takes " +
			"fewer bytes than it passes without an error, and the errors of the reader that Read returns.")
		e.line("func (d *%s) WriteTo(w %s.Writer) (int64, error) {", stream, e.std(ioPath))
		e.line("return d.s.WriteTo(w)")
		e.line("}")
		e.line("")
	}
	if len(slices) == 0 {
		return
	}
	bound := ""
	if len(bounded) > 0 {
		bound = " It fails at an element of " + series(bounded, "or") + " past the bound of the tag option max of " +
			"its field, before it reads the length, as DecodeKanon fails."
	}
	e.doc("Element reads the length of the next element of the current field, when the current field is " +
		series(slices, "or") + ", and returns it without reading the element. It returns the same length until a " +
		"decode method decodes the element. It checks the length as DecodeKanon checks it, and returns io.EOF " +
		"after the last element and when the current field is another field." + bound)
	e.line("func (d *%s) Element() (int64, error) {", stream)
	e.line("return d.s.Element()")
	e.line("}")
	e.line("")
	for _, f := range fields {
		if f.val.kind == kindSlice {
			e.decodeElement(f, stream)
		}
	}
}

// decodeElement writes the decode method of the streamed slice f on stream,
// the stream decoder of the struct of f, which decodes one element with the
// code that the decode of the slice runs for it: the unexported decode method
// of a -type struct of the package, and DecodeKanon of any other struct with
// a kanon codec. The wire.Stream checks the length of the element and the
// nesting limit before it.
func (e *emitter) decodeElement(f *field, stream string) {
	elem, w := f.val.elem, e.wire()
	call := method("e", decodeKanonName) + "(r.Data, " + w + "Nested(r.Slab(), 0, r.Depth))"
	if e.generates(elem.typ) {
		call = method("e", decodeMethod) + "(r.Data, r.Slab(), 0, r.Depth)"
	}
	name := decodePrefix + f.name
	bound := ""
	if f.tag.max != 0 {
		bound = " It fails at an element past the bound of the tag option max of " + f.name + ", as Element fails."
	}
	e.doc(name + " decodes the next element of " + f.name + " into e, when " + f.name + " is the current field, " +
		"with the code that DecodeKanon runs for an element of " + f.name + ", at the same depth. It reads the " +
		"length of the element first when Element has not read it." + bound + " It returns io.EOF after the last " +
		"element, and when the current field is another field. The strings of e are substrings of the buffers " +
		"of d until the next call of Next, Element or a decode method.")
	e.line("func (d *%s) %s(e *%s) error {", stream, name, e.p.typ(elem.typ))
	e.line("r, err := d.s.Value(%d)", f.num)
	e.line("if err != nil {")
	e.line("return err")
	e.line("}")
	e.line("if err := %s; err != nil {", call)
	e.line("return d.s.Fail(err)")
	e.line("}")
	e.line("return nil")
	e.line("}")
	e.line("")
}

// streamRun writes the method of stream, the stream decoder of the canonical
// struct m with a union, that decodes a run of fields: the canonical decode
// of the fields of m without the streamed fields, as [emitter.inOrder] writes
// it, with the bitmap of the unions whose member the runs before contain,
// which the method keeps in the stream. A member then fails after a member
// of its union in an earlier run, as it fails in the decode of m.
func (e *emitter) streamRun(m *target, stream string) {
	e.ret = retError
	results, params := errorName, ""
	if e.seenWords() > 0 {
		e.ret = retSeen
		results, params = "("+e.bitmap()+", "+errorName+")", ", seen "+e.bitmap()
	}
	e.doc(runName + " decodes the fields in data, a run of the encoding that starts at offset off of slab, " +
		"into the receiver of d, as the canonical decode of " + m.name + " decodes them, and records the " +
		"members of the unions of " + m.name + " for the runs after it. The decode enters depth levels below " +
		"the receiver at most.")
	e.line("func (d *%s) %s(data []byte, slab string, off, depth int%s) %s {", stream, runName, params, results)
	e.line("m, %s := d.m, d.%s", membersName, membersName)
	e.line("var %s uint64", priorName)
	e.line("i := 0")
	e.inOrder(m, slices.DeleteFunc(byNumber(m.fields), func(f *field) bool { return f.tag.stream }))
	e.line("d.%s = %s", membersName, membersName)
	e.fail(nilName)
	e.line("}")
	e.line("")
}

// series returns items joined as a series in a sentence, with conjunction,
// "and" or "or", before the last item: "a", "a and b", and "a, b and c".
func series(items []string, conjunction string) string {
	if len(items) == 1 {
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}
