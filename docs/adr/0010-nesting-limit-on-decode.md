---
adr: 0010
title: Bound the nesting depth of a decode, 100 levels by default
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0010: Bound the nesting depth of a decode, 100 levels by default

## Status

Accepted

## Context

The decoder recurses once per nested struct, slice, map, pointer and interface. A corrupt or
hostile encoding nests one level per two bytes, so a 16 MiB storage block can nest 8 million
levels. Go ends the process when a goroutine's stack exceeds 1 GB, and that failure cannot be
recovered. protobuf-go limits recursion to 10,000 levels for the same reason.

## Decision

We will bound the nesting a decode enters, at 100 levels by default and at the value the
caller sets in the decode options, because a corrupt block must produce an error and not end
the process.

## Alternatives Considered

### Bound the depth by the input size

Every level costs at least two bytes, so a frame or block size limit bounds the depth.
Rejected because a 16 MiB block allows 8 million levels, which exceed the stack limit.

### No limit

Rejected: a storage engine that reads a corrupt block would crash.

## Consequences

**Positive:**

- A corrupt block returns `ErrDepth` with the field and offset.

**Negative:**

- A legitimate value nested deeper than 100 levels needs a caller that raises the limit.
- Every decode function takes a depth and decrements it per level.

**Neutral:**

- Types that encode themselves decode through their own methods, which the limit does not
  reach.
