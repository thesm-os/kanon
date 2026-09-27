---
adr: 0004
title: The encoding carries no version
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0004: The encoding carries no version

## Status

Accepted

## Context

Encodings travel in containers: a frame on a stream, or a block in a storage engine. A format
version could be written in every encoding, in every container, or in both. A per-encoding
marker costs at least one byte per record, and nested values would need one too or be left
unversioned.

## Decision

We will write no version in an encoding and require every container of encodings, a frame or
a block, to carry the version, because a per-record byte is paid on every record while a
container carries one byte for all of them, and a change to the format is a new format.

## Alternatives Considered

### A version byte in every encoding

One byte per record, and a rule for nested values. Rejected because the container already
identifies its content and the byte would be repeated for every record in a block.

## Consequences

**Positive:**

- Zero overhead per record.
- A container names its version once.

**Negative:**

- A bare encoding is not self-identifying. A tool that reads one needs to know its format.
- Every container format must define a version field, and a new container format must too.
- The format cannot change in place. A change is a new format under a new container version.

**Neutral:**

- The frame format and the batch format each have a version byte.
