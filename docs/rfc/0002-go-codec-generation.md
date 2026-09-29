---
rfc: 0002
title: Generated Go codecs, their runtime and their public interface
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-09-27
updated: 2026-09-29
discussion: none
supersedes: none
superseded-by: none
produces-adr: ADR-0007, ADR-0008, ADR-0009, ADR-0010, ADR-0011, ADR-0012, ADR-0013, ADR-0021, ADR-0022
---

# RFC-0002: Generated Go codecs, their runtime and their public interface

## Summary

`kanon` is a code generator that reads Go struct types and writes their encoders and decoders
for the kanon wire format, as `stringer` writes `String` methods: a `//go:generate go tool
kanon -type=A,B` directive produces `<file>.kanon.go`, the methods of the named types, and
`<file>.kanon_test.go`, their conformance test. The generated code calls a small runtime: the
package `kanon` declares the `Message` interface that every generated type satisfies, the
decode `Options`, the error types and the version constants, and the package `kanon/wire`
provides the varint, time and skip functions that the generated code shares. The generated
methods encode with zero allocations into a sized buffer, decode with zero allocations into a
reused value, and keep exact presence for pointers. Field numbers are locked in the generated
file and checked against a git revision.

## Motivation

The wire format needs Go struct types as its schema, so that an application declares its data
model once. A codec written by hand costs about 35 lines per field, which is what the
generated code measures for a struct of 70 fields, and a class of bugs that the compiler does
not catch. Reflection at run time costs 5 to 20 times the time of generated code, measured as
gob and json against a generated prototype on 13 types.

Generated code needs a shared interface, so that a frame writer, a batch reader or an RPC
codec accepts any generated type. It needs typed errors, so that a storage engine tells a
truncated record from a corrupt one and from one that exceeds a limit. It needs decode
options, so that a caller passes a slab and a nesting limit without a method signature per
option. And it needs an allocation contract, so that a server with a million messages per
second does not spend its time in the garbage collector. A runtime package gives all four,
and one implementation of the varint, time and skip functions to fuzz and mutate instead of
one copy per generated file.

## Detailed design

### The directive

```go
//go:generate go tool kanon -type=Order,Line
```

- `-type` names one or more types of the package, separated by commas: struct types, and named
  bools, numbers, strings, slices, arrays and maps. It is required. A struct type gets its
  codec, and any other type gets the `ValidateKanon` method of the Types section.
- kanon writes `<base>.kanon.go` and `<base>.kanon_test.go`, where `<base>` is the name of the
  file with the directive, which `go generate` names in `GOFILE`. Outside `go generate`, the
  file is the only argument.
- kanon writes a file only when its content changes.
- `-views` also generates the view types of the Views section, one per struct type.
- `-validate=name` names a method `func (T) name() error` that each type of `-type` other
  than a struct declares on a value receiver. The method can be unexported. The generated
  `ValidateKanon` returns its error, and without the flag it returns nil. Generation fails
  when such a type lacks the method or declares `ValidateKanon` itself, and when `-validate`
  is set and `-type` names only struct types.
- With `KANON_CHECK=<revision>`, kanon checks the field numbers against the generated files at
  that git revision and writes nothing.
- The exit status is 0 on success, 1 when generation or the check fails, and 2 on a usage
  error.

### Field numbers

Every field encodes under a number. A field without a tag takes the smallest free number in
declaration order, and the generated file records the numbers of each type in a
`//kanon:numbers` line:

```go
//kanon:numbers Order ID=1 Lines=2 Note=3
```

Regeneration reads the line and keeps the numbers, so a struct can change and old data still
decodes:

- A field keeps its number when fields are added, removed or reordered.
- A new field takes the smallest number that is neither taken nor reserved.
- A removed field's number is reserved. kanon does not give it to another field, and a tag may
  name it.
- A tag that gives a recorded field another number fails the generation.
- A renamed field is a new field. Tag it with the number of its old name to keep the data.

The check mode compares the numbers of a change against a revision, such as the branch it
merges into, and fails on a renumbered field, on a removed number that is not reserved, and on
a reserved number that a field took without a tag naming it. A CI job runs it on every pull
request.

### Tags

```go
type Order struct {
	ID    string        `kanon:"7"`             // field number 7
	Skip  int           `kanon:"-"`             // not encoded
	Count int64         `kanon:",fixed"`        // fixed64 instead of a varint
	seq   uint64        `kanon:""`              // unexported, encoded because it has a tag
	Kind  Kind
	Text  string        `kanon:",union=Kind"`   // member of the union that Kind selects
	Num   int64         `kanon:",union=Kind"`
	Shape Shape         `kanon:",types=Circle|*Square"` // the concrete types of the interface
	Rest  []byte        `kanon:",unknown"`      // keeps unknown fields
}
```

- A field of a function or a channel type, or of a pointer to one at any depth, is not
  encoded, as gob leaves it out, and neither is an unexported field without a kanon tag. Any
  kanon tag on an unexported field other than `-`, `kanon:""` included, encodes it. Only the
  code of the field's package can read the field, so the generator rejects such a field in a
  struct of another package that it would encode inline or order as a map key, and a view has
  no method for it.
- An embedded field is a field named after its type, as gob encodes it: an embedded struct
  encodes as a nested struct, and an embedded struct of an unexported type is left out unless
  a kanon tag opts it in. kanon does not promote the fields of an embedded struct into the
  outer struct, as encoding/json does.
- `fixed` applies to the 32- and 64-bit integers in the field's type, except map keys. Varints
  are the default, and `fixed` is the opt-in for fields whose values are large or whose blocks
  compress better with fixed widths.
- `union=D` makes the field a member of the union whose discriminator is field `D`, an
  exported field of an integer type. The constant that selects the member is named after the
  discriminator's type and the member, with the member's first letter in upper case:
  `KindText` selects `Text`, or an unexported `text`, when `Kind` has type `Kind`, and
  `external.ChoiceText` when the type is `external.Choice`.
- `types=A|B|*C` lists the concrete types of every interface in the field's type, as gob
  registers types. Each entry is a type expression in the scope of the file. The generated
  file records the type numbers under the field's path, and regeneration keeps them as it
  keeps field numbers.
- `unknown` marks the one exported `[]byte` field of a struct that keeps unknown fields.

### Types

The generator encodes every type that gob and json encode:

- bool, the integer types, `time.Duration`, the float and complex types, string, `[]byte`,
  `[N]byte`, slices, arrays, maps with any comparable key type, pointers at any depth and
  `time.Time`.
- A struct type named by a `-type` directive of its package, through the methods of its own
  codec.
- Any other struct type, of this package or another, as an inline struct. The generated file
  declares its functions and records its field numbers under its type name, or under the path
  of its first field for an anonymous struct type.
- A type with `MarshalBinary` and `UnmarshalBinary`, `GobEncode` and `GobDecode`, or
  `MarshalText` and `UnmarshalText`, as an opaque value through the first of those families it
  has. `AppendBinary` or `AppendText` is used when the type has it. An opaque value is absent
  when it equals the zero value of its type and `==` compares every bit of the type, which
  rules out a float, a complex number and an interface in it. The generated code compares
  the value with the zero value and does not call a method of the type for it, so a zero
  value that the type cannot encode, such as a zero digest, leaves its field out. An opaque
  value of any other type is present when its encoding has at least one byte.
- A named type other than a struct with the method `func (T) ValidateKanon() error`, a
  `kanon.Validator`, as a value of its underlying type, ahead of its binary, gob and text
  methods. A `-type` directive of its package generates the method, and a method written by
  hand works alike. The generated code calls the method on every value of the type that it
  encodes or decodes, except a zero value that the encoding leaves out, and fails with its
  error. The tag option `fixed`, the order of map keys and the view methods treat the type as
  its underlying type. The generator rejects the method on a struct type, on a pointer
  receiver and with another signature, and on a type whose only value is its zero value,
  since a constant encoding has no value to check. A named interface type whose method set
  has the method encodes as any other interface. The wire format lists adding or removing the
  method on a type with binary, gob or text methods as incompatible in both directions.
- Interfaces with a `types` list, inside maps, slices, arrays, pointers and unions included.
- Generic struct instantiations, which share the field numbers of their generic type.

A map key may be any comparable type, including a struct, an array, a pointer, a time or an
interface whose listed types are comparable. A struct key must not contain an interface, since
its order depends on a type list of another field, nor a union member, whose encoding depends
on its discriminator. Struct keys order by field number.

A key encodes as its projection, as the wire format defines. The encode of a map fails with an
`*EncodeError` that wraps `ErrInvalidKey` for a key with a NaN component, and with one that
wraps `ErrAmbiguousKey` for two keys of one projection, such as two pointers to equal values
or two structs that differ only in a skipped field. A decoded key is a new value with the
projection of the encoded key. A pointer key decodes to a new pointer, and keys of one
projection decode to one entry.

A type may contain itself through a pointer, a slice, a map or an interface, and its values
encode as finite trees, as the fixtures test with a struct key that contains itself behind a
pointer, a named pointer type that points at itself, and trees of interfaces. `SizeKanon`,
`EncodeKanon`, `CloneKanon`, `Reset` and the order of map keys follow pointers and interfaces
without looking for a cycle, since `SizeKanon` returns no error. A value whose pointers or
interfaces form a cycle, such as a node whose `Next` points at the node itself, is outside
their contract, and its `SizeKanon` recurses without end. The methods through which an opaque
type encodes itself must write the same bytes for an unchanged value and return an error
rather than panic.

### The runtime: package kanon

```go
package kanon

// Message is the method set that kanon generates for a struct type T. *T
// implements it.
//
// # Allocation contract
//
// SizeKanon, EncodeKanon and AppendBinary into a buffer with spare capacity
// do not allocate, except for an opaque type without an append method, an
// opaque encoding longer than 128 bytes, a map of more than 16 keys other
// than bools, and a value whose ValidateKanon allocates. DecodeKanon with a
// Slab into a receiver that decoded before does not allocate, except for an
// opaque value, a time in a zone that the platform allocates, a map with
// more than 16 values that refer to memory, a map key that refers to
// memory, a value that an interface stores by value, and a value whose
// ValidateKanon allocates. Without a Slab, a decode allocates the copy of
// its input once when T contains a string.
type Message interface {
	encoding.BinaryAppender
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler

	// SizeKanon returns the length of the encoding in bytes, for a value
	// whose encode succeeds.
	SizeKanon() int

	// EncodeKanon writes the encoding into the last SizeKanon bytes of buf
	// and returns their count. When buf is shorter than SizeKanon bytes, it
	// writes nothing and returns io.ErrShortBuffer. It returns an
	// *EncodeError when an opaque value fails to encode itself, the
	// ValidateKanon of a Validator rejects a value, an interface stores a
	// type its list does not name, or a map has an invalid key or two keys
	// of one projection. After an error, buf does
	// not contain a valid encoding, and AppendBinary returns its buffer
	// unchanged.
	EncodeKanon(buf []byte) (int, error)

	// DecodeKanon sets the receiver to the value encoded in data and reuses
	// the memory of the receiver: its slices, maps, pointers and nested
	// values. It first clears the fields that the encoding leaves out, as
	// Reset does. Every decoded string is a substring of opts.Slab, or of
	// one copy of data when opts.Slab is empty. It returns a *DecodeError.
	// After an error the receiver contains the fields decoded before it,
	// and the failed field has an unspecified value.
	DecodeKanon(data []byte, opts Options) error

	// MergeKanon decodes data into the receiver without resetting it first,
	// with the options of DecodeKanon. Slices append, maps add entries,
	// nested structs merge, and other fields take the value in data. The
	// fields that the encoding leaves out keep their values. A map of the
	// receiver with two keys of one projection fails the merge with
	// ErrAmbiguousKey before the map changes.
	MergeKanon(data []byte, opts Options) error

	// Reset clears every field of the receiver, including the fields that
	// the encoding leaves out: skipped fields, unexported fields without a
	// kanon tag, and fields of a function or a channel type or of a pointer
	// to one. An encoded scalar becomes its zero value, and an encoded
	// pointer or interface becomes nil. An encoded slice or map becomes
	// empty and keeps its storage for a decode
	// to reuse: every element in the capacity of a slice is reset, a pointer
	// element keeps its allocation with its value reset, and the entries of
	// a map are deleted. A skipped slice or map becomes nil. The receiver
	// need not be reflect.DeepEqual to a new zero value, but no decoded
	// content remains in its values or in the storage that it keeps.
	Reset()
}

// Cloner is the method set of a generated *T with its clone method, which
// names T and so is not part of Message.
type Cloner[T any] interface {
	Message

	// CloneKanon returns a deep copy that shares no memory with the receiver
	// or with the slab it decoded from, with the zero value in the fields
	// that the encoding leaves out, map keys included. A map with two keys
	// of one projection, which does not encode, can have one entry for them
	// in the copy. A nil receiver returns nil.
	CloneKanon() *T
}

// Validator is the method of a named type other than a struct, which the
// generated code encodes as its underlying type. The generated code calls
// ValidateKanon on every value of the type that it encodes or decodes,
// except a zero value that the encoding leaves out, and on the value of a
// field that a view method reads, and fails the encode, the decode or the
// view with its error. A view method returns the zero value for a field that
// the encoding leaves out. SizeKanon, Reset and CloneKanon do not call it.
type Validator interface {
	// ValidateKanon returns nil for a value that kanon encodes and decodes,
	// and the reason that it rejects any other value.
	ValidateKanon() error
}

// Options control DecodeKanon and MergeKanon. The zero Options copy the
// input once and apply DefaultDepth.
type Options struct {
	// Slab is a string that data is a substring of, and Offset is the index
	// of data[0] in it. Decoded strings are substrings of Slab, so the
	// decode allocates nothing for them. An empty Slab makes the decode copy
	// data once and use the copy as the slab.
	Slab   string
	Offset int
	// Depth is the number of nested values the decode enters at most. Zero
	// means DefaultDepth.
	Depth int
}

// DefaultDepth is the nesting limit of a decode whose Options.Depth is 0.
const DefaultDepth = 100

// Causes of a DecodeError, which DecodeError.Unwrap returns. Truncation
// unwraps to io.ErrUnexpectedEOF, so a stream reader that retries on it
// needs no kanon-specific check.
var (
	ErrMalformed    = errors.New("kanon: malformed input")
	ErrRange        = errors.New("kanon: value outside the range of the type")
	ErrDepth        = errors.New("kanon: nested deeper than the limit")
	ErrUnknownType  = errors.New("kanon: interface type number not listed")
	ErrRepeatedView = errors.New("kanon: struct field occurs twice in a view")
)

// Causes of both an EncodeError and a DecodeError for a map key.
var (
	// ErrInvalidKey marks a map key with a NaN component.
	ErrInvalidKey = errors.New("kanon: map key has a NaN component")
	// ErrAmbiguousKey marks two keys of one map with the same projection.
	ErrAmbiguousKey = errors.New("kanon: two map keys encode alike")
)

// DecodeError is the error of DecodeKanon and MergeKanon: the struct type,
// the field and its number, the offset in the slab, the cause and a detail.
// Field is "" and Number is 0 when the struct itself is malformed.
type DecodeError struct {
	Type   string
	Field  string
	Number int
	Offset int
	Detail string
	Err    error
}

// Error returns "kanon: Type.Field (field N) at offset M: detail".
func (e *DecodeError) Error() string

// Unwrap returns the cause: io.ErrUnexpectedEOF, ErrMalformed, ErrRange,
// ErrDepth, ErrUnknownType, ErrInvalidKey, ErrAmbiguousKey,
// ErrRepeatedView, the error of an opaque value's own decoding or the error
// of a ValidateKanon.
func (e *DecodeError) Unwrap() error

// ErrUnlistedType is the cause of an EncodeError for an interface that
// stores a type its list does not name.
var ErrUnlistedType = errors.New("kanon: type not listed in the tag option types")

// EncodeError is the error of EncodeKanon: the struct type, the field and
// its number, and the cause, which is ErrUnlistedType, ErrInvalidKey,
// ErrAmbiguousKey, the error of an opaque value's own encoding or the error
// of a ValidateKanon.
type EncodeError struct {
	Type   string
	Field  string
	Number int
	Err    error
}

func (e *EncodeError) Error() string
func (e *EncodeError) Unwrap() error

// EnforceVersion is a compile-time check that a generated file and the
// runtime it imports agree. A generated file declares
//
//	const (
//		_ = kanon.EnforceVersion(1 - kanon.MinVersion)
//		_ = kanon.EnforceVersion(kanon.MaxVersion - 1)
//	)
//
// with the generator's version in place of 1. An unsigned constant below
// zero fails to compile, so a runtime that dropped support for the
// generator's version, or a generator newer than the runtime, is caught at
// build time.
type EnforceVersion uint

// MinVersion and MaxVersion are the oldest and newest generator versions
// this runtime supports.
const (
	MinVersion = 1
	MaxVersion = 1
)
```

`UnmarshalBinary(data)` is generated as `DecodeKanon(data, Options{})`, which copies the input
once. A caller that owns the buffer passes `Options{Slab: s, Offset: o}`, and with
`unsafe.String` over the buffer the decode allocates nothing and the decoded strings alias the
buffer.

Errors allocate. Every generated error path constructs a `*DecodeError` or `*EncodeError`,
and the allocation contract excludes failure paths.

### The runtime: package wire

`kanon/wire` is the support package that generated code calls. It is exported because
generated code in other modules imports it, and it is not an API for applications.

```go
package wire

// Uvarint reads the varint at the start of data and returns its value and
// length: 0 when data ends inside it, and -1 when it exceeds 64 bits.
func Uvarint(data []byte) (uint64, int)

// SizeUvarint returns the encoded length of v, 1 to 10.
func SizeUvarint(v uint64) int

// PutUvarint writes v backward so that it ends at buf[i] and returns the
// offset of its first byte.
func PutUvarint(buf []byte, i int, v uint64) int

// Zigzag and Unzigzag map between signed values and their varint form.
func Zigzag(v int64) uint64
func Unzigzag(u uint64) int64

// Skip returns the length of the value of a field with the given tag at the
// start of data, or an error for an invalid wire format or truncated data.
func Skip(data []byte, tag uint64) (int, error)

// CountValues returns the number of length-prefixed values in data, and
// CountVarints the number of varints, for pre-sizing a slice.
func CountValues(data []byte) int
func CountVarints(data []byte) int

// SizeTime, PutTime and Time encode and decode a time.Time without its
// length prefix.
func SizeTime(t time.Time) int
func PutTime(buf []byte, i int, t time.Time) int
func Time(data []byte) (time.Time, error)

// Take moves the value stored under key in a free list to its end and
// returns the new length, for map decodes that reuse values.
func Take[K comparable, V any](keys []K, values []V, n int, key K) int
```

The one-byte case of every varint read and write is generated inline at the call site, and
`wire.Uvarint` and `wire.PutUvarint` handle the longer forms. Go inlines a callee of cost
above 20 only into a function below 5,000 nodes, and the decode function of a struct with many
fields is above it, so an out-of-line call per one-byte read would cost about 2 ns per field.

### The decode model

- **Exact presence.** A pointer field that the input does not contain is nil after
  `DecodeKanon`, a pointer to a zero value round-trips as a pointer, and a nested struct
  present with an empty encoding round-trips as a zero struct.
- **Reuse.** A decode into a receiver that decoded before writes into its memory: slices are
  resliced and their elements decoded in place, map values are taken from a free list of the
  previous values, pointers keep their targets, and nested structs decode in place. A seen
  bitmap tracks the fields the input contains, and the decode sets the pointers, interfaces
  and maps it did not see to nil or empty at the end.
- **Merge.** `MergeKanon` and a repeated field number both merge as the wire format defines.
- **Map keys.** A decoded key with a NaN component fails with `ErrInvalidKey`. Keys of one
  projection are one key. When a key type can decode two such keys to distinct Go values, as
  pointers and times in a zone that the platform allocates do, the decode sorts those keys
  after it reads the map and deletes every entry of a key whose projection a later key
  repeats. A merge into a map with two keys of one projection fails with `ErrAmbiguousKey`
  before it changes the map.
- **Depth.** Each nested value costs one level. A decode that would enter a level below zero
  fails with `ErrDepth`.
- **Unknown fields.** Without an `unknown` field, unknown numbers are skipped. With one, their
  bytes are appended to it, and `EncodeKanon` writes the field's bytes after the known fields.
  A storage engine that rewrites records of a type without the field treats the records as
  opaque bytes, since a rewrite through the type drops the fields it does not know.
- **Range.** A value outside its integer type fails with `ErrRange`, including `int`, `uint`
  and `uintptr` on a platform where they are 32 bits wide.

### The encode model

`SizeKanon` computes the size in one pass, and `EncodeKanon` writes backward from the end of
the buffer in a second pass, so that each nested value's length is known when its prefix is
written and every value is written once. A map sorts its keys before it writes its entries:

- Integer and string keys with values other than structs, arrays and opaque values sort with
  their values as key-value pairs in a stack array, with an insertion sort, in a map of at
  most 16 entries. A larger map with such keys sorts its keys in a slice of its length, and a
  lookup finds the value of each key.
- A key type that a lookup cannot always find, such as a float, sorts with its values as
  key-value pairs in a stack array of 16 pairs.
- Any other key type sorts in a stack array of 16 keys, and a lookup finds the value of each
  key.
- Bool keys need no sort: the encode writes the entry of false before the entry of true.

The sort of a map of more than 16 entries allocates. A key writes every -0.0 as +0.0. The
encode fails with `ErrInvalidKey` for a key with a NaN component, and with `ErrAmbiguousKey`
when two adjacent sorted keys order equal, which neither check allocates for. Unions encode
with one `switch` on the discriminator. Every value that the wire format says is absent is
left out.

### Views

With `-views`, the generator declares a view type per struct type:

```go
// OrderView is the encoding of an Order. Each method reads one field by
// scanning the encoding, without decoding the rest, and returns the zero
// value when the encoding has no such field.
type OrderView []byte

// ID returns the bytes of the ID field, which alias the view.
func (v OrderView) ID() ([]byte, error)

// Line returns the encoding of the Line field, or nil.
func (v OrderView) Line() (LineView, error)

// Ref returns the value of the Ref field, an opaque value, which the
// UnmarshalBinary of its type decodes from the field's bytes.
func (v OrderView) Ref() (Ref, error)
```

A method exists for each exported field of a bool, integer, float, complex, string, byte slice,
byte array, time, struct or opaque type, and for a pointer to one. Slices, maps, interfaces and
union members have no method. A string or byte slice field returns its bytes, which alias
the view, so that a read needs neither a copy nor package unsafe. An opaque field returns the
value that the decode method of its family, `UnmarshalBinary`, `GobDecode` or
`UnmarshalText`, sets from the field's bytes, and the zero value when the encoding has no such
field. It allocates what that method allocates, and an error of that method is the cause of
the `*DecodeError` that it returns. A storage engine reads an index key or evaluates a filter
from the view without allocating.

A field of a `kanon.Validator` returns the value of its underlying type converted to its type,
after its `ValidateKanon` accepts the value: a byte slice aliases the view, and a string is a
copy of the bytes, which allocates. An error of `ValidateKanon` is the cause of the
`*DecodeError` at the offset of the value. A field that the encoding does not contain returns
the zero value without a call of `ValidateKanon`, as an opaque field returns it without a call
of its decode method.

A view checks the tags and lengths of the encoding and the wire format of the field that a
method reads. It does not check the rest of the schema, so a caller that needs a full check
decodes. A method of any field but a struct returns the last occurrence of the field, as a
decode does. A method of a struct field, or of a pointer to a struct, fails with a
`*DecodeError` that wraps `ErrRepeatedView` at the tag of a second occurrence, since a decode
merges the occurrences and one byte slice cannot hold the merge. For the record encoding
`72 03 0a 01 61 72 02 10 01`, a decode merges `Item{Name: "a", Count: 1}`, and
`RecordView.Item` fails with `ErrRepeatedView` at offset 5.

### The conformance suite

`kanontest` checks a generated codec against the wire format: every check runs from a `Spec`
that the generator writes into the test file. The suite reads and sets an unexported field
through its address with `reflect.NewAt`, since reflection neither sets such a field nor
returns its value as an interface. The suite builds samples from value tables and compares
each decode with a model derived from the wire format's rules, not from the decoder. It
compares each encoding with a reference encoder, probes malformed input for the required
rejections and error causes, checks the allocation contract, fuzzes the decoder, and pins the
encodings in a golden file per type. The suite defines conformance for this implementation,
and its golden files are test vectors for others. `kanontest.Codec[T]` constrains the codec to
`*T`, which implements `kanon.Message`.

The reference encoder and decoder call the `ValidateKanon` of a `kanon.Validator` as the
generated code does. A sample counts such a value as one that can fail to encode.
`kanontest.RunValue[T kanon.Validator]` checks the method of each type other than a struct
that a `-type` flag names, over the value tables of its underlying type:

- The method allocates nothing for a value that it accepts.
- For a type with binary, gob or text methods, the method accepts a value exactly when the
  encode method of the type's family accepts it, and the decode method of the family decodes
  the encoding of each accepted value to the value. A directive without the `-validate` that
  its type needs fails this check.
- A golden file pins the encoding of each value that the method accepts, as field 1 of a
  struct, and the error of each value that it rejects.

### Module layout

```text
go.thesmos.sh/kanon             Message, Cloner, Options, errors, versions
go.thesmos.sh/kanon/wire        functions for generated code
go.thesmos.sh/kanon/frame       messages in frames on a byte stream
go.thesmos.sh/kanon/batch       messages in batches for storage blocks
go.thesmos.sh/kanon/kanontest   conformance suite
go.thesmos.sh/kanon/cmd/kanon   the generator
```

A consumer's module requires `go.thesmos.sh/kanon` for the `tool` directive, and its
generated files import `kanon` and `kanon/wire`. One module version supplies the generator
and the runtime, so within a module they agree by construction. Across modules, the
`EnforceVersion` constants catch a mismatch at build time.

## Alternatives considered

### Generated code that imports only the standard library

Every generated file repeats the varint, time, skip and error functions, and consumers share
nothing but the standard library's sentinels.

**Why not:** consumers cannot classify decode errors beyond truncation, decode options grow
the method signatures, and the shared functions exist once per generated file, about 270 lines
each, with a mutation run per copy. The property it buys, a generated file that compiles
alone, has no consumer that needs it, and the `tool` directive already makes every consumer
depend on the module.

### Reflection at run time

One codec that walks any type with `reflect`, as gob does.

**Why not:** measured 5 to 20 times slower, with allocations on every decode, and no
allocation contract is possible.

### One method set per encoding, without a shared interface

**Why not:** a frame writer or a batch reader has to be written per type, or take `any` and
assert a method set of its own. One declared interface costs one small package.

### Generating from a schema language

A `.kanon` schema file compiled to Go, as protobuf, colfer and bebop do.

**Why not:** the application's data model is its Go types. A schema file is a second copy that
drifts, and it cannot express the types that only Go has, such as a struct key or an interface.

## Drawbacks

- Every consumer links the runtime, about 400 lines, and its binary depends on
  `go.thesmos.sh/kanon` at run time.
- The runtime promises compatibility with every generator version between `MinVersion` and
  `MaxVersion`. Dropping a version is a breaking change of the module.
- `kanon/wire` is a public package that applications should not use. Its documentation is
  the only thing that keeps them out, since Go has no export visible to generated code alone.
- A generated file is large: about 2,300 lines for a struct of 70 fields of distinct types,
  since each container type gets its own size, encode, decode and reset functions.
- The inline varint path adds about 5 lines per varint read and write site.
- A struct that keeps unknown fields has one exported `[]byte` field that is not part of its
  data model, and its encoding is canonical only while that field is nil.
- Views scan from the start on every call, so reading k fields costs k scans.
- A map whose keys can decode to distinct Go values of one projection, such as pointer keys,
  sorts those keys after every decode, which costs n log n and allocates above 16 keys.
- Every decode error allocates a `*DecodeError`.
- Adding `ValidateKanon` to a type with binary, gob or text methods, or removing it, changes
  the encoding of every field of the type. The field numbers stay the same, so the check mode
  passes. The golden files of the conformance suite fail.
- The view method of a string field of a `kanon.Validator` allocates a copy of the string.

## Unresolved and future work

- A schema export in JSON, with field numbers, types and type lists, for storage catalogs and
  readers in other languages.
- A gRPC codec adapter as a separate module, since it adds the grpc dependency.
- Test vectors in a machine-readable file for other implementations.
- A generate-time flag for the map key buffer of 16 keys, above which an encode allocates.

## References

| What | Where |
|---|---|
| stringer, the model for the directive | https://pkg.go.dev/golang.org/x/tools/cmd/stringer |
| gob's type registration, the model for type lists | https://pkg.go.dev/encoding/gob#Register |
| protobuf-go's version enforcement, the model for EnforceVersion | google.golang.org/protobuf v1.36.12, runtime/protoimpl/version.go |
| protobuf-go recursion limit, `protowire.DefaultRecursionLimit = 10000` | google.golang.org/protobuf v1.36.12, encoding/protowire/wire.go |
| Go inlining budget and the big-function rule | cmd/compile/internal/inline, Go 1.27.1 |
