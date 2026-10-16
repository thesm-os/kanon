---
rfc: 0009
title: A bound on the length of a string or a byte slice field
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-10-16
updated: 2026-10-16
discussion: https://github.com/thesm-os/kanon/issues/10
supersedes: none
superseded-by: none
produces-adr: tbd
---

<!--
  ~ Copyright ThesmOS B.V. 2026
  ~ SPDX-License-Identifier: Apache-2.0
-->

# RFC-0009: A bound on the length of a string or a byte slice field

## Summary

The tag option `max=N` also bounds the length of a string or a byte slice field, in bytes. The
encode rejects a longer value. The decode, the merge, the view method, `IndexKanon` and the
stream decoder fail at the byte past the bound with a `*kanon.DecodeError` that wraps
`kanon.ErrMax`. They fail before they copy the bytes or take the substring. The messages of
`kanon.ErrMax` and of `wire.MaxError` describe a value that is longer than its bound, so one
text fits a string, a byte slice, a slice and a map. Generated files declare generator
version 5.

## Motivation

For a slice or a map field, the tag option `max` bounds the elements. It leaves strings and byte
slices out, because their decode takes no more memory than their encoding. That reason is still
true. It does not cover a schema that limits the length of a field, such as an opaque event
type of at most 64 bytes. A request for such a field reverses the verdict that left strings and
byte slices out.

Without the option, the generated decode copies a byte slice into the receiver before the
caller can check its length. For a struct with a byte slice, a string and a `uint64`, whose
byte slice and string have 60,000 bytes each, the decode allocates:

| Decode | Allocated | Capacity of the byte slice afterwards |
|---|---:|---:|
| `DecodeKanon` with a slab, into a zero receiver | 65,536 bytes in 1 object | 65,536 |
| `DecodeKanon` with a slab, into a receiver whose byte slice has a capacity of 64 | 65,552 bytes in 2 objects | 65,536 |
| `UnmarshalBinary` into a zero receiver | 188,416 bytes in 2 objects | 65,536 |

The copy happens before a caller can refuse the value. A receiver that a pool reuses, such as
the call state of a server, then keeps a buffer of 64 KiB per receiver. The decode of the string
does not copy it with a slab. The schema has no place for its limit, so the caller checks the
string after the decode.

The bound belongs in the schema, beside the number of the field. The limit is then in one
place. The generated code checks it before the copy, and the conformance suite checks the
generated code against it.

## Detailed design

### Terms

- A **bounded field** is a field with the tag option `max=N`, and **N** is its **bound**.
- The **length** of a field is `len` of its value: the bytes of a string or a byte slice, the
  elements of a slice, and the entries of a map.
- The **byte past the bound** of a string or a byte slice of more than N bytes is its byte at
  index N.

### Components

| Component | Change |
|---|---|
| The generator | The tag option `max=N` on a field whose type is a string or a byte slice, named or not. The decode, the view method and `IndexKanon` of such a field check its length. The generator version is 5 |
| `kanon` | The message of `ErrMax` is "kanon: value longer than the max of its field", and `MaxVersion` is 5 |
| `kanon/wire` | The detail of `MaxError` is "longer than the max of N". `StreamField.Max` bounds a streamed string or byte slice too, and `Stream.Open` checks it. `Stream.Init` allocates the counts of the elements only for a streamed slice with a `Max` |
| `kanontest` | `Field.Max` on a string or a byte slice field. The samples, the reference encoder, the reference decoder, the reference view and the reference index apply it |
| `Message`, `Options`, `frame`, `batch` and `kanon inspect` | No change. None of them uses the tags of a struct type |

### Invariants

For a bounded string or byte slice field of a struct type `T`, with the bound N:

1. **Encode.** `EncodeKanon`, `AppendBinary` and `MarshalBinary` fail with a
   `*kanon.EncodeError` that wraps `kanon.ErrMax` for a value of `T` whose field is longer
   than N.
2. **Decode.** `DecodeKanon`, `MergeKanon` and `UnmarshalBinary` fail with a
   `*kanon.DecodeError` that wraps `kanon.ErrMax` at an occurrence of the field that is longer
   than N, at the offset of its byte past the bound. They check the length after the checks of
   the varint of the length, and before they copy the bytes or take the substring. Each
   occurrence replaces the value of the field, so the decode checks each occurrence on its own.
3. **Acceptance.** The decode accepts exactly the encodings that it accepts without the option
   and whose every occurrence of the field has N bytes at most. A decode without `-canonical`
   accepts a field that occurs twice, and fails when an occurrence before the last is longer
   than N. It fails the same way for an integer outside its range at any occurrence.
4. **Memory.** The decode copies N bytes at most into a bounded byte slice, and takes a
   substring of N bytes at most for a bounded string.
5. **Views.** The view method of the field fails as the decode fails, at the last occurrence of
   the field. `IndexKanon` fails at every occurrence, so the method of the index that reads the
   field returns the value alone.
6. **Stream.** The stream decoder of `T` returns the error of `DecodeKanon` for the field, at
   the same offset. For a streamed field, `Next` fails before it returns the number of the
   field, so the caller does not read a byte of it.

### The tag option

```go
//go:generate go tool kanon -type=Note -views

// Note is a label, a payload and a sequence number.
type Note struct {
	Label string `kanon:"1,max=2"`
	Body  []byte `kanon:"2,stream,max=3"`
	Seq   uint64 `kanon:"3"`
}
```

The option `max=N` takes a decimal N from 1 to 2147483647. It applies to a field whose type is a
string, a byte slice, a slice or a map, named or not. It combines with a field number,
`union=Name` and `stream`. On a `kanon.Validator` under `-validate`, the decode checks the
length before it calls `ValidateKanon`, and so does the encode.

The generation fails for the option on any other field, with these verdicts:

- An array, a byte array included, has its length in its type.
- No consumer has a pointer to a string, a byte slice, a slice or a map. A consumer with a
  bounded `*string` would reverse this verdict.
- The decode method of a type that encodes itself decodes the whole value.

The bound applies to the field itself. It does not bound the strings in the elements of a
`[]string` or in the values of a `map[K]string`. On those fields it bounds the elements and the
entries. A bound per element needs a second option, and no consumer has asked for one. A
consumer with a `[]string` whose strings need a bound would reverse this verdict.

### The generated code

The decode of the field checks the length in the case of the field, after the check of the
length and before the copy. A comment marks each added line:

```go
	case 2:
		if tag != 2<<3|wire.Bytes {
			return wire.FormatError(tag, wire.Bytes, "Note.Body", off+at)
		}
		l, n := wire.Uvarint(data[i:])
		if n <= 0 || uint64(len(data)-i-n) < l {
			return wire.ReadError(n, "Note.Body", 2, off+i)
		}
		i += n
		if l > 3 { // added
			return wire.MaxError("Note.Body", 2, off+i+3, 3) // added
		} // added
		m.Body = append(m.Body[:0], data[i:i+int(l)]...)
		i += int(l)
```

A string takes the substring `slab[off+i : off+i+int(l)]` after the same check. The view method
checks the last occurrence:

```go
func (v NoteView) Body() ([]byte, error) {
	i, err := wire.Find(v, 2<<3|wire.Bytes, "Note.Body")
	if err != nil || i < 0 {
		return nil, err
	}
	l, n := wire.Uvarint(v[i:])
	if l > 3 { // added
		return nil, wire.MaxError("Note.Body", 2, i+n+3, 3) // added
	} // added
	return []byte(v[i+n : i+n+int(l)]), nil
}
```

The scan of `IndexKanon` checks every occurrence, after `wire.Skip` has checked that the value
ends within the view:

```go
		case 2:
			if tag != 2<<3|wire.Bytes {
				return NoteIndex{}, wire.FormatError(tag, wire.Bytes, "Note.Body", at)
			}
			l, n := wire.Uvarint(v[i:]) // added
			if l > 3 { // added
				return NoteIndex{}, wire.MaxError("Note.Body", 2, i+n+3, 3) // added
			} // added
			ix.at[1] = i + 1
```

The method of the index reads the field without a check, and returns the value alone, as for a
field without the option. The encode of the field checks the length inside the block that
writes a present field, as it checks a bounded slice:

```go
	if len(m.Body) > 3 {
		return 0, wire.MarshalError(kanon.ErrMax, "Note.Body", 2)
	}
```

The schema of the stream decoder contains the bound of the streamed field:

```go
var _note_streamNote = wire.StreamSchema{
	Loc: "Note",
	Fields: []wire.StreamField{
		{Tag: 2<<3 | wire.Bytes, Loc: "Note.Body", Max: 3},
	},
}
```

### The stream decoder

`Stream.Open` reads the length of the streamed field after the run before it. For a string or a
byte slice whose `StreamField` has a `Max`, it fails with the error of `wire.MaxError` at the
offset of the byte past the bound when the length is above `Max`. It makes the check after the
checks of the varint of the length, so `Next` returns the error before it returns the field. The
caller then does not read a byte of the value.

A bounded string or byte slice without the tag option `stream` decodes in the run of the fields
that the stream decoder decodes into the receiver, with the code of `DecodeKanon`. A field
longer than the `Buffer` of the stream fails with `kanon.ErrLimit` before that decode. Every
field that does not stream fails this way.

### Errors and failure handling

| Failure | Where | Error | Offset |
|---|---|---|---|
| A string or a byte slice longer than N in a value to encode | `EncodeKanon`, `AppendBinary`, `MarshalBinary` | A `*kanon.EncodeError` that wraps `kanon.ErrMax`, for the field | None |
| An occurrence of the field longer than N | `DecodeKanon`, `MergeKanon`, `UnmarshalBinary`, and `Next` of a stream decoder for a field that does not stream | A `*kanon.DecodeError` that wraps `kanon.ErrMax`, for the field | The byte past the bound |
| The last occurrence of the field longer than N | The view method of the field | The same error | The same offset |
| Any occurrence of the field longer than N | `IndexKanon` | The same error | The same offset |
| A streamed field longer than N | `Next` of the stream decoder | The same error | The same offset |
| The option on a field that it does not apply to, a second option, or an N outside 1 to 2147483647 | The generator | An error that names the field and its position | None |

The text of a decode error contains the bound. For the second test vector it is
`kanon: Note.Label (field 1) at offset 4: longer than the max of 2`. The text of an encode error
contains the cause. For `{Label: "abc"}` it is
`kanon: Note.Label (field 1): value longer than the max of its field`.

### The runtime

```go
package kanon

// ErrMax is a cause of an [EncodeError] and of a [DecodeError]: a value that
// is longer than the tag option max of its field allows. A string and a byte
// slice are longer with more bytes, and a slice and a map with more elements.
// The encode of a value whose field is longer fails. The decode fails at the
// byte, the element or the entry that would take the field past the bound,
// before it copies or decodes it.
var ErrMax = errors.New("kanon: value longer than the max of its field")

const (
	MinVersion = 1
	MaxVersion = 5
)
```

```go
package wire

// MaxError returns the error of a decode for a field that is longer than the
// bound of its tag option max. loc and num locate the field, and bound is the
// bound. off is the offset of the byte at index bound of a string or a byte
// slice, or of the element or the map entry at index bound of a slice or a
// map. The error wraps kanon.ErrMax, and its detail contains bound.
func MaxError(loc string, num, off, bound int) error

type StreamField struct {
	// The fields Loc, Elem and Tag do not change.

	// Max is the bound of the tag option max: the most elements that a slice
	// takes over every occurrence of the field, and the most bytes of each
	// occurrence of a byte slice or a string. It is 0 for a field without the
	// option.
	Max int
}
```

### The conformance suite

The generated test file sets `Max` in the `Field` of a bounded string or byte slice:

```go
var kanonSpecNote = kanontest.Spec[Note]{
	Fields: []kanontest.Field{
		{Name: "Label", Number: 1, Max: 2},
		{Name: "Body", Number: 2, Stream: true, Max: 3},
		{Name: "Seq", Number: 3},
	},
	View: NoteView(nil),
	Stream: func(r io.Reader, size int64, m *Note, opts kanon.StreamOptions) kanontest.Stream[Note] {
		return kanonStreamNote{NewNoteStream(r, size, m, opts)}
	},
}
```

The suite fails a `Spec` that sets a `Max` below 0, or a `Max` on a field that is not a string,
a byte slice, a slice or a map. The checks change in these ways:

- **The reference decoder** checks the length of each occurrence after the check of the varint
  of the length, and fails at the byte past the bound with the error of the decode.
- **The reference view** checks the last occurrence, and **the reference index** every
  occurrence, as the generated code checks them.
- **The reference encoder** fails a value with the error of the encode. It already compares the
  length of every bounded field.
- **The samples** have N bytes at most in a bounded string or byte slice: the builder cuts the
  value that the source supplies to N bytes.
- **A bounded field counts as one value that can fail.** The sample of a table entry that fails
  at it sets the field at N + 1 bytes.
- **The bound samples** set the field alone at N bytes and at N + 1 bytes. The checks of every
  sample run on the first. The reference encoding of the second is the probe past the bound,
  which the decode checks decode.
- **The property** cuts the random bytes of a bounded field to N.

### Test vectors

The vectors decode with the `Note` type shown earlier, a type without `-canonical` with the
bounds 2 and 3:

| Input | Size | Result | Offset |
|---|---:|---|---:|
| `0a 02 61 62` | 4 | `{Label: "ab"}` | |
| `0a 03 61 62 63` | 5 | `kanon.ErrMax` for `Note.Label` | 4 |
| `0a 05 61 62 63` | 5 | `io.ErrUnexpectedEOF` for `Note.Label`, since the length runs past the input | 1 |
| `12 03 01 02 03` | 5 | `{Body: {1, 2, 3}}` | |
| `12 04 01 02 03 04` | 6 | `kanon.ErrMax` for `Note.Body` | 5 |
| `0a 03 61 62 63 0a 01 61` | 8 | `kanon.ErrMax` for `Note.Label` | 4 |

Without the option, the decode accepts the second, the fifth and the sixth input. The view
method `Label` of the sixth input returns `a`, the last occurrence, and `IndexKanon` fails at
offset 4. The stream decoder of `Note` fails in `Next` at offset 5 for the fifth input, before
it returns `NoteFieldBody`. The encode of `{Label: "abc"}` fails with a `*kanon.EncodeError`
that wraps `kanon.ErrMax` for `Note.Label`.

### Versions

The stream decoder of a type with a bounded streamed string or byte slice depends on the check
of `Stream.Open`, which a runtime with a `MaxVersion` of 4 does not make. Generated files
declare generator version 5, so such a file fails to compile against that runtime. A file of
version 1 to 4 compiles against a runtime with a `MaxVersion` of 5, and no file of those versions
has a bound on a string or a byte slice.

### Schema changes

The option does not change the encoding or a field number. A decode with the option accepts a
subset of the encodings that it accepts without it. A writer and a reader with different bounds
disagree about the lengths between the two bounds:

- Adding the option, or lowering N, makes a reader reject what an older writer writes. Upgrade
  every writer first.
- Removing the option, or raising N, makes a writer write what an older reader rejects. Upgrade
  every reader first.

### Cost

- The decode compares the length of each occurrence of a bounded field with N once. The view
  method compares once, and the scan of `IndexKanon` reads the varint of the length a second
  time for each occurrence of a bounded field, after `wire.Skip` has read it.
- The encode compares the length of each bounded field once.
- The generated code of a bounded field gains three lines in the decode, three in the view
  method, four in the scan of `IndexKanon` and three in the encode.
- Every generated file declares version 5, so every file that the generator writes changes
  beyond the code of its bounds.
- `kanontest` builds, per bounded string or byte slice, a sample of N bytes and a sample of
  N + 1 bytes, and per table entry a sample that fails at the field with N + 1 bytes.

The encoding of the motivation decodes into a type with `max=64` on its byte slice and its
string. Each decode fails at the byte past the bound of the byte slice, at offset 68:

| Decode | Allocated | Capacity of the byte slice afterwards |
|---|---:|---:|
| `DecodeKanon` with a slab, into a zero receiver | 112 bytes in 2 objects | 0 |
| `DecodeKanon` with a slab, into a receiver whose byte slice has a capacity of 64 | 112 bytes in 2 objects | 64 |
| `UnmarshalBinary` into a zero receiver | 122,992 bytes in 3 objects | 0 |

The 112 bytes are the `*kanon.DecodeError` and the text of its detail. `UnmarshalBinary` also
copies the input of 120,010 bytes into its slab before the decode, as it does for every input.

## Alternatives considered

### A check by the caller after the decode

The caller compares the length of the field with its limit after `DecodeKanon` returns.

**Why not:** the decode has copied the bytes by then, and a receiver keeps the capacity of the
longest value that it decoded. A caller that decodes into pooled receivers keeps one such buffer
per receiver.

### A named type that decodes itself

The field takes a named type whose `UnmarshalBinary` checks the length before it copies the
bytes.

**Why not:** every bounded field then needs a type of its own, and the bound is in code instead
of beside the number of the field. A view method and `IndexKanon` decode such a type with its
own method, so the bound costs a call of the method per read.

### A check of the raw field with the functions of kanon/wire

The caller finds the field with `wire.Find` and compares its length before the decode.

**Why not:** the caller restates the rules of the encoding outside kanon, and a difference
between the two copies is a parser differential.

### A limit in kanon.Options

The decode options take one limit for the length of every string and byte slice of a decode.

**Why not:** one limit cannot be 64 bytes for one field and 16 MiB for another field of the same
decode. The `MaxSize` of a frame reader and the `Buffer` of a stream decoder already bound
the bytes of a whole encoding.

### A check of the last occurrence alone

The decode accepts an encoding whose last occurrence of the field has N bytes at most, as the
value that it decodes to has.

**Why not:** the decode copies each occurrence before the next one replaces it, so an earlier
occurrence of 60,000 bytes costs the copy that the bound exists to prevent. The decode also
checks the range of an integer at every occurrence.

### The error at the offset of the length

The decode fails at the offset of the varint of the length.

**Why not:** the bound of a slice fails at the element past the bound, and a byte of a string or
a byte slice is its element. One rule for the four kinds gives one offset in the decode, the
view, the index and the stream decoder. Each of them computes it from the offset of the value
and N.

### A second error function for bytes

`wire.MaxError` keeps the detail "more elements than the max of N" for slices and maps, and a new
function returns the detail "more bytes than the max of N" for strings and byte slices.

**Why not:** it adds a function to the runtime for the text of a message. The message of
`kanon.ErrMax` appears in every encode error of a bound and has to fit the four kinds anyway.
`len` gives each of them a length. "Longer than the max" fits all four.

### The check in the generated code of the stream decoder

`Next` compares `Len` with the bound after `Open` returns the streamed field. The generator
version is then still 4.

**Why not:** the error needs the offset of the byte past the bound, and the generated code
cannot read the offset of the value from a `wire.Stream`. Such an accessor is new runtime API.
It needs version 5 too.

## Drawbacks

- The detail of `wire.MaxError` and the message of `kanon.ErrMax` change for slices and maps
  too. Code that compares the text of these errors breaks, and code that calls `errors.Is` with
  `kanon.ErrMax` does not.
- A file of generator version 5 needs a runtime with a `MaxVersion` of 5.
- A `kanon.Validator` whose value is longer than N fails with `kanon.ErrMax`, before its
  `ValidateKanon` runs.
- `IndexKanon` fails for an encoding whose earlier occurrence of the field is longer than N,
  where the view method of the field returns the last occurrence.
- The scan of `IndexKanon` reads the varint of the length of a bounded field twice.
- `kanontest` builds at least ten values of N or N + 1 bytes per bounded string or byte slice:
  the two bound samples and one sample per table entry that fails at the field. At N =
  16,777,216 they take at least 160 MiB, and each check of every sample encodes and decodes the
  16 MiB of the bound sample.

## Unresolved and future work

- A bound on the strings and the byte slices in the elements of a field, such as each string of
  a `[]string`, is not proposed here.
- A bound on a pointer to a string or a byte slice is not proposed here.

## References

| What | Where |
|---|---|
| The request, with the field of 64 bytes that needs the bound | <https://github.com/thesm-os/kanon/issues/10> |
| The measurement of the copy | a probe with a struct of a byte slice, a string and a `uint64`, kanon at 60c3a16 and Go 1.27.2 |
| The measurement with the bound | the same probe with `max=64` on both fields, kanon at 60c3a16 with this design and Go 1.27.2 |
| The length of a string in Go, its number of bytes | <https://go.dev/ref/spec#String_types> |
