---
rfc: 0008
title: A bound on the elements of a slice or a map field
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Draft
created: 2026-10-08
updated: 2026-10-08
discussion: https://github.com/thesm-os/kanon/issues/5
supersedes: none
superseded-by: none
produces-adr: tbd
---

# RFC-0008: A bound on the elements of a slice or a map field

## Summary

The tag option `max=N` bounds the number of elements of a slice field, or of entries of a map
field, at N. The encode rejects a value with more elements. The decode, the merge and the stream
decoder fail at the element or the entry that would take the field past N, before they decode
the element or the value of the entry. Their error is a `*kanon.DecodeError` that wraps the new
sentinel `kanon.ErrMax`. The first allocation of a bounded slice makes room for N elements at
most. The first allocation of every slice, bounded or not, also takes at most 10 MiB, and the
slice grows by append after that.

## Motivation

A generated decode of a nil slice makes the slice with room for every element that it counts
in the bytes of the field, before it decodes the first element. `wire.CountValues` counts the
length-prefixed values, and an empty value takes 1 byte. Every byte of the input can then count
as one element, and the first allocation takes the size of the element type for each of them:

| Element type | Input per counted element | Memory per counted element |
|---|---:|---:|
| `[]byte` | 1 byte | 24 bytes |
| `string` | 1 byte | 16 bytes |
| `int64`, counted with `wire.CountVarints` | 1 byte | 8 bytes |
| `crypto.Digest` of core, which decodes itself | 1 byte | 65 bytes |
| A struct type `T` | 1 byte | the size of `T` |

The author of the request measured a struct with the fields `Digests []crypto.Digest` and
`Items [][]byte`. Each input is one field whose bytes are all 0, so each byte is an element of
length 0. The decode reads from a slab and does not copy the input. The measurement reproduces
to the byte with kanon at 47ccacd and core at 7339784:

| Field | Input | Elements decoded | Allocated | Result |
|---|---:|---:|---:|---|
| `Digests` | 1,048,580 bytes | 0 | 68,157,520 bytes | the error of the first digest |
| `Digests` | 4,194,309 bytes | 0 | 272,629,856 bytes | the error of the first digest |
| `Items` | 1,048,580 bytes | 1,048,576 | 25,165,840 bytes | decoded |
| `Items` | 4,194,309 bytes | 4,194,304 | 100,663,296 bytes | decoded |

The fields differ in kind:

- The decode of `Digests` allocates for every counted value before the first element fails. A
  first allocation of at most 10 MiB caps that cost for every slice, without a change of the
  schema.
- The decode of `Items` accepts a valid encoding. Its memory grows with the input whatever rule
  the caller applies afterwards. Only a bound in the schema rejects the encoding before the
  decode allocates for it.

The ledger's verifier reads content indexes and boundary pages from an operator that it does not
trust. An index has one digest for each 64 KiB chunk of an object of at most 4 GiB, 65,536
digests at most, and a page has at most 1,024 items. In the request, an index of 4,456,501
bytes, the longest that the verifier reads, allocated 294,143,432 bytes before its first digest
failed. A page of 1 MiB allocated 69,207,440 bytes. The verifier runs up to 64 checks at once.

The bound belongs in the schema, beside the number of the field. The request needs bounds of
65,536 on two fields of an index, and bounds of 1,024 and 32 on two fields of a page, so one
limit for a whole decode cannot state them. A check by the caller after the decode comes after
the allocation.

## Detailed design

### Terms

- A **bounded field** is a field with the tag option `max=N`, and **N** is its **bound**.
- An **element** is an element of a slice or an entry of a map.
- The **length** of a field is `len` of its value: the number of its elements.
- The **projection** of a map key is its encoding, with every float component of -0.0 written as
  +0.0.

### Components

| Component | Change |
|---|---|
| The generator | The tag option `max=N`. The read function of a slice type or a map type that a bounded field calls takes the bound as a parameter, and its other calls pass `math.MaxInt`. The encode of a bounded field checks its length. The first allocation of a slice takes its capacity from `wire.SliceCap`. The test file sets `Max` in the `Field` of a bounded field. The generator version is 4 |
| `kanon` | `ErrMax`, the causes that the docblocks of `DecodeError` and `EncodeError` list, and a `MaxVersion` of 4 |
| `kanon/wire` | `MaxError`, `SliceCap` and `StreamField.Max`. A `Stream` counts the elements of each bounded streamed slice |
| `kanontest` | `Field.Max`, the bound in the reference encoder and the reference decoder, and the samples and the probes of the bound |
| Views and their indexes | No change. A view has no method for a slice field or a map field |
| `Message`, `Options`, `frame`, `batch` and `kanon inspect` | No change. None of them uses the tags of a struct type |

### Invariants

For a bounded field of a struct type `T`, with the bound N:

1. **Encode.** `EncodeKanon` and `AppendBinary` fail with a `*kanon.EncodeError` that wraps
   `kanon.ErrMax` for a value of `T` whose field has a length above N. Every encoding that the
   encode writes has at most N elements in the field.
2. **Decode.** `DecodeKanon`, `MergeKanon` and `UnmarshalBinary` fail with a `*kanon.DecodeError`
   that wraps `kanon.ErrMax` at the element that would take the length of the field past N.
   They fail before they decode that element, and for a map entry before they decode its value.
   The length counts the elements that the field has when the element arrives: the elements of
   an earlier occurrence of the field in the encoding, and in a merge the elements of the
   receiver.
3. **Acceptance.** The decode accepts exactly the encodings that it accepts without the option
   and that give the field a length of N at most. A map whose keys of one projection can differ
   under ==, such as a map with pointer keys, is the one exception. A decode without `-canonical`
   and a merge of such a map also reject an input whose keys pass N before the decode deletes
   the entries of a repeated projection. The encode never writes two keys of one projection.
4. **Memory.** The first allocation of a bounded slice makes room for N elements at most, and the
   decode appends N elements at most. The first allocation of any slice takes at most 10 MiB,
   or one element when one element takes more.
5. **Stream.** The stream decoder of `T` returns the error of `DecodeKanon` for a streamed
   bounded slice, at the same offset.

### The tag option

```go
//go:generate go tool kanon -type=Page -canonical

// Page is a list of items and a map of named counts.
type Page struct {
	Items [][]byte          `kanon:"1,max=2"`
	Meta  map[string]uint64 `kanon:"2,max=1"`
}
```

The option `max=N` takes a decimal N from 1 to 2147483647, the largest `int` on a platform with
a 32-bit `int`, so that the generated code compiles on every platform. It applies to a field
whose type is a slice or a map, named or not. It combines with every other word of a tag: a
number, `fixed`, `union=Name`, `types=T1|T2` and `stream`. The word `unknown` takes no other
word.

The generation fails for the option on any other field, with these verdicts:

- The decode of a byte slice or a string takes no more memory than its encoding, so it amplifies
  no input. A consumer that needs the schema to state the length of a string, such as an origin
  of 255 bytes, would reverse this verdict.
- An array has its length in its type.
- No consumer has a pointer to a slice or a map. A consumer with a bounded `*[]T` would reverse
  this verdict.
- The decode method of a type that encodes itself decodes the whole value.

The generation also fails for a second `max` in one tag, and for an N that is not a decimal from
1 to 2147483647.

The bound applies to the field itself, and not to the elements of a `[][]T` or the values of a
`map[K][]V`.

### The generated code

The read function of a slice type or a map type that a bounded field of the code file calls
takes the bound as a parameter. Fields of one slice type with different bounds then share one
read function. A call without a bound passes `math.MaxInt`: a field without the option, and a
slice or a map in an element or a value. The generator writes the parameter and the check only
into a read function that a bounded field calls, so the code of a type without a bounded field
has no branch that its tests cannot run.

In the read function of `[][]byte`, a comment marks each changed line:

```go
// _page_readSliceSliceByte appends the elements that data encodes, a [][]byte
// without its length, to *dst. data starts at offset off of slab, and the
// decode enters depth levels below the value at most. It fails at an element
// that would take the slice past bound elements. loc and num locate the field
// in errors.
func _page_readSliceSliceByte(dst *[][]byte, data []byte, slab string, off, depth, bound int, loc string, num int) error { // changed
	if depth < 0 {
		return wire.DepthError(loc, num, off)
	}
	x := *dst
	if x == nil && len(data) > 0 {
		x = make([][]byte, 0, min(wire.SliceCap[[]byte](wire.CountValues(data)), bound)) // changed
	}
	for i := 0; i < len(data); {
		if len(x) >= bound { // added
			return wire.MaxError(loc, num, off+i, bound) // added
		} // added
		if len(x) < cap(x) {
			x = x[:len(x)+1]
		} else {
			x = append(x, nil)
		}
		last := len(x) - 1
		l, n := wire.Uvarint(data[i:])
		if n <= 0 || uint64(len(data)-i-n) < l {
			return wire.ReadError(n, loc, num, off+i)
		}
		if n > 1 && data[i+n-1] == 0 {
			return wire.LongFormError(n, loc, num, off+i)
		}
		i += n
		x[last] = append(x[last][:0], data[i:i+int(l)]...)
		i += int(l)
	}
	*dst = x
	return nil
}
```

The check comes first in the loop, before the decode reads the length of the element, so a
malformed element past the bound fails with `kanon.ErrMax`. The check compares with `>=`, so a
merge into a receiver that has more than N elements already fails at the first element that it
appends.

A map read function takes the bound after `depth` too. It checks after the key of an entry, and
in a canonical decode after the order of the key, since a key that the map has already does not
add an entry:

```go
		if len(x) >= bound { // added
			if _, ok := x[mk]; !ok { // added
				return wire.MaxError(loc, num, off+at, bound) // added
			} // added
		} // added
```

The offset `at` is the offset of the key of the entry, which every map read function with a
bound declares. A canonical decode rejects a key that does not ascend, so its map has no key
twice, and the check fails at the entry past the bound.

A map whose keys of one projection can differ under ==, such as a map with pointer keys, keeps
one entry per projection. Its read function deletes the earlier entries of a repeated
projection after the last entry. The check runs before that deletion and counts every key that
the map does not have under ==. So a decode without `-canonical` and a merge of such a map also
reject an input whose keys pass N before the deletion.

The decode of a field passes its bound to the read function:

```go
		if err := _page_readSliceSliceByte(&m.Items, data[i:i+int(l)], slab, off+i, depth-1, 2, "Page.Items", 1); err != nil {
			return seen, err
		}
```

The encode of a bounded field checks the length before it writes the field:

```go
	if len(m.Items) > 2 {
		return 0, wire.MarshalError(kanon.ErrMax, "Page.Items", 1)
	}
```

`SizeKanon` does not check the bound, since its contract covers a value whose encode succeeds.
The docblocks of `EncodeKanon`, `AppendBinary` and `MarshalBinary` of a type with a bounded
field state the error of the bound.

The put function of a map with integer or string keys sorts up to 16 entries in a stack array,
and a larger map in a slice that it allocates. When every field that the function writes has a
bound of 16 or less, the encode fails for a larger map before the call, so the function sorts in
the stack array alone and has no branch that a value can never take.

### The first allocation of a slice

`wire.SliceCap[T](n)` returns n when n elements of `T` take 10 MiB at most. Otherwise it returns
the number of elements of `T` that fit in 10 MiB, and 1 at least. The read function of every
slice whose elements the decode counts makes its first allocation with it. An element past that
room grows the slice by `append`. `encoding/gob` caps the first allocation of a slice and of a
map in the same way, through `internal/saferio`, at the same 10 MiB.

A map grows with its entries already, since its read function makes it without a size.

### The stream decoder

`StreamField.Max` is the bound of a streamed slice, and 0 for any other streamed field. A
`Stream` counts the elements that `Value` returns for each bounded streamed slice, across the
occurrences of the field. `Element` fails with the error of `wire.MaxError` at the offset of the
next element when the count is N, before it reads the length of the element. `Value` reads the
length through `Element`. The `Next` of a generated stream decoder reads each element that the
caller skipped with `Value` and decodes it, so every element meets the check. `Reset` sets the
counts to 0.

`Init` allocates the counts when a field of the schema has a bound, so the constructor of a
stream decoder with a bounded streamed slice allocates once more. `Reset`, `Element` and `Value`
allocate nothing for the counts.

### Errors and failure handling

| Failure | Where | Error | Offset |
|---|---|---|---|
| A field with a length above N in a value to encode | `EncodeKanon`, `AppendBinary`, `MarshalBinary` | A `*kanon.EncodeError` that wraps `kanon.ErrMax`, for the field | None |
| Input that takes a bounded field past N | `DecodeKanon`, `MergeKanon`, `UnmarshalBinary`, and the `Next` of a stream decoder for a field that does not stream | A `*kanon.DecodeError` that wraps `kanon.ErrMax`, for the field | The offset of the element past the bound, or of the key of the entry |
| A streamed slice with more than N elements | `Element`, the decode method of the slice, or `Next` for a skipped element | The same error | The same offset |
| The option on a field that it does not apply to, a second option, or an N outside 1 to 2147483647 | The generator | An error that names the field and its position | None |

The detail of the decode error states the bound, as in
`kanon: Page.Items (field 1) at offset 6: more elements than the max of 2`.

A merge into a receiver whose field already has more than N elements fails at the first
element that it appends. A merge that appends no element to the field does not check it.

### The runtime

```go
package kanon

// ErrMax marks a slice or a map with more elements than the tag option max
// of its field allows. It is the cause of the DecodeError at the element past
// the bound, and of the EncodeError of a value with such a field.
var ErrMax = errors.New("kanon: more elements than the max of the field")

// The generator versions that the runtime supports, from MinVersion to
// MaxVersion.
const (
	// MinVersion is the oldest generator version whose files compile
	// against the runtime.
	MinVersion = 1
	// MaxVersion is the newest generator version whose files compile
	// against the runtime.
	MaxVersion = 4
)
```

```go
package wire

// MaxError returns the error of a decode for the element or the map entry
// at offset off that would take the slice or the map of the field that loc
// and num locate past bound elements. It wraps kanon.ErrMax, and its detail
// states bound.
func MaxError(loc string, num, off, bound int) error

// SliceCap returns the capacity of the first allocation of a slice of n
// elements of T: n when n elements take 10 MiB at most, and otherwise the
// number of elements that fit in 10 MiB, 1 at least.
func SliceCap[T any](n int) int

type StreamField struct {
	// The fields Tag, Loc and Elem do not change.

	// Max is the bound of the tag option max of a streamed slice, and 0 for
	// any other streamed field.
	Max int
}
```

### The conformance suite

The test file sets `Max` in the `Field` of a bounded field:

```go
var kanonSpecPage = kanontest.Spec[Page]{
	Fields: []kanontest.Field{
		{Name: "Items", Number: 1, Max: 2},
		{Name: "Meta", Number: 2, Max: 1},
	},
	Canonical: true,
}
```

```go
type Field struct {
	// The other fields do not change.

	// Max is the bound of the tag option max: the most elements of a slice,
	// or entries of a map, that a value of the field has, and 0 for a field
	// without the option.
	Max int
}
```

The suite fails a `Spec` that sets a `Max` below 0, or a `Max` on a field that is not a slice or
a map. The checks change in these ways:

- **The reference encoder and decoder** apply the bound as the generated code applies it. The
  encoder fails a value with the error of the encode. The decoder fails at the element past the
  bound, with the error and at the offset of the decode.
- **The samples** have at most N elements in a bounded field. A table sample has up to 2
  elements, and a wide sample has 1 element in a slice and 33 entries in a map, each lowered to
  N when N is smaller. A struct with a kanon codec in a sample takes the bound that the tag of
  its field states.
- **The bound sample** of each bounded field sets that field alone at N elements, or a map at
  every key of a key type with fewer than N values. The checks that run once on every sample run
  on it, such as the round trip, the reuse of a receiver, Reset and the stream checks. The probes
  that cut a sample at each byte, write each value of it wrong, or leave out its elements one at
  a time leave it out, so the time of its checks grows linearly with N. The golden file records
  its encoding by a digest.
- **A merge of the bound sample into its decode** fails at its first element for a slice. For a
  map, whose keys the decode has already, the merge succeeds.
- **A bounded field counts as one value that can fail.** The sample of a table entry that fails
  at it sets the field at N + 1 elements. The encode checks expect `kanon.ErrMax`, and a merge
  of a table sample into it fails at the first element that the merge appends to the field.
- **The probe past the bound** is the encoding of the bound sample with one more element at its
  end, or one more entry with a key above its keys. The decode, the merge and the stream decoder
  fail at that element with `kanon.ErrMax`. A map whose key type has N values or fewer, such as
  a `map[bool]V` with a bound of 2, has no such probe.
- **The property** generates at most N elements for a bounded field.

### Test vectors

The vectors decode with the `Page` type shown earlier, a canonical type with the bounds 2 and 1:

| Input | Size | Result | Offset |
|---|---:|---|---:|
| `0a 04 01 61 01 62` | 6 | `{Items: {"a", "b"}}` | |
| `0a 06 01 61 01 62 01 63` | 8 | `kanon.ErrMax` for `Page.Items` | 6 |
| `0a 03 00 00 00` | 5 | `kanon.ErrMax` for `Page.Items` | 4 |
| `0a 05 01 61 01 62 05` | 7 | `kanon.ErrMax` for `Page.Items`, before the length that runs past the field | 6 |
| `12 03 01 78 01` | 5 | `{Meta: {"x": 1}}` | |
| `12 06 01 78 01 01 79 02` | 8 | `kanon.ErrMax` for `Page.Meta` | 5 |

The encode of `{Items: {"a", "b", "c"}}` fails with a `*kanon.EncodeError` that wraps
`kanon.ErrMax` for `Page.Items`. Without the option, the decode accepts the second, the third
and the sixth input, and rejects the fourth at offset 6 with `io.ErrUnexpectedEOF`.

### Versions

Every read function with a bound calls `wire.MaxError`, and the first allocation of a slice
calls `wire.SliceCap`, so generated files declare generator version 4. A file of version 1 to 3
compiles against the runtime, and a file of version 4 fails to compile against a runtime whose
`MaxVersion` is 3.

### Schema changes

The option does not change the encoding or a field number. A decode with the option accepts a
subset of the encodings that it accepts without it. A writer and a reader with different bounds then
disagree about the lengths between the two bounds:

- Adding the option, or lowering N, makes a reader reject what an older writer writes. Upgrade
  every writer first.
- Removing the option, or raising N, makes a writer write what an older reader rejects. Upgrade
  every reader first.

### Cost

- The read function of a slice type or a map type that a bounded field calls takes one more
  parameter. The first allocation of every slice calls `wire.SliceCap`, so every generated file
  of a type with a slice changes beyond its version constants.
- The decode compares the length of a slice with its bound once per element, and the length of
  a map once per entry, in each read function with a bound. The encode compares the length of
  each bounded field once.
- The first allocation of a slice of more than 10 MiB grows by `append` afterwards, which costs
  more allocations and copies on a decode into a nil slice of that size. A decode into a
  receiver that decoded before reuses its slice.
- kanontest builds, per bounded field, a sample of N elements and a sample of N + 1 elements,
  and per table entry a sample that fails at the field with N + 1 elements. A decode of 65,536
  elements of 32 bytes into a zero receiver took 1.6 to 1.7 ms, and their encode took 0.35 ms, on
  one core of an AMD Ryzen 9 9950X3D with Go 1.27.1. The reference encoder and decoder of
  kanontest, which use reflection, are not measured on such a sample.

## Alternatives considered

### A limit in kanon.Options

The decode options take one limit for the elements of every slice and every map of a decode.
fxamacker/cbor limits every array and every map of a decode in this way, at 131,072 elements and
131,072 pairs by default.

**Why not:** one limit for a whole decode cannot state that a page has at most 1,024 items while
an index of the same verification has up to 65,536 digests. A consumer that decodes untrusted
input with many unbounded fields would reverse this verdict.

### A check by the caller before the decode

The caller parses the bytes of the field with the functions of `kanon/wire` and counts the
elements.

**Why not:** the caller restates the rules of the encoding outside kanon, and a difference
between the two copies is a parser differential.

### ValidateKanon on a named slice type

The field takes a named slice type whose `ValidateKanon` rejects a length above N.

**Why not:** `ValidateKanon` runs after the decode of the value, so the decode has allocated
for every element before the check.

### The tag option stream

The verifier streams the bounded fields and counts the elements itself.

**Why not:** the option applies to a byte slice, a string and a slice of a struct type with a
kanon codec, and `[]crypto.Digest` and `[][]byte` are neither. A stream also accepts every
valid list, whatever its length.

### One byte slice for the digests

The ledger writes its digests into one byte slice field. It splits the field into digests after
the decode.

**Why not:** it changes the persisted format of the ledger. The items of a page remain a list of
byte slices.

### A bounded first allocation alone

The first allocation of a slice takes at most 10 MiB. The schema states no bound.

**Why not:** it caps the cost of the `Digests` rows at 10 MiB. The cost of the `Items` rows does
not change, since a valid encoding of many empty elements still allocates for each of them.

### A bound per occurrence of the field

The decode counts the elements of each occurrence of the field in the encoding.

**Why not:** a type without `-canonical` accepts a field more than once, and a slice appends
the elements of each occurrence. An encoding could repeat the field to give the value any
length. The bound states the length of the value.

### A bound in bytes

The option bounds the length of the encoding of the field in bytes.

**Why not:** the `MaxSize` of a frame reader and the `Buffer` of a stream decoder bound bytes
already. The memory of a decode grows with the number of elements, which a bound in bytes does
not limit.

## Drawbacks

- The public API of kanon grows by one error in `kanon`, two functions and one field in
  `kanon/wire`, and one field in `kanontest`.
- Every generated file with a slice changes, since the first allocation of a slice calls
  `wire.SliceCap`.
- A field without a bound that shares a read function with a bounded field compares the length
  of its value once per element.
- A bound applies to its field, not to the slices and the maps in its elements.
- A decode into a nil slice of more than 10 MiB allocates more than once.
- A reader that adds or lowers a bound rejects encodings that older writers wrote. A writer
  that removes or raises a bound writes encodings that older readers reject.
- For a map whose keys of one projection can differ under ==, a decode without `-canonical` and
  a merge reject some inputs that would leave N entries at most after the deletion of a
  repeated projection.
- The checks of a bound of 65,536 decode and encode samples of 65,536 elements. The time of the
  reference encoder and decoder of kanontest on them is not measured.
- A file of generator version 4 needs a runtime with a `MaxVersion` of 4.

## Unresolved and future work

- A bound on the length of a string or a byte slice is not proposed here.
- A bound on the slices and maps nested in the elements of a field is not proposed here.

## References

| What | Where |
|---|---|
| The request, with the measurement and the ledger's bounds | https://github.com/thesm-os/kanon/issues/5 |
| The reproduced measurement | the probe of the request, run with kanon at 47ccacd and core at 7339784 |
| The first allocation of a slice of the generated decode | `readSlice` in `internal/kanon/decode.go:967-970`, and `CountValues` in `wire/bytes.go:20-36`, at 47ccacd |
| The map of the generated decode, made without a size | `readMap` in `internal/kanon/decode.go:1085`, at 47ccacd |
| The decode and the encode of 65,536 elements of 32 bytes | a benchmark of a generated `Page` with kanon at 47ccacd and Go 1.27.1, on one core of an AMD Ryzen 9 9950X3D |
| `encoding/gob`, which caps the first allocation of a slice and a map at 10 MiB | `src/encoding/gob/decode.go:599` and `:665`, and `chunk` in `src/internal/saferio/io.go:19`, Go 1.27.1 |
| fxamacker/cbor, which limits the arrays and the maps of a decode | `defaultMaxArrayElements` and `defaultMaxMapPairs` in `decode.go:989-995`, at 7847abf |
| The bytes of the test vectors | a generated `Page` without the option, kanon at 47ccacd |
