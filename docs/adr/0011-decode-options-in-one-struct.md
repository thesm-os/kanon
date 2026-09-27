---
adr: 0011
title: Pass decode options in one struct
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0011: Pass decode options in one struct

## Status

Accepted

## Context

A decode takes a slab and an offset, so that decoded strings alias one buffer, and a nesting
limit. Each could be a parameter of `DecodeKanon` and `MergeKanon`, which are methods of the
`Message` interface that every generated type and every consumer depend on. Adding a
parameter changes that interface and every implementation and caller of it.

## Decision

We will pass the slab, its offset and the nesting limit to `DecodeKanon` and `MergeKanon` in
one `Options` struct whose zero value copies the input once and applies the default limit,
because a positional parameter per option changes the interface every time an option is
added.

## Alternatives Considered

### Positional parameters

`DecodeKanon(data, slab, slabOff, depth)`. Rejected because the fourth parameter was already
the second change to the signature, and the next option would be the third.

### The nesting limit as a generated constant only

No per-call limit. Rejected because a caller that decodes a deep but legitimate value could
not raise it, and one that decodes untrusted input could not lower it.

## Consequences

**Positive:**

- The interface is stable when an option is added.
- The zero value copies the slab and applies 100 levels, which is the safe default.

**Negative:**

- Every call writes `kanon.Options{...}`, 32 bytes passed by value.
- The meaning of each zero field must be documented and tested.

**Neutral:**

- `UnmarshalBinary(data)` is `DecodeKanon(data, Options{})`.
