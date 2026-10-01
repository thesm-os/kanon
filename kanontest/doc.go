// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package kanontest checks the codecs that the kanon generator writes
// against the kanon wire format. The test file that kanon generates beside
// a code file declares one [Spec] per struct type and calls [Run], [Bench]
// and [Fuzz] with it, and calls [RunValue] on each named type that is not a
// struct. The package that declares a kanon.Exact type calls [RunExact] on
// it in a test of its own. [Checks], [ValueChecks] and [ExactChecks] return
// the checks of [Run], [RunValue] and [RunExact], so that a test runs them
// against a codec of its own.
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
//   - per value that can fail to encode, and per way to fail it, a sample in
//     which it alone fails that way: a value of a type that encodes itself
//     whose SizeKanon as a kanon.Sizer is below 0, whose method rejects it,
//     or that encodes to another length than its SizeKanon, an interface
//     that stores a type that its field does not list, or a map with a key
//     with a NaN component or with two keys of one projection;
//   - the key sample, whose map keys differ in one part each and have no
//     NaN component;
//   - wide samples, whose maps have more entries than a decode reuses, and
//     per side of the entries of a map whose keys or values can fail to
//     encode, a wide sample in which one entry of such a map fails at that
//     side.
//
// A value of a type that encodes itself takes the value of its Go kind, and
// for a struct the values of the fields that kanon would encode. A type for
// which that gives no value other than the zero value, such as a struct whose
// fields are all unexported, takes the values that its decode method decodes
// from byte strings of every length that it accepts, up to the size of the
// type in memory.
//
// A struct with a kanon codec that a value of T contains, and a struct that
// encodes itself, take the values of their fields in the order of the field
// names. A reorder of the field declarations, which keeps the field numbers,
// leaves the samples unchanged.
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
//     one value of a kanon.Validator that its ValidateKanon rejects, with
//     every byte changed by one or with its continuation bit flipped, with
//     its tag in every other wire format, written twice, and under every
//     depth limit;
//   - each sample with every field of every inline struct written twice;
//   - every two table samples, one after the other;
//   - each sample in a slab, at an offset.
//
// The golden file of T pins the number and a digest of the probes of each
// family, and the encodings of the samples. [Fuzz] decodes its inputs with
// both decoders as well.
//
// # Canonical Specs
//
// A Spec with Canonical set describes a type whose decode accepts only the
// canonical encoding of a value. Its reference decoder applies the rules of
// a canonical decode, in the order and at the offsets of the generated code.
// Each field of a sample is then probed alone, without unknown fields around
// it. Eight more families each break one rule of the canonical encoding at
// one site, the first in each field of a sample alone and the others in the
// encoding of each sample:
//
//   - a varint one byte longer than its shortest form: a tag, a length, an
//     integer, a bool, a type number or the length of a complex128;
//   - two adjacent fields of a struct, swapped;
//   - two adjacent entries of a map, swapped;
//   - the unknown fields of every wire format, at a boundary between fields;
//   - a field written at the zero value of its type, as a selected union
//     member is written, which gives a union a second member;
//   - a malformed tag at a boundary between fields: a tag cut short, the tag
//     of field number 0, and that tag one byte longer than its shortest
//     form;
//   - a float component of a map key written as -0.0;
//   - the bytes of a value with a length written as zero bytes, which decode
//     a type that encodes itself to its zero value.
//
// A further check tests the reference decoder itself. The reference decode
// of the encoding of each sample and of each probe succeeds exactly when the
// input round-trips: the decode without the canonical rules succeeds, and
// the reference encoding of the decoded value is the input. [Fuzz] runs the
// same check on its inputs.
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
// When the view type has IndexKanon, a check calls it on the same inputs.
// It returns the first error, by offset, of the reference scans of the
// fields that the view reads, and without an error each method of the index
// returns what the reference view of its field returns.
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
// # Validators
//
// The reference encoder and decoder call the ValidateKanon method of a
// kanon.Validator on each value of the type that they encode or decode. A
// sample counts such a value as one that can fail to encode: the sample that
// fails at it takes a value that the method rejects, and every other sample
// a value that the method accepts. A probe writes each such value of a
// field, one at a time, as a value that the method rejects, which a decode,
// a view and an index return the error of the method for.
//
// The generated code does not call the method of a type whose directive has
// no -validate, since that method returns nil for every value. The reference
// encoder and decoder call it on such a type too. When the directive adds
// -validate, the checks of code generated before fail for a value of the
// value tables that the method rejects.
//
// [RunValue] checks the method itself: it allocates nothing for a value that
// it accepts, it accepts a value exactly when the encode method of the
// binary, gob or text family of the type accepts it, and the golden file of
// the type pins the encoding or the error of each value of the value
// tables.
//
// # Exact types
//
// The reference encoder and decoder treat a value of a kanon.Exact type as
// the value of any other type that encodes itself. The reference decoder
// encodes it again, so the decode checks of a struct with a field of such a
// type fail for a type whose decode method accepts bytes that its append
// method does not write. A type whose append method fails for the value of
// a field, or appends another length than SizeKanon, panics the generated
// encode. [RunExact] checks both guarantees of kanon.Exact on the type
// itself:
//
//   - For the zero value, every value of the value tables and every value
//     that the decode method decodes, SizeKanon returns no negative value,
//     and the append method appends as many bytes as SizeKanon returns
//     without an error. The append method can fail for the zero value.
//   - The decode method decodes the encoding of every value of the value
//     tables, every prefix of it, every change of one of its bytes, the
//     encoding with one more byte, and a byte string of every length up to
//     the size of the type in memory. For every input that it accepts, the
//     append method writes the input for the decoded value.
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
