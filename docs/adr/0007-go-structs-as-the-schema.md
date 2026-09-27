---
adr: 0007
title: Go struct types are the schema, generated in stringer's model
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0007: Go struct types are the schema, generated in stringer's model

## Status

Accepted

## Context

An application declares its data model as Go struct types. A codec needs a schema to number
fields and to know their types. The schema can be the Go types themselves, a schema file in a
language of its own compiled to Go, or the Go types read by reflection at run time. Measured
on 2026-09-27 on 13 types, the reflection codecs of the standard library take 5 to 20 times
the time of generated code and allocate on every decode.

## Decision

We will take the Go struct types as the schema and generate their codecs from a
`//go:generate go tool kanon -type=A,B` directive, as stringer generates `String` methods,
because the data model exists once in the Go types and a schema file is a second copy that
drifts and cannot express struct map keys or interfaces.

## Alternatives Considered

### A schema language compiled to Go, as protobuf, colfer and bebop use

A `.proto`, `.colf` or `.bop` file defines the types. Rejected because the application's
types would be generated from the schema or mirrored by hand, which is the drift the
application avoided by declaring them once, and because the schema languages have no struct
keys, no interfaces and no fixed arrays.

### Reflection at run time, as gob

One codec for any type. Rejected: 5 to 20 times slower, allocations on every decode, and no
allocation contract is possible.

## Consequences

**Positive:**

- One source of truth for the data model, with the compiler checking every field.
- No reflection at run time, and an allocation contract per method.

**Negative:**

- A generated file of about 2,300 lines for a struct of 70 fields of distinct types, since
  every container type gets its own functions.
- A regeneration step after every change to a struct, and a generated file to commit.
- A struct type without a codec, from this package or another, is encoded as an inline struct
  with functions in the referring package's file.

**Neutral:**

- Types with `MarshalBinary`, `GobEncode` or `MarshalText` encode through those methods, as
  gob encodes them.
