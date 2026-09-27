---
adr: 0018
title: Compression stays with the storage engine
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0004
---

# ADR-0018: Compression stays with the storage engine

## Status

Accepted

## Context

Block compression is where storage density comes from. Measured on 2026-09-27, zstd over
blocks of 1,000 records took the benchmark record from 58.6 to 32.6 bytes, and every codec
measured ended within 30% of the others after it, while compressing one record at a time
added 13 bytes to each. A storage engine chooses its compressor, its level and its
dictionary, and often compresses its own block header together with the records.

## Decision

We will leave compression to the storage engine and keep the batch package free of
compression, because the engine chooses the compressor, the level and the dictionary, and a
compressor in the package would add a dependency and fix that choice.

## Alternatives Considered

### zstd inside the batch package

A compressed batch in one call. Rejected because it adds a dependency to every consumer,
fixes the compressor and level, and separates the batch from the block header the engine
compresses with it.

### Compression per record

Independent of any block layout. Rejected: measured 13 bytes of overhead per record, which
made every record larger than uncompressed.

## Consequences

**Positive:**

- The batch package depends on the standard library and `kanon` alone.
- The engine tunes compression to its data.

**Negative:**

- Every engine wires compression itself.
- A tool that reads a stored batch must know the engine's compression.

**Neutral:**

- The batch layout with its ascending offsets compresses well, since the offsets differ by
  small amounts.
