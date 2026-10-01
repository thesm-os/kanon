---
rfc: 0006
title: Decoding that accepts only the canonical encoding
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-09-30
updated: 2026-10-01
discussion: none
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0006: Decoding that accepts only the canonical encoding

## Summary

A directive flag, `-canonical`, makes the struct types of a kanon directive accept only their
canonical encoding. The canonical encoding of a value is the input that a conformant encoder
writes for it, and it contains no unknown field. `DecodeKanon`, `MergeKanon` and `UnmarshalBinary`
of such a type reject every other input with `ErrNotCanonical`. The flag changes only the code of
the types that its directive names. This document states the rules byte by byte, with negative
test vectors, so that a decoder in another language accepts the same inputs as the Go decoder.
The generated code checks each rule as it reads the input, with the presence, key order and
projection logic of the encoder. The conformance suite tests the check against its reference
decoder and against a round trip through its reference encoder. A type that encodes itself can
declare `kanon.Exact`: its encode does not fail for a value other than its zero value, and its
decode accepts only the bytes that its encode writes. The generated code writes a field of such a
type without error handling and decodes it without a second encode, and the conformance suite
checks the declaration.

## Motivation

A tamper-evident log hashes the stored bytes of a record and decodes them. The writer of those
bytes is the operator of the log, and a verifier does not trust the operator. The wire format
gives each value one encoding, but only as a promise of the encoder. A decoder accepts many
encodings of one value:

- a varint that is not in its shortest form
- fields in any order, and a field number twice, whose occurrences merge
- a bool above 1
- map entries in any order, and two entries of one key
- unknown field numbers
- a field at a value that the encoder leaves out, such as `10 00` for a `Count` of 0

A decoder accepts these inputs because a conformant writer never produces them, and because a
check of the shortest form of every varint costs time on the hot path of every consumer. For an
evidence format, the non-conformant writer is the adversary, and neither reason applies:

- **Malleability.** The operator can store a second encoding of a record. The record then has
  another hash, and every verifier decodes it to the same value.
- **Parser differentials.** Two verifiers that implement the merge rules differently decode one
  byte string to two values. A record whose second `Stream` field names another stream is one
  example. The ledger has a verifier in Go and an independent verifier in another language, and
  every merge rule that the second one copies imperfectly is such a differential.

A consumer cannot check the encoding without a second pass. It can decode, encode the value
again and compare the bytes. That encode adds 45% to 243% to the time of the decode, as the
alternative of a re-encode in the decode measures below. Each consumer also copies the rules of
kanon's canonical form into its own code.

Canonicality is a property of a format, so the schema declares it. The readers of an evidence
format include a verifier, monitors and the admission path of a log. A choice that each reader makes
per call fails without an error in the reader that forgets it. A choice that the type makes
applies to every reader, `UnmarshalBinary` and code that decodes through
`encoding.BinaryUnmarshaler` included.

The check belongs in the generated decode, which reads every byte of the input with the schema in
hand. Most rules then cost one comparison at the byte that they concern. The conformance suite
proves the check for every canonical type, in the generated test of the type's own package.

Other canonical formats reject non-canonical input at the decoder:

- The Distinguished Encoding Rules of ITU-T X.690 give each value of an X.509 certificate one
  encoding. Go's `encoding/asn1` rejects an integer that is not minimally encoded, a length that
  is not minimal, and a boolean other than `00` and `ff`.
- dCBOR requires its decoders to reject a serialization that is not the preferred one, map keys
  out of order and duplicate map keys. Its security considerations describe the attack:
  semantically equivalent documents that serialize to different bytes.
- Bitcoin Core rejects a CompactSize integer that is not in its shortest form, with the error
  `non-canonical ReadCompactSize()`.

The generated code calls the methods of a type that encodes itself, such as the digest, the
identifier and the instant of the core module, and it guards every result: an encode error, a
`SizeKanon` that differs from the encoding, and in a canonical decode the zero value and a second
encode. The generator cannot know that a type keeps its methods consistent. For a type that does,
the guards never run:

- The ledger's record package has 39 such statements in 5 code files, in the code of its fields
  of core digests and identifiers. No test can run them, so no test covers its generated code in
  full.
- The ledger's content package has 2 more, in the put function of the elements of a slice of
  digests: the check of the room and the check of the length.
- The two second encodes take 6.5 ns of the 11.8 ns of the canonical decode of `Opaque`.

A declaration by the type, which the conformance suite checks, removes both.

## Detailed design

### Terms

- A **canonical type** is a struct type that a directive with the `-canonical` flag names.
- A **canonical decode** is `DecodeKanon`, `MergeKanon` or `UnmarshalBinary` of a canonical type.
- A **default decode** is the decode of any other struct type.
- An **opaque type** is a type that encodes itself through the methods of a family: binary, gob
  or text.
- An **Exact type** is an opaque type that declares `kanon.Exact`.

### Components

| Component | Change |
|---|---|
| The generator | The `-canonical` flag. The decode of each canonical type applies the rules of this document, and the generation fails for the cases that the directive section lists. A field of an Exact type encodes without error handling and decodes without a second encode |
| `kanon` | `ErrNotCanonical`, `Exact`, `Appender`, `ErrExact`, one sentence in the contract of `DecodeKanon`, and `MaxVersion` of 2 |
| `kanon/wire` | `CanonicalTime`, nine error constructors that wrap `ErrNotCanonical`, `MustExact` and `ExactError` |
| `kanontest` | `Spec.Canonical`, the canonical rules in the reference decoder, the round-trip check, new probe families, values of opaque types from their decode method, `ExactChecks` and `RunExact` |
| `Options`, `wire.Nested`, `wire.Time`, views, indexes, `frame`, `batch` and `kanon inspect` | No change |

### Invariants

- A canonical decode accepts an input exactly when the wire format lets a decoder accept it, the
  input contains only field numbers of the schema, and the encode of the decoded value is the
  input.
- A canonical decode accepts every encoding that `EncodeKanon` writes for a value of its type,
  when the decode method of each opaque type returns a value that encodes to the bytes it decoded.
- A canonical decode rejects a non-canonical encoding of the value of an Exact type only when the
  type keeps its declaration. The conformance suite checks the declaration on every value that it
  builds and on every probe.
- An input that a canonical decode accepts decodes to the value that the default rules of the
  wire format give it.
- The flag changes only the code of the types that its directive names and of the inline structs
  of their fields.
- A canonical decode allocates what a default decode of the same fields allocates, except for the
  re-encode of an opaque value that is not Exact, and the encodes of two opaque map keys that it
  compares, of a type without an append method or with an encoding longer than 128 bytes. The
  allocation contract of `Message` already lists every value of a type that decodes itself.

### The directive

```go
//go:generate go tool kanon -type=Header,State -canonical
```

The usage of the generator becomes
`kanon -type=T[,T...] [-views] [-validate=method] [-canonical] [file]`. The flag applies to the
struct types of `-type`, and to the inline structs in the types of their fields, which the code
file of the directive encodes. A named type other than a struct keeps its `ValidateKanon` method,
and the code of each canonical type checks the encoding of its underlying type.

The generation fails when:

- `-canonical` is set and `-type` names only types other than structs
- a canonical type or one of its inline structs has a field that keeps unknown fields, since a
  canonical decode rejects every unknown field and the field would always be empty
- a canonical type contains a struct type with a kanon codec of its own whose directive does not
  set `-canonical`, as a field, an element, a map key or value, the value of a pointer or a
  concrete type of an interface
- a canonical type contains a struct type whose kanon codec is written by hand

The generator reads the flag from the directive of the contained type, in the package or in a
dependency, as it reads that directive to find the codec of the contained type. A contained codec
is then canonical wherever a canonical type contains it, and a nested decode through
`wire.Nested` does not need a flag of its own.

### Rules

A canonical decode applies the decoder requirements of the wire format and these rules. A port
applies the same rules in the same order, so that it rejects the same inputs at the same offsets.

| Rule | Applies to | Offset of the error |
|---|---|---|
| Every varint has its shortest form. The last byte of a varint of two or more bytes is not `00` | Every varint: tags, lengths, values, interface type numbers and elements | The first byte of the varint |
| The field numbers of a struct encoding strictly ascend | Every struct encoding, and the payload of a time | The tag of the field |
| Every field number is in the schema | Every struct encoding | The tag of the field |
| A struct encoding contains at most one member of each union | Union members | The tag of the second member |
| A bool is `00` or `01` | Bool fields, elements, keys and values, and the bool that a pointer points at | The value |
| No field is present at a value that the presence table of the wire format makes absent | Every field other than a pointer and a union member | The tag of the field |
| Map keys strictly ascend in the key order of the wire format | Every map | The key |
| Each map key is its projection, without a float component of `-0.0` | Keys with a float component | The key |
| A time payload contains fields 1 to 3 only, and neither field 1 nor field 2 is 0 | The payload of every time | The tag of the field |
| The bytes of an opaque value are the bytes that the encode method of its type writes for the value that its decode method returns | Every opaque value. An Exact type keeps the rule by its declaration, and the decode does not check it | The value |

Strict ascent also excludes repetition:

- A field number does not repeat, so the occurrences of a field never merge.
- A map does not contain two keys of one projection.

The presence rule rejects these inputs, among others:

- an integer, a float or a complex number whose bits are all zero
- a bool field of `00`
- an empty string, byte slice, slice or map
- an array whose elements are all zero values, and a byte array whose bytes are all zero
- a struct field that is not a pointer, with an encoding of no bytes
- an interface field with type number 0
- the zero time in UTC
- an opaque value at the zero value of its type, for a type whose `==` compares every bit, and
  an opaque value with an empty encoding for any other type

A pointer field, a union member and every value inside a slice, an array, a map, a pointer or an
interface are written whatever their value, so the presence rule does not apply to them. The
fields of a struct inside any of them follow the rule.

### Order of checks

A decode checks the input in reading order, and it checks one position in a fixed order. At a
tag, the decode checks, in this order:

1. that the varint of the tag reads
2. that the varint has its shortest form
3. that the field number is not 0
4. that the number is above the number before it
5. that the number is in the schema
6. that the wire format is valid and is the wire format of the field
7. that the field is the first member of its union in the input

At a value, the decode applies the requirements and rules of the value as it reads the bytes of
the value. At a map key, it checks the requirements of the key's value first. It then checks that
the key has no NaN component, that the key is its projection, and last that the key is above the
key before it. When the value of a field ends, the decode applies the presence rule. A field of
an opaque type whose `==` compares every bit is the exception: the decode applies the presence
rule after the decode method of the type and before its encode method, since the encoder never
passes the zero value of such a type to the encode method. A field of an Exact type checks an
error of the decode method first and the presence rule second, and no encode follows.

An input can break a requirement of the wire format and a rule of this document. The decode
reports the first failure in this order. A canonical decode can return `ErrNotCanonical` for an
input that a default decode rejects with `ErrMalformed`. The tag `80 00` is an example: a varint
that is not in its shortest form, for field number 0.

### Failure handling

A failure leaves the fields decoded before it in the receiver, as it does in a default decode,
and returns a `*kanon.DecodeError` that wraps `kanon.ErrNotCanonical` with these fields:

| Failure | Field and number | Detail |
|---|---|---|
| Varint not in its shortest form | The struct for a tag, the field for a value | `varint of N bytes has a shorter form` |
| Field number not above the one before it | The struct | `field N after field M` |
| Field number not in the schema | The struct | `field N not in the schema` |
| Second member of a union | The member | `second member of the union of D` |
| Bool above 1 | The field | `bool N, want 0 or 1` |
| Field at a value that the encoding leaves out | The field | `field at a value that the encoding leaves out` |
| Map key not above the key before it | The map field | `key not above the key before it` |
| Map key with a float component of `-0.0` | The map field | `key with a float component of -0.0` |
| Time payload field 1 or 2 of 0 | The time field | `time field N of 0` |
| Opaque bytes that the encode method does not write | The field | `bytes differ from the encoding of the decoded value` |

A time payload with another field, or with its fields out of order, fails with the detail of the
struct rule.

### Test vectors

The vectors decode with the vector types of the wire format, declared canonical, and with one more
type:

**Nest** `{Inner Inner = 1}`, a struct field that is not a pointer, with `Inner {Label string = 1,
Count int64 = 2}`

A canonical decode rejects each of these inputs with `ErrNotCanonical`, at the offset given:

| Type | Input | Rule | Offset |
|---|---|---|---:|
| Inner | `90 00 0e` | tag not in its shortest form | 0 |
| Inner | `10 8e 00` | value not in its shortest form | 1 |
| Inner | `0a 81 00 78` | length not in its shortest form | 1 |
| Lists | `0a 04 01 80 00 02` | element not in its shortest form | 3 |
| Inner | `10 0e 0a 01 78` | fields out of order | 2 |
| Inner | `10 0e 10 0e` | field twice | 2 |
| Repeats | `0a 01 02 0a 01 04` | union member twice | 3 |
| Repeats | `42 03 01 01 02 42 03 01 01 04` | interface twice | 5 |
| Inner | `18 01` | field 3 not in the schema | 0 |
| Holder | `12 01 74 18 12` | two members of one union | 3 |
| Numbers | `28 02` | bool field of 2 | 1 |
| Numbers | `48 02` | bool of 2 behind a pointer | 1 |
| Inner | `10 00` | `Count` of 0 | 0 |
| Inner | `0a 00` | empty `Label` | 0 |
| Numbers | `28 00` | bool field of false | 0 |
| Fixture | `42 20` and 32 bytes of `00` | byte array of zero bytes | 0 |
| Nest | `0a 00` | struct field of no bytes | 0 |
| Holder | `0a 01 00` | nil interface | 0 |
| Opaque | `12 04 00 00 00 00` | `Stamp` at its zero value | 0 |
| Times | `0a 07 08 ff db 8f f9 ce 03` | the zero time in UTC | 0 |
| Times | `0a 05 10 f4 03 08 02` | time fields out of order | 5 |
| Times | `0a 02 08 00` | time field 1 of 0 | 2 |
| Times | `0a 02 20 01` | time field 4 | 2 |
| Maps | `0a 08 01 62 01 32 01 61 01 31` | keys descending | 6 |
| Maps | `0a 08 01 61 01 31 01 61 01 32` | one key twice | 6 |
| Keys | `0a 0a 00 00 00 00 00 00 00 80 01 61` | key of `-0.0` | 2 |

Every encoding vector of the wire format is an input that a canonical decode accepts. The decode
vectors of the wire format that repeat a field are negative vectors. These inputs are accepted
too:

| Type | Input | Value |
|---|---|---|
| Numbers | `48 00` | `{I: &false}`, a pointer to a zero value |
| Container | `12 00` | `{Inner: &Inner{}}` |
| Holder | `18 00` | `{Kind: selects Num, Num: 0}`, a selected member at its zero value |
| Holder | `0a 02 02 00` | `{Shape: (*Square)(nil)}`, an interface that stores a nil pointer |
| Times | `0a 00` | `{CreatedAt: 1970-01-01T00:00:00Z}` |
| Times | `1a 08 07 08 ff db 8f f9 ce 03` | `{Stamps: {zero time}}`, the zero time as an element |
| Nest | `0a 02 10 02` | `{Inner: {Count: 1}}` |

### Schema changes

For a canonical reader, adding and removing a field are compatible in one direction only, since
the reader rejects a field number that its schema does not list:

| Change | Canonical reader, data of the other version | Rollout |
|---|---|---|
| Add a field with a number never used before | An old reader rejects new data that sets the field | Upgrade every canonical reader before any writer sets the field |
| Remove a field and reserve its number | A new reader rejects old data that set the field | Keep a reader of the old schema for the old data, or rewrite the data |

A format with canonical readers versions its schema, so that a reader decodes each record with
the schema of the version that the record states.

### Exact types

A type declares `kanon.Exact` with one method, `ExactKanon`. It can declare it when:

- it encodes itself through an append method
- it is a `kanon.Sizer`
- its `==` compares every bit, so that a field leaves out its zero value

A type that declares it keeps two guarantees:

1. `SizeKanon` returns no negative value. The append method returns no error for a value other
   than the zero value, and appends as many bytes as `SizeKanon` returns whenever it returns no
   error.
2. The decode method accepts a byte string only when the append method writes that byte string
   for the value that the decode method sets.

The second guarantee makes the decode method its own canonical check. The first guarantee lets
the append method fail for the zero value, since the zero digest of the core module has no
encoding. The generation fails for a type with an `ExactKanon` method and without an append
method, without `SizeKanon`, or with an `==` that does not compare every bit.

The code of a field of an Exact type differs from the code of a field of another opaque type:

| Step | Another opaque type | An Exact type |
|---|---|---|
| Encode | An error of the append method, and a `SizeKanon` that differs from the length of the encoding, fail the encode | The encode has no error path. A type that breaks guarantee 1 panics with a `*kanon.EncodeError` that wraps `kanon.ErrExact` |
| Canonical decode | The decode method, the presence rule, and a second encode compared with the input | The decode method and the presence rule in one statement, and no second encode |
| Default decode | The decode method | No change |

These rules apply to a field that the encoding leaves out at its zero value. The other positions
write the zero value, whose append method can fail:

- an element of a slice or an array
- a key or a value of a map
- the target of a pointer
- a union member
- the value of an interface

In these positions the put function of an Exact type returns the error of the append method,
which guarantee 1 allows for the zero value alone. It checks neither the room nor the length,
which guarantee 1 rules out, and passes the encoding to `wire.MustExact`, which panics for a type
that breaks the guarantee.

An Exact type whose append method fails for no value, its zero value included, declares
`kanon.Appender` with a second method, `AppendKanon(dst []byte) []byte`. It appends the bytes of
the append method and has no error result, so the compiler rules out the error that guarantee 1
allows. The put function of such a type calls `AppendKanon` in every position and passes the
encoding to `wire.MustExact` for its length, without an error path. A field of the type writes
through the same put function. A slice, an array, a pointer, a map value or a union member of
the type cannot fail, and neither can a struct whose other fields cannot fail. The generation
fails for a type with an `AppendKanon` method and without `ExactKanon`, and for an `AppendKanon`
of another signature.

Guarantee 2 applies to every byte string, so a canonical decode skips the second encode of an
Exact value in every position. A struct encodes without an error path when none of its fields can
fail, and an Exact field counts as a field that cannot fail.

The declaration removes the statements of the ledger's evidence format that its conformance suite
cannot run, 39 in the record package and 2 in the content package:

- per field of a digest or an identifier: the error return after the put function, the three
  failure returns of the put function, and the error return of the second encode
- per field of a digest: the presence check after the decode method, since no input decodes to
  the zero digest
- per struct: the error return of `AppendBinary`
- per put function of a digest in another position: the check of the room and the check of the
  length

`AppendKanon` on the identifier and the instant removes 16 more, in the protection package of
the same format: the error return after each element of a slice of identifiers, in the put
functions of those slices, in the encodes of the structs that contain them, and in their
`AppendBinary`.

The digest, the identifier and the instant of the core module keep both guarantees:

- `crypto.Digest` decodes 32, 48 or 64 bytes to a value whose append method writes those bytes,
  and it rejects every other length. Its append method fails for the zero digest alone.
- `id.ID` decodes 16, 20 or 32 bytes the same way, and the empty input to its zero value, whose
  encoding is empty. Its append method never fails.
- `clock.Instant` decodes 16 bytes to its three fields and appends them in the same order. Its
  `SizeKanon` returns 16 for every value, and its append method never fails.

Each of the three adds `ExactKanon` and a test that runs `kanontest.RunExact`. The identifier
and the instant also add `AppendKanon`, and the digest does not, since it has no encoding for
the zero digest. The core module already depends on kanon for the codec of its signatures.

### The runtime

```go
package kanon

var (
	// ErrNotCanonical marks input that the decode of a type whose directive
	// has the -canonical flag rejects, and that the wire format lets a
	// decoder accept: input that is not the canonical encoding of the value
	// that it decodes to.
	ErrNotCanonical = errors.New("kanon: input is not the canonical encoding")
	// ErrExact marks the panic of an encode that meets a value of an Exact
	// type that breaks its guarantee: an append method that appends another
	// number of bytes than SizeKanon returns, or that fails for the value of
	// a field, which is never the zero value.
	ErrExact = errors.New("kanon: a type that declares ExactKanon breaks its guarantee")
)

// Exact is implemented by a type that encodes itself through an append
// method, is a Sizer, has an == that compares every bit, and keeps two
// guarantees. SizeKanon returns no negative value, and the append method
// returns no error for a value other than its zero value and appends
// SizeKanon bytes whenever it returns no error. The decode method accepts a
// byte string only when the append method writes that byte string for the
// value that the decode method sets. The generated code writes a field of
// such a type without an error path and decodes it without a second encode,
// and kanontest checks both guarantees.
type Exact interface {
	Sizer
	// ExactKanon marks the type. No code calls it.
	ExactKanon()
}

// Appender is implemented by an Exact type whose append method returns no
// error for any value, its zero value included. AppendKanon appends the
// bytes of the append method without an error result. The generated code
// writes a value of such a type through AppendKanon in every position
// without an error path, and kanontest checks that AppendKanon appends the
// bytes of the append method.
type Appender interface {
	Exact
	// AppendKanon appends the encoding of the receiver to dst and returns the
	// extended slice.
	AppendKanon(dst []byte) []byte
}

// The generator versions that the runtime supports.
const (
	MinVersion = 1
	MaxVersion = 2
)
```

The contract of `DecodeKanon` in `Message` gains one sentence: the decode of a type whose
directive has the `-canonical` flag also returns a `*DecodeError` that wraps `ErrNotCanonical` for
input that is not the canonical encoding of its value. `Options` does not change. The slab, the
offset and the nesting limit apply to a canonical decode as they apply to a default decode.

Generated files declare generator version 2. A file of version 1 still compiles against the
runtime, and a file of version 2 fails to compile against a runtime whose `MaxVersion` is 1.

### The wire package

```go
package wire

// CanonicalTime decodes data as Time does, and accepts only the encoding
// that PutTime writes: fields 1 to 3 in ascending order, each at most once,
// each varint in its shortest form, and fields 1 and 2 not 0. It matches the
// tag bytes of the fields in their order, and a byte that is not the tag of
// a field after the fields before it fails at its offset. The errors of the
// canonical rules wrap kanon.ErrNotCanonical.
func CanonicalTime(data []byte, loc string, num, off int) (time.Time, error)

// Each error constructor of a canonical decode returns a *kanon.DecodeError
// that wraps kanon.ErrNotCanonical, names loc and num, and gives the offset
// off. OrderError and UnknownFieldError name the struct with a num of 0, or
// the field of a time.
func LongFormError(n int, loc string, num, off int) error
func OrderError(field, prev uint64, loc string, num, off int) error
func UnknownFieldError(field uint64, loc string, num, off int) error
func MemberError(disc, loc string, num, off int) error
func BoolError(v uint64, loc string, num, off int) error
func AbsentError(loc string, num, off int) error
func KeyOrderError(loc string, num, off int) error
func NegativeZeroError(loc string, num, off int) error
func EncodingError(loc string, num, off int) error

// MustExact checks enc, the encoding that the append method of an Exact type
// returned with err for a value whose SizeKanon is n. The put function of a
// field passes the error, since a field never writes the zero value. The put
// function of any other position returns the error itself and passes nil.
// MustExact panics with a *kanon.EncodeError that names loc and num and wraps
// kanon.ErrExact and err, when err is not nil or enc does not have n bytes.
func MustExact(enc []byte, err error, n int, loc string, num int)

// ExactError returns the error of the canonical decode of a field of an Exact
// type, whose decode method returned err or set the zero value: the error of
// UnmarshalError at the offset valueOff of the value for an error, and the
// error of AbsentError at the offset tagOff of the tag otherwise.
func ExactError(err error, loc string, num, tagOff, valueOff int) error
```

`ReadError`, `TagError` and the other constructors of a default decode apply to both kinds of
decode.

### The generated code

The decode functions of a canonical type have the signatures of a default decode, and apply the
rules without a flag. The canonical encoding writes the fields in ascending field number, so the
decode matches the tag bytes of each field in that order, as Go's `encoding/asn1` walks the
elements of a SEQUENCE and skips an OPTIONAL element whose tag does not match. A tag that matches
is in its shortest form, above the field before it, in the schema and of the wire format of its
field, so the decode does not read a tag as a varint or check one. A byte that begins no field
after the fields before it goes to a function that returns the error of the rules for it. The
generator writes this decode for `Inner` as a canonical type:

```go
func (m *Inner) mergeKanon(data []byte, slab string, off, depth int) error {
	var prior uint64
	i := 0
	if i < len(data) && data[i] == 1<<3|wire.Bytes {
		at := i
		i++
		prior = 1
		l, n := wire.Uvarint(data[i:])
		if n <= 0 || uint64(len(data)-i-n) < l {
			return wire.ReadError(n, "Inner.Label", 1, off+i)
		}
		if n > 1 && data[i+n-1] == 0 {
			return wire.LongFormError(n, "Inner.Label", 1, off+i)
		}
		i += n
		m.Label = slab[off+i : off+i+int(l)]
		i += int(l)
		if l == 0 {
			return wire.AbsentError("Inner.Label", 1, off+at)
		}
	}
	if i < len(data) && data[i] == 2<<3|wire.Varint {
		at := i
		i++
		prior = 2
		u, n := wire.Uvarint(data[i:])
		if n <= 0 {
			return wire.ReadError(n, "Inner.Count", 2, off+i)
		}
		if n > 1 && data[i+n-1] == 0 {
			return wire.LongFormError(n, "Inner.Count", 2, off+i)
		}
		m.Count = wire.Unzigzag(u)
		i += n
		if m.Count == 0 {
			return wire.AbsentError("Inner.Count", 2, off+at)
		}
	}
	if i != len(data) {
		return _vector_tagInner(data, i, prior, off)
	}
	return nil
}

func _vector_tagInner(data []byte, i int, prior uint64, off int) error {
	tag, n := wire.Uvarint(data[i:])
	if n <= 0 {
		return wire.ReadError(n, "Inner", 0, off+i)
	}
	if n > 1 && data[i+n-1] == 0 {
		return wire.LongFormError(n, "Inner", 0, off+i)
	}
	if tag>>3 == 0 {
		return wire.TagError(tag, "Inner", 0, off+i)
	}
	if tag>>3 <= prior {
		return wire.OrderError(tag>>3, prior, "Inner", 0, off+i)
	}
	switch tag >> 3 {
	case 1:
		return wire.FormatError(tag, wire.Bytes, "Inner.Label", off+i)
	case 2:
		return wire.FormatError(tag, wire.Varint, "Inner.Count", off+i)
	}
	return wire.UnknownFieldError(tag>>3, "Inner", 0, off+i)
}
```

A tag of a field number above 15 has two bytes or more, and the decode compares them as one
string, such as `string(data[i:i+2]) == "\x82\x01"` for field 16 in the wire format bytes.

The error function reproduces the order of checks at a tag. Every field after `prior` failed its
match at the same byte, so a known field number above `prior` has another wire format. The decode
reports the error that a loop over the tags reports for the same input, at the same offset.

A varint of one byte has no shorter form, so the check of the shortest form of a value is one
comparison of the length for most varints.

The other checks of the generated code:

- **Presence.** The condition is the absence condition of the field, the negation of the presence
  condition that the size pass of the encoder emits for it: `m.Count == 0` for `m.Count != 0`. A
  slice, map or struct field tests the length in the input, so that a merge checks the input and
  not the merged value. An interface field is absent when it is nil, which type number 0 decodes
  to.
- **Unions.** The decode of a struct with a union of two members or more keeps one bit per union
  for the members that the input contains. A member with a member of a smaller number in its
  union fails with `wire.MemberError` when the bit is set, and a member with a member of a larger
  number sets it. The member with the smallest number follows no member of its union, since the
  fields ascend, and a union of one member does not keep a bit.
- **Bools.** Every read of a bool rejects a varint above 1.
- **Map keys.** The read of a map stores the previous key in a local variable, and requires the
  compare function of the key order to return a negative result for it and the current key. A
  helper built like the helper that finds a NaN component finds a component of `-0.0`. An opaque
  key compares through the compare function of its type, which encodes both keys. The opaque rule
  makes the encoding of a decoded key the bytes of its input. A key type of one value, such as
  `struct{}`, has no local variable, and its second key fails.
- **Keys of one projection.** `DecodeKanon` and `UnmarshalBinary` of a map with pointer keys or
  zone keys do not collect keys for the sort that removes keys of one projection, since the order
  check already rejects such keys. `MergeKanon` collects them, because the receiver can contain a
  key of the projection of an input key.
- **Opaque values.** After the decode method of the type, the code encodes the value again and
  compares the bytes with the input. It encodes through the append method into a stack array of
  128 bytes, or through the encode method of the type's family. A field of a type whose `==`
  compares every bit checks its presence between the decode method and the encode method. An
  Exact type skips the second encode.
- **Times.** A time decodes through `wire.CanonicalTime`.

The encode of the field `Chain` of `Head`, of type `crypto.Digest`, which declares `kanon.Exact`,
calls a put function that has no error:

```go
if !m.Chain.IsZero() {
	w := _head_exactputCryptoDigest(buf[:i], &m.Chain, "Head.Chain", 2)
	i -= w
	i = wire.PutUvarint(buf, i, uint64(w))
	i = wire.PutTag(buf, i, 2<<3|wire.Bytes)
}

func _head_exactputCryptoDigest(buf []byte, x *crypto.Digest, loc string, num int) int {
	n := x.SizeKanon()
	i := len(buf) - n
	enc, err := x.AppendBinary(buf[i:i:len(buf)])
	wire.MustExact(enc, err, n, loc, num)
	return n
}
```

The put function of an element of the field `Digests` of `Index`, a `[]crypto.Digest`, returns
the error of the append method, which the zero digest returns, and passes the encoding of every
other digest to `wire.MustExact`:

```go
func _index_putCryptoDigest(buf []byte, x *crypto.Digest, loc string, num int) (int, error) {
	n := x.SizeKanon()
	i := len(buf) - n
	enc, err := x.AppendBinary(buf[i:i:len(buf)])
	if err != nil {
		return 0, wire.MarshalError(err, loc, num)
	}
	wire.MustExact(enc, nil, n, loc, num)
	return n, nil
}
```

The canonical decode of the field checks the decode method and the presence rule in one
statement, after the length of the value:

```go
m.Chain = crypto.Digest{}
if err := m.Chain.UnmarshalBinary(data[i : i+int(l)]); err != nil || m.Chain.IsZero() {
	return wire.ExactError(err, "Head.Chain", 2, off+at, off+i)
}
i += int(l)
```

The docblock of `DecodeKanon` of a canonical type states the rule:

```go
// DecodeKanon sets m to the value encoded in data and reuses the memory of
// m. It accepts only the canonical encoding of a value, the input that
// EncodeKanon writes for it, and returns a *kanon.DecodeError that wraps
// kanon.ErrNotCanonical for any other input that the wire format lets a
// decoder accept.
func (m *Header) DecodeKanon(data []byte, opts kanon.Options) error
```

### Cost

- The code of a type without the flag is the code that the generator writes for it without this
  design. The generated files of the existing fixtures change only in their version comment and
  their two version constants.
- A canonical decode compares the tag bytes of each field once, where a default decode reads each
  tag as a varint and switches on its number. It adds a comparison per varint of two or more
  bytes, a presence check per field, a call of the compare function per map key, one encode per
  opaque value that is not Exact, and two more per opaque map key.
- A canonical type has one decode. Its code file grows by the checks, not by a second decode.

Measured on 2026-09-30 at `GOMAXPROCS=4`, the median of 8 runs, with `DecodeKanon` of the table
sample of each vector type, with a slab, into a receiver that decoded before. Each type decodes
once with the flag and once without it, in two test binaries that run in alternation:

| Type | Default decode | Canonical decode | Change |
|---|---:|---:|---:|
| `Inner` | 5.64 ns | 5.17 ns | -8.4% |
| `Times` | 21.84 ns | 19.88 ns | -9.0% |
| `Maps` | 32.25 ns | 32.94 ns | +2.1% |
| `Holder` | 122.0 ns | 124.4 ns | +2.0% |
| `Repeats` | 3.74 ns | 3.91 ns | +4.5% |
| `Opaque`, two values of types that encode themselves | 5.56 ns | 11.82 ns | +113% |
| Geometric mean of the 15 vector types | 10.71 ns | 10.87 ns | +1.4% |

The change for `Keys`, +1.6% at p = 0.088, is within the noise, and the other 8 types decode 2%
to 9% faster. The re-encodes of `Opaque` include the allocation of `MarshalBinary` of `Stamp`,
which has no append method. `wire.CanonicalTime` decodes a time 11% to 15% faster than
`wire.Time`, with no allocation: 7.86 ns against 9.27 ns in UTC, and 12.65 ns against 14.41 ns in
the local zone.

The second encodes take most of the time of the canonical decode of `Opaque`. Its generated code
without them, which an Exact declaration of both of its types gives, measured 5.32 ns on
2026-10-01, against 11.82 ns with them and 5.51 ns for the default decode. Each binary ran 8 times,
in alternation with the other two.

### The conformance suite

```go
package kanontest

type Spec[T any] struct {
	Fields  []Field
	Structs []Struct
	Keys    []Struct
	Unknown string
	View    any
	// Canonical reports that the directive of T has the -canonical flag, so
	// that the decode of T accepts only the canonical encoding of a value.
	// The reference decode then applies the canonical rules, and the checks
	// compare it with the round trip through the reference encode.
	Canonical bool
}
```

The generator writes `Canonical: true` into the Spec of a canonical type.

- **Reference decoder.** For a canonical Spec, the reference decoder of the suite applies the
  rules of this document in the order of this document. Every check of the suite then compares
  the generated decode with it, error for error, as it does for a default Spec. The families that
  probe one field of a sample probe it alone. A default Spec puts unknown fields around it, which
  a canonical decode would reject before it reads the field.
- **Round trip.** The check `DecodeKanon/accepts exactly the inputs that the reference encode
  writes back` runs on the encoding of every sample and on every probe. It asserts that the
  reference decoder succeeds exactly when the input round-trips: the reference decode without
  the rules of this document succeeds, and the reference encode of the decoded value is the
  input. An unknown field fails the round trip, since the encode writes only the fields of the
  schema. The round trip does not depend on the rules, so it checks the reference decoder
  itself. The reference decoder sets a float32 and the parts of a complex64 bit for bit, as the
  generated code does, so that a signaling NaN round-trips.
- **Probes.** A family of probes per rule breaks the rule at one site, one probe per site. The
  family of varints changes each field of a sample alone, which it writes whatever its presence,
  and the others change the encoding of each sample:
  - a varint one byte longer than its shortest form: a tag, a length, an integer, a bool, a
    type number or the length of a complex128
  - two adjacent fields of a struct, swapped
  - two adjacent entries of a map, swapped
  - the unknown fields of every wire format, at a boundary between two fields
  - a field at the zero value of its type, written as a selected union member is written: in
    place of its value or where the encoding leaves it out, which gives a union a second member
  - a malformed tag at a boundary between two fields: a tag cut short, the tag of field number
    0, and that tag one byte longer than its shortest form
  - a zero float component of a map key written as `-0.0`, a field of a struct key included
  - the bytes of a value with a length written as zero bytes, which decode an opaque value to
    its zero value

  The families of every Spec break the other rules. A field twice is a family of its own, and
  one changed byte turns a bool into 2.
- **Fuzzing.** `Fuzz` compares the generated decode of a canonical Spec with its reference decoder
  on every input, and runs the round-trip check on every input.
- **Values of opaque types.** A sample takes a value of an opaque type from its decode method when
  reflection gives only the zero value, as for a struct whose fields are all unexported. The
  decode method decodes byte strings of every length up to the size of the type in memory. A
  field of a digest or an identifier of the core module then has a sample in which it is present.
- **Exact fields.** The reference decoder encodes every opaque value again, an Exact one
  included. A type that breaks guarantee 2 fails the decode checks, since the generated code
  accepts an input that the reference decoder rejects. A type that breaks guarantee 1 panics the
  generated encode, which fails the check that encodes.

The package of an Exact type checks the declaration itself:

```go
// ExactChecks returns the checks of T, a type that declares kanon.Exact,
// over the values that the value tables and its decode method give it. The
// append method and SizeKanon keep guarantee 1 for the zero value and for
// every such value. The decode method keeps guarantee 2 for the encoding
// of every such value, every prefix of it, every change of one of its bytes,
// the encoding with one more byte, and a byte string of every length up to
// the size of T in memory. For a type that does not encode itself through
// an append method, or whose == does not compare every bit, ExactChecks
// returns one check, which fails with the reason.
func ExactChecks[T kanon.Exact]() []Check

// RunExact runs the checks of T that ExactChecks returns, each as a
// parallel subtest of t.
func RunExact[T kanon.Exact](t *testing.T)
```

The vectors of this document run as tests of fixture types that match the vector types of the
wire format, generated with `-canonical`, with the offset of each rejection.

## Alternatives considered

### A decode option per call

`Options` gains a field `Canonical`, and each call of `DecodeKanon` or `MergeKanon` chooses the
mode. One type can then decode both ways.

**Why not:**

- The option costs every decode of every type. The unexported decode functions below
  `DecodeKanon` take the slab, the offset and the depth as arguments, not the options, so the flag
  becomes an argument of each of them: 1,029 functions in the 124 code files of the fixtures. A
  read function passes 11 integer words, against 9 integer argument registers on amd64, so the
  flag moves one more argument of each call to the stack. Every field of every type also runs a
  branch on the flag.
- A reader that forgets the option decodes leniently without an error. `UnmarshalBinary` and code
  that decodes through `encoding.BinaryUnmarshaler` decode leniently.
- A nested codec from a generator without the option ignores it. Only a runtime that supports
  generator version 2 alone catches that, which makes every module regenerate every code file.
- A struct that keeps unknown fields needs a rule at run time for a flag that it cannot honor.
- No reader of an evidence format decodes leniently, so the choice per call has no consumer.

### Canonical decoding by default

Every decode rejects every non-canonical input, and no flag exists.

**Why not:** a default decode that rejects unknown fields breaks the rolling upgrade that the
wire format puts first among its properties. An old reader would fail on every record of a new
writer that sets a new field. Every consumer would also run the checks, and most consumers read
the output of their own writer.

### A re-encode and a comparison in the decode

The decode of a canonical type decodes, encodes the value again into scratch memory and compares
the bytes. The check cannot drift from the encoder, because it is the encoder.

Measured on 2026-09-30 at `GOMAXPROCS=4`, the median of 5 runs, with `EncodeKanon` into a buffer of
the length of the encoding and `DecodeKanon` with a slab into a receiver that decoded before:

| Type | DecodeKanon | EncodeKanon | Added by the re-encode |
|---|---:|---:|---:|
| `view.Record`, 21 fields, 522 bytes | 157.1 ns | 70.5 ns | 45% |
| `view.Item`, 12 bytes | 5.66 ns | 4.99 ns | 88% |
| The maps shape of the shape benchmarks | 261 ns | 635 ns | 243% |

`EncodeKanon` runs `SizeKanon` first, so each percentage lacks only the comparison.

**Why not:** the encode adds 45% to 243% to the decode. The comparison needs scratch memory of
the length of the input, which a decode allocates or takes from the caller. An error reports the
offset of the first byte that differs, and no field. The round-trip check of the conformance
suite keeps this comparison, where its cost does not matter.

### A re-encode and a comparison in each consumer

The consumer decodes, encodes and compares with the methods that the generated code already
declares.

**Why not:** it costs what the re-encode in the decode costs. Each consumer also writes its own
copy of the check.

### A check function per type without a decode

The generator declares `CheckKanon(data []byte) error` per type, which walks the encoding and
checks it without building a value, for a reader that reads fields through views.

**Why not:** the function is a second parser beside the decode, and the two can disagree about
an input, which is the kind of differential that this design removes. Every reader that needs
the check decodes the record anyway.

### A lenient and a canonical decode per canonical type

The generator emits both decodes for a canonical type, as two methods.

**Why not:** the decode functions are 36,154 of the 97,481 lines of the code files of the
fixtures, 37.1%. A second decode would add about that share to every canonical code file, and no
reader of an evidence format calls the lenient one.

### Rejecting unknown fields alone

A canonical decode rejects only field numbers that the schema does not list.

**Why not:** the record then keeps every other encoding of its value: fields out of order,
repeated fields, varints that are not in their shortest form, and fields at their zero value.
Each changes the hash without changing the value, and the merge rules remain for an independent
implementation to copy.

### A canonical type that keeps unknown fields

A canonical type with a field that keeps unknown fields decodes as a default type does, since the
kept bytes have no canonical order.

**Why not:** the flag then has no effect for such a type, and nothing tells the author. The
generation fails instead, when the author sets the flag.

### Trusting every opaque type

The generator takes every opaque type to keep the guarantees of `kanon.Exact`, without a
declaration. The generated code uses the results of their methods without checks, and skips the
second encode of every opaque value.

**Why not:** many opaque types break the guarantees on purpose. A reading that is not a number has
no encoding, and a text type can accept leading zeros. The guarantee of this design would then
depend on every opaque type of every consumer, and no check would find the one that breaks it. An
Exact type states the guarantees, and the conformance suite checks them.

### Byte arrays in the schema

A field that contains a digest or an identifier is a byte array, such as `[32]byte`, and the
application converts it to and from the core types. The generated code of a byte array calls no
method, and the conformance suite runs every statement of it.

**Why not:** a byte array fixes one length per field. The core digest has 32, 48 or 64 bytes and
the identifier 16, 20 or 32. A format that keeps the choice of algorithm then needs a field per
length, or a byte slice, whose length the generated decode does not check.

### A kanon codec for the core types

The digest, the identifier and the instant get kanon codecs, whose encode the generator knows
cannot fail.

**Why not:** a codec encodes a struct as its fields. A digest then writes its 64-byte array and
its size, about 70 bytes in a field where its 32-byte form takes 34, in another wire format than
the raw digest.

### Guards in functions of the runtime

The generated code passes the results of the methods of every opaque type to functions of the
wire package, which return one error for every way in which they can fail. No type declares
anything.

**Why not:** the guards of the decode fold into one statement that a decode error runs. The encode
still returns the error of a field whose type cannot fail, and so does the `AppendBinary` of its
struct. 11 of the 39 statements of the ledger's record package remain, and the canonical decode
keeps its second encode.

### An Exact type without a check at run time

The generated code of an Exact field uses the results of the append method without checking them.

**Why not:** a type that breaks guarantee 1 then writes a wrong encoding without an error. The
check is two comparisons, and its panic shows the breach at the first encode.

### Generated code outside the coverage gate of a consumer

A consumer leaves its generated files out of its coverage gate, since the tests of kanon cover
every kind of statement that the generator writes.

**Why not:** the consumer then runs generated code that none of its gates measures, and the
canonical decode keeps its second encode.

## Drawbacks

- A canonical type decodes canonically in every reader. No method decodes it leniently. `kanon
  inspect` reads a rejected record without its type.
- Every struct codec that a canonical type contains needs the flag. A shared type, such as the
  signature codec of the core module, becomes canonical for all its consumers, and its module
  regenerates it with the flag.
- A codec written by hand cannot be part of a canonical type.
- A canonical reader rejects every field that its schema does not list. A format with canonical
  readers versions its schema. Every canonical reader upgrades before any writer sets a new
  field.
- The code file of a canonical type grows by a presence check per field, a check per union
  member, an order check and a projection check per map, and a re-encode per opaque value that is
  not Exact.
- A canonical decode encodes every opaque value that is not Exact again, and every opaque map key
  twice more for its order. It allocates for a type without an append method and for an encoding
  longer than 128 bytes. The decode of `Opaque` takes 113% more time.
- A type that breaks guarantee 1 of `kanon.Exact` panics the encode of a struct that contains a
  value of it: for an error of the append method in a field, and for a length other than
  `SizeKanon` in any position. The conformance suite finds the breach only for the values that it
  builds.
- For an Exact type whose zero value encodes, such as `id.ID` and `clock.Instant`, the error
  return of the put function in a position that writes the zero value cannot run, and neither can
  the error returns that pass its error on. A declaration that the zero value encodes too would
  remove them, and `crypto.Digest` could not make it. This proposal has no such declaration.
- `wire.MustExact` is the one panic of the library code, which otherwise returns every error. The
  lint configuration exempts its file from the rule that forbids `panic`.
- A canonical decode relies on guarantee 2 for an Exact value. A type that breaks it lets another
  encoding of its value through. The conformance suite checks the guarantee on the values and
  probes that it builds, not on every input.
- The generator writes the put functions of an opaque type and the canonical decode of its field
  in two forms, one for Exact types and one for the others, and its tests cover both.
- The public API of kanon grows by an interface, an error, two functions of the wire package and
  two functions of kanontest. Each Exact type of the core module adds a marker method and a test.
- Views and indexes of a canonical type check no canonical rule, so a reader that reads through
  them does not check the encoding.
- A port reports the offsets of the vectors only when it checks in the order of this document.
- A file of generator version 2 needs a runtime with a `MaxVersion` of 2.
- The guarantee covers struct encodings. The headers of frames and batches follow their own
  rules.

## Unresolved and future work

- The negative vectors of this document are not part of a machine-readable vector file. The wire
  format lists that file as future work.
- A check without a decode, for a reader that measures that a decode into a reused receiver
  exceeds its budget, is not proposed here.

## References

| What | Where |
|---|---|
| ITU-T X.690, the Distinguished Encoding Rules: one encoding for each value | https://www.itu.int/rec/T-REC-X.690 |
| Go's `encoding/asn1` in Go 1.27.1: `invalid boolean` at lines 59 and 72, `integer not minimally-encoded` at line 90, `non-minimal length` at line 622 | `src/encoding/asn1/asn1.go` of Go 1.27.1 |
| Go's register ABI in Go 1.27.1: RAX, RBX, RCX, RDI, RSI, R8, R9, R10 and R11 for integer arguments on amd64 | `src/cmd/compile/abi-internal.md` of Go 1.27.1 |
| dCBOR, draft-mcnally-deterministic-cbor-18: sections 2.2 to 2.4 on the rejections of a decoder, section 5 on the attack, and section 7.2 on invalid encodings | https://datatracker.ietf.org/doc/html/draft-mcnally-deterministic-cbor-18 |
| Bitcoin Core, `ReadCompactSize` and `non-canonical ReadCompactSize()` | https://github.com/bitcoin/bitcoin/blob/v0.14.2/src/serialize.h |
| The ledger's record commitment design, which requires every reader of an evidence format to accept only the canonical encoding | the ledger repository, `docs/rfc/0003-record-commitment.md` |
| The benchmarks of `view.Record` and `view.Item`, run 2026-09-30 | `go test -run '^$' -bench 'BenchmarkKanon(Record\|Item)$' -benchtime=200ms -count=5 ./internal/fixture/view/` |
| The maps shape of the shape benchmarks, run 2026-09-29 | `docs/benchmarks.md` |
| The cost of a canonical decode, run 2026-09-30 | the types of `internal/fixture/canonical/vector.go` generated with and without `-canonical`, one test binary each, run in alternation 8 times with `-test.run '^$' -test.bench '^BenchmarkKanon' -test.benchtime=200ms`, compared with `benchstat` |
| The decode of a time, run 2026-09-30 | `go test -run '^$' -bench 'BenchmarkTime/(Canonical)?Time/' -benchtime=200ms -count=8 ./wire/` |
| The canonical decode of `Opaque` without its second encodes, run 2026-10-01 | the generated code of `Opaque` with `-canonical`, the same code with its two second encodes removed, and the code without `-canonical`, one test binary each, run in alternation 8 times with `-test.run '^$' -test.bench '^BenchmarkKanonOpaque$/^DecodeKanon$' -test.benchtime=200ms`, compared with `benchstat` |
| The statements of the ledger's record package that its conformance suite cannot run, 2026-10-01 | `go test -coverprofile` of `evidence/format/record` in the ledger repository, generated by this design with values of opaque types from their decode method: 39 blocks in `head`, `link`, `manifest`, `precondition` and `unit` |
| The statements of the ledger's content package that its conformance suite cannot run, 2026-10-01 | the coverage of `evidence/format/content/index.kanon.go` in the ledger repository, generated by kanon `f6f835e`: lines 484 and 492 of `_index_putCryptoDigest` |
| The count of decode functions and lines in the code files of the fixtures, 2026-09-30 | a `go/parser` walk of `internal/fixture/**/*.kanon.go` |
| The encode and decode methods of `crypto.Digest`, `id.ID` and `clock.Instant` | `go.thesmos.sh/core` at `9caa95a`: `crypto/digest.go:107` and `:137`, `id/id.go:106` and `id/binary.go:17` and `:62`, `clock/instant.go:97`, `:133` and `:155` |
