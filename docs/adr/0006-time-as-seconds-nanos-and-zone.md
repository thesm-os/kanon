---
adr: 0006
title: Encode a time as seconds, nanoseconds and zone offset
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0001
---

# ADR-0006: Encode a time as seconds, nanoseconds and zone offset

## Status

Accepted

## Context

A `time.Time` has an instant with nanosecond precision over the years 1 to 292277026596, and
a location. gob and json keep the zone offset on a round trip, and kanon encodes what they
encode. A time encoded as one int64 of nanoseconds covers the years 1678 to 2262 and no zone.
Measured on the benchmark record on 2026-09-27, the full time costs 7 bytes and 20 ns more
than the int64 form: 5 bytes for nanosecond precision, 3 for the zone offset of a local time.
A UTC time of whole seconds costs 7 bytes in the chosen encoding and 10 as int64
nanoseconds.

## Decision

We will encode a time as a nested value of three fields, Unix seconds as a zigzag varint,
nanoseconds as a varint when not zero, and the zone offset as a zigzag varint when the
location is not UTC, because it keeps the full range and the zone of a `time.Time` and costs
less than an int64 of nanoseconds for a time of whole seconds.

## Alternatives Considered

### One zigzag varint of Unix nanoseconds

Saves 2 bytes for a time with nanoseconds and costs 3 bytes for a time of whole seconds.
Rejected because it cannot express the years outside 1678 to 2262 or a zone, and storage
timestamps are often whole seconds or milliseconds.

### Plain int64 seconds, as protobuf's Timestamp encodes them

Compatible with protobuf's Timestamp for the instant. Rejected with protobuf compatibility on
2026-09-27, and a pre-1970 time would cost 10 bytes as a plain varint.

## Consequences

**Positive:**

- Every `time.Time` round-trips with its instant and its zone offset.
- A whole-second UTC time costs 7 bytes.

**Negative:**

- A time with nanosecond precision costs 13 bytes, and a time in a zone 3 more.
- Decoding a nested value costs more than one varint, part of the 20 ns measured.
- The zone name is not encoded. A decoder yields a fixed zone with the offset, or the local
  zone when the offset matches it.

**Neutral:**

- A duration is one zigzag varint of nanoseconds.
