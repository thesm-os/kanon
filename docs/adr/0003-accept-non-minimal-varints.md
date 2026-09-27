---
adr: 0003
title: Decoders accept non-minimal varints
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0003: Decoders accept non-minimal varints

## Status

Accepted

## Context

A varint has one shortest form and many longer ones with leading zero groups, such as `80 00`
for 0. Canonical bytes, which storage engines hash and deduplicate, need encoders to write the
shortest form. A decoder could also reject the longer forms, so that only canonical input
decodes, at the cost of a check on every varint it reads.

## Decision

We will make decoders accept a varint of up to 10 bytes in any form while encoders write the
shortest form, because canonical bytes are a property of the encoder and a strictness check
per varint costs time on the hot path and rejects nothing that matters.

## Alternatives Considered

### Reject non-minimal varints

Only canonical input decodes. Rejected because the check runs on every varint of every
decode, and the data it rejects comes only from a non-conformant writer, which a decoder
cannot repair anyway.

## Consequences

**Positive:**

- The decoder's varint read is one loop without a form check.
- Input from a tolerant writer decodes.

**Negative:**

- The output of a non-conformant writer decodes and re-encodes to different bytes, so a
  deduplication by hash must re-encode instead of hashing the input.
- A decoder still rejects an 11th byte and a 10th byte above 1, so the limit check stays.

**Neutral:**

- The conformance suite tests the accepting behaviour with a non-minimal vector.
