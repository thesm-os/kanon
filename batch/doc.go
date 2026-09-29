// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package batch stores many kanon messages in one buffer and reads any one
// of them without reading the others. A batch is a version byte, a flags
// byte, the encodings of the messages one after another, an index of their
// offsets and the number of messages. A [Writer] appends messages without
// knowing their number, and [Parse] and [ParseAlias] check a batch once, so
// that [Batch.Record] and [Batch.Decode] find message i with one read of
// the index.
//
// # Layout
//
// The index has one little-endian offset per message, counted from the
// first encoding: 4 bytes wide, or 8 bytes wide when bit 0 of the flags is
// set. The count is a little-endian uint32 at the end. The first offset is
// 0 and the offsets are nondecreasing, since an empty encoding has no
// bytes. With 4-byte offsets the encodings are at most 2^32-1 bytes long,
// and in both widths a batch has at most 2^32-1 messages.
//
// # Slabs
//
// Every message of a batch decodes with the whole batch as its slab. The
// records and the decoded strings of a Batch from [Parse] alias one copy of
// the input. Those of a Batch from [ParseAlias] alias the input, which must
// not change while they are in use. CloneKanon copies a message out of the
// slab.
//
// The package does not compress a batch. A storage engine compresses it as
// a block with the compressor of its choice.
//
// # Allocation contract
//
// [Writer.Append] and [Writer.Bytes] do not allocate when an earlier batch
// of the writer was at least as long and had at least as many messages.
// [ParseAlias], [Batch.Len], [Batch.Width], [Batch.Offset] and [Batch.Record]
// do not allocate, and [Parse] allocates the copy. [Batch.Decode] does not allocate for a message whose
// decode with a slab does not allocate.
//
// # Concurrency
//
// A Writer is not safe for concurrent use. The methods of a Batch only
// read, so a Batch is safe for concurrent use.
//
// # Dependency position
//
// batch imports go.thesmos.sh/kanon, and encoding/binary, errors, math,
// reflect, slices and unsafe from the standard library.
package batch
