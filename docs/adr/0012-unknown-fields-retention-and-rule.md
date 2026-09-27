---
adr: 0012
title: Keep unknown fields through an opt-in field, and rewrite records as opaque bytes otherwise
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0012: Keep unknown fields through an opt-in field, and rewrite records as opaque bytes otherwise

## Status

Accepted

## Context

A decoder skips a field whose number it does not know. A binary that decodes a record written
by a newer binary, changes it and re-encodes it drops the fields it did not know. A proxy that
rewrites messages and a storage compaction that rewrites records both do this during a rolling
upgrade. protobuf keeps unknown fields in its messages and writes them back.

## Decision

We will let a struct declare one `[]byte` field tagged `unknown` that receives the bytes of
every unknown field on decode and is written after the known fields on encode, and we will
require a storage engine that rewrites records of a type without that field to treat them as
opaque bytes, because a rewrite through an older type must not lose what a newer one wrote.

## Alternatives Considered

### Skip unknown fields and keep nothing

The smallest decoder and no extra field. Rejected because a proxy or a compaction would lose
data written by a newer binary.

### The retention field alone

Every type that a storage engine rewrites would need the field. Rejected as the only
mechanism, because storage engines rewrite records of every type and treating them as bytes
is simpler than a field in each.

### The opaque-rewrite rule alone

A proxy that decodes a message cannot follow it. Rejected as the only mechanism.

## Consequences

**Positive:**

- No silent loss of fields during a rolling upgrade, whichever path a record takes.

**Negative:**

- The retention field is an exported `[]byte` that is not part of the data model.
- A struct with unknown bytes has no canonical encoding, because the bytes keep the order of
  their source.
- Two mechanisms to document, and a rule the storage engine must follow without the compiler
  checking it.

**Neutral:**

- `Reset` clears the field, `MergeKanon` appends to it and `CloneKanon` copies it.
