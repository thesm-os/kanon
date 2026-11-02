---
rfc: 0010
title: Strict types that encode themselves
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-11-02
updated: 2026-11-02
discussion: https://github.com/thesm-os/kanon/issues/17
supersedes: none
superseded-by: none
produces-adr: none
---

<!--
  ~ Copyright ThesmOS B.V. 2026
  ~ SPDX-License-Identifier: Apache-2.0
-->

# RFC-0010: Strict types that encode themselves

## Summary

A type that encodes itself through its binary, gob or text methods can declare `kanon.Strict` with
one marker method, `StrictKanon`. The declaration states that the decode method of the type
accepts a byte string only when the encode method writes that byte string for the value that the
decode method sets. A canonical decode then does not encode a value of the type again, in any
position. The canonical decode of a field of every type that encodes itself and whose `==`
compares every bit checks an error of the decode method and the presence rule in one statement.
`kanontest.RunStrict` checks the declaration in the package that declares the type. The canonical
decode of an enumeration with text methods then has no statement that a test cannot run.

## Motivation

A canonical decode checks a field of a type that encodes itself and whose `==` compares every bit
in three steps after its length. The decode method sets the value, the presence rule rejects the
zero value, and a second encode must write the bytes of the input. Each step returns its error
from a statement of its own.

The decode method of an enumeration with text methods accepts only the names of its variants. Its
encode method writes the same names, and its zero value is often not a variant. dokimi writes such
methods for the enumerations that it generates. For a field of such an enumeration, two of the
three statements cannot run:

- The decode method never sets the zero value, so the presence rule never fails.
- The encode method writes the bytes that the decode method accepted, so the comparison never
  fails.

kanon v0.3.1 writes these statements for the canonical decode of a field `Mode` of such an
enumeration:

```go
m.Mode = 0
if err := m.Mode.UnmarshalText(data[i : i+int(l)]); err != nil {
	return wire.UnmarshalError(err, "Scalars.Mode", 40, off+i)
}
if m.Mode == 0 {
	return wire.AbsentError("Scalars.Mode", 40, off+at)
}
{
	enc, err := m.Mode.MarshalText()
	if err != nil || !bytes.Equal(enc, data[i:i+int(l)]) {
		return wire.EncodingError("Scalars.Mode", 40, off+i)
	}
}
i += int(l)
```

For such a type, the return of `AbsentError` and the return of `EncodingError` are dead code. A
module that requires full statement coverage of its generated code fails that requirement, and a
consumer with two such fields reported four statements that its specs cannot cover.

The second encode also runs on every decode. Take a canonical record of a string, such an
enumeration and a `uint32`. Its decode into a receiver that decoded it before takes 22.8 ns and
2 allocations when the enumeration has the text methods that dokimi writes. Without the second
encode it takes 10.9 ns and 1 allocation, which the decode method of the enumeration makes.

The canonical decode of a field of an Exact type has neither statement. An enumeration keeps only
one of the two guarantees of Exact:

- Its encode method fails for a value outside its variants, such as `Mode(9)`. The first
  guarantee allows the append method to fail for the zero value alone, and the generated code
  writes an Exact field without an error path.
- Its decode method accepts only the bytes that its encode method writes for the decoded value.
  This is the second guarantee.

Exact also requires an append method and `SizeKanon`, and an enumeration with `MarshalText` alone
has neither.

The generator cannot read the second guarantee from the method set of a type. The text type
`Count` of kanon's fixtures has the methods of such an enumeration, and its decode method accepts
`007`, which its encode method writes as `7`. The type states the guarantee, and the conformance
suite checks the statement.

## Detailed design

### Terms

- An **opaque type** is a type that encodes itself through the methods of a family: binary, gob
  or text.
- A **strict type** is an opaque type that declares `kanon.Strict`.
- A **bitwise type** is a type whose `==` compares every bit. A field of a bitwise opaque type
  leaves out its zero value.
- A **canonical decode** is `DecodeKanon`, `MergeKanon` or `UnmarshalBinary` of a struct type
  whose directive sets `-canonical`.

### Components

| Component | Change |
|---|---|
| The generator | Reads `StrictKanon()` on an opaque type. A canonical decode does not encode a value of a strict type again. The canonical decode of a field of a bitwise opaque type checks the decode error and the presence rule in one statement. The generation fails for a type that declares `StrictKanon` and is not opaque. The generator version is 5 |
| `kanon` | The interface `Strict` |
| `kanon/wire` | The contract of `ExactError` covers a field of every bitwise opaque type |
| `kanontest` | `StrictChecks` and `RunStrict` |
| `Message`, `Options`, the default decode, the encode, views, indexes, `frame`, `batch` and `kanon inspect` | No change |

### Invariants

1. A canonical decode of a strict type that keeps its guarantee accepts the inputs that it accepts
   without the declaration. For every other input it returns the same error at the same offset.
2. The single statement of a field of a bitwise opaque type returns the error of the decode method
   first and the error of the presence rule second. The decode reports the same first failure as
   with two statements.
3. The encode of a value of a strict type returns the error of its encode method, as the encode of
   every opaque type that is not Exact does.
4. A default decode encodes no value again, and it does not change.
5. A canonical decode of a strict type that breaks its guarantee accepts an input that is not the
   canonical encoding of its value. `RunStrict` and the decode checks of a canonical Spec detect
   such a type on the inputs that they build.

### The marker

```go
package kanon

// Strict is the method set of a type that encodes itself through its binary,
// gob or text methods, and whose decode method accepts a byte string only
// when the encode method of its family writes that byte string for the value
// that the decode method sets. The guarantee is the second guarantee of
// [Exact], without the first, so the encode method can fail for any value.
// An enumeration whose text methods write and parse the names of its
// variants keeps it, and fails to encode a value outside its variants.
//
// # Encoding
//
// A canonical decode does not encode a decoded value of a Strict type again,
// in any position. The encode returns the error of the encode method, as it
// does for every type that encodes itself and is not Exact. The generation
// fails for a type that declares StrictKanon and does not encode itself
// through the methods of a family, such as a [Validator] or a struct with a
// kanon codec.
//
// [go.thesmos.sh/kanon/kanontest.RunStrict] checks the guarantee, in the
// package that declares the type.
type Strict interface {
	// StrictKanon marks the type. No code calls it.
	StrictKanon()
}
```

The contract of `Exact` gains one sentence: the second guarantee is the guarantee of `Strict`, so
an Exact type need not declare `StrictKanon`.

A type declares the marker beside its methods:

```go
// Mode is an enumeration whose text methods write and parse the names of
// its variants. The zero Mode is not a variant.
type Mode uint8

// MarshalText returns the name of m. It fails with ErrModeValue for a value
// outside the variants.
func (m Mode) MarshalText() ([]byte, error)

// UnmarshalText sets m to the variant that text names. It fails with
// ErrModeText for any other text.
func (m *Mode) UnmarshalText(text []byte) error

// StrictKanon marks Mode as a kanon.Strict type.
func (Mode) StrictKanon() {}
```

The generator finds the method in the method set of the pointer to the type, as it finds
`ExactKanon`. A type can declare both markers, and the generated code then treats it as an Exact
type. The generation fails for a type that declares `StrictKanon` and is a struct with a kanon
codec, a `kanon.Validator` or a type without the methods of a family. For a field `T` of a struct
`A`, whose type `T` declares the marker and has no such methods, the generator reports:

```text
kanon: a.go:8:2: A.T: example.com/m.T declares StrictKanon, and does not encode itself through binary, gob or text methods
```

### The generated code

The canonical decode of a field of a bitwise opaque type sets the field to its zero value, calls
the decode method, and checks its error and the presence rule in one statement. `wire.ExactError`
returns the error of `UnmarshalError` at the offset of the value for an error of the method. It
returns the error of `AbsentError` at the offset of the tag otherwise. A type that is neither
Exact nor strict encodes the value again after the statement:

```go
m.Count = 0
if err := m.Count.UnmarshalText(data[i : i+int(l)]); err != nil || m.Count == 0 {
	return wire.ExactError(err, "Scalars.Count", 30, off+at, off+i)
}
{
	var scratch [128]byte
	enc, err := m.Count.AppendText(scratch[:0])
	if err != nil || !bytes.Equal(enc, data[i:i+int(l)]) {
		return wire.EncodingError("Scalars.Count", 30, off+i)
	}
}
i += int(l)
```

A field of a strict type ends after the statement:

```go
m.Mode = 0
if err := m.Mode.UnmarshalText(data[i : i+int(l)]); err != nil || m.Mode == 0 {
	return wire.ExactError(err, "Scalars.Mode", 40, off+at, off+i)
}
i += int(l)
```

The encoding writes an element, a map key, a map value, the target of a pointer, a union member
and the value of an interface whatever its value, so the presence rule applies to fields alone. In
the other positions the decode returns the error of the decode method, and it does not encode a
value of a strict type again:

```go
x[last] = 0
if err := x[last].UnmarshalText(data[i : i+int(l)]); err != nil {
	return wire.UnmarshalError(err, loc, num, off+i)
}
i += int(l)
```

The order of the checks at a field does not change. The decode reports an error of the decode
method first, the presence rule second, and the bytes of the second encode last. A map with keys
of a strict type still orders its keys by their encodings, which the compare function of the key
type writes. The stream decoder of a canonical type decodes the fields that do not stream through
the same code.

Generated files declare generator version 5, because the generated code calls only functions that
a runtime of version 5 declares. The marker is a method of the type, and the runtime declares no
function for it.

### The conformance suite

```go
package kanontest

// StrictChecks returns the check of T, a type that declares kanon.Strict:
// for every input that the decode method of T accepts, the encode method of
// its family writes the input for the value that the decode method sets. The
// check decodes the encoding of every value of the value tables, every prefix
// of it, every change of one of its bytes and the encoding with one more
// byte, and a byte string of every length up to the size of T in memory.
//
// For a type that kanon does not encode, and for a type that does not encode
// itself through the methods of a family, StrictChecks returns one check,
// which fails with the reason. kanon encodes a kanon.Validator as its
// underlying type, so a Validator is such a type.
func StrictChecks[T kanon.Strict]() []Check

// RunStrict runs the checks of T that [StrictChecks] returns, each as a
// parallel subtest of t.
func RunStrict[T kanon.Strict](t *testing.T)
```

`StrictChecks` names its checks `StrictKanon/marks a type that encodes itself through the methods
of a family` and `StrictKanon/decodes only the bytes that the encode method writes for the decoded
value`. The decode check of `ExactChecks` derives the same inputs from the same values, and the
two checks share their code.

The reference decoder of a canonical Spec encodes every opaque value again, a strict one included.
The decode checks of a struct with a field of a strict type then fail for a type that breaks its
guarantee, since the generated code accepts an input that the reference decoder rejects. The
samples and the probes do not change, and neither do the golden files.

A package that declares a strict type runs the checks in a test of its own:

```go
func TestMode(t *testing.T) {
	t.Parallel()
	kanontest.RunStrict[Mode](t)
}
```

### Cost

- `kanon` gains one exported interface, and `kanontest` gains two exported functions.
- The canonical decode of a field of a bitwise opaque type loses one statement: the return of
  `AbsentError`.
- A field of a strict type also loses the second encode: a block with the call of the encode
  method and the return of `EncodingError`.

The canonical decode of the record of a string, an enumeration and a `uint32` measured these
times into a receiver that decoded it before, at `GOMAXPROCS=4`, 12 runs per side:

| Text methods of the enumeration | Without the marker | With the marker |
|---|---:|---:|
| The methods that dokimi writes, whose `MarshalText` calls `fmt.Errorf` for a value outside the variants | 22.80 ns, 2 allocations | 10.93 ns, 1 allocation |
| A `MarshalText` that the compiler inlines, which returns a constant name | 6.90 ns, 0 allocations | 5.73 ns, 0 allocations |

## Alternatives considered

### Declaring kanon.Exact

The enumeration declares `ExactKanon`, adds `AppendText` and `SizeKanon`, and a canonical decode
treats its fields as Exact fields.

**Why not:** an Exact field encodes without an error path, and `wire.MustExact` panics for an
append method that fails for a field. An enumeration fails to encode a value outside its variants.
Under Exact, the encode of a struct with `Mode(9)` panics where its caller expects an error.
`SizeKanon` of a value outside the variants has no length to return either.

### A guarantee that the decode method never sets the zero value

The marker states a second property: the decode method never sets the zero value. The generated
code then leaves out the presence check of a field of the type, as well as the second encode.

**Why not:** the single statement of the decode error and the presence rule removes the separate
presence check of every bitwise opaque field, without a declaration. The condition of the rule is
part of the statement and runs with it. The property is also false for an enumeration with a
variant at its zero value, such as an enumeration of strings whose empty string is a variant. A
type that declares the property falsely gets a field that decodes to the zero value, which the
encoder never writes.

### The second encode inside the statement of the decode method

A function of `wire` encodes the value again and compares the bytes. The generated code calls it
in the condition of the single statement, after the presence rule. No type declares anything, and
every statement can run.

**Why not:** the second encode still runs on every decode, with the allocation of an encode method
that returns a new slice. For a strict type its condition is never true, and no test can tell it
from `false`. The function also needs a type parameter for each family of methods, and its call
of the method goes through the dictionary of the type parameter.

### A directive flag that names strict types

The directive of the struct lists the strict types of its fields, as `-strict=Mode`, and no type
declares a method.

**Why not:** the guarantee is a property of the type, not of a struct that contains it. Every
directive with a field of the type repeats its name, and a type of another package needs a
qualified name in the flag. The suite has no declaration to check in the package of the type, so
no check fails for a flag that lists a type that breaks the guarantee.

### The method set of an enumeration

The generator treats a type with `String`, `IsValid` and text methods as strict, since a generator
of enumerations writes these methods together.

**Why not:** a method set does not state a guarantee. `Count` has text methods and accepts `007`.
A type of another generator can have the same methods and parse names without regard to case. The
generator would also follow the conventions of one generator of enumerations.

## Drawbacks

- The public API grows by one interface in `kanon` and two functions in `kanontest`. Each strict
  type adds a marker method and a test that runs `RunStrict`.
- A type that declares the marker and breaks the guarantee makes a canonical decode accept a
  second encoding of a value. `RunStrict` checks the inputs that it builds, not every byte string.
- The single statement changes the generated code of every canonical field of a bitwise opaque
  type. A consumer regenerates its code files after the upgrade.
- `wire.ExactError` returns the errors of fields of types that are not Exact. Its name refers to
  one of the kinds of types that it serves.
- A type can declare both markers. `StrictKanon` then has no effect, and the generation does not
  report it.

## Unresolved and future work

- A declaration that the decode method of a type never sets the zero value is not proposed here.
- A generator of enumerations that declares `StrictKanon` on the types that it writes is a change
  of that generator, which this proposal does not make.

## References

| What | Where |
|---|---|
| Go's `encoding/base64`, `Encoding.Strict`: strict decoding requires the zero padding bits of RFC 4648, at Go 1.27.3 | `go doc encoding/base64 Encoding.Strict` |
| RFC 4648, section 3.5, Canonical Encoding: decoders may reject an encoding whose pad bits are not zero | https://www.rfc-editor.org/rfc/rfc4648#section-3.5 |
| dokimi's enumeration `Status`, whose text methods write and parse the names of its variants, and whose zero value is not a variant | https://github.com/dokimasia/dokimi/blob/main/corpus/go/enum/status/iface_enum.go |
| dokimi's enumeration `Region`, whose zero value is the variant `Unset` | https://github.com/dokimasia/dokimi/blob/main/corpus/go/enum/zerolast/iface.go |
| The decode of the record with and without the marker, run 2026-11-02 | Two packages of one module, which differ in `StrictKanon` alone, generated with `-canonical` by the generator of this design. Each test binary ran `-test.bench '^BenchmarkDecode$' -test.benchtime=200ms -test.count=3` in four alternating rounds, compared with `benchstat` |
