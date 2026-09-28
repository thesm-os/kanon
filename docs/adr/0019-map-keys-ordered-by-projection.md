---
adr: 0019
title: Order map keys by their projection, and fail the encode of NaN and tied keys
status: Accepted
date: 2026-09-28
supersedes: ADR-0005
superseded-by: none
rfc: RFC-0001
---

# ADR-0019: Order map keys by their projection, and fail the encode of NaN and tied keys

## Status

Accepted

## Context

Storage engines hash, deduplicate and sign records, so equal values must encode to identical
bytes, and a decode must return every entry that an encode wrote. Go iterates a map in random
order. A map key may be any comparable type that gob encodes: a struct, an array, a pointer,
a time, an interface or a type that encodes itself. A struct key can be ordered by its fields
in declaration order or in field-number order, and the two differ after a schema author
reorders the fields.

A Go map compares its keys with `==`, which differs from the equality of their encodings:

- `-0.0` and `+0.0` are equal under `==`, so a map keeps one of them. Their IEEE 754 bits
  differ.
- A NaN is unequal to itself, so a map can keep any number of NaN keys. No lookup finds them,
  and no order sorts them.
- Two keys can differ under `==` and encode alike: two pointers to equal values, two structs
  that differ only in a skipped field, two times with one instant and offset in zones of
  different names, and two values of a type that encodes itself to the same bytes.

Without a rule for these keys, a map that contains them has no canonical encoding. The decode
of its encoding keeps one entry for the keys that encode alike and loses the others without an
error.

## Decision

We will write the entries of a map in ascending order of the projections of their keys, the
encoding of each key with every `-0.0` float component written as `+0.0`, under one total order
over every key type that compares struct keys field by field in ascending field number, and
fail the encode of a map with a key that has a NaN component or with two keys of one
projection, because equal values must encode to identical bytes and no decode may lose an
entry that an encode wrote.

## Alternatives Considered

### Reject every key type that can tie

Reject pointer, time, opaque, skipped-field and interface key types when the code file is
generated, so that Go's `==` decides which keys are equal. Rejected because it removes key
types that gob encodes and that the fixtures use. The rule of this decision keeps them and
fails only a map that has two keys of one projection.

### Leave ties and NaN keys without an order

Write such keys in whatever order the sort leaves them. Rejected because the encoding of such
a map is not canonical, and its decode loses entries without an error.

### Iteration order

No sorting cost. Rejected because two encodings of one map differ, which breaks hashing and
deduplication.

### Struct keys in declaration order

The order a reader sees in the source. Rejected because reordering the fields of a key type,
which the field numbers make a compatible change, would change the bytes of every stored map
with that key.

## Consequences

**Positive:**

- A map that encodes has one encoding, whatever order Go iterates it in.
- A decode keeps every entry that an encode wrote.
- A reader in another language can reproduce the order and the projection from the
  specification.

**Negative:**

- A map with two keys of one projection, such as two distinct pointers to equal values, fails
  to encode with `kanon.ErrAmbiguousKey`, where gob encodes it and its decode keeps one entry.
- A key with a NaN component fails to encode with `kanon.ErrInvalidKey`.
- A key of `-0.0` decodes as `+0.0`.
- Encoding a map sorts its keys, which costs time proportional to n log n and a stack buffer
  of 16 keys, above which the encode allocates. For a key type that can tie, the encode also
  compares neighbouring keys, and a decode or a merge compares keys by projection.
- A struct key cannot contain an interface, since its order would depend on the type list of
  another field, nor a union member, whose encoding depends on its discriminator.

**Neutral:**

- A decoder accepts entries in any order, so the order is a promise of the encoder alone.
- A merge into a map that already has two keys of one projection fails with
  `kanon.ErrAmbiguousKey` before it changes the map.
