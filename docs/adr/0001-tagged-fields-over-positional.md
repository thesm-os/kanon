---
adr: 0001
title: Encode structs as tagged fields with locked numbers
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0001: Encode structs as tagged fields with locked numbers

## Status

Accepted

## Context

kanon encodes records for two uses: messages between servers of one fleet, and records in
storage engines. A fleet runs two binary versions during every rolling upgrade, and stored
records are read for years after the binary that wrote them is gone. Both uses need a reader
to decode data written under a schema with a field added, removed or reordered.

The fastest Go codecs are positional: they write values in declaration order without field
numbers. Measured on 2026-09-27 against the public Go serialization benchmark, they encode
and decode in 0.43 to 0.7 of kanon's time, and with the same data model their record is 2.7
bytes smaller than kanon's 51.7. After zstd over blocks of 1,000 records the difference is 5%.
Those are the costs of the tag byte per field and the length prefix per nested value.

## Decision

We will encode every struct as tagged fields under field numbers that the generated file
locks, because stored records and mixed-version fleets must decode across schema changes
without a migration, and a positional format cannot.

## Alternatives Considered

### A positional format, as mus, benc and gencode write

Values in declaration order, no tags, no lengths for fixed-size values. Every change to a
struct is a new version of the whole type, so stored data needs a migration or a version
prefix per type with a converter per version, and a rolling upgrade needs both binaries to
agree on the version of every message. Rejected: the measured saving is 2.7 bytes raw and 5%
compressed, and evolution is worth more than that for petabytes of stored records.

### A self-describing format, as MessagePack, CBOR or JSON

Field names or type tags with every value. Measured on the benchmark record: 97 bytes for
MessagePack and 152 for JSON against 58.6, and the reader still needs the schema to type the
values. Rejected for size.

### A protobuf-compatible subset

The tag and wire formats match protobuf's, so a protobuf reader could read the fields whose
types protobuf has. Measured on 2026-09-27: protobuf's map entries cost 60.3 bytes against
49.4 raw and 5.7 against 4.4 compressed for a record with two small maps, and repeated
strings cost one byte more per element. Struct, float, time and interface keys, nullable
elements, interfaces and complex numbers would stay kanon-only. Rejected on 2026-09-27 as
not needed.

## Consequences

**Positive:**

- A field can be added, removed or reordered, and old and new readers both read the other's
  data.
- A decoder skips fields it does not know by their wire format alone.
- The concatenation of two encodings decodes to their merge, so records can be merged without
  decoding.

**Negative:**

- Every present field costs a tag of 1 byte for numbers 1 to 15 and 2 bytes above, and every
  nested value costs a length prefix.
- Encoding takes a size pass before the write pass, because lengths precede values.
- Encode and decode take 1.4 to 2.3 times the time of the positional codecs on the benchmark
  record.

**Neutral:**

- The wire format has the framing of protobuf without its type system or compatibility.
