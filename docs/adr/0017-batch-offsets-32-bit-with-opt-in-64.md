---
adr: 0017
title: Batch offsets are 32 bits wide, with 64 bits as an opt-in
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0004
---

# ADR-0017: Batch offsets are 32 bits wide, with 64 bits as an opt-in

## Status

Accepted

## Context

A batch index stores one offset per record. A 32-bit offset addresses 4 GiB and costs 4 bytes
per record, 7% of a 58-byte record before compression. A 64-bit offset costs 8. Compression
blocks are far smaller than 4 GiB, and a batch is compressed as a block.

## Decision

We will store batch offsets as 32-bit integers and let the writer opt into 64-bit offsets
with a flag bit, because a compression block never approaches 4 GiB and 4 extra bytes per
record would cost 7% of a small record for no use.

## Alternatives Considered

### 64-bit offsets always

One index width for readers. Rejected for the 4 bytes per record on every batch.

### 32-bit offsets only

Rejected because a batch that is not a compression block, such as an export file, may exceed
4 GiB.

## Consequences

**Positive:**

- The index costs 4 bytes per record for every batch below 4 GiB.

**Negative:**

- Readers implement two index widths, selected by the flag.
- A writer without the option fails with `ErrTooBig` when the payloads exceed 4 GiB.

**Neutral:**

- The count stays 32 bits in both layouts.
