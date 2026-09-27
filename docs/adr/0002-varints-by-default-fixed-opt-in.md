---
adr: 0002
title: Encode integers as varints by default, fixed widths per field on request
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0002: Encode integers as varints by default, fixed widths per field on request

## Status

Accepted

## Context

A varint costs 1 byte for values below 128 and up to 10 bytes for the largest, and a fixed
width costs 4 or 8 bytes whatever the value. Small integers dominate most records: counts,
enumerations, lengths and flags. Under zstd over blocks of 1,000 records, fixed widths
compressed better than varints in one measurement on 2026-09-27: the codec with fixed
integers and dates was the largest raw, 55 bytes, and the smallest compressed, 34.2 bytes,
against 49 and 36.7 for the smallest varint codec. That measurement is one record type with
random values.

## Decision

We will encode integers as varints by default and let a field opt into fixed 32- or 64-bit
widths through the `fixed` tag option, because small values dominate most records and the
schema author knows which fields hold large values or compress better at fixed width.

## Alternatives Considered

### Fixed widths by default

Better block compression in one measurement, and 3 to 7 extra bytes raw for every small
integer. Rejected until a measurement on real records shows the compressed gain outweighs
the raw cost, which the `fixed` option lets a schema author apply per field.

### Varints only

No per-field choice. Rejected because a field of large values, such as a hash or a 64-bit
identifier, costs 9 or 10 bytes as a varint against 8 fixed, and compresses worse.

## Consequences

**Positive:**

- The smallest raw encoding for the common case.
- A field can change to fixed width where a measurement shows a gain.

**Negative:**

- Two encodings per integer type, which a reader in another language must both implement.
- A change from varint to fixed changes the wire format of the field, so it needs a new field
  number.
- The block-compression question is open per field until real records are measured.

**Neutral:**

- Map keys are always varints, since the option applies to values.
