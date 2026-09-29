---
adr: 0020
title: Leave out the zero value of an opaque type whose equality compares every bit
status: Accepted
date: 2026-09-29
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0020: Leave out the zero value of an opaque type whose equality compares every bit

## Status

Accepted

## Context

A type with the binary, gob or text methods encodes itself: kanon writes the bytes that its
method returns as an opaque value. An absent field decodes to the zero value of its type. An
opaque field was absent when its encoding had no bytes, so the encode called the method of the
type for every value, the zero value included. Two kinds of type break that rule:

- A type whose zero value has no encoding fails the encode of every struct with such a field at
  the zero value. A digest has no encoding for the zero digest, and the ledger uses the zero
  digest as the absent value, such as the previous digest of the first entry of a stream.
- A type that encodes its zero value to bytes writes them in every record that leaves the
  field unset. A 16-byte instant costs 18 bytes, its tag and its length included.

Every other type of the wire format is absent at its zero value. A float is absent only when
every bit is zero. Under `==`, -0.0 equals +0.0, and a decode of the absent field would lose the
sign.

## Decision

We will leave out an opaque field whose value equals the zero value of its type, compared with
`==` and without a call of a method of the type, when `==` compares every bit of the type,
because an absent field decodes to the zero value, and the method of the type can have no
encoding for that value or can write bytes for it.

## Alternatives Considered

### An IsZero method

A field is absent when the `IsZero` method of its type reports true, for a type that has one,
as the omitzero option of `encoding/json` reads it. Rejected because a value that `IsZero`
calls zero, but that differs from the zero value, decodes to the zero value. Its round trip
changes it.

### A tag option per field

`kanon:",omitzero"` on each optional field. Rejected because presence is a property of the
type. A digest field without the option still fails its encode at the zero digest, and a
missing option shows only at run time, on the first zero value.

### Pointer fields in the application

The application declares `*Digest` fields, since a nil pointer is absent. Rejected because the
application then changes its data model for the codec. Every read of an optional digest also
gains a nil check.

### An encoding for the zero digest

The digest's `AppendBinary` returns no bytes for the zero digest. Rejected because the digest's
package gives the zero digest no encoding on purpose. A missing digest then cannot pass for a
valid one.

## Consequences

**Positive:**

- A struct with a zero digest, a zero instant or a zero identifier encodes, and each zero value
  costs no bytes.
- The size and the encode skip the method of the type for a zero value, so they do not allocate
  for it.

**Negative:**

- The codec of such a type must not decode the encoding of another value to the zero value. A
  codec that breaks this rule, such as one that encodes an integer as its lowest bit, loses the
  field on a second encode of the decoded value. No check enforces the rule.
- Whether the zero value of a type is absent depends on the kinds of its components. A type
  that gains a float, complex or interface component writes its zero value again when its
  encoding has bytes, which changes the encoding of the records that contain it. Every decoder
  reads both forms.

**Neutral:**

- An opaque type with a float, complex or interface component keeps the rule of the encoding
  length. A slice, a map and a function have no `==`, so a type that contains one keeps it too.
- A value other than the zero value whose encoding has no bytes writes its tag and a zero
  length. Under the rule of the encoding length, it was absent and decoded to the zero value.
- The rule applies to the encode alone. A decode reads an absent field as the zero value and a
  present one through the method of its type.
