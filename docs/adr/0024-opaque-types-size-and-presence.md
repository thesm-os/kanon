---
adr: 0024
title: Size an opaque value with SizeKanon and test its presence with IsZero
status: Accepted
date: 2026-09-29
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0024: Size an opaque value with SizeKanon and test its presence with IsZero

## Status

Accepted

## Context

The generated code sizes an opaque value by encoding it into a zeroed stack array of 128 bytes,
and writes it by encoding it into another such array and copying the result. Each opaque value
costs two calls of its encode method and two zeroed arrays per encode. A field of a type whose
`==` compares every bit is absent at its zero value, and the generated code tests that by
comparing the whole value with the zero value, in the size pass and again in the write pass.

The ledger's record header has 10 opaque fields. Core's `crypto.Digest` contains 64 bytes of
room and a size byte, and its `id.ID` 32 bytes and a size byte. A profile of the header's
`EncodeKanon` spent 29% of its time in those comparisons and about 40% in the encode helpers
of the opaque values. A struct laid out like the header, with byte arrays in place of the
opaque types, encodes in half the time.

## Decision

We will size a value of a type that declares `SizeKanon() int` with that method and append
its encoding into that room in place, and test the presence of a value of a type that
declares `IsZero() bool` with that method, because both methods give the generated code what
it now derives by encoding and by comparing every byte.

## Alternatives Considered

### A size method per family

One size method for the binary methods and another for the text methods. Rejected because
kanon encodes a type through one family, and one method covers that family.

### A size method with a name of its own

A new name, since `SizeKanon` on a struct codec is part of `kanon.Message`. Rejected because on
an opaque type `SizeKanon` means what it means on a struct codec: the length of the bytes that
kanon writes for the value. A type with `SizeKanon` and binary methods alone is not a
`kanon.Message`, so kanon still encodes it as an opaque value.

### The method Size

Core's `id.ID` and `crypto.Digest` already declare `Size() int` with the right meaning.
Rejected because `Size` means other things on other types, such as the length of a file, and a
kanon method name states that the type declares it for kanon.

### No change

Rejected because the opaque fields cost the ledger's header about half of its encode time.

## Consequences

**Positive:**

- On a struct of 17 fields with 6 opaque values laid out like core's types, an `AppendBinary`
  takes 32 ns instead of 64 ns, the time of the same struct with byte arrays.
- A `kanon.Sizer` with an append method encodes a value of any length without an allocation.
- A type without the methods encodes as before, so no existing encoding changes.

**Negative:**

- `IsZero` must report true for the zero value alone. A type whose `IsZero` reports true for
  another value loses that value, and the conformance suite detects that only for the values
  that it samples.
- A `SizeKanon` that disagrees with the encode method fails every encode of such a value with
  `kanon.ErrSize`.
- Core has to declare `SizeKanon`, and `id.ID.IsZero` has to test the size byte, before the
  ledger gains the full measured time.

**Neutral:**

- A value encodes to the same bytes with both methods, and the zero value is absent as
  before.
