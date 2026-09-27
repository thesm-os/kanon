---
rfc: 0001
title: The kanon wire format
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-09-27
updated: 2026-09-27
discussion: none
supersedes: none
superseded-by: none
produces-adr: ADR-0001, ADR-0002, ADR-0003, ADR-0004, ADR-0005, ADR-0006
---

# RFC-0001: The kanon wire format

## Summary

kanon is a binary encoding for typed records. A record is a struct with numbered fields. Each
field is a tag followed by a value, integers are variable-length, nested values are
length-prefixed, and map keys are sorted, so equal values encode to identical bytes. The
format encodes every type that Go's gob and json encode: nested slices and maps at any depth,
maps with struct, array, pointer, time and interface keys, interfaces with a listed set of
concrete types, unions, fixed-size arrays, complex numbers, and times with their zone offset.
This document is the specification. It is complete enough to implement an encoder and a
decoder in any language from it alone, and its test vectors are exact bytes.

## Motivation

Servers of one fleet exchange messages in this format, and storage engines keep records in
it. Both uses need these properties, in this order:

1. **Evolution without migration.** A field can be added, removed or reordered while old
   readers read new data and new readers read old data. Stored records are read for years
   after the binary that wrote them has been replaced, and a fleet runs two versions during
   every rolling upgrade.
2. **Identical bytes for equal values.** Storage engines hash, deduplicate and sign records,
   so the encoding of a value must not depend on the order of a map's iteration or on the
   encoder.
3. **Density.** Bytes on disk and on the wire cost money at petabyte scale.
4. **Every Go type that gob and json encode.** A codec that rejects a struct map key or a
   nested map forces the application to change its data model for the codec.
5. **Portability.** A reader in another language must be able to read stored records from
   this specification and its test vectors.

A prototype of this format was measured on 2026-09-27 against the public Go serialization
benchmark, on one machine, with a record of two strings, a time, an integer, a bool and a
float. The six field tags cost 6 bytes. Against the smallest positional format with the same
data model, the encoding was 2.7 bytes larger, 51.7 against 49.0, because absent zero values
and varints give some tag bytes back. After zstd over blocks of 1,000 records, the difference
to that positional format was 5%, and every codec measured ended within 30% of the others.
So the decisive property is evolution, which a positional format lacks, and the tag cost is
the price of it.

## Detailed design

### Conventions

- A **byte** is an octet. Multi-byte fixed-width values are little-endian.
- **Must** marks a rule whose violation makes an encoder non-conformant or makes a decoder
  reject the input. **Accepts** marks input a decoder reads without error.
- The type names of this document are Go's: struct, field, bool, int8 to int64, uint8 to
  uint64, float32, float64, complex64, complex128, string, byte slice (`[]byte`), slice,
  array (fixed size), map, pointer, interface, and time. A port maps them to its own types.
  A **struct** is a record of named, typed fields. A **slice** is a variable-length sequence.
  An **array** is a fixed-length sequence. A **pointer** is an optional value: nil or a value.
  An **interface** is a value of one of a listed set of concrete types, or nil.
- The grammar below uses `|` for alternatives, `*` for zero or more repetitions, and `n *`
  for exactly n repetitions.

### Primitive encodings

**uvarint.** A 64-bit unsigned integer, 7 bits per byte, least significant group first, with
bit 7 set on every byte but the last:

```text
uvarint = *( %x80-FF ) %x00-7F
```

- An encoder must write the shortest form: no byte of value 0x80 except as part of a longer
  value, and no more than 10 bytes.
- A decoder accepts a form with leading zero groups, up to 10 bytes. It must reject an 11th
  byte and a 10th byte with a value above 1, both of which mean a value outside 64 bits.
- 0 encodes as `00`, 127 as `7f`, 128 as `80 01`, 300 as `ac 02`.

**zigzag.** A 64-bit signed integer maps to an unsigned one before it is written as a uvarint:
`(n << 1) ^ (n >> 63)`, with an arithmetic shift. 0 maps to 0, -1 to 1, 1 to 2, -2 to 3,
`math.MinInt64` to `math.MaxUint64`.

**fixed32, fixed64.** 4 or 8 bytes, little-endian. Signed integers are two's complement.
Floats are the IEEE 754 binary32 or binary64 bit pattern. An encoder writes the bit pattern of
the value as it is, so a round trip changes neither a NaN payload nor the sign of zero.

### Structs

The encoding of a struct is a sequence of fields, each a tag followed by a value:

```text
struct = *field
field  = tag value
tag    = uvarint ; number<<3 | wire
```

The **field number** is a positive integer that identifies the field in the schema. It is the
tag shifted right by three bits. The low three bits are the **wire format**, which tells a
decoder how long the value is without knowing the field:

| Wire | Name | Value |
|---|---|---|
| 0 | varint | one uvarint |
| 1 | fixed64 | 8 bytes |
| 2 | bytes | a uvarint length, then that many bytes |
| 5 | fixed32 | 4 bytes |

Wire formats 3, 4, 6 and 7 are invalid. A decoder must reject a tag with one of them, and a
tag whose field number is 0.

**Order.** An encoder must write the fields in ascending field number. A decoder accepts any
order.

**Presence.** A field is written only when it is present, and an absent field decodes to the
zero value of its type. A field is present when:

| Type | Present when |
|---|---|
| bool | true |
| integer, float, complex | not zero. A float is present when its bit pattern is not all zero, so -0.0 and NaN are present |
| string, byte slice, slice, map | not empty. A nil and an empty slice or map encode alike |
| array | at least one element is not the zero value of its type |
| byte array | at least one byte is not zero |
| time | not the zero time (January 1, year 1, 00:00:00 UTC) |
| struct | its encoding has at least one byte |
| opaque | its encoding has at least one byte |
| pointer, interface | not nil. A pointer to a zero value is present |
| union member | its discriminator selects it, whatever the value |

**Unknown fields.** A decoder must skip a field whose number it does not know, by its wire
format, and continue with the next field. It may keep the skipped bytes, tag included, and an
encoder may write them after its known fields, unchanged. A decoder must reject a known field
number whose wire format is not the one the schema gives that field.

**Repeated fields.** A field number may occur more than once in one struct encoding. The
occurrences merge, as the concatenation of two encodings merges:

| Type | Second occurrence |
|---|---|
| slice | appends its elements |
| map | adds its entries; a key present in both takes the later value |
| struct, pointer to a struct | merges field by field under these rules |
| interface | replaces the value. If the concrete type is the same, the values merge as their type merges |
| union member | replaces the union: the other members become zero, and this member takes the value |
| any other | replaces the value |

An encoder must not write a field number twice in one struct encoding.

### Values

Every value encodes the same way wherever it occurs: as a field, as a slice or array element,
as a map key or value, as the value a pointer points at, and as the value an interface stores.
A pointer and an interface encode differently as a field, where the tag makes a byte
redundant, as the table marks.

| Type | Wire | Value |
|---|---|---|
| bool | varint | 0 or 1. A decoder accepts any non-zero value as true |
| int8, int16, int32, int64, int, duration | varint | zigzag |
| uint8, uint16, uint32, uint64, uint, uintptr | varint | uvarint |
| int32, int64, uint32, uint64 with the fixed option | fixed32, fixed64 | two's complement |
| float32, float64 | fixed32, fixed64 | IEEE 754 |
| complex64 | fixed64 | the real part as binary32, then the imaginary part |
| complex128 | bytes | length 16: the real part as binary64, then the imaginary part |
| string, byte slice | bytes | the bytes. A string is not required to be valid UTF-8 |
| byte array `[N]byte` | bytes | length N, the bytes. A decoder must reject any other length |
| array `[N]T` | bytes | the N elements, back to back, each as a value. A decoder must reject fewer or more |
| slice `[]T` | bytes | the elements, back to back, each as a value |
| map | bytes | keys and values alternating, each as a value, keys ascending |
| struct | bytes | its fields, as defined above |
| opaque | bytes | the bytes that the application's own encoding of the value produces |
| time | bytes | a struct of three fields, defined below |
| pointer | the wire of its target | a presence byte, 0 for nil or 1, then the value when 1. As a field: the value alone, and the tag is the presence |
| pointer to a pointer or an interface, as a field | bytes | the inner value, wrapped in a length |
| interface | the wire of its type | a uvarint type number, 0 for nil, then the value of the concrete type. As a field: wrapped in bytes with a length |

`int`, `uint` and `uintptr` are 64-bit on the wire. A decoder whose native type is narrower
must reject a value outside its range.

**Range.** A decoder must reject a varint value outside the range of the field's type: a
zigzag-decoded value outside an int8, int16 or int32, and a uvarint above a uint8, uint16 or
uint32.

**Nested lengths.** A length counts bytes, not elements. Every length-prefixed value is
self-delimiting, so a decoder can skip any value from its tag and length alone. A decoder must
reject a length that runs past the end of the enclosing value.

### Time

A time is a struct with three fields, encoded as bytes:

| Field | Type | Value | Present when |
|---|---|---|---|
| 1 | int64, zigzag | seconds since 1970-01-01T00:00:00 UTC | not 0 |
| 2 | uint32, uvarint | nanoseconds within the second, 0 to 999,999,999 | not 0 |
| 3 | int32, zigzag | the offset of the zone in seconds east of UTC | the location is not UTC |

- A time in UTC has no field 3. A time in a zone with offset 0 that is not UTC has field 3
  with value 0.
- A decoder must reject a nanosecond value above 999,999,999, and a zone offset outside the
  32-bit signed range.
- A decoder yields a time in a fixed zone with the decoded offset, or in UTC. The name of the
  zone is not encoded. An implementation may yield its local zone when the offset matches it.
- The zero time is 0001-01-01T00:00:00 UTC. As a field it is absent. As an element or a map
  value it encodes as length 0.
- A duration is an int64 count of nanoseconds, zigzag, and has no fields.

Examples:

- `time.Unix(1, 0)` in UTC is `08 02`.
- The same instant with 500 nanoseconds is `08 02 10 f4 03`.
- The same instant in a zone one hour east of UTC is `08 02 18 a0 38`.
- One second before the epoch in UTC is `08 01`.

### Unions

A union is a group of fields of one struct with a discriminator, a field of an integer type
whose value selects at most one member. The discriminator is not on the wire:

- An encoder writes the selected member, whatever its value, and none of the other members.
  A member of the zero value is written, as `18 00` for an int64 member with number 3.
- A discriminator that selects no member leaves every member out.
- A decoder that reads a member sets the discriminator to the value that selects it, and sets
  the other members to their zero values.
- A pointer member that is nil while selected encodes as a pointer to the zero value of its
  target type, so that it is present.

### Interfaces

An interface value is a uvarint type number, then the value of the concrete type. The schema
of the field lists the concrete types and their numbers, from 1. The number 0 is nil and has
no value after it.

- The concrete value encodes as its type encodes as a value: a string with its length, a
  struct with its length, a pointer with its presence byte. A nil pointer of a listed pointer
  type encodes as its type number followed by presence 0.
- As a field, the interface is wrapped in bytes: tag, length, type number, value.
- A decoder must reject a type number the schema does not list.
- A listed type may contain the interface again, as `[]any` in the list of an `any` field, so
  a value can be a tree.

Examples, for a field with number 1 that lists `Circle` as type 1 and `*Square` as type 2,
where `Circle` has one float64 field and `Square` one int32 field:

- `Circle{1.5}` is `0a 0b 01 09 09 00 00 00 00 00 00 f8 3f`.
- `&Square{3}` is `0a 05 02 01 02 08 06`.
- A nil `*Square` is `0a 02 02 00`.

### Map key order

An encoder must write the entries of a map in ascending key order under this total order,
so that equal maps encode to identical bytes:

| Key type | Order |
|---|---|
| bool | false before true |
| signed and unsigned integers | numeric |
| float | numeric, with NaN before every other value. -0.0 and +0.0 are equal |
| complex | by the real part, then by the imaginary part, each as a float |
| string, byte slice, byte array | bytewise, shorter prefix first |
| array | element by element |
| struct | field by field, in ascending field number, each as its type orders |
| pointer | nil first, then by the value pointed at |
| time | by instant (seconds, then nanoseconds), then a time in UTC before a time in a zone, then by zone offset |
| interface | by type number, then by value as the concrete type orders |
| opaque | by encoding, bytewise |

Two keys that are equal under this order but not equal as values, such as two NaNs or -0.0
and +0.0, have no defined order between them. A map with such keys has no canonical encoding.
A decoder accepts entries in any order, and a key that occurs twice takes the later value.

Example, a map from a struct `{X, Y int32}` to string with entries `{2,1}: "b"` and
`{1,9}: "a"`, as field 6: `32 0e 04 08 02 10 12 01 61 04 08 04 10 02 01 62`. The key `{1,9}`
sorts first on X.

### Decoder requirements

A decoder must reject each of these inputs, with an error that gives the field and the byte
offset:

- Input that ends inside a tag, a length, a fixed-width value or a length-prefixed value.
  This is truncation, and the error must say so, since a stream reader retries on it.
- A uvarint of 11 bytes, or of 10 bytes with a last byte above 1.
- A tag with field number 0 or wire format 3, 4, 6 or 7.
- A known field with a wire format other than the schema's.
- An array or byte array with another number of elements or bytes than its length.
- A value outside the range of a narrower integer type.
- A nanosecond count above 999,999,999, or a zone offset outside 32 bits.
- An interface type number the schema does not list.
- A nesting deeper than its limit.

A decoder accepts non-minimal uvarints, fields in any order, map entries in any order,
duplicate map keys, a bool above 1 and unknown field numbers.

A decoder must bound its nesting depth, since every nested struct, slice, map, pointer and
interface is a level and a hostile encoding can nest one level per two bytes. The
recommended default is 100 levels, and the limit must be a parameter. A decoder must not
allocate from a declared length before it has checked the length against the input.

### Encoder requirements

A conformant encoder writes the canonical form: minimal uvarints, fields in ascending number,
absent fields left out, map entries in key order, no repeated field numbers, and no unknown
fields other than the bytes it decoded and kept. Two encoders that follow these rules encode
equal values to identical bytes, except for maps with keys of undefined mutual order and for
kept unknown fields.

The encoding has no version of its own. A container of encodings, such as a stream frame or a
storage block, carries the version, and a change to this format is a new format.

### Schema

The schema of a struct is its list of fields, each with a number and a type, its unions, and
the type lists of its interface fields with their numbers. The schema is not on the wire. Two
parties share it by sharing the struct definition.

These changes are compatible, and old and new readers both read the other's data after them:

- Adding a field with a number never used before.
- Removing a field, whose number is then never used again.
- Reordering fields.
- Adding a concrete type to an interface's list with a new number.
- Widening an integer type within the same wire format, such as int32 to int64.

These changes are incompatible:

- Reusing a number.
- Changing the wire format of a number, such as int32 to a fixed int32, or a string to a
  struct.
- Moving a field into or out of a union.
- Changing a type number.
- Narrowing an integer type.

### Test vectors

Each vector is the encoding of one struct value. Field numbers are given per struct.

**Inner** `{Label string = 1, Count int64 = 2}`

| Value | Bytes |
|---|---|
| `{}` | (empty) |
| `{Label: "x", Count: 7}` | `0a 01 78 10 0e` |
| `{Count: -1}` | `10 01` |
| `{Count: 300}` | `10 d8 04` |

**Numbers** `{A uint32 = 1, D int64 fixed = 4, E bool = 5, F int32 = 6, H *int32 = 8, I *bool = 9, J *int64 fixed = 10}`

| Value | Bytes |
|---|---|
| `{A: 1, D: 2, E: true, F: -1}` | `08 01 21 02 00 00 00 00 00 00 00 28 01 30 01` |
| `{H: &-7, I: &true, J: &2}` | `40 0d 48 01 51 02 00 00 00 00 00 00 00` |

**Lists** `{Values32 []int32 = 1}`

| Value | Bytes |
|---|---|
| `{Values32: {-1, 0, 1}}` | `0a 03 01 00 02` |
| `{Values32: {}}` | (empty) |

**Maps** `{Attrs map[string]string = 1, Counts map[string]int64 = 2}`

| Value | Bytes |
|---|---|
| `{Attrs: {"b": "2", "a": "1"}}` | `0a 08 01 61 01 31 01 62 01 32` |
| `{Counts: {"n": -2}}` | `12 03 01 6e 03` |

**Container** `{Name string = 1, Inner *Inner = 2, Children []*Inner = 3}`

| Value | Bytes |
|---|---|
| `{Inner: &Inner{}}` | `12 00` |
| `{Children: {nil, &Inner{Label: "c"}}}` | `1a 06 00 01 03 0a 01 63` |

**Tree** `{Label string = 1, Children []*Tree = 2}`

| Value | Bytes |
|---|---|
| `{Label: "r", Children: {{Label: "a"}}}` | `0a 01 72 12 05 01 03 0a 01 61` |

**Times** `{CreatedAt time = 1, Timeout duration = 2}`

| Value | Bytes |
|---|---|
| `{CreatedAt: 1970-01-01T00:00:01Z}` | `0a 02 08 02` |
| `{CreatedAt: 1970-01-01T00:00:01.000000500Z}` | `0a 05 08 02 10 f4 03` |
| `{CreatedAt: 1970-01-01T01:00:01+01:00}` | `0a 05 08 02 18 a0 38` |
| `{CreatedAt: 1969-12-31T23:59:59Z}` | `0a 02 08 01` |
| `{CreatedAt: zero time}` | (empty) |
| `{Timeout: 1s}` | `10 80 a8 d6 b9 07` |

**Bytes** `{Payload []byte = 1}`

| Value | Bytes |
|---|---|
| `{Payload: {1, 2, 3}}` | `0a 03 01 02 03` |
| `{Payload: {}}` | (empty) |

**Fixture** `{ID string = 1, Score int64 = 4, Enabled bool = 6, Ref [32]byte = 8, Tags []string = 9}`

| Value | Bytes |
|---|---|
| `{ID: "a", Score: -1, Enabled: true}` | `0a 01 61 20 01 30 01` |
| `{Tags: {"b", "a"}}` | `4a 04 01 62 01 61` |

**Patch** `{Fixed64Val int64 fixed = 7, BlobRef [32]byte = 8}`

| Value | Bytes |
|---|---|
| `{Fixed64Val: 1}` | `39 01 00 00 00 00 00 00 00` |
| `{BlobRef: {1, 0, ...}}` | `42 20 01` then 31 zero bytes |

**Holder** `{Shape interface = 1 (Circle = 1, *Square = 2), Kind discriminator, Text string = 2 (union), Num int64 = 3 (union), PP **int32 = 4, Arr [2]int32 = 5, Points map[Point]string = 6, Times map[time]int32 = 7, Opt *Point = 8, Any interface = 9 (string = 1, int64 = 2, []any = 3)}`, with `Point {X int32 = 1, Y int32 = 2}`, `Circle {Radius float64 = 1}` and `Square {Side int32 = 1}`

| Value | Bytes |
|---|---|
| `{Shape: Circle{1.5}}` | `0a 0b 01 09 09 00 00 00 00 00 00 f8 3f` |
| `{Shape: &Square{3}}` | `0a 05 02 01 02 08 06` |
| `{Shape: (*Square)(nil)}` | `0a 02 02 00` |
| `{Kind: selects Text, Text: "t", Num: 9}` | `12 01 74` |
| `{Kind: selects Num, Text: "t", Num: 9}` | `18 12` |
| `{Kind: selects Num, Num: 0}` | `18 00` |
| `{Kind: selects nothing, Text: "t"}` | (empty) |
| `{PP: &&1}` | `22 02 01 02` |
| `{PP: &nil}` | `22 01 00` |
| `{Arr: {0, 5}}` | `2a 02 00 0a` |
| `{Points: {{2,1}: "b", {1,9}: "a"}}` | `32 0e 04 08 02 10 12 01 61 04 08 04 10 02 01 62` |
| `{Times: {1s UTC: 1, 1s at +01:00: 2, 1s at -01:00: 3}}` | `3a 12 02 08 02 02 05 08 02 18 9f 38 06 05 08 02 18 a0 38 04` |
| `{Opt: &Point{}}` | `42 00` |
| `{Any: []any{"s", int64(-1)}}` | `4a 07 03 05 01 01 73 02 01` |
| `{Any: ""}` | `4a 02 01 00` |

The reference implementation also pins the encoding of every sample of every fixture type in
golden files, one line per sample, as `sample N: <hex>`.

## Alternatives considered

### A positional format without field numbers

Values in declaration order, as mus, benc and gencode write them. Measured 5% to 20% smaller
raw and 1.4 to 2.3 times faster to encode and decode.

**Why not:** every change to a struct is a new version of the whole type. Stored records need a
migration or a version prefix per type and a converter per version, and a rolling upgrade needs
both binaries to agree on the version of every message. After block compression the size
difference to this format was 5%.

### A self-describing format

MessagePack, CBOR or JSON write field names or type tags with every value, so a reader needs
no schema.

**Why not:** names cost the most bytes of any option, 97 against 58 for the benchmark record
in MessagePack, and the reader still needs the schema to give the values types. The uses here
share the struct definition anyway.

### Protobuf compatibility

The tag and wire formats here match protobuf's, so a protobuf reader could read a subset.

**Why not:** protobuf has no struct, array, float or interface map keys, no fixed arrays, no 8-
or 16-bit integers, no complex numbers, and its timestamp has no zone. Compatibility would cut
the type set to protobuf's. The shared framing is kept because it is a good design for
skipping, not for interoperability.

### Fixed-width integers by default

Fixed widths compressed better than varints in the block measurement: the codec with fixed
integers and dates was the largest raw and the smallest compressed.

**Why not:** the measurement is one record type with random values. Varints are smaller raw
for the small integers that dominate most records, and the fixed option gives a schema author
the choice per field.

### One varint of nanoseconds for a time

**Why not:** it saves 2 bytes for a time with nanoseconds and costs 3 bytes for a time of
whole seconds. It cannot express the years outside 1678 to 2262, and it has no zone offset.

### A terminator instead of a length for nested values

A struct could end with a sentinel tag instead of starting with its length, which would let an
encoder write forward in one pass.

**Why not:** a decoder could not skip an unknown nested value without parsing it, and a hostile
value could not be bounded before it is read. The length costs an encoder a size pass, which
the reference encoder makes once and reuses for the buffer allocation.

## Drawbacks

- Every present field costs a tag of 1 byte for numbers 1 to 15 and 2 bytes for 16 to 2047,
  and every nested value costs a length of 1 byte below 128 bytes. On the benchmark record,
  4 bytes of 58.
- Encoding takes two passes, one for sizes and one for bytes, because lengths precede values.
- Map encoding sorts the keys, which costs time linear in the number of entries times the
  log of it, and needs memory for the sorted keys.
- Varints compress worse than fixed widths in blocks, measured as 5% on one record type.
- A time costs 4 bytes more than an int64 of nanoseconds when it has nanosecond precision,
  and 3 bytes for a zone offset when it is not in UTC.
- Random access within a struct is a scan of its fields from the start.
- A map with NaN keys, or with -0.0 and +0.0 keys, has no canonical encoding.

## Unresolved and future work

- Test vectors in a machine-readable file, one per line with the value in JSON and the bytes
  in hex, for conformance suites in other languages.
- A columnar layout for blocks of records, with each field's values adjacent across records.
- An order-preserving encoding for storage keys.

## References

| What | Where |
|---|---|
| Go serialization benchmark, with the prototype added as a serializer | https://github.com/alecthomas/go_serialization_benchmarks, run of 2026-09-27 |
| MUS format specification, a positional format | https://github.com/mus-format/specification |
| Protocol Buffers encoding, the origin of the tag and wire format design | https://protobuf.dev/programming-guides/encoding/ |
| IEEE 754-2019, binary32 and binary64 interchange formats | IEEE Std 754-2019 |
| zstd used for the density measurement | github.com/klauspost/compress v1.20.0 |
