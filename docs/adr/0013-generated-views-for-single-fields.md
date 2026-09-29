---
adr: 0013
title: Generate view types that read one field from an encoding
status: Superseded
date: 2026-09-27
supersedes: none
superseded-by: ADR-0021
rfc: RFC-0002
---

# ADR-0013: Generate view types that read one field from an encoding

## Status

Superseded by [ADR-0021](0021-views-read-opaque-fields.md).

## Context

A storage engine evaluates filters and extracts index keys from records far more often than
it needs whole records. A full decode of a record allocates for its strings, slices and maps
and reads every field. The encoding is a sequence of tagged fields, so one field can be found
by scanning tags and skipping values.

## Decision

We will generate, behind the `-views` flag, a view type per struct with one method per scalar,
string, byte, time or struct field that scans the encoding for that field, because a filter
or an index key needs one field without a decode and its allocations.

## Alternatives Considered

### A reflection-based scanner

One scanner that takes a `reflect.Type` and a field name. Rejected because it looks the field
up by name on every call and boxes the result, while a generated view knows the field number
and type.

### Decode the whole record

Rejected for the allocations and the time of decoding fields the caller does not read.

## Consequences

**Positive:**

- A filter or an index key costs one scan and no allocation.

**Negative:**

- Reading k fields costs k scans of the encoding.
- About 15 generated lines per field when the flag is set.
- Slices, maps, interfaces, union members and opaque types have no view method, so a caller
  that needs one decodes.

**Neutral:**

- A string or byte slice returned by a view aliases the view.
