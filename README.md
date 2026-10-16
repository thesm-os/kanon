<!--
  ~ Copyright ThesmOS B.V. 2026
  ~ SPDX-License-Identifier: Apache-2.0
-->

# kanon

[![CI](https://github.com/thesm-os/kanon/actions/workflows/ci.yml/badge.svg)](https://github.com/thesm-os/kanon/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/go.thesmos.sh/kanon.svg)](https://pkg.go.dev/go.thesmos.sh/kanon)
[![Release](https://img.shields.io/github/v/release/thesm-os/kanon)](https://github.com/thesm-os/kanon/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/thesm-os/kanon)](go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

kanon generates binary codecs for Go struct types.
You name the types in a `go:generate` directive, as you name them for stringer:

```go
//go:generate go tool kanon -type=Order,Line
```

kanon writes the encode, decode, copy and reset methods of each type, and a test that checks them against a reference codec.

## Features

- The Go types are the schema. You do not write a schema in another language, and the generated code does not use reflection.
- kanon encodes every type that `encoding/gob` and `encoding/json` encode: slices and maps nested to any depth, maps with any comparable key type, struct keys included, arrays, pointers, complex numbers, `time.Time`, and interfaces with a list of their concrete types.
- A type with binary, gob or text methods, such as `MarshalBinary` and `UnmarshalBinary`, encodes through them.
- Each field encodes under a number that the generated file records, so a struct can gain and lose fields between an encode and a decode.
- Equal values encode to identical bytes, because map keys are sorted and absent fields are left out.
- An encode into a buffer with room for the encoding does not allocate.
- A decode into a value that decoded before does not allocate when you pass the input as the slab.
- A decode error contains the struct type, the field, the field number and the offset of the malformed input.
- A view reads one field of an encoding without a decode of the others.
- The `-canonical` flag makes a decode accept only the bytes that the encode writes for the value.
- A stream decoder reads an encoding too large for memory from an `io.Reader`, one field at a time.
- The tag option `max` bounds the bytes of a string or a byte slice and the elements of a slice or a map. A decode checks the bound before it copies the bytes or allocates the elements.
- The generated test compares each codec with a reference encoder and decoder on fixed samples, on generated values and on malformed input, and checks its allocations.

## Install

kanon needs Go 1.27.2 or later.
Add the generator to your module as a tool:

```sh
go get -tool go.thesmos.sh/kanon/cmd/kanon@latest
```

The module also contains the packages that the generated code imports: `go.thesmos.sh/kanon` and `go.thesmos.sh/kanon/wire`, and `go.thesmos.sh/kanon/kanontest` for the generated tests.

## Quick start

1. Name the struct types in a directive in the file that declares them, here `shop/order.go`:

   ```go
   package shop

   import "time"

   //go:generate go tool kanon -type=Order,Line

   type Order struct {
   	ID     string
   	Lines  []Line
   	Placed time.Time
   	Note   *string
   }

   type Line struct {
   	SKU      string
   	Quantity int
   	Cents    int64 `kanon:",fixed"`
   }
   ```

2. Run the generator, and add the requirements of the generated tests to `go.mod`:

   ```sh
   go generate ./...
   go mod tidy
   ```

   kanon writes `order.kanon.go`, with the methods of `*Order` and `*Line`, and `order.kanon_test.go`, which runs the conformance test of each type.

3. Write the golden files of the conformance tests:

   ```sh
   go test -run '^TestKanon' ./shop -update
   ```

4. Commit `order.kanon.go`, `order.kanon_test.go` and `shop/testdata` with `order.go`.

`order.kanon.go` records the field numbers, and regeneration reads them, so a deleted file loses them.
The golden files pin the encoding of every sample.
A change that alters an encoding fails the tests until you run the command with `-update` again.

The conformance test also checks values that [`go.dokimi.dev/assert/prop`](https://pkg.go.dev/go.dokimi.dev/assert/prop) generates from a new seed on each run.
prop shrinks a failing case to the smallest case that fails the same check, and stores it in `testdata/prop`.
Every later run tries a stored case first, until you delete its file.
In CI, set `DOKIMI_ASSERT_PROP_PROFILE=ci`.
Each run then takes its seed from the property, and every run of a commit checks the same values.

## Encode and decode

```go
note := "leave at the door"
order := shop.Order{
	ID:     "o-1001",
	Lines:  []shop.Line{{SKU: "tea", Quantity: 2, Cents: 450}},
	Placed: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
	Note:   &note,
}

data, err := order.MarshalBinary()
if err != nil {
	return err
}

var got shop.Order
if err := got.UnmarshalBinary(data); err != nil {
	return err
}
```

To encode without an allocation, append to a buffer that you reuse:

```go
buf, err = order.AppendBinary(buf[:0])
```

For each struct type `T` of the directive, `*T` implements [`kanon.Message`](https://pkg.go.dev/go.thesmos.sh/kanon#Message) and [`kanon.Cloner[T]`](https://pkg.go.dev/go.thesmos.sh/kanon#Cloner):

| Method | Contract |
|---|---|
| `SizeKanon() int` | Returns the length of the encoding in bytes. |
| `EncodeKanon(buf []byte) (int, error)` | Writes the encoding into the last `SizeKanon()` bytes of `buf`, and returns their count. |
| `AppendBinary(b []byte) ([]byte, error)` | Appends the encoding to `b`. |
| `MarshalBinary() ([]byte, error)` | Returns the encoding in a new slice. |
| `DecodeKanon(data []byte, opts kanon.Options) error` | Sets the receiver to the value in `data`, and reuses the memory of its slices, maps and pointers. |
| `UnmarshalBinary(data []byte) error` | Decodes as `DecodeKanon` does with the zero `kanon.Options`, with one copy of `data`. |
| `MergeKanon(data []byte, opts kanon.Options) error` | Decodes `data` into the receiver without a reset. A slice appends, and a map adds entries. |
| `Reset()` | Clears every field, and keeps the storage of slices and maps for the next decode. |
| `CloneKanon() *T` | Returns a deep copy that shares no memory with the receiver or with the input that it decoded from. |

`kanon.Options` sets the slab of a decode, as [Decode without a copy](#decode-without-a-copy) describes, and the nesting limit, which is 100 levels by default.

## Handle errors

A decode returns a `*kanon.DecodeError`.
For the order of the preceding example without its last byte, the error is:

```go
err := got.UnmarshalBinary(data[:len(data)-1])
fmt.Println(err)
// kanon: Order.Note (field 4) at offset 36: unexpected EOF
```

Truncated input unwraps to `io.ErrUnexpectedEOF`, so a stream reader can wait for more bytes without a check of its own.
Every other cause is a sentinel error of package kanon, such as `kanon.ErrMalformed` and `kanon.ErrRange`, or the error of a type that decodes itself.
Classify an error with `errors.Is` and `errors.AsType`:

```go
if errors.Is(err, io.ErrUnexpectedEOF) {
	// Wait for more input.
}
if derr, ok := errors.AsType[*kanon.DecodeError](err); ok {
	log.Printf("field %s of %s at offset %d", derr.Field, derr.Type, derr.Offset)
}
```

An encode returns a `*kanon.EncodeError` in seven cases:

- A map key has a NaN component.
- Two keys of one map encode alike.
- An interface contains a type that its list does not name.
- A type that encodes itself returns an error.
- The encoding of a type that declares `SizeKanon` has another length than `SizeKanon` returns, and the error wraps `kanon.ErrSize`.
- The `ValidateKanon` method of a type rejects a value.
- A value is longer than the tag option `max` of its field allows, and the error wraps `kanon.ErrMax`.

A value of a type that declares `kanon.Exact` and breaks its guarantee panics the encode with a `*kanon.EncodeError` that wraps `kanon.ErrExact`.
[The documentation of package kanon](https://pkg.go.dev/go.thesmos.sh/kanon#pkg-variables) lists every sentinel error.

## Add, remove and rename fields

A field without a tag takes the smallest free number in declaration order.
`order.kanon.go` records the numbers of each type:

```go
//kanon:numbers Order ID=1 Lines=2 Placed=3 Note=4
//kanon:numbers Line SKU=1 Quantity=2 Cents=3
```

Regeneration keeps the recorded numbers, so data that you encoded before an edit of a struct still decodes after it:

- A field keeps its number when you add, remove or reorder fields.
- A new field takes the smallest number that is neither taken nor reserved.
- kanon reserves the number of a removed field, and gives it to no other field.
- A tag that gives a recorded field another number fails the generation.
- A renamed field is a new field. To keep decoding the data of the old name, tag the field with the old number.

To check the numbers against a git revision, such as the branch that a change merges into, set `KANON_CHECK`.
The check does not write a file, and `-run` keeps the other generators of your module from running:

```sh
KANON_CHECK=origin/main go generate -run 'go tool kanon' ./...
```

The check fails in three cases:

- A field has another number than at the revision.
- A number that the revision reserves, or gives a field that the change removes, is neither reserved nor taken by a field whose tag names it.
- A struct of another package that the revision records as an inline struct now has a kanon codec of its own, and the code file of its package breaks one of these rules against that record, or does not record numbers for it.

## Tag fields

Tags are optional.
A tag is a comma-separated list of at most one field number and any of the options, as in `kanon:"7,fixed"`:

| Tag | Effect |
|---|---|
| `kanon:"7"` | Encodes the field under number 7. |
| `kanon:"-"` | Leaves the field out of the encoding. |
| `kanon:""` | Encodes an unexported field. Every tag other than `-` opts an unexported field in. |
| `kanon:",fixed"` | Writes the 32- and 64-bit integers of the field at their fixed width instead of as varints. |
| `kanon:",union=Kind"` | Makes the field a member of the union whose discriminator is the field `Kind`. |
| `kanon:",types=Circle\|*Square"` | Lists the concrete types of the interfaces in the type of the field, as `gob.Register` registers types. |
| `kanon:",unknown"` | Marks the `[]byte` field in which a decode keeps the fields that it does not know. |
| `kanon:",stream"` | Makes the stream decoder of the struct return the value of the field to its caller, as [Decode an encoding too large for memory](#decode-an-encoding-too-large-for-memory) describes. |
| `kanon:",max=1024"` | Bounds a string or a byte slice field at 1024 bytes, or a slice or a map field at 1024 elements, as [Bound the length of a field](#bound-the-length-of-a-field) describes. |

The discriminator of a union is a field of a named integer type, which the encoding leaves out.
Declare one constant of that type per member, named after the type and the member.
For a discriminator of type `Kind`, the constant `KindText` selects the member `Text`.
The constants must differ.
None of them can be 0, because 0 does not select a member.

gob and kanon both leave out a field of a function or a channel type.
They both encode an embedded struct as one field named after its type.
Unlike `encoding/json`, kanon does not promote the fields of an embedded struct.

## Validate a named type

A named type that is not a struct can encode as its underlying type.
Name it in a directive.
Name the method that checks a value with `-validate`:

```go
type Fixed64 int64

//go:generate go tool kanon -type=Fixed64 -validate=valid

func (f Fixed64) valid() error {
	if f == math.MinInt64 {
		return errOutOfRange
	}
	return nil
}
```

The generated method `ValidateKanon() error` calls `valid`.
The generated code calls it on every value of the type that it encodes or decodes.
A value that it rejects fails the encode or the decode.
Without `-validate`, the generated `ValidateKanon` returns nil, and the generated code does not call it.

kanon reads the directive of the type when it generates the code of a struct with a field of that type.
After you upgrade a dependency whose directive adds `-validate`, run `go generate` again.
Code generated before the upgrade does not call the method.

> [!NOTE]
> Adding `ValidateKanon` to a type with binary, gob or text methods, or removing it, changes the encoding of every field of the type.
> A check of the field numbers does not detect that change.

## Encode a type through its own methods

A type with binary, gob or text methods encodes as the bytes that the first of those families returns.
To make its encode cheaper, declare these methods:

- `SizeKanon() int` returns the length of that encoding. kanon sizes the value with it instead of an encode, and appends the encoding into its buffer in place. The type is then a [`kanon.Sizer`](https://pkg.go.dev/go.thesmos.sh/kanon#Sizer).
- `IsZero() bool` reports whether the value is the zero value, which a field leaves out. kanon calls it in place of `==`, so it must report true for the zero value alone.
- `ExactKanon()` declares [`kanon.Exact`](https://pkg.go.dev/go.thesmos.sh/kanon#Exact) for a type with an append method, `SizeKanon` and an `==` that compares every bit. kanon then writes a field of the type without an error path, and a canonical decode does not encode its values again.
- `AppendKanon(dst []byte) []byte` declares [`kanon.Appender`](https://pkg.go.dev/go.thesmos.sh/kanon#Appender) for an Exact type whose append method does not return an error for any value. kanon writes such a type without an error path in every position.

An Exact type keeps two guarantees:

- `SizeKanon` does not return a negative value. The append method does not return an error for a value other than the zero value, and appends `SizeKanon` bytes whenever it succeeds.
- The decode method accepts only the bytes that the append method writes for the decoded value.

kanon writes the zero value of an Exact type in a slice, a map, a pointer, a union or an interface.
In those positions the error of its append method fails the encode.
An append method that fails for a field, or that appends another length than `SizeKanon`, panics the encode.
Check both guarantees, and the bytes that `AppendKanon` appends, in a test of the package that declares the type:

```go
func TestDigest(t *testing.T) {
	kanontest.RunExact[Digest](t)
}
```

## Read one field with a view

Add `-views` to the directive to generate a view type per struct type:

```go
//go:generate go tool kanon -type=Order,Line -views
```

A view is the encoding of a value.
Each method of a view reads one field without a decode of the others:

```go
id, err := shop.OrderView(data).ID() // id aliases data
```

A string or a byte slice field returns its bytes, which alias the view.
A slice, a map, an interface and a union member do not have a method.

Each method scans the encoding from its start.
To read two or more fields, index the view once:

```go
index, err := shop.OrderView(data).IndexKanon()
if err != nil {
	return err
}
id := index.ID()
placed, err := index.Placed()
```

`IndexKanon` scans the encoding once, and records the offset of every field that the view reads.
It also fails at each occurrence of an integer field whose value is outside the range of its Go type, and of a string or a byte slice field whose value is longer than the tag option `max` of the field allows.
A method of the view checks the range and the bound at the last occurrence of its field alone.

Each method of the index returns the value that the method of the view with the same name returns.
The methods of a time, a `complex128`, a byte array, a type that encodes itself and a type with `-validate` also return the error of the view method.
The other methods return the value alone.

The view of a struct without a field that a view reads does not have such a method.
Its `IndexKanon` returns the index without a scan and without an error.

## Accept only canonical encodings

Add `-canonical` to the directive, so that the decode of its struct types accepts only the bytes that `EncodeKanon` writes for a value:

```go
//go:generate go tool kanon -type=Header,State -canonical
```

A canonical decode rejects every other input that a default decode accepts.
The error is a `*kanon.DecodeError` that wraps `kanon.ErrNotCanonical`, at the field and the offset of the first byte that breaks one of these rules:

- A varint is in its shortest form.
- Each field number is above the one before it, so no field occurs twice.
- The struct declares every field number of the input.
- A union has one member at most.
- A bool is 0 or 1.
- A field at a value that the encoding leaves out is absent, such as an integer of 0 or an empty string.
- Each map key is above the key before it, and does not have a float component of -0.0.
- A time has the bytes that the encoder writes for it.
- A type with methods of its own has the bytes that its encode method writes for the value that its decode method returns.

Use the flag for data whose bytes are evidence, such as a hashed record.
The hash of such a record then changes only when its value changes.
The flag changes the decode of the named types alone, and their encode and their views do not change.

A canonical reader rejects a field that its struct does not declare.
Upgrade every canonical reader before a writer sets a new field.
The generation fails for a canonical type that contains a struct with a kanon codec whose directive does not set `-canonical`, a struct whose codec is written by hand, or a field tagged `unknown`.

## Bound the length of a field

A decode makes a slice with room for every element that it counts in its input, before it decodes the first one.
An empty element takes one byte of input, so 1 MiB of untrusted input can make a slice of a million elements.
A byte slice field can take the whole input, and a decode copies it into the memory of the receiver before the caller can check its length.
Bound a string, a byte slice, a slice or a map field with the tag option `max`:

```go
type Page struct {
	Title string            `kanon:"1,max=256"`
	Body  []byte            `kanon:"2,max=65536"`
	Items [][]byte          `kanon:"3,max=1024"`
	Path  []string          `kanon:"4,max=32"`
	Meta  map[string]string `kanon:"5,max=16"`
}
```

The bound of a string or a byte slice counts bytes, and the bound of a slice or a map counts elements.

- The encode of a value whose field is longer than its bound fails with a `*kanon.EncodeError` that wraps `kanon.ErrMax`.
- A decode, a merge and a stream decoder fail with a `*kanon.DecodeError` that wraps `kanon.ErrMax`. A string or a byte slice fails at the byte past its bound, before they copy the bytes or take the substring. A slice or a map fails at the element that would take the field past its bound, before they decode it.
- A method of a view and `IndexKanon` fail at the same byte for a string or a byte slice.
- The first allocation of a bounded slice makes room for its bound of elements at most.
- The first allocation of every slice takes at most 10 MiB, bounded or not, and the slice grows by `append` after it.

The bound of a string or a byte slice applies to each occurrence of the field in the encoding, because a later occurrence replaces the value.
The bound of a slice or a map counts the elements of every occurrence of the field, and in a merge the elements of the receiver.
It applies to the field itself, and not to the strings, slices and maps in its elements.
The option applies to a string, a byte slice, a slice and a map, other than a type that encodes itself.

The option does not change the encoding, but it changes what a reader accepts:

- To add the option or to lower the bound, upgrade every writer first.
- To remove the option or to raise the bound, upgrade every reader first.

## Decode an encoding too large for memory

A decode reads the whole encoding from one byte slice.
For an encoding too large to read into memory, such as an object of 4 GiB, mark its large fields with the tag option `stream`:

```go
//go:generate go tool kanon -type=Bundle,Inner -canonical

type Bundle struct {
	Name    string
	Data    []byte  `kanon:",stream"`
	Members []Inner `kanon:",stream"`
}
```

The option applies to a byte slice, a string and a slice of a struct type with a kanon codec.
It changes neither the encoding nor the other methods of the type.
kanon writes the stream decoder `BundleStream` for the type.
The stream reads an encoding of a known length from an `io.Reader`.
It decodes the other fields into the receiver and stops at each streamed field:

```go
s := NewBundleStream(r, size, &b, kanon.StreamOptions{})
for {
	f, err := s.Next()
	if errors.Is(err, io.EOF) {
		return nil // b contains every field but Data and Members
	}
	if err != nil {
		return err
	}
	switch f {
	case BundleFieldData:
		if _, err := io.Copy(h, s); err != nil { // hashes the bytes of Data
			return err
		}
	case BundleFieldMembers:
		for {
			n, err := s.Element() // the length, before the element is read
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
			if n > maxMember {
				return errMemberTooLong
			}
			if err := s.DecodeMembers(&member); err != nil {
				return err
			}
		}
	}
}
```

- `Len` returns the declared length of the current field before the stream reads its value.
- `Read` reads the bytes of a streamed byte slice or string, and `WriteTo` writes them to an `io.Writer`. `io.Copy` calls `WriteTo`, so the copy does not allocate.
- `Element` returns the length of the next element of a streamed slice, and the decode method of the slice, here `DecodeMembers`, decodes it.
- `Next` discards the bytes that you do not read. It also decodes the elements that you skip, so that their checks run.

The stream applies the checks of `DecodeKanon` and returns its errors at the same offsets.
It returns a streamed value before it reads the rest of the encoding.
A later byte can still fail the stream, so act on the values only after `Next` returns `io.EOF`.

`kanon.StreamOptions` sets the nesting limit and the buffer limit, which is 16 MiB by default.
The decoded fields together, and each element, must fit within the buffer limit, or the stream fails with `kanon.ErrLimit`.
A streamed slice with the tag option `max` fails with `kanon.ErrMax` at the element past its bound, before the stream reads the length of the element.
A streamed string or byte slice with the option fails with `kanon.ErrMax` at the byte past its bound, before `Next` returns the field.
The strings of the receiver alias the buffer of the stream until the next `Reset`.
The strings of an element alias it until the next call of `Next`, `Element` or a decode method.

## Decode without a copy

The slab of a decode is the string `opts.Slab`, which contains `data` at the offset `opts.Offset`.
Every string that the decode returns is a substring of the slab.
With an empty `Slab`, `DecodeKanon` copies `data` into a slab of its own, as `UnmarshalBinary` does.
For a type with strings, that copy is one allocation per decode.
If you control the input buffer, pass the slab yourself.

To alias the input buffer, pass the slab that `unsafe.String` returns, which shares the memory of `data`:

```go
slab := unsafe.String(unsafe.SliceData(data), len(data))
if err := m.DecodeKanon(data, kanon.Options{Slab: slab}); err != nil {
	return err
}
```

Leave `data` unchanged while `m` or a string decoded from it is in use.

To decode many values from one buffer, copy the buffer into one slab, and pass the offset of each value:

```go
slab := string(batch)
for _, r := range records { // r.off and r.n locate one encoding in batch
	if err := m.DecodeKanon(batch[r.off:r.off+r.n], kanon.Options{Slab: slab, Offset: r.off}); err != nil {
		return err
	}
}
```

The batch then allocates one slab, and the offset in an error is the offset in `batch`.
Package `batch` decodes this way.

> [!WARNING]
> An aliased buffer must not change while a value decoded from it is in use:
>
> - A change to an aliased buffer changes every string decoded from it, without an error. A reuse of the buffer for the next read changes the values of the previous read.
> - A string map key changes with the buffer. The map hashed the key with its old bytes, so a lookup by the old bytes or by the new ones can miss it.
> - The garbage collector frees a slab only after every string decoded from it is unreachable. One short string that a value retains keeps the whole batch in memory. Retain the copy that `strings.Clone` returns instead.

Decoded byte slices do not share memory with `data`, because `DecodeKanon` copies each one into the memory of the receiver.

## Write messages to a stream

Package `frame` writes messages to a byte stream, one frame per message.
A frame contains its length, a version byte, a flags byte, a type ID that you assign and the encoding.
If you set `Checksum` on the writer, each frame also ends with a CRC-32C:

```go
w := frame.NewWriter(conn)
if err := w.Write(orderType, &order); err != nil {
	return err
}

r := frame.NewReader(conn)
id, _, err := r.Next()
if err != nil {
	return err
}
if id == orderType {
	var m shop.Order
	if err := r.Decode(&m); err != nil {
		return err
	}
}
```

The strings of a decoded message alias the buffer of the reader until the next call of `Next`.
A `frame.Registry` maps type IDs to constructors.

## Store messages in blocks

Package `batch` stores many messages in one buffer with an index of their offsets.
A storage engine then reads message i without the others:

```go
var w batch.Writer
for i := range orders {
	if err := w.Append(&orders[i]); err != nil {
		return err
	}
}
block := w.Bytes()

b, err := batch.Parse(block)
if err != nil {
	return err
}
var m shop.Order
if err := b.Decode(42, &m); err != nil {
	return err
}
```

`batch.Parse` copies a batch into one slab.
`Batch.Decode` passes the offset of each message in it.
`batch.ParseAlias` uses the buffer of the batch as the slab and does not allocate.
The buffer must then not change while its messages are in use.
Package `batch` does not compress.
Compress a batch as a block with the compressor of your choice.

## Inspect an encoding

`kanon inspect` prints the fields of an encoding without its Go types.
It prints them in the text notation of [protoscope](https://github.com/protocolbuffers/protoscope), or as JSON with `-json`:

```sh
echo '0a 01 78 10 0e' | go tool kanon inspect -hex
```

```text
1: {"x"}
2: 14  # zigzag 7
```

The command reads the file that you name.
Without a file, it reads the standard input.
It reads one struct encoding, a stream of frames with `-frames`, or a batch with `-batch`.
With `-hex`, it reads hexadecimal digits instead of bytes.

## Performance

These numbers come from an AMD Ryzen 9 9950X3D with Go 1.27.1 and `GOMAXPROCS=4`:

- In [go_serialization_benchmarks](https://github.com/alecthomas/go_serialization_benchmarks), with a struct of six fields, `AppendBinary` into a reused buffer takes 40.4 ns and `DecodeKanon` with the input as the slab 40.7 ns, without an allocation. 5 of the 58 serializers do not allocate in either.
- `MarshalBinary` takes 56.1 ns and `UnmarshalBinary` 53.1 ns, with one allocation each.
- In a harness of 13 codec variants on four shapes of data, kanon with the input as the slab has the lowest time of an encode and a warm decode on slices and maps, and the second lowest on nested structs and on a page of 4 KB.
- The warm decode of kanon with the input as the slab does not allocate in any of the four shapes.
- kanon encodes nested structs in as many bytes as protobuf, and slices, maps and the page in fewer.

[docs/benchmarks.md](docs/benchmarks.md) has the setup, the method, every table and the encoded sizes.

## Compatibility

| Concern | Rule |
|---|---|
| Go | kanon needs Go 1.27.2 or later. CI runs the tests on Linux, macOS and Windows, and on 386, where `int`, `uint` and `uintptr` are 32 bits wide. |
| Generated code | Each generated file states the version of the generator that wrote it, and does not compile against a runtime that lacks support for that version. If a build fails this way after an upgrade, run `go generate ./...` again. |
| Wire format | [The kanon wire format](docs/rfc/0001-wire-format.md) specifies every byte, with test vectors. A change to it needs an RFC, and the data that earlier releases encode must still decode. |
| Field numbers | Regeneration keeps the number of every field, as [Add, remove and rename fields](#add-remove-and-rename-fields) describes. |
| Versions | Releases follow [Semantic Versioning](https://semver.org). Before 1.0.0, a minor release can change the API of the runtime and of the generated code. |

## Packages

| Package | Contents |
|---|---|
| [`go.thesmos.sh/kanon`](https://pkg.go.dev/go.thesmos.sh/kanon) | The method set of a generated codec, the decode options, the errors and the version constants |
| [`go.thesmos.sh/kanon/wire`](https://pkg.go.dev/go.thesmos.sh/kanon/wire) | The primitives that the generated code calls. Applications use package `kanon` instead. |
| [`go.thesmos.sh/kanon/frame`](https://pkg.go.dev/go.thesmos.sh/kanon/frame) | Messages on a byte stream, one frame per message |
| [`go.thesmos.sh/kanon/batch`](https://pkg.go.dev/go.thesmos.sh/kanon/batch) | Many messages in one buffer, with an index of their offsets |
| [`go.thesmos.sh/kanon/kanontest`](https://pkg.go.dev/go.thesmos.sh/kanon/kanontest) | The conformance checks that the generated tests run, and `RunExact` |
| [`go.thesmos.sh/kanon/cmd/kanon`](https://pkg.go.dev/go.thesmos.sh/kanon/cmd/kanon) | The generator and `kanon inspect` |

## Design documents

RFCs under [docs/rfc](docs/rfc/README.md) specify the design, and ADRs under [docs/adr](docs/adr/README.md) record the decisions:

- [The kanon wire format](docs/rfc/0001-wire-format.md) specifies the encoding, with test vectors as exact bytes for an implementation in another language.
- [Generated Go codecs](docs/rfc/0002-go-codec-generation.md) specifies the generator, the runtime and the public interface.
- [Frames](docs/rfc/0003-frames.md) specifies messages on a stream.
- [Batches](docs/rfc/0004-batches.md) specifies messages in storage blocks.
- [Inspection](docs/rfc/0005-inspection.md) specifies `kanon inspect`.
- [Canonical decoding](docs/rfc/0006-canonical-decoding.md) specifies the `-canonical` flag.
- [Stream decoding](docs/rfc/0007-stream-decoding.md) specifies the tag option `stream` and the stream decoder.
- [Element bounds](docs/rfc/0008-element-bounds.md) specifies the tag option `max`.
- [Length bounds](docs/rfc/0009-length-bounds.md) extends the tag option `max` to strings and byte slices.

## Contributing

The Makefile runs the build, the tests, the linters and the release through [ergon](https://github.com/dokimasia/ergon), and `make check` runs the gate that CI runs.
[CONTRIBUTING.md](CONTRIBUTING.md) describes the setup, the rules of a change and the review.
To report a vulnerability, follow [SECURITY.md](SECURITY.md).

## License

kanon is licensed under the [Apache License 2.0](LICENSE).
