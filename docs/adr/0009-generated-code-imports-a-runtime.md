---
adr: 0009
title: Generated code imports a small runtime
status: Accepted
date: 2026-09-27
supersedes: none
superseded-by: none
rfc: RFC-0002
---

# ADR-0009: Generated code imports a small runtime

## Status

Accepted

## Context

Generated code can import only the standard library and repeat its helper functions, about
270 lines per generated file, or import a runtime package that declares them once. Without a
runtime, consumers share only the standard library's error sentinels, so a storage engine
cannot tell a corrupt record from one that exceeds a limit, and every decode option is a
parameter added to the method signatures. A consumer's module already requires
`go.thesmos.sh/kanon` for the `tool` directive that runs the generator. protobuf-go ties
generated files to its runtime with version constants that fail to compile on a mismatch.

The rule that generated code imports only the standard library was set earlier and replaced
on 2026-09-27: "small runtime, agreed. compatibility is ok. We write once, set the standards
and are good."

## Decision

We will make the generated code import `go.thesmos.sh/kanon` for the `Message` interface,
the decode options, the typed errors and the version constants, and `go.thesmos.sh/kanon/wire`
for the varint, time and skip functions, because typed errors and options need a shared
package and the consumer depends on the module already.

## Alternatives Considered

### Generated code that imports only the standard library

Every generated file compiles alone and repeats the helpers. Rejected because consumers get
no typed errors, decode options grow the method signatures, the helpers exist once per file
with a mutation run per copy, and no consumer needs a file that compiles alone.

## Consequences

**Positive:**

- One implementation of the shared functions to fuzz and mutate.
- `DecodeError` and `EncodeError` with causes a consumer can test with `errors.Is`.
- Generated files shrink by about 270 lines each.

**Negative:**

- The runtime promises compatibility with every generator version from `MinVersion` to
  `MaxVersion`, and dropping a version is a breaking change of the module.
- Every generated file declares two `EnforceVersion` constants.
- `kanon/wire` is a public package that applications should not use. Its documentation is
  what keeps them out, since Go cannot restrict an export to generated code.
- Every consumer's binary links the runtime.

**Neutral:**

- Within one module, the `tool` directive makes the generator and the runtime the same
  version.
