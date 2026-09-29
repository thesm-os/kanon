# kanon

kanon generates binary codecs for Go struct types, as stringer generates `String` methods.
A `go:generate` directive names the types:

```go
//go:generate go tool kanon -type=Order,Line
```

- The Go types are the schema, so you do not write a schema in another language.
- The generated code does not use reflection.
- kanon encodes every type that `encoding/gob` and `encoding/json` encode: slices and maps nested to any depth, maps with any comparable key type, struct keys included, arrays, pointers, complex numbers, `time.Time`, and interfaces with a list of their concrete types.
- A type with binary, gob or text methods, such as `MarshalBinary` and `UnmarshalBinary`, encodes through them.
- A named type that is not a struct can encode as its underlying type instead, with a check of its values that kanon calls on every encode and decode.
- Equal values encode to identical bytes, because map keys are sorted and absent fields are left out.
- Each field encodes under a number that the generated file records, so a struct can gain and lose fields between an encode and a decode.
- An encode into a buffer with spare capacity does not allocate.
- A decode into a value that decoded before allocates one copy of its input when the type contains a string, and nothing when you pass a slab. `kanon.Message` lists the values that allocate in either case.
- A decode error contains the struct type, the field, the field number and the offset of the malformed input.
- kanon writes a conformance test per type. The test compares the codec with a reference encoder and decoder and checks the allocation contract. A fuzz target runs the decoder on arbitrary input.

## Install

kanon needs Go 1.27 or later.
Add the generator to your module as a tool:

```sh
go get -tool go.thesmos.sh/kanon/cmd/kanon@latest
```

The module also contains the packages that the generated code imports: `kanon`, `kanon/wire`, and `kanon/kanontest` for the generated tests.

## Generate a codec

Name the struct types in a directive in the file that declares them:

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

Run `go generate ./...`.
kanon writes two files beside `order.go`:

- `order.kanon.go` contains the methods of `*Order` and `*Line`.
- `order.kanon_test.go` runs the conformance suite on each type.

The conformance suite compares the encodings of its samples with golden files.
Create them with the first run:

```sh
go test -run '^TestKanon' ./shop -update
```

Commit the generated files and `testdata/golden` with `order.go`:

- `order.kanon.go` records the field numbers. Regeneration reads them, and deleting the file loses them.
- The golden files pin the encoding of every sample. A change that alters an encoding fails the tests until you run the command with `-update` again.

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

For each type `T` of the directive, `*T` implements `kanon.Message` and `kanon.Cloner[T]`:

| Method | Contract |
|---|---|
| `SizeKanon() int` | Returns the length of the encoding in bytes. |
| `EncodeKanon(buf []byte) (int, error)` | Writes the encoding into the last `SizeKanon()` bytes of `buf` and returns their count. |
| `AppendBinary(b []byte) ([]byte, error)` | Appends the encoding to `b`. |
| `MarshalBinary() ([]byte, error)` | Returns the encoding in a new slice. |
| `DecodeKanon(data []byte, opts kanon.Options) error` | Sets the receiver to the value in `data` and reuses the memory of its slices, maps and pointers. |
| `UnmarshalBinary(data []byte) error` | Calls `DecodeKanon` with the zero `kanon.Options`, which copy `data` once. |
| `MergeKanon(data []byte, opts kanon.Options) error` | Decodes `data` into the receiver without clearing it first. Slices append, and maps add entries. |
| `Reset()` | Clears every field and keeps the storage of slices and maps for the next decode. |
| `CloneKanon() *T` | Returns a deep copy that shares no memory with the receiver or with the input it decoded from. |

`kanon.Options` sets the slab of a decode, as [Decoding without a copy](#decoding-without-a-copy) describes, and the nesting limit, which is 100 levels by default.

## Errors

A decode returns a `*kanon.DecodeError`.
For the order of the preceding example without its last byte, the error is:

```go
err := got.UnmarshalBinary(data[:len(data)-1])
fmt.Println(err)
// kanon: Order.Note (field 4) at offset 36: unexpected EOF
```

Truncated input unwraps to `io.ErrUnexpectedEOF`, so a stream reader can wait for more bytes without a check of its own.
The other causes are the sentinel errors of package kanon, such as `kanon.ErrMalformed` and `kanon.ErrRange`, and the errors of types that decode themselves.
Classify an error with `errors.Is` and `errors.AsType`:

```go
if errors.Is(err, io.ErrUnexpectedEOF) {
	// Wait for more input.
}
if derr, ok := errors.AsType[*kanon.DecodeError](err); ok {
	log.Printf("field %s of %s at offset %d", derr.Field, derr.Type, derr.Offset)
}
```

An encode returns a `*kanon.EncodeError` in six cases:

- A map key has a NaN component.
- Two keys of one map encode alike.
- An interface contains a type that its list does not name.
- A type that encodes itself returns an error.
- The encoding of a type that declares `SizeKanon` has another length than `SizeKanon` returns, and the error wraps `kanon.ErrSize`.
- The `ValidateKanon` method of a type rejects a value.

## Field numbers

A field without a tag takes the smallest free number in declaration order.
`order.kanon.go` records the numbers of each type:

```go
//kanon:numbers Order ID=1 Lines=2 Placed=3 Note=4
//kanon:numbers Line SKU=1 Quantity=2 Cents=3
```

Regeneration keeps the recorded numbers, so data that you encoded before an edit of a struct still decodes after it:

- A field keeps its number when you add, remove or reorder fields.
- A new field takes the smallest number that is neither taken nor reserved.
- kanon reserves the number of a removed field and gives it to no other field.
- A tag that gives a recorded field another number fails the generation.
- A renamed field is a new field. To keep decoding the data of the old name, tag the field with the old number.

To check the numbers against a git revision, such as the branch that a change merges into, set `KANON_CHECK`.
The check does not write files.
The `-run` flag keeps the other generators of your module from running:

```sh
KANON_CHECK=origin/main go generate -run 'go tool kanon' ./...
```

The check fails in three cases:

- A field has another number than at the revision.
- A number that the revision reserves, or gives a field that the change removes, is neither reserved nor taken by a field whose tag names it.
- A struct of another package that the revision records as an inline struct now has a kanon codec of its own, and the code file of its package breaks one of these rules against that record, or records no numbers for it.

## Tags

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

The constant that selects a union member is named after the type of the discriminator and the member.
When `Kind` has type `Kind`, the constant `KindText` selects the member `Text`.

kanon leaves out a field of a function or a channel type, as gob does.
An embedded struct encodes as one field named after its type, as in gob.
kanon does not promote the fields of an embedded struct, which `encoding/json` does.

## Types with methods of their own

A type with binary, gob or text methods encodes as the bytes that the first of those families returns.
Such a type can also declare these methods, which make its encode cheaper:

- `SizeKanon() int` returns the length of that encoding. kanon sizes the value with it instead of encoding it twice, and appends the encoding into its buffer in place.
- `IsZero() bool` reports whether the value is the zero value, which kanon leaves out of a field. kanon calls it instead of comparing the whole value with `==`, so it must report true for the zero value alone.

A named type that is not a struct can encode as its underlying type instead.
Name it in the directive, and name a method that checks a value with `-validate`:

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

kanon generates `ValidateKanon() error`, which calls `valid`, and calls it on every value that it encodes or decodes.
The encode or the decode of a value that it rejects fails.
Adding `ValidateKanon` to a type with binary, gob or text methods, or removing it, changes the encoding of every field of the type.
A check of the field numbers does not detect that change.

## Views

Add `-views` to the directive to generate a view type per struct type:

```go
//go:generate go tool kanon -type=Order,Line -views
```

A view is the encoding of a value.
Each method of a view reads one field without decoding the rest:

```go
id, err := shop.OrderView(data).ID() // id aliases data
```

A string or byte slice field returns its bytes, which alias the view.
Slices, maps, interfaces and union members have no method.

Each method call scans the encoding from its start.
To read two or more fields, index the view once:

```go
index, err := shop.OrderView(data).IndexKanon()
if err != nil {
	return err
}
id, err := index.ID()
```

`IndexKanon` scans the encoding once and records the offset of every field that the view reads.
Each method of the index returns what the method of the view with the same name returns.

## Frames and batches

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

The strings of a decoded message alias the reader's buffer until the next call of `Next`.
A `frame.Registry` maps type IDs to constructors.

Package `batch` stores many messages in one buffer with an index of their offsets, so that a storage engine can read message i without reading the others:

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

Package `batch` does not compress.
Compress a batch as a block with the compressor of your choice.

## Inspect an encoding

`kanon inspect` prints the fields of an encoding without its Go types, in the text notation of [protoscope](https://github.com/protocolbuffers/protoscope), or as JSON with `-json`:

```sh
echo '0a 01 78 10 0e' | go tool kanon inspect -hex
```

```text
1: {"x"}
2: 14  # zigzag 7
```

The command reads the file that you name, or the standard input without one.
It reads one struct encoding, a stream of frames with `-frames`, or a batch with `-batch`.
With `-hex`, it reads hexadecimal digits instead of bytes.

## Performance

We ran the following benchmarks on 2026-09-29 at commit `c5af2b0`, on an AMD Ryzen 9 9950X3D with Go 1.27.1 and `GOMAXPROCS=4`.
[docs/benchmarks.md](docs/benchmarks.md) has the full tables, the encoded sizes and the method.

### A small struct

[go_serialization_benchmarks](https://github.com/alecthomas/go_serialization_benchmarks) encodes and decodes a struct of six fields, one of them a `time.Time`, with 58 serializers.
The following rows are a selection of the 58, ranked by the total time of the encode and the decode.
Times are in nanoseconds per operation.
Allocs counts the allocations of the encode and the decode together.

| # | Serializer | Encode ns | Decode ns | Allocs | Bytes |
|---:|---|---:|---:|---:|---:|
| 1 | baseline/unsafe_reuse | 17.0 | 10.5 | 0 | 47.0 |
| 2 | mus/unsafe_reuse | 18.4 | 17.0 | 0 | 49.0 |
| 18 | gogo/protobuf | 38.3 | 41.5 | 3 | 53.0 |
| 19 | **kanon/unsafe_reuse** | 40.4 | 40.7 | 0 | 57.8 |
| 21 | **kanon/unixns** | 46.2 | 40.8 | 2 | 51.6 |
| 22 | msgp | 43.0 | 54.1 | 3 | 97.0 |
| 26 | **kanon** | 56.1 | 53.1 | 2 | 58.6 |
| 30 | protobuf-go | 87.5 | 92.3 | 3 | 51.6 |
| 33 | flatbuffers | 200 | 51.7 | 8 | 95.1 |
| 50 | json | 322 | 585 | 2 | 151.8 |
| 58 | gob | 1,288 | 6,210 | 199 | 172.6 |

- `kanon` encodes into a new slice and decodes with one copy of the input.
- `kanon/unsafe_reuse` appends to a reused buffer and decodes strings that alias the input.
- `kanon/unixns` encodes the time as `int64` Unix nanoseconds instead of a `time.Time`.
- `kanon` and `kanon/unsafe_reuse` write the same encoding. Their Bytes differ because each benchmark sets every time to `time.Now()` when it starts, and kanon writes the nanoseconds of a time as a varint of 4 or 5 bytes.
- Of the 18 serializers ahead of `kanon/unsafe_reuse`, 10 need code written by hand per struct, and 8 generate it. Four of the 18 encode the time with its zone offset, as kanon does.

### Four shapes

A second harness compares 13 codec variants on four shapes of data: nested structs, slices, maps, and a page of about 4 KB.
Each cell is the time of an encode and a warm decode in nanoseconds, with the rank among the 13 in parentheses.
A warm decode decodes into a value that decoded another sample before.
The `/alias` variants decode strings that alias the input.

| Codec | nested | slices | maps | large |
|---|---:|---:|---:|---:|
| **kanon/alias** | 83 (2) | 389 (1) | 896 (1) | 1,974 (2) |
| **kanon** | 105 (3) | 447 (3) | 949 (3) | 2,373 (4) |
| benc/alias | 61 (1) | 402 (2) | 918 (2) | 1,457 (1) |
| vtprotobuf/alias | 124 (4) | 491 (4) | 1,505 (8) | 2,273 (3) |
| msgp | 186 (7) | 671 (7) | 952 (4) | 3,189 (7) |
| mus/alias | 127 (5) | 781 (8) | 1,036 (5) | 3,622 (8) |
| protobuf-go | 548 (10) | 2,196 (10) | 6,238 (12) | 9,942 (10) |
| cbor | 1,008 (11) | 4,219 (11) | 2,735 (10) | 19,554 (11) |
| json | 1,580 (12) | 8,338 (12) | 4,591 (11) | 33,771 (13) |
| gob | 9,821 (13) | 11,683 (13) | 9,548 (13) | 21,559 (12) |

- `kanon/alias` ranks first on slices and maps. On nested and large it ranks second, behind `benc/alias`. The harness's benc functions are written by hand and write the fields in order, without field numbers.
- The warm decode of `kanon/alias` allocates nothing in any shape. The warm decode of `kanon` allocates one copy of the input.
- kanon encodes the four shapes in 139, 535, 373 and 4,101 bytes on average, and protobuf in 139, 554, 463 and 4,154 bytes.

## Decoding without a copy

The codec that kanon generates for a struct type has a `DecodeKanon(data []byte, opts kanon.Options) error` method.
`opts.Slab` is a string that contains `data` at offset `opts.Offset`.
Every string that `DecodeKanon` decodes is a substring of the slab.
With an empty `Slab`, `DecodeKanon` copies `data` into a slab of its own, and `UnmarshalBinary` decodes this way.
For a type with strings, that copy is one allocation per decode.
If you control the input buffer, pass the slab yourself.

### Alias the input buffer

`unsafe.String` returns a slab that shares the memory of `data`.
The decode then does not allocate a slab:

```go
slab := unsafe.String(unsafe.SliceData(data), len(data))
if err := m.DecodeKanon(data, kanon.Options{Slab: slab}); err != nil {
	return err
}
```

Leave `data` unchanged while `m` or a string decoded from it is in use.

### Share one slab across a batch

If you decode many values from one buffer, copy the buffer into one slab and pass the offset of each value:

```go
slab := string(batch)
for _, r := range records { // r.off and r.n locate one encoding in batch
	if err := m.DecodeKanon(batch[r.off:r.off+r.n], kanon.Options{Slab: slab, Offset: r.off}); err != nil {
		return err
	}
}
```

The batch then allocates one slab.
The offset in an error is the offset in `batch`.
Package `batch` decodes this way: `batch.Parse` copies a batch into one slab, and `Batch.Decode` passes the offset of each message.

### Hazards

- A change to an aliased buffer changes every string decoded from it.
  If you reuse the buffer for the next read, the values of the previous read change without an error.
- A string map key decoded from an aliased buffer changes with the buffer.
  The map computed the hash of the key from its old bytes, so a lookup by the old bytes or by the new ones can miss it.
- The garbage collector frees a slab only after every string decoded from it is unreachable.
  A value that retains one short string from a batch prevents the collector from freeing the whole batch.
  If you retain a decoded string, retain the copy that `strings.Clone` returns.

Decoded byte slices do not share memory with `data`.
`DecodeKanon` copies each one into the memory of the receiver.

## Design documents

The design is in RFCs under [docs/rfc](docs/rfc/README.md), and the decisions are in ADRs under [docs/adr](docs/adr/README.md):

- [The kanon wire format](docs/rfc/0001-wire-format.md) specifies the encoding, with test vectors as exact bytes for an implementation in another language.
- [Generated Go codecs](docs/rfc/0002-go-codec-generation.md) specifies the generator, the runtime and the public interface.
- [Frames](docs/rfc/0003-frames.md) specifies messages on a stream.
- [Batches](docs/rfc/0004-batches.md) specifies messages in storage blocks.
- [Inspection](docs/rfc/0005-inspection.md), a draft, specifies `kanon inspect`.

## Development

The build, test, lint and release targets come from [ergon](https://go.thesmos.sh/ergon):

```sh
make bootstrap      # install dev tools
make install        # go mod download
make check          # full pre-merge gate (mod verify + lint + test + checks)
make test-386       # run the tests where int, uint and uintptr are 32 bits wide
make check-numbers  # fail when a change renumbers a field of BASE (default origin/main)
make build          # compile every module's source
```

`make help` lists every target.

## License

kanon is licensed under the [Apache License 2.0](LICENSE).
