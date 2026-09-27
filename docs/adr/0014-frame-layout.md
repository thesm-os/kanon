---
adr: 0014
title: Frame messages on a stream with length, version, flags, type ID and optional CRC-32C
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0003
---

# ADR-0014: Frame messages on a stream with length, version, flags, type ID and optional CRC-32C

## Status

Accepted

## Context

An encoding has no length and no type of its own. Two servers on a stream need the length to
find where a message ends and the type to know which struct decodes it. Without a shared
frame, each service frames its own way and two services cannot talk. A fleet also needs to
change the frame layout during a rolling upgrade, and a transport without integrity checks,
such as a plain pipe or a file, needs a checksum.

## Decision

We will frame each message as a uvarint length, a version byte, a flags byte, a uvarint type
ID, the payload and an optional CRC-32C, because a stream needs the length and type that the
encoding lacks, a version byte lets the layout change under a rolling upgrade, and the
checksum is an option for transports that do not verify integrity.

## Alternatives Considered

### gRPC as the transport

It frames, multiplexes and flow-controls. Rejected as the only transport because it needs
HTTP/2 and its dependency tree, and a fleet that streams over TCP or a message bus needs only
a frame. A gRPC codec adapter of six lines is possible beside the frame.

### Fixed-width length and type

A 4-byte length and a 4-byte type ID. Rejected: 8 bytes per frame against 2 to 3 for the
small frames that dominate fleet traffic.

### No version byte

Rejected because a frame layout that cannot change without a flag day never changes.

### Checksum always on

Rejected because TLS and TCP verify integrity already, and the checksum costs 4 bytes per
frame and time per byte.

## Consequences

**Positive:**

- One frame layout for every service, with a registry from type IDs to constructors.
- A reader rejects an unknown version or flag before it reads the payload.

**Negative:**

- 4 to 6 bytes per frame, plus 4 for the checksum.
- A decoded message aliases the reader's buffer until the next frame, which the race detector
  does not catch.
- The default limit of 16 MiB per frame must be raised by callers with larger messages.

**Neutral:**

- A frame with an empty payload is valid.
