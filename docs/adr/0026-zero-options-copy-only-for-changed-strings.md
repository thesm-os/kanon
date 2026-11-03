---
adr: 0026
title: Pass decode options in one struct whose zero value copies the input only for a string that changes
status: Accepted
date: 2026-11-03
supersedes: ADR-0011
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright ThesmOS B.V. 2026
  ~ SPDX-License-Identifier: Apache-2.0
-->

# ADR-0026: Pass decode options in one struct whose zero value copies the input only for a string that changes

## Status

Accepted

## Context

`DecodeKanon` and `MergeKanon` take the slab, its offset and the nesting limit in one `Options`
struct. With an empty `Slab`, the generated decode of a struct that contains a string copies the
whole input into a string before it reads a field, and every decoded string is a substring of that
copy. The decode makes the copy for an input that contains no string. It also makes the copy when
every decoded string equals the string that the receiver already has.

[Issue 20](https://github.com/thesm-os/kanon/issues/20) measured the copy at kanon v0.4.0 with
`-canonical`, with `testing.AllocsPerRun` over 1,000 decodes into one reused value:

| Input | Allocations per decode |
|---|---|
| `Record{Count uint64; Lines []Line}` with `Line{Name string; Value []byte}`, without lines, 2 bytes | 1 |
| The same `Record` with one line, 95 bytes, decoded again into the value of the last decode | 1 |
| `Plain{Count uint64}`, 2 bytes | 0 |

A record type that gains an optional list of named entries allocates in every decode, also when
the list is empty and when its names repeat. A caller avoids the copy with a slab over its input,
which takes `unsafe.String`. The decoded strings then alias memory that the caller reuses for the
next input.

The decode already reuses the memory of the receiver: the values that its pointers point at, its
nested structs, the elements in the capacity of its slices, and up to 16 values of each map whose
values refer to memory. A seen bitmap records the pointers, interfaces, structs and maps that the
input contains, and the decode clears the others after the fields. In core,
`note.Note.UnmarshalText` reuses its signature lines the same way. It keeps each name that equals
the name of the line at its position, and allocates one string for the names that change.

## Decision

We will pass the decode options in one `Options` struct, whose empty `Slab` makes a decode keep
each string of the receiver that the input repeats at its place and copy the input once for the
first string that changes, because a decode into a receiver that decoded the same strings before
then does not allocate without a slab.

A decode with an empty `Slab` follows these rules:

- A decoded string keeps the string of the receiver when their bytes are equal, at every place
  where the decode reuses memory. Those places are a field, an element in the capacity of a slice,
  an element of an array, the value that a pointer points at, a map value that the decode takes
  from the values of the map, and the value of an interface that stores the same concrete type.
- The decode copies its input once, for the first decoded string that is not empty and that the
  receiver does not have at its place. Every later such string of the decode is a substring of
  that copy.
- The strings of a map key, and of a map value that does not refer to memory, take substrings of
  the copy, since the decode clears a map before it reads its entries.
- A nested struct of another package decodes with the copy as its slab when the decode has made
  the copy, and with an empty `Slab` otherwise. It then applies these rules to its own encoding.

A decode with a slab in its `Options` takes every decoded string from the slab. It does not keep
a string of the receiver.

## Alternatives Considered

### A copy for the first non-empty string, without kept strings

The decode copies its input when it decodes its first string that is not empty. It does not keep
a string of the receiver. Rejected because only an input without strings then decodes without the
copy. The record of the issue that decodes again into the value of its last decode still copies.

### Kept strings under a slab

A decode with a slab also keeps a string of the receiver that equals the decoded bytes. Rejected
because a kept string can alias memory that the caller reuses. The caller expects every decoded
string to be a substring of its slab. A stream decoder appends the runs of one encoding to a
buffer that it reuses for the next encoding. A string kept from the previous encoding can alias
bytes past the current run. A later run of the same decode writes over those bytes.

### A buffer of the changed strings

The decode allocates a buffer of the length of the input for its first changed string and copies
each changed string into it. `note.Note` collects its changed names into one string the same way.
Rejected because the decoded strings then alias a byte slice that the decode writes until it ends.
Such strings take `unsafe.String` in package wire. `note.Note` avoids that with a second pass over
its lines. A decode reads its input in one pass, and the copy of the input is an immutable string.

### A field of Options for a copy that nested decodes share

`Options` gains a field through which the decode of a nested struct of another package uses the
copy of the outer decode. Rejected because `Options` is the interface of every decode, and the
field would expose a mechanism of the generated code to every caller. The field saves one copy for
each nested value with a string that changes.

### A copy before every nested struct of another package

The decode makes its copy before it calls the decode of a nested struct of another package, and
passes the copy as the slab. Rejected because every decode of a record with such a field then
copies its input, also when every string repeats. The issue measures that cost.

### Offsets from Offset under an empty Slab

A decode with an empty `Slab` counts the offsets of its errors from `Offset`, so that a nested
struct of another package reports offsets in the outer input. Rejected because code generated
before this decision counts them from 0. A nested struct whose package has older code would then
report offsets from the start of its own encoding. The decode passes `Offset` 0 to such a struct
instead, and adds the offset of the struct to the offset of its error. Code of both versions then
reports the same offsets.

### No change

A caller that needs a decode without an allocation passes a slab over its input. Rejected because
the slab takes package unsafe, and the decoded strings then alias memory that the caller reuses.

## Consequences

The measurements compare 12 runs per side in alternating rounds with benchstat, with Go 1.27.3 on
one processor of an AMD Ryzen 9 9950X3D.

**Positive:**

- A decode with an empty `Slab` into a receiver that decoded the same input before does not
  allocate, except for a map with a string that is not empty in a key, or in a value that does not
  refer to memory. The exceptions of a decode with a slab still apply, such as a type that decodes
  itself.
- The decodes that the issue measures do not allocate: a `Record` without lines, a `Record` of one
  line that decodes again into the value of its last decode, and a `Plain`. The `Record` of one
  line takes 13.5 ns instead of 30.9 ns, and one of four lines 40.0 ns instead of 78.6 ns.
- `UnmarshalBinary` decodes with the zero `Options`, so it decodes the same way.
- A decode with a slab, a stream decoder, a frame and a batch take every string from their slab.

**Negative:**

- A kept string keeps the memory that it refers to. When an earlier decode took a string of the
  receiver from a slab over a buffer that the caller reuses, a decode with an empty `Slab` keeps
  that string, and the string changes with the buffer. `Reset` clears the strings of a receiver,
  so a decode after `Reset` does not keep such a string.
- A decode with an empty `Slab` compares each decoded string with the string at its place before
  it copies the input. A `Record` of one line whose name changes takes 34.2 ns instead of 31.0 ns,
  and one of four such lines 88.0 ns instead of 78.4 ns.
- A decode with a slab reads every string through `wire.Slab.String` in place of a substring of the
  slab. A `Record` of four lines takes 35.9 ns instead of 32.3 ns. The decodes of the fixtures of
  the conformance suite take 4% to 21% more time, such as 7.5 ns instead of 6.2 ns for a struct of
  a string and a `uint32`, and 332 ns instead of 316 ns for `canonical.Deep`.
- A map with a string that is not empty in a key, or in a value that does not refer to memory,
  copies the input in every decode with an empty `Slab`.
- A record with n nested values of a struct of another package can make n copies of their
  encodings and one copy of its own, where it made one copy before.
- The decode of a struct with a string field records the field in its seen bitmap, and clears it
  after the fields when the input leaves it out. Such a struct gains the loop function of a struct
  with tracked fields.

**Neutral:**

- The encoding does not change, and neither do `Options`, `kanon.Message` and the signatures of
  the generated methods.
- The generated decode passes one pointer to a `wire.Slab` through its functions in place of the
  slab string. `wire.Slab.Reset` sets it in place, so a decode with a slab and without strings
  takes the time that it took before: 2.9 ns for a `Record` without lines. The `Slab` contains the
  slab of the options, or the input and the copy once the decode makes it.
- The loop function of a struct with tracked fields takes the seen bitmap by pointer, so that a
  tracked string does not add an operand to its error returns. With the bitmap as a result, the loop
  function of `canonical.Scalars` passed the 5,000 nodes above which the compiler inlines no
  `wire.Uvarint`, and its decode took 504 ns instead of 169 ns.
- A slice whose elements contain a string decodes into the elements in its capacity, as a slice
  whose elements refer to memory does.
- A nested struct of another package without the copy decodes with `Offset` 0, and the decode adds
  the offset of the struct to the offset of its error, as a stream decoder adds the offset of a
  run.
- The offsets of the errors of a decode with an empty `Slab` count from 0.
- Code generated before this decision compiles against the new runtime and copies its input as
  before. Generated files declare generator version 6, since they call `wire.Slab`, which earlier
  runtimes lack.
