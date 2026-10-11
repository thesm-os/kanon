---
adr: 0025
title: Generate an index per view type whose methods return an error only for a read that can fail
status: Accepted
date: 2026-10-11
supersedes: ADR-0023
superseded-by: none
rfc: RFC-0002
---

<!--
  ~ Copyright ThesmOS B.V. 2026
  ~ SPDX-License-Identifier: Apache-2.0
-->

# ADR-0025: Generate an index per view type whose methods return an error only for a read that can fail

## Status

Accepted

## Context

Each method of a view type scans the encoding from its start to the last occurrence of its
field. `IndexKanon` records in one scan the offset of the last occurrence of each field that the
view reads. Each method of the index reads its field at the recorded offset, and returns the
value and the error of the view method of the same name.

The scan checks that every value of the encoding is complete, so the read of most fields cannot
fail after it. A bool, a 64-bit integer, a fixed-size number, a float, a `complex64`, a string, a
byte slice and a struct do not need another check. The read of an integer narrower than 64 bits
checks the range of the value, and the scan does not check it. The read of a time, a
`complex128`, a byte array, a type that encodes itself and a `kanon.Validator` makes a check of
the value that the scan cannot make.

The ledger reads every field of its `engine.Op`, a struct of nine fields, through an index in
the decode of each operation of a batch. Six of the nine methods return a nil error on every
path. The other three fail only for a value outside the range of `uint32` or `uint8`. The
ledger joins the ten errors of each operation. At kanon e296e64 with Go 1.27.2 on one
processor, the read of every field takes 34.16 ns with the errors joined and 32.90 ns with the
errors discarded (−3.69%, benchstat, n=12). A review that requires every error checked refuses
the second form. [Issue 8](https://github.com/thesm-os/kanon/issues/8) records the measurement.

## Decision

We will generate beside each view type an index whose scan also checks the range of every
integer narrower than 64 bits, and whose methods return an error only for a read that can fail
after that scan, because a caller then checks one error for all the fields that the scan covers.

## Alternatives Considered

### An error from every method of the index

Each method of the index returns the value and the error of the view method, and the caller
joins the errors. Rejected because seven of the ten errors that the ledger joins per operation
cannot be non-nil after `IndexKanon` returns nil. The other three repeat a check that the scan
can make once.

### Errors that the caller discards

The methods keep both results. The caller discards the errors that cannot be non-nil. Rejected
because a review that requires every error checked refuses that form. The signatures also do not
tell the caller which errors are nil.

### A range check after the scan, at the last occurrence

After its scan, `IndexKanon` checks the integer at each recorded offset. It then fails exactly
where the view methods of those fields fail. Rejected because `IndexKanon` then does not return
the first error in the order of the encoding. A malformed tag after an integer outside its range
fails the scan before the check. A decode returns the first error in that order. It also rejects
a value outside the range at any occurrence.

### A second read of the varint for the range check

The scan skips every value with `wire.Skip`. The range check then reads the varint of the
integer again. Rejected because the second read made `IndexKanon` up to 14.6% slower on the
fixtures that the consequences measure. The scan of this decision reads each varint once.

### A skip in package wire that returns the value of a varint

Package wire gains a function that skips a value as `wire.Skip` does and returns the value of a
varint, so that the range check does not read the varint twice. Rejected because the function
exceeds the inlining budget of the compiler, at a cost of 240 against 80. Its call in front of
`wire.Skip` made `IndexKanon` up to 22.6% slower on the same fixtures.

### A length check of byte arrays and complex numbers in the scan

The scan also checks the length of each byte array and `complex128`, so that their methods
return the value alone too. Rejected because no caller reads such a field through an index.
The ledger, the only caller of an index, reads byte slices and integers through it.

## Consequences

**Positive:**

- A caller that reads only fields whose read cannot fail checks one error, the error of
  `IndexKanon`. Every field of the ledger's `engine.Op` is such a field.
- The scan reads the varint of a field of the wire format Varint itself, as a decode reads it,
  instead of in a call of `wire.Skip`. The range checks take their values from that read.
- On the Record fixture, a struct of 20 fields with 4 narrow integers, the read of its 14 fields
  whose read cannot fail takes 63.9 ns instead of 72.1 ns (−11.4%) with one-byte integers. With
  integers of two to five bytes, it takes 68.5 ns instead of 76.4 ns (−10.3%). On the Item
  fixture, a struct of a string and a `uint32`, the read of both fields takes 7.9% and 2.5% less
  time. benchstat compared 12 runs per side in alternating rounds, with Go 1.27.2 on one
  processor of an AMD Ryzen 9 9950X3D.

**Negative:**

- Each call of an index method whose read cannot fail changes, because the method returns one
  result instead of two. The ledger calls 11 such methods in its batch code.
- `IndexKanon` fails for an integer outside its range at any occurrence. It fails where the view
  method of the field returns a valid last occurrence, and for a field that the caller does not
  read.
- `IndexKanon` alone takes up to 7.1% more time. On Record with integers of two to five bytes it
  takes 39.2 ns instead of 37.9 ns, and on Item with a five-byte `uint32` 14.8 ns instead of
  13.8 ns.
- The error of a varint that does not read has two implementations, `wire.Skip` and the scan of
  each `IndexKanon`. The conformance suite compares their errors on every probe.
- The generated code calls the `ValidateKanon` of a type only under `-validate`, and the
  conformance suite does not read the directive. The suite accepts either form of the index
  method of a `kanon.Validator` whose read cannot fail otherwise.
- Code generated before this decision compiles against the new runtime, and its conformance
  test fails at the index check until `go generate` runs again.

**Neutral:**

- The methods of the view type keep their errors, because each of them scans the encoding.
- A method of the index reads the last occurrence of its field, as a decode does.
- The generated code does not call a function that the runtime lacks, so generated files keep
  generator version 4.
