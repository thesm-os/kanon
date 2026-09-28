// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package kanontest checks the codecs that the kanon generator writes
// against the kanon wire format. The test file that kanon generates beside
// a code file declares one [Spec] per struct type and calls [Run], [Bench]
// and [Fuzz] with it. [Checks] returns the checks of [Run], so that a test
// runs them against a codec of its own.
//
// # Reference
//
// The checks compare the generated methods with a reference encoder and a
// reference decoder, which derive the encoding of a value, and the value
// and the error of an encoding, from the rules of the wire format by
// reflection. A field of a struct type with a kanon codec encodes and
// decodes through the methods of that type, which the checks of its Spec
// cover. A field of a type that encodes itself goes through the methods of
// its family. The reference decoder returns the error of the generated code
// for malformed input: the same cause, location and offset.
//
// The checks compare decoded values by their fingerprints. The fingerprint
// of a value is its encoding extended by every union member and every
// discriminator, in which every NaN has one bit pattern and the entries of
// each map follow the order of their keys.
//
// # Samples
//
// The checks run on these values of T:
//
//   - the zero value;
//   - a table sample per entry of the value tables, which cover the
//     boundaries of every type;
//   - a sample per field, which sets that field alone;
//   - per value that can fail to encode, a sample in which it alone fails:
//     a value of a type that encodes itself that its method rejects, an
//     interface that stores a type that its field does not list, or a map
//     with a key with a NaN component or with two keys of one projection;
//   - the key sample, whose map keys differ in one part each and have no
//     NaN component;
//   - wide samples, whose maps have more entries than a decode reuses.
//
// An int, a uint and a uintptr take values of 32 bits in every sample, so
// the samples and their encodings are the same on every platform. The map
// keys of the samples other than the key sample have a value in every field
// that the encoding leaves out, which no decode yields.
//
// A check merges the encoding of each table sample into each of its
// samples that fail to encode. In such a merge, a map with two keys of one
// projection fails with kanon.ErrAmbiguousKey before the map changes, and
// any other failing value merges as a value that encodes.
//
// # Probes
//
// The decode checks decode the encodings of the samples, and inputs that
// they derive from them, the probes, with the generated code and with the
// reference decoder:
//
//   - each field of a sample alone, between unknown fields of every wire
//     format, cut at every byte, with one nested value cut short,
//     lengthened by a byte or, for an integer, widened out of range, with
//     every byte changed by one or with its continuation bit flipped, and
//     under every depth limit;
//   - each sample with every field of every inline struct written twice;
//   - every two table samples, one after the other;
//   - each sample in a slab, at an offset.
//
// The golden file of T pins the number and a digest of the probes of each
// family, and the encodings of the samples. [Fuzz] decodes its inputs with
// both decoders as well.
//
// # Views
//
// When the Spec names a view type, a check calls each of its methods on the
// encoding of every sample and on every probe. The method returns what the
// reference view of the field that it names returns: the value of the last
// occurrence of the field, or of the value that the field points at,
// decoded alone, and the bytes after the length of a string, a byte slice
// and a struct. The method of a struct fails with kanon.ErrRepeatedView at
// the tag of a second occurrence, since a decode merges the occurrences.
//
// # Fields left out and unexported fields
//
// The checks of Reset, DecodeKanon, MergeKanon and CloneKanon first set
// every field that the encoding leaves out to a value, in the keys and the
// values of maps too, and read every such field afterwards. Reset,
// DecodeKanon and CloneKanon clear them, and MergeKanon keeps them where the
// reference decode keeps their struct.
// Reflection neither sets an unexported field nor returns its value as an
// interface, so the checks read and set every unexported field through its
// address with reflect.NewAt: a field that a kanon tag opts into the
// encoding, and a field that the encoding leaves out, a field of a struct of
// another package included.
//
// # Allocations
//
// The allocation checks count the allocations of SizeKanon, of EncodeKanon
// and AppendBinary into a buffer with room for the encoding, and of
// DecodeKanon with a slab into a receiver that decoded the encoding before.
// Each allows none. They measure every sample except the wide samples and
// the samples that contain a value that [go.thesmos.sh/kanon.Message] lists
// as allocating, and the golden file pins the samples that they measure.
// They count through testing.AllocsPerRun, which panics while a parallel
// test runs, so [Run] runs them first and serially, and [Check.Serial]
// marks them.
//
// # Dependency position
//
// kanontest imports go.thesmos.sh/kanon, go.thesmos.sh/kanon/wire,
// go.dokimi.dev/assert and the standard library. The test files that kanon
// generates import it, and the code files do not.
package kanontest
