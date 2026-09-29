---
adr: 0023
title: Generate an index per view type that reads its fields after one scan
status: Accepted
date: 2026-09-29
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0023: Generate an index per view type that reads its fields after one scan

## Status

Accepted

## Context

Each method of a view type scans the encoding from its start to the last occurrence of its
field, so a caller that reads k fields scans the encoding k times. The storage adapters of the
ledger read the stream and the sequence number of every entry through `HeaderView`. Two view
methods take 74 ns on a struct of 17 fields, measured on one core.

A view method returns the last occurrence of its field, as a decode does. It cannot stop at
the first occurrence, because an encoding can repeat a field.

## Decision

We will generate, beside each view type, an index type and a method `IndexKanon` that records
the offset of the last occurrence of every field that the view reads in one scan, because a
caller that reads two or more fields then scans the encoding once while each field keeps the
result of its view method.

## Alternatives Considered

### An index whose methods take the view again

The index of the feature request records the offsets alone. Each of its methods takes the view
as an argument. Rejected because a caller can pass another view than the one that the index
scanned, and the offsets then read other bytes.

### A method named Index

The same request names the method `Index`. Rejected because a view type declares a method per
field, named after the field, so a field named `Index` would collide with it.

## Consequences

**Positive:**

- Reading two fields takes 48 ns through the index instead of 74 ns through two view methods,
  measured on the same struct.
- Each method of the index returns what the view method of the same name returns, and the
  conformance suite checks both against one reference.

**Negative:**

- `IndexKanon` fails where any method of the view fails before it reads a value, so an
  encoding with a malformed field that the caller does not read fails the index and not the
  view method of another field.
- Each view type gains an index type, one method per field, and about 20 generated lines per
  field.
- The index takes one `int` per field that the view reads, which `IndexKanon` returns by
  value.

**Neutral:**

- A view method still returns the last occurrence of its field, as a decode does, and the
  index keeps that rule.
