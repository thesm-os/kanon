---
rfc: 0004
title: Batches of messages for storage blocks
author: Roy Klopper <roy.klopper@stealthscale.io>
status: Accepted
created: 2026-09-27
updated: 2026-09-29
discussion: none
supersedes: none
superseded-by: none
produces-adr: ADR-0016, ADR-0017, ADR-0018
---

# RFC-0004: Batches of messages for storage blocks

## Summary

A batch is many kanon encodings in one buffer, with a two-byte header and an index at its
end, so that a writer appends messages without knowing their count and a reader finds message
i with one index read. The package `kanon/batch` builds batches and decodes their messages
from one slab, so a block of a thousand records costs one allocation to decode instead of a
thousand. Offsets are 32 bits wide, and a writer opts into 64-bit offsets for a batch above
4 GiB. A storage engine compresses a batch as a block.

## Motivation

A storage engine reads and writes records in blocks. Decoding each record from its own
buffer costs one allocation per record for the slab, and the decoder was designed for one slab
per block: `DecodeKanon` takes the slab and the offset of the record in it. The block needs a
layout that every engine and every tool share, and that layout must allow random access, since
an engine reads one record of a block far more often than all of them. Compression is the
storage lever: zstd over blocks of 1,000 records took the benchmark record from 58.6 to 32.6
bytes, while compressing one record at a time added 13 bytes to each.

## Detailed design

### The batch

```text
batch   = version flags payload* offset* count
version : 1 byte, value 1
flags   : 1 byte; bit 0 set when offsets are 8 bytes wide; other bits must be 0
payload : the kanon encoding of one message
offset  : uint32 little-endian, or uint64 when flags bit 0 is set; the start of
          payload i, counted from the first payload
count   : uint32 little-endian, the number of payloads
```

- A reader reads `count` from the last 4 bytes, the offsets from the bytes before it, and
  payload i as the bytes from offset i to offset i+1, or to the start of the offsets for the
  last payload.
- Offsets are nondecreasing: an empty payload has no bytes, so two offsets can be equal. With
  at least one payload, the first offset is 0, every offset is within the payload region, and
  the last payload ends at the start of the offsets. With no payload, the payload region is
  empty.
- A reader must reject a version other than 1, a flags byte with a bit above 0 set, an index
  that does not fit the length, and offsets that break the rules above. It checks the count,
  the width of the index, their product and the start of the index without an integer
  overflow, before it allocates or slices, and it never allocates according to a count that it
  has not checked.
- With 32-bit offsets, the payload region is at most 2^32 - 1 bytes. In both widths, the count
  limits a batch to 2^32 - 1 payloads.

The index is at the end so that a writer appends in one pass and the payload region is a
plain concatenation. The header is at the start so that a reader rejects a batch of another
version before it reads the index.

### The package

```go
package batch

// Errors of the Writer, Parse, ParseAlias and Batch.Decode.
var (
	ErrVersion    = errors.New("batch: unsupported version")
	ErrFlags      = errors.New("batch: unknown flag")
	ErrLayout     = errors.New("batch: index does not match the data")
	ErrTooBig     = errors.New("batch: the batch does not fit its offsets or an int")
	ErrFinalized  = errors.New("batch: append after Bytes")
	ErrNilMessage = errors.New("batch: nil message")
)

// Writer appends messages to one buffer. Append into a buffer with capacity
// allocates nothing. A Writer is not safe for concurrent use.
type Writer struct {
	// Wide selects 64-bit offsets, for a batch above 4 GiB. The writer
	// reads it at the first successful Append, or at Bytes for a batch
	// without messages, and keeps that width until Reset.
	Wide bool
	/* ... */
}

// Append encodes m at the end of the buffer and records its offset. It
// fails when m fails to encode, with ErrNilMessage for a nil m, a typed
// nil pointer included, with ErrFinalized after Bytes until Reset, and
// with ErrTooBig when the batch with m, its header, index and count
// included, would not fit its offsets or an int. A failed Append changes
// neither the length nor the bytes of the writer, and does not fix the
// width of the offsets.
func (w *Writer) Append(m kanon.Message) error

// Len returns the number of appended messages.
func (w *Writer) Len() int

// Bytes appends the index to the buffer at its first call and returns the
// batch, which aliases the buffer until Reset. Later calls return the same
// bytes without a second index.
func (w *Writer) Bytes() []byte

// Reset empties the writer and keeps its buffer. The bytes that Bytes
// returned before are no longer valid.
func (w *Writer) Reset()

// Batch reads the messages of one batch.
type Batch struct { /* ... */ }

// Parse checks the layout of data and returns its Batch. Its slab is one
// copy of data, made after the check, so Record and decoded strings do not
// alias data. It returns ErrLayout for a count or an offset that an int of
// the platform cannot hold.
func Parse(data []byte) (Batch, error)

// ParseAlias checks the layout of data as Parse does and returns its Batch
// with data itself as the slab. Record and decoded strings alias data,
// which must not change while they are in use.
func ParseAlias(data []byte) (Batch, error)

// Len returns the number of messages.
func (b Batch) Len() int

// Width returns the length in bytes of the offsets of the index: 4, or 8
// when bit 0 of the flags is set, and 0 for the zero Batch.
func (b Batch) Width() int

// Offset returns the offset in the batch of the encoding of message i, the
// offset that Decode passes in its kanon.Options. It panics when i is out
// of range, as a slice index does.
func (b Batch) Offset(i int) int

// Record returns the encoding of message i. It panics when i is out of
// range, as a slice index does.
func (b Batch) Record(i int) []byte

// Decode decodes message i into m with the batch's slab and the record's
// offset as the kanon.Options, and kanon.DefaultDepth. It panics when i is
// out of range, and returns ErrNilMessage for a nil m, a typed nil pointer
// included. Parse checks the layout alone, so Decode reports a malformed
// payload for its message alone.
func (b Batch) Decode(i int, m kanon.Message) error
```

### Use in a storage engine

```go
var w batch.Writer
for _, r := range records {
	if err := w.Append(&r); err != nil {
		return err
	}
}
block := compress(w.Bytes()) // the engine's compressor, zstd for example

// ...

b, err := batch.ParseAlias(decompress(block))
if err != nil {
	return err
}
var rec Record
for i := range b.Len() {
	if err := b.Decode(i, &rec); err != nil {
		return err
	}
	if keep(&rec) {
		out = append(out, rec.CloneKanon()) // detach from the block
	}
}
```

With views enabled, `RecordView(b.Record(i)).Key()` reads an index key of record i without
decoding it.

### Failure behaviour

- `Parse` and `ParseAlias` validate the header and the index once. `Record` and `Decode` then
  need no bounds checks beyond the slice index.
- A decode error of record i does not affect the other records.
- A batch decoded with `ParseAlias` keeps the block alive while any decoded string is in use.
  `CloneKanon` detaches a record.

### Test vectors

Counts and offsets are little-endian:

| Batch | Bytes |
|---|---|
| Empty, 32-bit offsets | `01 00 00 00 00 00` |
| One empty record, 32-bit offsets | `01 00 00 00 00 00 01 00 00 00` |
| Two empty records, 32-bit offsets | `01 00 00 00 00 00 00 00 00 00 02 00 00 00` |
| One record `08 02`, 32-bit offsets | `01 00 08 02 00 00 00 00 01 00 00 00` |
| One empty record, 64-bit offsets | `01 01 00 00 00 00 00 00 00 00 01 00 00 00` |

A reader must reject these inputs with `ErrLayout`, and must not allocate according to their
counts:

| Input | Why |
|---|---|
| `01 00 01 00 00 00 01 00 00 00` | the first offset is 1 |
| `01 00 ff ff ff ff` | the count of 2^32 - 1 records has no index |

The tests of the package cover both index widths, repeated empty records, an empty batch, a
first offset other than 0, decreasing and out-of-range offsets, an overflowing count or
index, repeated calls of `Bytes`, an `Append` that fails and leaves the writer unchanged, the
lifetime of a copied and of an aliased batch, the panic of an out-of-range index, and a decode
error of one record.

## Alternatives considered

### An index at the start

**Why not:** the writer must know the count before it writes, so it buffers the payloads or
writes the index in a second pass. The index at the end costs the reader one read of the last
4 bytes.

### Uvarint lengths instead of fixed offsets

Each payload preceded by its length, as a stream of frames.

**Why not:** finding record i costs i reads instead of one. A block is read by record more
often than sequentially.

### 64-bit offsets always

**Why not:** 4 extra bytes per record, 7% of a 58-byte record, for a block size that no
compression unit reaches. The flag costs one byte per batch.

### The version and count in the engine's block header

**Why not:** a batch that is copied out of its block, or read by a tool, has no header. Two
bytes per batch make it self-describing.

### Compression inside the package

**Why not:** the engine chooses the compressor, the level and the dictionary, and it often
compresses the block together with its own header. The package depends on the standard
library and `kanon` alone.

### One slab per record

**Why not:** one allocation per record, measured as the single allocation of `UnmarshalBinary`
for a type with strings. The batch slab is one allocation per block, or none with
`ParseAlias`.

## Drawbacks

- 4 bytes of index per record and 6 per batch, about 7% on a 58-byte record before
  compression, and less after, since nondecreasing offsets compress well.
- A batch has at most 2^32 - 1 records, and 4 GiB of payloads unless `Wide` is set.
- Aliasing keeps the whole block alive while one string of it is in use.
- One package of about 220 lines with about 280 lines of tests, at 100% coverage and
  mutation.

## Unresolved and future work

- A columnar layout, with each field's values adjacent across the records of a batch, for
  better compression and column scans. Not measured.
- A record count and byte size in the engine's block header, for statistics without parsing.

## References

| What | Where |
|---|---|
| Density measurement, raw and zstd over 1,000-record blocks | run of 2026-09-27, see the wire format RFC's references |
| zstd used for the measurement | github.com/klauspost/compress v1.20.0 |
