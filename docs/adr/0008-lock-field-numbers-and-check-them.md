---
adr: 0008
title: Lock field numbers in the generated file and check them against a revision
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0008: Lock field numbers in the generated file and check them against a revision

## Status

Accepted

## Context

A field is identified on the wire by its number alone. A field that changes number decodes
another field's data, silently, into the wrong place. protobuf's language guide names the
consequences of a reused number: parse errors at best, leaked data and corruption otherwise.
kanon numbers fields without tags automatically, so the numbers must survive regeneration
after fields are added, removed or reordered, and a review must catch a change that
renumbers a field.

## Decision

We will record the number of every field and every interface type in a `//kanon:numbers`
line of the generated file, keep the recorded numbers across regeneration, reserve the number
of a removed field, and check the numbers against a git revision with `KANON_CHECK`,
because a renumbered field misreads stored data and nothing on the wire can detect it.

## Alternatives Considered

### Numbering by declaration order on every generation

No record to keep. Rejected because removing or reordering a field renumbers the fields after
it, and every stored record then decodes into the wrong fields without an error.

## Consequences

**Positive:**

- Adding, removing and reordering fields are compatible changes.
- A pull request that renumbers a field fails CI.

**Negative:**

- The generated file must be committed with its source. Deleting it loses the record.
- A renamed field is a new field, and keeping its data needs a tag with the old number.
- The check needs git and the base revision's generated files.

**Neutral:**

- A tag may name a reserved number on purpose, which the check accepts.
