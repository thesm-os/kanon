---
adr: 0021
title: Generate view types that read one field from an encoding, opaque fields included
status: Accepted
date: 2026-09-29
supersedes: ADR-0013
superseded-by: none
rfc: RFC-0002
---

# ADR-0021: Generate view types that read one field from an encoding, opaque fields included

## Status

Accepted

## Context

A storage engine evaluates filters and extracts index keys from records far more often than
it needs whole records. A full decode of a record allocates for its strings, slices and maps
and reads every field. The encoding is a sequence of tagged fields, so one field can be found
by scanning tags and skipping values.

The index keys of most records are identifiers and digests, types that encode themselves as
opaque values. A view without a method for them sends the engine back to a full decode. The
ledger reads the stream and the sequence number of every entry on its append path, whose
budget is 1 µs per entry, and a decode of the header costs every field of the record.

## Decision

We will generate, behind the `-views` flag, a view type per struct with one method per field
of a scalar, string, byte, time, struct or opaque type, or of a pointer to one, that scans the
encoding for that field and decodes an opaque value with the decode method of its type,
because a filter or an index key needs one field without a decode of the record and its
allocations.

## Alternatives Considered

### A reflection-based scanner

One scanner that takes a `reflect.Type` and a field name. Rejected because it looks the field
up by name on every call and boxes the result, while a generated view knows the field number
and type.

### Decode the whole record

Rejected for the allocations and the time of decoding fields the caller does not read.

### No method for opaque fields

Opaque fields keep no view method, and a caller that reads one decodes the record. Rejected
because identifiers and digests are the index keys that views exist to read.

### The bytes of an opaque field

The method returns the bytes of the field, and the caller decodes them. Rejected because every
caller then repeats the decode that the type of the field defines.

## Consequences

**Positive:**

- A filter or an index key costs one scan, and no allocation for a type whose decode method
  allocates nothing.
- An identifier or a digest reads from a record without a decode of the record.

**Negative:**

- Reading k fields costs k scans of the encoding.
- About 15 generated lines per field when the flag is set.
- Slices, maps, interfaces and union members have no view method, so a caller that needs one
  decodes.
- The method of an opaque field allocates what the decode method of its type allocates, and
  fails with the error of that method, which the view wraps with the field and the offset of
  the value.

**Neutral:**

- A string or byte slice returned by a view aliases the view. An opaque value is decoded, so it
  aliases the view only where the decode method of its type keeps the bytes.
