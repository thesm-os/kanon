---
adr: 0005
title: Write map entries in a total key order, struct keys by field number
status: Superseded
date: 2026-09-27
supersedes: none
superseded-by: ADR-0019
rfc: RFC-0001
---

# ADR-0005: Write map entries in a total key order, struct keys by field number

## Status

Superseded by [ADR-0019](0019-map-keys-ordered-by-projection.md).

## Context

Storage engines hash, deduplicate and sign records, so equal values must encode to identical
bytes. Go iterates a map in random order, and protobuf's serialization is not deterministic
by its own documentation. A map key may be any comparable type, including a struct, an array,
a pointer, a time or an interface, so the order must be defined for every one of them. A
struct key can be ordered by its fields in declaration order or in field-number order, and
the two differ after a schema author reorders the fields.

## Decision

We will write the entries of a map in ascending key order under one total order over every
key type, with struct keys compared field by field in ascending field number, because equal
values must encode to identical bytes and the bytes must not change when a struct's fields
are reordered.

## Alternatives Considered

### Iteration order

No sorting cost. Rejected because two encodings of one map differ, which breaks hashing and
deduplication.

### Struct keys in declaration order

The order a reader sees in the source. Rejected because reordering the fields of a key type,
which the field numbers make a compatible change, would change the bytes of every stored map
with that key.

## Consequences

**Positive:**

- Equal values encode to identical bytes, except for the cases below.
- A reader in another language can reproduce the order from the specification.

**Negative:**

- Encoding a map sorts its keys, which costs time proportional to n log n and a stack buffer
  of 16 keys, above which the encode allocates.
- Two keys that compare equal but are not equal, two NaNs or -0.0 and +0.0, have no defined
  order, so a map with such keys has no canonical encoding.
- A struct key cannot contain an interface, since its order would depend on the type list of
  another field.

**Neutral:**

- A decoder accepts entries in any order, so the order is a promise of the encoder alone.
