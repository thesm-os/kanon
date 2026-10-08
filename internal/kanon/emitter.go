// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"encoding/binary"
	"go/types"
	"strconv"
	"strings"
)

// Limits of the generated code.
const (
	// binaryScratch is the length of the stack array into which the
	// generated code appends the encoding of a type with an append method.
	// A longer encoding allocates.
	binaryScratch = 128
	// mapKeyBuffer is the number of map keys, or of keys with their values,
	// that the encode of a map sorts in a stack array, and the number of
	// keys that the decode of an ambiguous key type collects in one. A
	// larger map allocates a slice per encode, and a decode that collects
	// more keys allocates one.
	mapKeyBuffer = 16
	// freeListLength is the number of values of a map that its decode keeps
	// in a stack array, its free list, for the entries it decodes. Only a map
	// whose values refer to memory has a free list, and a decoded entry
	// beyond it allocates its value.
	freeListLength = 16
	// docWidth is the column at which the comments of the generated
	// functions wrap.
	docWidth = 80
	// maxPutTag is the smallest tag that takes three bytes, the tag of field
	// number 2048. wire.PutTag writes the smaller ones.
	maxPutTag = 1 << 14
)

// Import path and name of package wire, whose functions the generated code
// calls.
const (
	wirePath    = "go.thesmos.sh/kanon/wire"
	wirePackage = "wire"
)

// generatorVersion is the version of the generator, which the EnforceVersion
// constants of every code file state.
const generatorVersion = 3

// Names of the parameters of the helpers that locate a value in errors: the
// location, "Type.Field", and the field number.
const (
	locParam = "loc"
	numParam = "num"
)

// mergeParam is the name of the parameter of a read function that tells it
// to merge the decoded value into the value that the destination holds, as
// a repeated field merges.
const mergeParam = "merge"

// Prefixes of the error in the return statements of a generated function,
// which [emitter.ret] holds.
const (
	// retError precedes the error of a function that returns an error alone.
	retError = ""
	// retLength precedes the error of a function that returns a length.
	retLength = "0, "
	// retSeen precedes the error of a decode that returns its seen bitmap.
	retSeen = "seen, "
)

// Go spellings of the predeclared constants and of nil, which the generated
// code writes as values, and of the blank identifier.
const (
	trueName  = "true"
	falseName = "false"
	nilName   = "nil"
	blankName = "_"
)

// Names of the unexported methods of a -type struct, which the generated
// code of its package calls.
const (
	// encodeMethod writes the encoding without the length check of
	// EncodeKanon.
	encodeMethod = "encodeKanon"
	// decodeMethod and mergeMethod decode with a slab, an offset and a depth
	// instead of options.
	decodeMethod = "decodeKanon"
	mergeMethod  = "mergeKanon"
	// fieldsMethod decodes the fields for decodeMethod and mergeMethod.
	fieldsMethod = "fieldsKanon"
	// cloneMethod copies the receiver into a struct that exists.
	cloneMethod = "cloneKanon"
)

// emitter writes the declarations of one code file: the methods of its
// structs and the helpers that they call.
type emitter struct {
	p *printer
	// prefix begins the name of every helper that the file declares.
	prefix string
	// names maps each helper that the code calls to its name, and pending
	// lists the helpers that the file does not declare yet, in the order in
	// which the code first calls them.
	names   map[helperKey]string
	pending []helper
	// words maps the id of a value to the word that names its helpers, and
	// owners maps each word to that id.
	words  map[string]string
	owners map[string]string
	// bits maps each field that the decode of the struct being written
	// tracks, as [tracked] numbers them, to its bit in the seen bitmap.
	bits map[*field]int
	// ret precedes the error in the return statements of the function being
	// written: retSeen, retLength or retError.
	ret string
	// cls classifies the fields of the struct types of map keys in lookup
	// mode.
	cls classifier
	// generates reports whether a kanon directive of the package of the
	// file names the struct type t, whose code file declares the unexported
	// methods that the code of the package calls.
	generates func(t types.Type) bool
	// keyStructs maps the type string of each struct type with a kanon codec
	// in a map key to the numbers of its fields, which order its keys.
	keyStructs map[string]keyStruct
	// views marks the declarations of the -type structs whose view types
	// the code file declares.
	views map[types.Object]bool
	// merging marks the ids of the values whose read functions take the
	// merge flag, as [mergingValues] collects them.
	merging map[string]bool
	// inKey reports that the emitter writes the size or the encoding of a
	// map key, whose projection writes every float component of -0.0 as
	// +0.0: a float field is present when it is not zero, and a value that
	// contains a float, as [emitter.floats] reports, takes the helpers of
	// opKeySize, opKeyPut and opKeyPresent.
	inKey bool
	// keyTargets maps the id of each struct with a kanon codec in a map key
	// that contains a float to the target that writes its projection field by
	// field, as [emitter.keyTarget] builds it.
	keyTargets map[string]*target
	// canonical makes the decode of every struct of the file accept only the
	// canonical encoding of a value, as [emitter.fieldsBody] and
	// [emitter.read] check it.
	canonical bool
}

// line writes format, expanded with args, as one line of the file.
func (e *emitter) line(format string, args ...any) {
	e.p.line(format, args...)
}

// doc writes text as a comment, as [wrap] lays it out.
func (e *emitter) doc(text string) {
	for _, line := range wrap(text) {
		e.line("%s", line)
	}
}

// wire returns the name that the file imports package wire by, followed by
// a dot, and records the import.
func (e *emitter) wire() string {
	return e.p.use(wirePath, wirePackage) + "."
}

// rt returns the name that the file imports the runtime package kanon by,
// followed by a dot, and records the import.
func (e *emitter) rt() string {
	return e.p.use(runtimePath, runtimeName) + "."
}

// std returns the name that the file imports the standard-library package
// importPath by, and records the import.
func (e *emitter) std(importPath string) string {
	return e.p.std(importPath)
}

// tag returns the expression of the tag of field num with the wire format
// w: 3<<3 | wire.Bytes.
func (e *emitter) tag(num, w int) string {
	return strconv.Itoa(num) + "<<3 | " + e.wire() + wireName(w)
}

// putTag writes the statement that writes the tag of field num with the
// wire format w before buf[i] and moves i to its first byte.
func (e *emitter) putTag(num, w int) {
	if uint64(num)<<3|uint64(w) < maxPutTag {
		e.line("i = %sPutTag(buf, i, %s)", e.wire(), e.tag(num, w))
		return
	}
	e.line("i = %sPutUvarint(buf, i, %s)", e.wire(), e.tag(num, w))
}

// fail writes the return statement of the function being written with the
// error expression err.
func (e *emitter) fail(err string) {
	e.line("return %s%s", e.ret, err)
}

// check writes the statements that return err, the error variable of the
// generated code, when it is not nil.
func (e *emitter) check() {
	e.line("if err != nil {")
	e.fail("err")
	e.line("}")
}

// cast returns x, an expression of the basic type kind, as an expression of
// the type of v: x itself when v has that type, and a conversion otherwise.
func (e *emitter) cast(v *value, x string, kind types.BasicKind) string {
	if types.Identical(v.typ, types.Typ[kind]) {
		return x
	}
	return e.p.typ(v.typ) + "(" + x + ")"
}

// zero returns the zero value of the type t.
func (e *emitter) zero(t types.Type) string {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		if u.Info()&types.IsBoolean != 0 {
			return falseName
		}
		if u.Info()&types.IsString != 0 {
			return `""`
		}
		if u.Kind() == types.UnsafePointer {
			return nilName
		}
		return "0"
	case *types.Struct, *types.Array:
		return e.p.typ(t) + "{}"
	default:
		return nilName
	}
}

// as returns x, an expression of a value of v, as an expression of the
// basic type kind: x itself when v has that type, and a conversion
// otherwise.
func as(v *value, x string, kind types.BasicKind) string {
	if types.Identical(v.typ, types.Typ[kind]) {
		return x
	}
	return types.Typ[kind].Name() + "(" + x + ")"
}

// composite returns the zero value of the struct or array type t as an
// operand of a comparison: (T{}).
func (e *emitter) composite(t types.Type) string {
	return "(" + e.p.typ(t) + "{})"
}

// wrap returns the lines of text as a comment, wrapped at docWidth columns. A
// word longer than the width takes a line of its own.
func wrap(text string) []string {
	var out []string
	line := "//"
	for w := range strings.FieldsSeq(text) {
		if line != "//" && len(line)+1+len(w) > docWidth {
			out = append(out, line)
			line = "//"
		}
		line += " " + w
	}
	return append(out, line)
}

// tagSize returns the length of the tag of field num. The wire format takes
// the low three bits, so it does not change the length.
func tagSize(num int) int {
	return varintSize(uint64(num) << 3)
}

// tagBytes returns the bytes of the tag of field num with the wire format w:
// the shortest varint of the tag, the only form that a canonical decode
// matches.
func tagBytes(num, w int) []byte {
	return binary.AppendUvarint(nil, uint64(num)<<3|uint64(w))
}

// varintSize returns the length of the shortest varint of v.
func varintSize(v uint64) int {
	return len(binary.AppendUvarint(nil, v))
}

// deref returns the expression of the value that x points at: *x. x is a
// primary expression, such as m.Next or x[k], or a dereference in turn, so
// the operator applies to all of x.
func deref(x string) string {
	return "*" + x
}

// addr returns the expression of the address of the addressable value x:
// the pointer p for a dereference *p, and &x otherwise.
func addr(x string) string {
	if p, ok := strings.CutPrefix(x, "*"); ok {
		return p
	}
	return "&" + x
}

// primary returns x as the operand of an index, a slice or a selector:
// (*p) for a dereference *p, which binds looser than they do, and x
// otherwise.
func primary(x string) string {
	if strings.HasPrefix(x, "*") {
		return "(" + x + ")"
	}
	return x
}

// quoted returns s as a Go string literal.
func quoted(s string) string {
	return strconv.Quote(s)
}

// fieldLoc returns the Go literal of the location of the field f of m in
// errors: "Order.Count".
func fieldLoc(m *target, f *field) string {
	return quoted(m.name + fieldStep + f.name)
}

// structLoc returns the Go literal of the location of m in errors, for an
// error of the struct itself: "Order".
func structLoc(m *target) string {
	return quoted(m.name)
}
