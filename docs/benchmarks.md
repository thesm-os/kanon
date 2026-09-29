# Benchmarks

[go_serialization_benchmarks](https://github.com/alecthomas/go_serialization_benchmarks) measures kanon and 54 other serializers on one small struct.
A shape harness measures kanon and 8 other codecs, 13 variants in all, on four shapes of data: nested structs, slices, maps, and a page of about 4 KB.

## Setup

| Item | Value |
|---|---|
| CPU | AMD Ryzen 9 9950X3D, 16 cores |
| Operating system | Linux 7.1.10, Fedora 44 |
| Go | 1.27.1 |
| CPU limit | `GOMAXPROCS=4`, in a systemd user scope with `CPUQuota=400%` |
| Benchmark time | `-benchtime 200ms` for every benchmark |
| kanon | Commit `c5af2b0`, measured on 2026-09-29 |

## A small struct

### Method

We added four kanon serializers to go_serialization_benchmarks at commit `833ed64`.
The suite measures every serializer on this struct:

```go
type SmallStruct struct {
	Name     string    // 16 hexadecimal characters
	BirthDay time.Time // time.Now() in the local zone
	Phone    string    // 10 hexadecimal characters
	Siblings int       // 0 to 4
	Spouse   bool
	Money    float64 // rand.Float64()
}
```

- The suite generates 1,000 random values.
- The encode benchmark encodes one of the 1,000, chosen at random, per operation.
- The decode benchmark decodes the encoding of one of the 1,000, chosen at random, per operation, into a struct that it zeroes first.
- kanon generates methods on a type of the package with the directive, so the kanon serializers copy the fields into a struct of their own. Their times include the copy.
- The suite ran 5 times. A time is the median of the 5 runs, and an allocation count is the largest of the 5. Bytes is the mean of the 5.

The API column is the suite's classification of the code that a serializer needs:

- Codegen: code that a generator writes.
- Manual: code written by hand for each struct.
- Reflect: no code, since the serializer uses reflection at run time.

The Time column is the suite's classification of how a serializer encodes a `time.Time`:

| Time | Encoding |
|---|---|
| FullTzOffset | The full range of `time.Time` with the zone offset, without the zone name |
| FullRange | The full range without a zone |
| UnixNs | An `int64` of nanoseconds since the Unix epoch |
| UnixMs | An `int64` of milliseconds since the Unix epoch |
| RFC3339Ns | An RFC 3339 string with nanoseconds |
| Custom | Restrictions of the serializer's own |
| NoSupport | No `time.Time`, which the caller converts to a primitive |
| Unknown | Not documented |

The kanon serializers differ in their buffers and in their time:

| Serializer | Encode | Decode | Time |
|---|---|---|---|
| kanon | `MarshalBinary` into a new slice | `UnmarshalBinary`, which copies the input once | `time.Time` with its zone offset |
| kanon/unsafe_reuse | `AppendBinary` into one reused buffer | `DecodeKanon` with the input as the slab, so that strings alias the input | `time.Time` with its zone offset |
| kanon/utc | As kanon, with the time converted to UTC | As kanon | `time.Time` in UTC, which encodes without a zone offset |
| kanon/unixns | As kanon, with the time as `int64` Unix nanoseconds | As kanon | `int64` |

### Results

The rows are ranked by the total time of the encode and the decode.
Times are in nanoseconds per operation.
Allocs counts the allocations of one operation.
Bytes is the mean length of an encoding.

| # | Serializer | API | Time | Encode ns | Allocs | Decode ns | Allocs | Total ns | Bytes |
|---:|---|---|---|---:|---:|---:|---:|---:|---:|
| 1 | baseline/unsafe_reuse | Manual | NoSupport | 17.0 | 0 | 10.5 | 0 | 27.5 | 47.0 |
| 2 | mus/unsafe_reuse | Manual | NoSupport | 18.4 | 0 | 17.0 | 0 | 35.4 | 49.0 |
| 3 | benc/usafe | Manual | NoSupport | 25.6 | 1 | 15.4 | 0 | 41.0 | 51.0 |
| 4 | fastape | Manual | UnixNs | 30.5 | 1 | 18.9 | 0 | 49.3 | 55.0 |
| 5 | gencode/unsafe_reuse | Codegen | FullTzOffset | 20.3 | 0 | 29.1 | 2 | 49.4 | 46.0 |
| 6 | baseline | Manual | NoSupport | 24.7 | 1 | 26.9 | 2 | 51.6 | 47.0 |
| 7 | 200sc/bebop/reuse | Codegen | Custom | 21.3 | 0 | 31.8 | 2 | 53.1 | 55.0 |
| 8 | benc | Manual | NoSupport | 25.1 | 1 | 29.0 | 2 | 54.1 | 51.0 |
| 9 | baseline_rw/unsafe_reuse | Manual | NoSupport | 25.6 | 0 | 29.7 | 0 | 55.4 | 47.0 |
| 10 | wellquite/bebop/reuse | Codegen | Custom | 22.1 | 0 | 34.1 | 2 | 56.1 | 55.0 |
| 11 | idr/reuse | Manual | FullTzOffset | 21.6 | 0 | 34.7 | 2 | 56.4 | 54.0 |
| 12 | mus | Manual | NoSupport | 27.3 | 1 | 32.6 | 2 | 59.9 | 49.0 |
| 13 | 200sc/bebop | Codegen | Custom | 30.9 | 1 | 31.4 | 2 | 62.3 | 55.0 |
| 14 | wellquite/bebop | Codegen | Custom | 30.4 | 1 | 34.4 | 2 | 64.8 | 55.0 |
| 15 | idr | Manual | FullTzOffset | 30.7 | 1 | 35.3 | 2 | 66.0 | 54.0 |
| 16 | gencode | Codegen | FullTzOffset | 36.8 | 1 | 32.5 | 2 | 69.3 | 53.0 |
| 17 | colfer | Codegen | Custom | 35.2 | 1 | 36.5 | 2 | 71.7 | 51.1 |
| 18 | gogo/protobuf | Codegen | NoSupport | 38.3 | 1 | 41.5 | 2 | 79.8 | 53.0 |
| 19 | **kanon/unsafe_reuse** | Codegen | FullTzOffset | 40.4 | 0 | 40.7 | 0 | 81.1 | 57.8 |
| 20 | calmh/xdr | Codegen | NoSupport | 45.6 | 1 | 35.8 | 2 | 81.4 | 60.0 |
| 21 | **kanon/unixns** | Codegen | UnixNs | 46.2 | 1 | 40.8 | 1 | 87.0 | 51.6 |
| 22 | msgp | Codegen | FullRange | 43.0 | 1 | 54.1 | 2 | 97.0 | 97.0 |
| 23 | **kanon/utc** | Codegen | FullRange | 54.5 | 1 | 46.7 | 1 | 101 | 55.6 |
| 24 | baseline_rw | Manual | NoSupport | 55.3 | 3 | 47.3 | 2 | 103 | 47.0 |
| 25 | shamaton/msgpackgen/array | Reflect | Unknown | 52.3 | 2 | 54.7 | 3 | 107 | 50.0 |
| 26 | **kanon** | Codegen | FullTzOffset | 56.1 | 1 | 53.1 | 1 | 109 | 58.6 |
| 27 | flatbuffers/unsafe_reuse | Codegen | NoSupport | 84.6 | 0 | 35.7 | 0 | 120 | 95.2 |
| 28 | gotiny | Reflect | UnixNs | 98.2 | 5 | 41.8 | 2 | 140 | 48.0 |
| 29 | shamaton/msgpackgen/map | Reflect | Unknown | 66.0 | 2 | 88.6 | 3 | 155 | 92.0 |
| 30 | protobuf-go | Codegen | RFC3339Ns | 87.5 | 1 | 92.3 | 2 | 180 | 51.6 |
| 31 | hprose2 | Manual | Custom | 108 | 0 | 126 | 3 | 234 | 85.3 |
| 32 | shamaton/msgpack/array | Reflect | Unknown | 140 | 4 | 103 | 4 | 242 | 50.0 |
| 33 | flatbuffers | Codegen | NoSupport | 200 | 6 | 51.7 | 2 | 252 | 95.1 |
| 34 | pulsar | Codegen | NoSupport | 133 | 7 | 119 | 6 | 252 | 51.6 |
| 35 | shamaton/msgpack/map | Reflect | Unknown | 158 | 4 | 127 | 4 | 285 | 92.0 |
| 36 | ikea | Manual | NoSupport | 138 | 8 | 166 | 10 | 304 | 55.0 |
| 37 | dedis/protobuf | Reflect | UnixNs | 172 | 7 | 176 | 3 | 348 | 52.0 |
| 38 | avro2/binary | Manual | NoSupport | 246 | 9 | 222 | 10 | 468 | 47.0 |
| 39 | msgpack | Reflect | FullRange | 214 | 4 | 259 | 3 | 473 | 92.0 |
| 40 | hprose | Manual | Custom | 200 | 7 | 274 | 8 | 475 | 85.3 |
| 41 | jsoniter | Reflect | Custom | 191 | 3 | 303 | 5 | 494 | 141.4 |
| 42 | davecgh/xdr | Reflect | RFC3339Ns | 292 | 12 | 227 | 4 | 520 | 92.0 |
| 43 | ugorji/msgpack | Reflect | Unknown | 249 | 3 | 271 | 3 | 520 | 91.0 |
| 44 | ugorji/binc | Reflect | FullTzOffset | 267 | 4 | 265 | 3 | 532 | 95.0 |
| 45 | easyjson | Codegen | Unknown | 273 | 7 | 262 | 2 | 535 | 151.4 |
| 46 | alecthomas/binary | Reflect | NoSupport | 311 | 15 | 263 | 13 | 574 | 61.0 |
| 47 | capnproto | Codegen | NoSupport | 513 | 6 | 116 | 5 | 629 | 96.0 |
| 48 | bson | Reflect | UnixMs | 237 | 10 | 468 | 18 | 705 | 110.0 |
| 49 | mongobson | Reflect | UnixMs | 373 | 9 | 407 | 14 | 780 | 110.0 |
| 50 | json | Reflect | RFC3339Ns | 322 | 1 | 585 | 1 | 908 | 151.8 |
| 51 | fastjson/reuse | Manual | NoSupport | 392 | 7 | 583 | 11 | 974 | 133.8 |
| 52 | fastjson | Manual | NoSupport | 508 | 13 | 577 | 11 | 1,085 | 133.8 |
| 53 | sereal | Reflect | Unknown | 708 | 22 | 705 | 15 | 1,413 | 142.0 |
| 54 | ssz | Manual | NoSupport | 906 | 5 | 511 | 2 | 1,417 | 55.0 |
| 55 | avro2/text | Manual | NoSupport | 837 | 20 | 692 | 30 | 1,529 | 133.8 |
| 56 | goavro | Manual | NoSupport | 412 | 18 | 1,259 | 52 | 1,672 | 47.0 |
| 57 | gogo/jsonpb | Codegen | RFC3339Ns | 3,184 | 73 | 3,508 | 30 | 6,692 | 125.6 |
| 58 | gob | Reflect | FullTzOffset | 1,288 | 22 | 6,210 | 177 | 7,498 | 172.6 |

- `kanon/unsafe_reuse` is one of 5 serializers that allocate nothing in the encode and the decode.
- `kanon` and `kanon/unsafe_reuse` write the same encoding. Their Bytes differ because each benchmark sets every time to `time.Now()` when it starts, and kanon writes the nanoseconds of a time as a uvarint of 4 or 5 bytes.
- Of the 18 serializers ahead of it, 10 need code written by hand for each struct, and 8 generate it. Four of the 18 encode the time with its zone offset, as kanon does.
- A `time.Time` with its zone offset costs kanon 22 ns and 6 or 7 bytes more than an `int64` of nanoseconds: `kanon` takes 109 ns and 58.6 bytes, and `kanon/unixns` 87.0 ns and 51.6 bytes.

## Four shapes

### Method

The shape harness encodes the same values with every codec.
The shapes are these Go types, which each codec declares in its own form, such as a `.proto` file for protobuf:

```go
// Order is the nested shape: three levels of structs.
type Order struct {
	ID       uint64
	Customer Customer
	Shipping Address
	Total    float64
	Paid     bool
}

type Customer struct {
	ID      int64
	Name    string
	Email   string
	Address Address
}

type Address struct {
	Street  string
	City    string
	Zip     string
	Country string
}

// Series is the slice shape: 32 integers, 8 strings and 16 structs.
type Series struct {
	Name   string
	Values []int64
	Tags   []string
	Points []Point
}

type Point struct {
	X     float64
	Y     float64
	Label string
}

// Index is the map shape: 16 counters and 8 labels, keyed by string.
type Index struct {
	Counts map[string]int64
	Labels map[string]string
}

// Page is the large shape: 64 records with 16 bytes of data each, and a
// 512-byte blob.
type Page struct {
	Cursor  string
	Records []Record
	Blob    []byte
}

type Record struct {
	ID    int64
	Name  string
	Score float64
	Flags uint32
	Data  []byte
}
```

- A generator with a fixed seed draws 1,000 samples per shape. Strings have 2 to 24 random characters.
- A test checks that every codec decodes all 1,000 samples of every shape to values equal to the samples, into a new value and into a reused one.
- Each benchmark cycles through the first 16 samples, one sample per operation.
- An encode appends the encoding to a buffer that the benchmark reuses. `json.Marshal` has no append form and returns a new slice. gob encodes each sample with a new `gob.Encoder`, so that each encoding contains its type descriptors, as a message of a request-response protocol does.
- A cold decode decodes into a new value.
- A warm decode decodes into the value that the previous warm decode decoded into. The warm decode of vtprotobuf calls `ResetVT` first, and that of gob zeroes the value, since both merge into the value. In the maps shape, json, cbor and gob decode into emptied maps, since they add to a map instead of replacing it.
- The `/alias` variants decode strings that alias the input:
  - kanon passes the input as the slab of `DecodeKanon`.
  - vtprotobuf calls `UnmarshalVTUnsafe`.
  - mus and benc call their unsafe functions.
- The harness's functions for mus and benc are written by hand, as the documentation of each shows. They write the fields in declaration order, without field numbers.
- Each benchmark ran 5 times with `-count 5`, and each number is the median of the 5 runs.

| Codec | Module | Version |
|---|---|---|
| benc | `github.com/deneonet/benc` | v1.1.8 |
| cbor | `github.com/fxamacker/cbor/v2` | v2.9.4 |
| gob | `encoding/gob` | Go 1.27.1 |
| json | `encoding/json` | Go 1.27.1 |
| msgp | `github.com/tinylib/msgp` | v1.6.4 |
| mus | `github.com/mus-format/mus-go` | v0.10.2 |
| protobuf-go | `google.golang.org/protobuf` | v1.36.12 |
| vtprotobuf | `github.com/planetscale/vtprotobuf` | v0.6.0 |

### Results

The rows of each shape are ranked by the time of an encode and a warm decode.
Times are in nanoseconds per operation.
Allocs counts the allocations of one operation.
Bytes is the mean length of the encodings of the 16 samples.

#### nested

| # | Codec | Bytes | Encode ns | Allocs | Cold decode ns | Allocs | Warm decode ns | Allocs | Encode + warm ns |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | benc/alias | 129 | 35 | 0 | 54 | 1 | 27 | 0 | 61 |
| 2 | **kanon/alias** | 140 | 36 | 0 | 75 | 1 | 47 | 0 | 83 |
| 3 | **kanon** | 140 | 36 | 0 | 97 | 2 | 69 | 1 | 105 |
| 4 | vtprotobuf/alias | 140 | 42 | 0 | 142 | 4 | 82 | 0 | 124 |
| 5 | mus/alias | 121 | 68 | 0 | 84 | 1 | 59 | 0 | 127 |
| 6 | benc | 129 | 34 | 0 | 132 | 11 | 101 | 10 | 135 |
| 7 | msgp | 230 | 35 | 0 | 184 | 11 | 151 | 10 | 186 |
| 8 | mus | 121 | 68 | 0 | 164 | 11 | 131 | 10 | 199 |
| 9 | vtprotobuf | 140 | 42 | 0 | 230 | 14 | 163 | 10 | 205 |
| 10 | protobuf-go | 140 | 205 | 0 | 363 | 14 | 344 | 13 | 548 |
| 11 | cbor | 230 | 231 | 0 | 818 | 11 | 777 | 10 | 1,008 |
| 12 | json | 298 | 557 | 1 | 1,075 | 6 | 1,023 | 5 | 1,580 |
| 13 | gob | 344 | 1,871 | 20 | 8,131 | 236 | 7,950 | 235 | 9,821 |

#### slices

| # | Codec | Bytes | Encode ns | Allocs | Cold decode ns | Allocs | Warm decode ns | Allocs | Encode + warm ns |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | **kanon/alias** | 537 | 192 | 0 | 391 | 4 | 198 | 0 | 389 |
| 2 | benc/alias | 674 | 157 | 0 | 268 | 4 | 245 | 3 | 402 |
| 3 | **kanon** | 537 | 191 | 0 | 443 | 5 | 256 | 1 | 447 |
| 4 | vtprotobuf/alias | 556 | 208 | 0 | 609 | 27 | 283 | 0 | 491 |
| 5 | benc | 674 | 156 | 0 | 443 | 29 | 419 | 28 | 575 |
| 6 | vtprotobuf | 556 | 210 | 0 | 800 | 52 | 449 | 25 | 659 |
| 7 | msgp | 733 | 167 | 0 | 650 | 29 | 504 | 25 | 671 |
| 8 | mus/alias | 468 | 322 | 0 | 472 | 4 | 459 | 3 | 781 |
| 9 | mus | 468 | 326 | 0 | 662 | 29 | 639 | 28 | 965 |
| 10 | protobuf-go | 556 | 772 | 0 | 1,448 | 52 | 1,424 | 51 | 2,196 |
| 11 | cbor | 728 | 1,074 | 0 | 3,377 | 32 | 3,145 | 25 | 4,219 |
| 12 | json | 1,246 | 2,835 | 1 | 6,472 | 35 | 5,503 | 19 | 8,338 |
| 13 | gob | 782 | 2,828 | 24 | 8,782 | 242 | 8,855 | 241 | 11,683 |

#### maps

| # | Codec | Bytes | Encode ns | Allocs | Cold decode ns | Allocs | Warm decode ns | Allocs | Encode + warm ns |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | **kanon/alias** | 374 | 635 | 0 | 792 | 10 | 261 | 0 | 896 |
| 2 | benc/alias | 458 | 443 | 0 | 483 | 11 | 475 | 10 | 918 |
| 3 | **kanon** | 374 | 644 | 0 | 850 | 11 | 305 | 1 | 949 |
| 4 | msgp | 418 | 406 | 0 | 728 | 39 | 546 | 32 | 952 |
| 5 | mus/alias | 370 | 535 | 0 | 525 | 7 | 501 | 6 | 1,036 |
| 6 | benc | 458 | 448 | 0 | 788 | 43 | 787 | 42 | 1,235 |
| 7 | mus | 370 | 548 | 0 | 861 | 39 | 829 | 38 | 1,377 |
| 8 | vtprotobuf/alias | 464 | 686 | 0 | 832 | 10 | 818 | 9 | 1,505 |
| 9 | vtprotobuf | 464 | 690 | 0 | 1,122 | 42 | 1,098 | 41 | 1,788 |
| 10 | cbor | 414 | 835 | 0 | 2,182 | 43 | 1,900 | 36 | 2,735 |
| 11 | json | 517 | 2,195 | 29 | 3,178 | 41 | 2,396 | 31 | 4,591 |
| 12 | protobuf-go | 464 | 2,766 | 96 | 3,534 | 138 | 3,472 | 137 | 6,238 |
| 13 | gob | 505 | 2,246 | 69 | 7,382 | 219 | 7,302 | 212 | 9,548 |

#### large

| # | Codec | Bytes | Encode ns | Allocs | Cold decode ns | Allocs | Warm decode ns | Allocs | Encode + warm ns |
|---:|---|---:|---:|---:|---:|---:|---:|---:|---:|
| 1 | benc/alias | 3,997 | 666 | 0 | 805 | 2 | 792 | 1 | 1,457 |
| 2 | **kanon/alias** | 4,103 | 882 | 0 | 2,241 | 67 | 1,092 | 0 | 1,974 |
| 3 | vtprotobuf/alias | 4,155 | 941 | 0 | 2,564 | 72 | 1,332 | 0 | 2,273 |
| 4 | **kanon** | 4,103 | 886 | 0 | 2,603 | 68 | 1,487 | 1 | 2,373 |
| 5 | benc | 3,997 | 664 | 0 | 2,180 | 132 | 2,161 | 131 | 2,825 |
| 6 | vtprotobuf | 4,155 | 937 | 0 | 4,167 | 202 | 2,167 | 65 | 3,104 |
| 7 | msgp | 5,553 | 876 | 0 | 3,245 | 132 | 2,313 | 65 | 3,189 |
| 8 | mus/alias | 3,715 | 1,345 | 0 | 2,283 | 67 | 2,277 | 66 | 3,622 |
| 9 | mus | 3,715 | 1,345 | 0 | 3,102 | 132 | 3,097 | 131 | 4,442 |
| 10 | protobuf-go | 4,155 | 3,508 | 0 | 6,498 | 202 | 6,434 | 201 | 9,942 |
| 11 | cbor | 5,492 | 4,418 | 0 | 16,198 | 133 | 15,136 | 130 | 19,554 |
| 12 | gob | 4,341 | 6,921 | 27 | 14,573 | 463 | 14,638 | 462 | 21,559 |
| 13 | json | 8,292 | 10,740 | 1 | 26,716 | 136 | 23,031 | 63 | 33,771 |

- The warm decode of `kanon/alias` allocates nothing in any shape, and that of `kanon` allocates one copy of the input.
- `benc/alias` ranks first on nested and large, and `kanon/alias` on slices and maps.

## Sizes

### A small struct

The size measurement draws 10,000 records from a seeded generator, as the suite draws its values.
It reports the mean bytes per record for two datasets that differ only in their times:

- In the equal-times dataset, every time is `time.Now()` at generation, so the times of a block are nearly equal.
- In the spread-times dataset, the times spread over one year.

The compressor is `github.com/klauspost/compress/zstd` v1.20.1.
Each column is one measurement:

| Column | Dataset | Measurement |
|---|---|---|
| Raw | Equal times | The encoding |
| zstd per record | Equal times | Each record compressed alone at the default level |
| zstd with a dictionary | Equal times | Each of the last 5,000 records compressed alone with a dictionary that `zstd.BuildDict` builds from the first 5,000, with at most 4 KiB of their bytes as its content |
| Block, equal times | Equal times | Blocks of 1,000 records, each record after its length as a uvarint, compressed at the default level |
| Block, spread times | Spread times | The same blocks at the default level |
| Block, spread times, best level | Spread times | The same blocks at the best level |

The variants that reuse buffers or alias strings encode as their base serializers do.
The following table leaves them out and ranks the rows by its last column.

| Serializer | Raw | zstd per record | zstd with a dictionary | Block, equal times | Block, spread times | Block, spread times, best level |
|---|---:|---:|---:|---:|---:|---:|
| 200sc/bebop | 55.0 | 68.0 | 68.8 | 28.7 | 34.2 | 35.1 |
| wellquite/bebop | 55.0 | 68.0 | 67.6 | 28.7 | 34.2 | 35.1 |
| baseline_rw | 47.0 | 60.0 | 61.0 | 29.5 | 35.5 | 35.6 |
| baseline | 47.0 | 60.0 | 61.0 | 29.5 | 35.5 | 35.7 |
| bson | 110.0 | 123.0 | 67.3 | 29.5 | 36.0 | 36.0 |
| mongobson | 110.0 | 123.0 | 67.3 | 29.4 | 36.0 | 36.0 |
| mus | 49.0 | 62.0 | 63.0 | 30.3 | 36.7 | 36.0 |
| benc | 51.0 | 64.0 | 65.0 | 29.7 | 35.9 | 36.1 |
| fastape | 55.0 | 68.0 | 62.5 | 29.5 | 36.1 | 36.3 |
| ssz | 55.0 | 68.0 | 69.0 | 30.7 | 36.0 | 36.4 |
| avro2/binary | 47.0 | 60.0 | 61.0 | 30.8 | 44.9 | 36.7 |
| dedis/protobuf | 52.0 | 65.0 | 66.0 | 30.2 | 36.8 | 36.8 |
| goavro | 47.0 | 60.0 | 61.0 | 30.8 | 44.9 | 36.8 |
| capnproto | 96.0 | 103.4 | 72.0 | 34.1 | 38.5 | 36.9 |
| gotiny | 48.0 | 61.0 | 62.0 | 31.9 | 49.0 | 37.1 |
| calmh/xdr | 60.0 | 73.0 | 73.9 | 31.9 | 36.7 | 37.1 |
| flatbuffers | 95.1 | 108.1 | 69.8 | 34.2 | 39.6 | 37.3 |
| idr | 54.0 | 67.0 | 68.0 | 31.0 | 37.3 | 37.3 |
| ikea | 55.0 | 68.0 | 69.0 | 31.8 | 36.7 | 37.4 |
| jsoniter | 141.4 | 154.4 | 68.8 | 28.2 | 39.1 | 37.5 |
| colfer | 51.1 | 64.1 | 65.1 | 32.7 | 38.4 | 37.7 |
| gencode | 53.0 | 66.0 | 67.0 | 32.4 | 38.4 | 37.7 |
| shamaton/msgpack/array | 50.0 | 63.0 | 64.0 | 32.1 | 45.1 | 37.7 |
| shamaton/msgpackgen/array | 50.0 | 63.0 | 64.0 | 32.2 | 45.1 | 37.7 |
| pulsar | 51.6 | 64.6 | 65.5 | 31.4 | 38.6 | 37.8 |
| protobuf-go | 51.6 | 64.6 | 65.5 | 31.4 | 38.6 | 37.8 |
| gogo/protobuf | 53.0 | 66.0 | 67.0 | 30.8 | 37.6 | 37.8 |
| **kanon/unixns** | 51.6 | 64.6 | 65.5 | 31.5 | 38.7 | 37.9 |
| alecthomas/binary | 61.0 | 74.0 | 64.5 | 33.2 | 39.1 | 38.0 |
| hprose | 85.3 | 98.3 | 75.7 | 30.7 | 38.8 | 38.5 |
| hprose2 | 85.3 | 98.3 | 75.9 | 30.7 | 38.8 | 38.5 |
| msgp | 97.0 | 110.0 | 67.7 | 33.0 | 38.4 | 39.2 |
| ugorji/binc | 95.0 | 108.0 | 67.1 | 32.8 | 39.2 | 39.5 |
| gob | 172.6 | 185.6 | 73.6 | 33.1 | 39.6 | 39.6 |
| msgpack | 92.0 | 105.0 | 67.4 | 33.3 | 38.8 | 39.8 |
| ugorji/msgpack | 91.0 | 104.0 | 67.6 | 33.2 | 38.7 | 39.8 |
| gogo/jsonpb | 125.5 | 138.5 | 72.6 | 33.2 | 39.9 | 39.8 |
| shamaton/msgpack/map | 92.0 | 105.0 | 66.7 | 33.2 | 38.8 | 39.8 |
| shamaton/msgpackgen/map | 92.0 | 105.0 | 67.3 | 33.3 | 38.8 | 39.8 |
| fastjson | 133.8 | 146.8 | 73.7 | 33.6 | 40.2 | 40.0 |
| **kanon/utc** | 54.6 | 67.6 | 68.5 | 32.4 | 42.1 | 40.2 |
| **kanon** | 57.6 | 70.6 | 71.4 | 32.4 | 41.7 | 40.6 |
| davecgh/xdr | 92.0 | 105.0 | 73.3 | 32.7 | 43.4 | 41.7 |
| avro2/text | 133.8 | 146.8 | 75.4 | 35.6 | 42.1 | 41.7 |
| sereal | 142.0 | 155.0 | 71.7 | 36.9 | 43.3 | 42.8 |
| easyjson | 151.7 | 164.7 | 73.8 | 34.0 | 44.6 | 43.6 |
| json | 151.8 | 164.8 | 74.0 | 34.0 | 44.7 | 43.7 |

- kanon writes the nanoseconds of a time as a uvarint of 4 bytes below 2^28 and 5 bytes from 2^28. The times of the equal-times dataset are nearly equal, so that varint has one length in almost every record of a run. kanon's raw size measured 57.6 bytes in one run on 2026-09-29 and 58.6 bytes in another.
- zstd adds 13 bytes to a record that it compresses alone, for 46 of the 47 serializers.
- In blocks of spread times at the best level, every serializer takes between 35.1 and 43.7 bytes per record. Raw, they take between 47.0 and 172.6 bytes.

### Shapes

Each cell is the mean bytes per sample over the 1,000 samples of the shape: the raw encoding, then the encoding compressed in one block of all 1,000 samples at zstd's best level.
In the block, each sample follows its length in decimal digits and a colon.

| Codec | nested | slices | maps | large |
|---|---:|---:|---:|---:|
| **kanon** | 139 / 105 | 535 / 488 | 373 / 298 | 4,101 / 3,900 |
| protobuf-go | 139 / 105 | 554 / 482 | 463 / 330 | 4,154 / 3,882 |
| vtprotobuf | 139 / 105 | 554 / 482 | 463 / 330 | 4,154 / 3,882 |
| msgp | 228 / 107 | 730 / 491 | 417 / 310 | 5,553 / 3,892 |
| cbor | 229 / 104 | 725 / 490 | 414 / 298 | 5,493 / 3,882 |
| json | 298 / 101 | 1,244 / 535 | 516 / 304 | 8,292 / 3,939 |
| gob | 342 / 103 | 780 / 482 | 504 / 304 | 4,342 / 3,856 |
| mus | 120 / 94 | 466 / 455 | 369 / 294 | 3,713 / 3,682 |
| benc | 128 / 94 | 672 / 472 | 457 / 296 | 3,998 / 3,702 |

- mus is the smallest raw encoding in every shape.
- kanon's raw encoding is as small as protobuf's on nested and smaller on the other three shapes.
