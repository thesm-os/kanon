---
adr: 0015
title: Frame type IDs are constants the application assigns
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0003
---

# ADR-0015: Frame type IDs are constants the application assigns

## Status

Accepted

## Context

A frame names the type of its payload by an integer ID, and a registry maps IDs to
constructors. The ID can be a constant the application assigns and registers, or a hash of a
registered type name computed by the library.

## Decision

We will identify message types on the wire by integer IDs that the application assigns as
constants and registers, because a hash of a type name changes when the type is renamed and
collides silently.

## Alternatives Considered

### A hash of the registered name

No constants to maintain. Rejected because a rename changes the ID, which breaks every peer
and every stored stream, and two names can hash to one ID without an error.

## Consequences

**Positive:**

- IDs are small varints and stable across renames.
- A duplicate registration fails at startup.

**Negative:**

- The application maintains the ID list by hand. An ID that was used for one type cannot be
  used for another.

**Neutral:**

- The registry is part of the application's wire contract, like its field numbers.
