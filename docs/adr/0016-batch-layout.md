---
adr: 0016
title: Lay out a batch as header, payloads, then offsets and count
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0004
---

# ADR-0016: Lay out a batch as header, payloads, then offsets and count

## Status

Accepted

## Context

A storage engine writes records in blocks and reads one record of a block far more often
than all of them. A batch of encodings needs an index for random access, and a writer that
appends records as they come does not know their count until the end. A batch that is copied
out of its block, or read by a tool, has no block header to identify it.

## Decision

We will lay out a batch as a version byte and a flags byte, then the payloads, then the
offsets and the count, because the writer then appends in one pass, the reader locates record
i with one index read, and the header rejects a batch of another version before the index is
read.

## Alternatives Considered

### The index at the start

The reader finds the index without reading the tail. Rejected because the writer must know
the count before it writes, which means buffering the payloads or a second pass.

### Uvarint lengths before each payload

The layout of a stream of frames. Rejected because finding record i costs i reads instead of
one.

### Version and count in the engine's block header

No header bytes in the batch. Rejected because a batch outside its block is then not
self-describing, and two bytes per batch make it so.

## Consequences

**Positive:**

- One-pass writes, one-read random access, and a self-describing batch.

**Negative:**

- 4 bytes of index per record and 6 bytes per batch.
- A batch has at most 2^32 - 1 records.

**Neutral:**

- The payload region is a plain concatenation, so a reader can also scan it sequentially.
