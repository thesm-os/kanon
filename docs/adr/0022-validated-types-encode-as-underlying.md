---
adr: 0022
title: Encode a named type other than a struct with ValidateKanon as its underlying type
status: Accepted
date: 2026-09-29
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0022: Encode a named type other than a struct with ValidateKanon as its underlying type

## Status

Accepted

## Context

kanon encodes a named type with its own binary, gob or text methods as an opaque value through
those methods, ahead of its underlying type. A type can keep such methods for another purpose,
such as a fixed-width form that callers sign and persist. `epoch.Epoch`, a uint64, and
`fixed.Fixed64`, an int64, keep 8-byte big-endian binary forms, which core has frozen. As an
opaque value, each field of either type costs 10 bytes, against 2 bytes for the epoch 7 as a
varint, measured with the generator on a struct with that one field set. The ledger has 15
fields of type `epoch.Epoch`.

Some of these types admit only part of their underlying type. `Fixed64` excludes
`math.MinInt64`, which its binary methods reject. An encoding as the underlying type has to
keep that check on both the encode and the decode.

`-type` named struct types only. The package of such a type could not generate a kanon encoding
for it. Every consumer encoded the type through its opaque methods.

## Decision

We will encode a named type other than a struct that has `ValidateKanon() error` on a value
receiver as its underlying type, ahead of its binary, gob and text methods, and call the method
on every value that the generated code encodes or decodes, because one method gives every
underlying kind the encoding of that kind while it keeps the domain of the type.

## Alternatives Considered

### A method pair per form

The type declares a pair such as `UintKanon() (uint64, error)` and `SetUintKanon(uint64) error`
per underlying kind. `database/sql` converts values this way through `Valuer` and `Scanner`.
serde does it through `into` and `try_from`. Rejected because it needs a pair per underlying
kind, a range check in every setter of a type narrower than 64 bits, and a rule for the width
that the tag option `fixed` writes. The pair buys a mapping to a form other than the type's own
value. The nine types with such methods in core, the ledger and sharder encode their own
values.

### A hand-written ValidateKanon without generation

kanon finds a method written by hand, so the encoding works without `-type`. Rejected as the
only way, because nothing checks a hand-written method against the type's own methods, and no
golden file pins the encoding in the type's package. A hand-written method still works.

### A generated ValidateKanon that calls the type's own encode method

The generated method encodes the value into a stack array with the type's own method and
returns its error, so the domain of the kanon form matches the type's own form without a flag.
Rejected because every encode and decode then calls the binary or text method, also for a type
that accepts every value, such as `Epoch`, and a type without an append method allocates. The
conformance test catches a missing `-validate` instead.

### A tag option or a directive in the consumer's file

A tag or a directive in the consumer's file makes a field encode as its underlying type. The
`shim` and `replace` directives of msgp work this way. Rejected because every consumer has to
mark every field, and a consumer that forgets writes another encoding of the same value. The
decode also skips the check of the type and admits `math.MinInt64` into a `Fixed64`.

### The underlying type ahead of the binary methods for every named type

Rejected because it changes the encoding of every existing type with such methods without an
opt-in, and it does not check a decoded value.

### Records that store uint64 and int64 and convert at their boundary

Rejected because the records lose the domain types and their checks, and the Go types stop
being the schema.

### The kanon.Message method set on the type

Rejected because its `AppendBinary`, `MarshalBinary` and `UnmarshalBinary` collide with the
binary methods that the type keeps.

## Consequences

**Positive:**

- A field of type `epoch.Epoch` costs 2 bytes for the epoch 7 instead of 10.
- The tag option `fixed`, the order of map keys and the view methods treat the type as its
  underlying type.
- A decode rejects a value outside the domain of the type with a `*DecodeError` at the offset
  of the value, and an encode with an `*EncodeError`.
- `-type` generates the method in the package of the type, and `-validate` names the method
  that it calls. `kanontest.RunValue` checks that the method accepts a value exactly when the
  type's own encode method accepts it, and pins the encoding of each value in a golden file.

**Negative:**

- Adding the method to a type with its own binary, gob or text methods, or removing it,
  changes the encoding of every field of the type, incompatibly in both directions. The field
  numbers stay the same, so their check passes. The golden files of the conformance suite
  fail.
- A `ValidateKanon` that allocates makes the encode and the decode allocate. The conformance
  test catches it for a type that `-type` names, and not for a method written by hand.
- The view method of a string field of such a type copies the string.
- The generator type-checks a package without its generated files, so a generated
  `ValidateKanon` does not make a type of the same package fit an interface that lists the
  method.

**Neutral:**

- The generator rejects the method on a struct type, on a pointer receiver and with another
  signature, and on a type whose only value is its zero value.
- A named interface type whose method set has the method encodes as any other interface.
- The wire format does not change. The type takes the wire format of its underlying type.
