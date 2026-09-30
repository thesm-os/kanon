---
rfc: 0006
title: Decoding that accepts only the canonical encoding
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-09-30
updated: 2026-09-30
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
decoder and against a round trip through its reference encoder.

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

## Detailed design

### Terms

- A **canonical type** is a struct type that a directive with the `-canonical` flag names.
- A **canonical decode** is `DecodeKanon`, `MergeKanon` or `UnmarshalBinary` of a canonical type.
- A **default decode** is the decode of any other struct type.

### Components

| Component | Change |
|---|---|
| The generator | The `-canonical` flag. The decode of each canonical type applies the rules of this document, and the generation fails for the cases that the directive section lists |
| `kanon` | `ErrNotCanonical`, one sentence in the contract of `DecodeKanon`, and `MaxVersion` of 2 |
| `kanon/wire` | `CanonicalTime`, and nine error constructors that wrap `ErrNotCanonical` |
| `kanontest` | `Spec.Canonical`, the canonical rules in the reference decoder, the round-trip check and new probe families |
| `Options`, `wire.Nested`, `wire.Time`, views, indexes, `frame`, `batch` and `kanon inspect` | No change |

### Invariants

- A canonical decode accepts an input exactly when the wire format lets a decoder accept it, the
  input contains only field numbers of the schema, and the encode of the decoded value is the
  input.
- A canonical decode accepts every encoding that `EncodeKanon` writes for a value of its type,
  when the decode method of each opaque type returns a value that encodes to the bytes it decoded.
- An input that a canonical decode accepts decodes to the value that the default rules of the
  wire format give it.
- The flag changes only the code of the types that its directive names and of the inline structs
  of their fields.
- A canonical decode allocates what a default decode of the same fields allocates, except for the
  re-encode of an opaque value, and the encodes of two opaque map keys that it compares, of a type
  without an append method or with an encoding longer than 128 bytes. The allocation contract of
  `Message` already lists every value of a type that decodes itself.

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
| The bytes of an opaque value are the bytes that the encode method of its type writes for the value that its decode method returns | Every opaque value | The value |

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
passes the zero value of such a type to the encode method.

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

### The runtime

```go
package kanon

var (
	// ErrNotCanonical marks input that the decode of a type whose directive
	// has the -canonical flag rejects, and that the wire format lets a
	// decoder accept: input that is not the canonical encoding of the value
	// that it decodes to.
	ErrNotCanonical = errors.New("kanon: input is not the canonical encoding")
)

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
		if !(m.Count != 0) {
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

- **Presence.** The condition is the presence condition that the size pass of the encoder emits
  for the field, negated. A slice, map or struct field tests the length in the input, so that a
  merge checks the input and not the merged value. An interface field tests the type number 0.
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
  compares every bit checks its presence between the decode method and the encode method.
- **Times.** A time decodes through `wire.CanonicalTime`.

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
  bytes, a presence check per field, a call of the compare function per map key, and one encode
  per opaque value, two more per opaque map key.
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

### A contract for opaque types

The types that encode themselves promise that their decode method accepts only the bytes that
their encode method writes, and a canonical decode skips the re-encode of their values. The
digest, the identifier and the instant of the core module keep this promise: each accepts one
encoding per value.

**Why not:** no check enforces the promise, and any opaque type that an application declares can
break it. The guarantee of this design would then depend on every opaque type of every consumer.
The re-encode costs at most the time that the encode of the struct spends on the value.

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
  member, an order check and a projection check per map, and a re-encode per opaque value.
- A canonical decode encodes every opaque value again, and every opaque map key twice more for
  its order. It allocates for a type without an append method and for an encoding longer than
  128 bytes. The decode of `Opaque` takes 113% more time.
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
- An opaque type could declare that its decode method accepts one encoding per value, with a
  conformance check of the declaration, so that a canonical decode skips its re-encode. It is not
  proposed here. A benchmark of a record with many opaque values, such as the ledger's header,
  shows whether the saving justifies the contract.

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
| The count of decode functions and lines in the code files of the fixtures, 2026-09-30 | a `go/parser` walk of `internal/fixture/**/*.kanon.go` |
| The decode methods of `crypto.Digest`, `id.ID` and `clock.Instant` | `go.thesmos.sh/core` at `df6887b`: `crypto/digest.go`, `id/binary.go`, `clock/instant.go` |
