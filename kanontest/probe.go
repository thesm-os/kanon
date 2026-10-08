// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"reflect"
	"slices"
	"strconv"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// edit is a change that a probe makes to one byte of an encoding.
type edit struct {
	// name names the change in the name of a probe.
	name   string
	change func(b byte) byte
}

// edits lists the changes of the probes: the next value and the previous
// one, which move a length, a presence byte, a type number, the wire format
// of a tag and the last byte of a varint by one, and the byte with its
// continuation bit flipped, which joins a varint to the byte after it or
// ends it early.
var edits = [...]edit{
	{"plus 1", func(b byte) byte { return b + 1 }},
	{"minus 1", func(b byte) byte { return b - 1 }},
	{"with its continuation bit flipped", func(b byte) byte { return b ^ 0x80 }},
}

// unknownFields is an unknown field of each wire format, with the field
// number 2147483648, one more than the largest number that a struct can
// give a field, so that no struct knows it: a varint, eight bytes, a length
// and two bytes, and four bytes. Each tag takes five bytes. A piece puts a
// field of a sample between two copies of them, so that the field starts
// at an offset other than 0 and bytes follow it.
var unknownFields = []byte{
	0x80, 0x80, 0x80, 0x80, 0x40, 0x01,
	0x81, 0x80, 0x80, 0x80, 0x40, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
	0x82, 0x80, 0x80, 0x80, 0x40, 0x02, 0xaa, 0xbb,
	0x85, 0x80, 0x80, 0x80, 0x40, 0x01, 0x02, 0x03, 0x04,
}

// slabGuard surrounds the encoding in the slab of a slab probe.
const slabGuard = "\xaa"

// wireFormats lists the wire formats that a tag can name: a varint, eight
// bytes, a length and bytes, and four bytes.
var wireFormats = [...]uint64{wire.Varint, wire.Fixed64, wire.Bytes, wire.Fixed32}

// formatBits masks the bits of a tag that are its wire format.
const formatBits = 7

// piece is a field of a sample alone, which the probes cut and change.
type piece struct {
	// name names the sample and the field.
	name string
	f    *field
	// x is the value of the field in the sample.
	x reflect.Value
	// field is the field as [suite.pieceField] writes it, and enc is field as
	// [suite.wrap] places it.
	field []byte
	enc   []byte
}

// probeFamily is a family of probes and the check that decodes them.
type probeFamily struct {
	// check names the check, which also labels the digest of the family in
	// the golden file.
	check  string
	probes func() []probe
}

// families returns the families of the probes of the decode checks, and for
// a canonical Spec also the families of [suite.breachFamilies].
func (s *suite[T, P]) families() []probeFamily {
	if !s.r.canonical {
		return s.decodeFamilies()
	}
	return slices.Concat(s.decodeFamilies(), s.breachFamilies())
}

// decodeFamilies returns the families of the probes of the decode checks of
// every Spec, whose checks name the probes of a piece by [suite.subject], and
// for a T with a field past whose bound a sample goes, as [suite.overs] lists
// them, the family of [suite.pasts].
func (s *suite[T, P]) decodeFamilies() []probeFamily {
	field := s.subject()
	var past []probeFamily
	if len(s.overs) > 0 {
		past = []probeFamily{{
			"DecodeKanon/decodes a field with one element past its bound as the reference decode", s.pasts,
		}}
	}
	return slices.Concat([]probeFamily{
		{"DecodeKanon/decodes each prefix of " + field + " as the reference decode", s.prefixes},
		{"DecodeKanon/decodes " + field + " with one value written wrong as the reference decode", s.faults},
		{
			"DecodeKanon/decodes " + field + " with a value that its ValidateKanon rejects as the reference decode",
			s.rejections,
		},
		{"DecodeKanon/decodes " + field + " with one changed byte as the reference decode", s.changes},
		{"DecodeKanon/decodes " + field + " with its tag in another wire format as the reference decode", s.formats},
		{"DecodeKanon/decodes " + field + " written twice as the reference decode", s.repeats},
		{"DecodeKanon/decodes " + field + " under each depth limit as the reference decode", s.limits},
		{"DecodeKanon/decodes an encoding that writes every field twice as the reference decode", s.redundancies},
		{"DecodeKanon/decodes two concatenated encodings as the reference decode", s.concatenations},
		{"DecodeKanon/decodes an encoding in a slab at its offset as the reference decode", s.slabs},
	}, past)
}

// pasts returns the probes of the bound: the reference encoding of each
// sample of [suite.overs], which sets a field with the tag option max alone
// at one element past its bound, the last in the order of the encoding. The
// reference encoder writes the encoding, whose encode fails.
func (s *suite[T, P]) pasts() []probe {
	out := make([]probe, len(s.overs))
	for k, x := range s.overs {
		out[k] = probe{name: x.name, data: x.enc}
	}
	return out
}

// breachFamilies returns the families of the probes of a canonical Spec that
// break one rule of the canonical encoding, one family per breach: in each
// piece for breachLong, as [suite.longForms] returns them, and in the
// encoding of each sample for the others, as [suite.breaching] returns them.
func (s *suite[T, P]) breachFamilies() []probeFamily {
	return []probeFamily{
		{
			"DecodeKanon/decodes a field with a varint one byte longer than its shortest form as the reference decode",
			s.longForms,
		},
		{
			"DecodeKanon/decodes an encoding with two adjacent fields swapped as the reference decode",
			s.breaching(breachFields),
		},
		{
			"DecodeKanon/decodes an encoding with two adjacent map entries swapped as the reference decode",
			s.breaching(breachKeys),
		},
		{
			"DecodeKanon/decodes an encoding with unknown fields at a field boundary as the reference decode",
			s.breaching(breachUnknown),
		},
		{
			"DecodeKanon/decodes an encoding with a field written at its zero value as the reference decode",
			s.breaching(breachZero),
		},
		{
			"DecodeKanon/decodes an encoding with a malformed tag at a field boundary as the reference decode",
			s.breaching(breachTag),
		},
		{
			"DecodeKanon/decodes an encoding with a float component of a map key of -0.0 as the reference decode",
			s.breaching(breachNegZero),
		},
		{
			"DecodeKanon/decodes an encoding with a value written as zero bytes as the reference decode",
			s.breaching(breachZeroBytes),
		},
	}
}

// subject returns the input of the probes of a piece in the names of their
// checks: a field between unknown fields, and a field alone for a canonical
// Spec, as [suite.wrap] places it.
func (s *suite[T, P]) subject() string {
	if s.r.canonical {
		return "a field"
	}
	return "a field between unknown fields"
}

// piecesOf returns the pieces of samples: per sample that encodes, per
// field of T, the field alone as [suite.pieceField] writes it, whatever its
// presence. A field that fails to encode, a nil pointer, and a piece that
// another sample has already given are left out.
func (s *suite[T, P]) piecesOf(samples []sample[T]) []piece {
	var out []piece
	seen := make(map[string]bool)
	for _, x := range encodes(samples) {
		v := reflect.ValueOf(&x.value).Elem()
		for _, f := range s.l.fields {
			fx := f.of(v)
			enc, err := s.pieceField(f, fx, nil, 0)
			if err != nil || len(enc) == 0 || seen[string(enc)] {
				continue
			}
			seen[string(enc)] = true
			out = append(out, piece{
				name:  x.name + ", field " + f.Name,
				f:     f,
				x:     fx,
				field: enc,
				enc:   s.wrap(enc),
			})
		}
	}
	return out
}

// pieceField returns the field f of T, whose value is x, as the encoder of
// the pieces writes it, and the error of the encoding: the redundant
// encoding for a Spec that is not canonical, and the reference encoding for
// a canonical one, whose probes change a canonical encoding. The value that
// ft targets is written wrong when ft is not nil, and the first from fields
// of every inline struct and the first from elements and entries of every
// slice and map are left out.
func (s *suite[T, P]) pieceField(f *field, x reflect.Value, ft *fault, from int) ([]byte, error) {
	e := &encoder{r: s.r, mode: s.pieceMode(), fault: ft, from: from}
	enc := e.appendField(nil, s.l.loc(f), f, x, true)
	return enc, e.err
}

// pieceMode returns the mode of the encoder of the pieces: modeRedundant, and
// modeReference for a canonical Spec.
func (s *suite[T, P]) pieceMode() mode {
	if s.r.canonical {
		return modeReference
	}
	return modeRedundant
}

// wrap returns the encoding of a field of a piece as its probes decode it:
// between two copies of unknownFields, and alone for a canonical Spec, whose
// decode rejects the unknown fields before the field.
func (s *suite[T, P]) wrap(field []byte) []byte {
	if s.r.canonical {
		return field
	}
	return slices.Concat(unknownFields, field, unknownFields)
}

// margin returns the number of bytes that [suite.wrap] puts on each side of
// the field of a piece.
func (s *suite[T, P]) margin() int {
	if s.r.canonical {
		return 0
	}
	return len(unknownFields)
}

// thinned returns the encodings of the piece p that leave out the first n
// fields of every inline struct, and the first n elements and entries of
// every slice and map, for n from 0 until the encoding stops changing,
// each as [suite.wrap] places it: in one of them, every field, element and
// entry comes first in the order of the decode.
func (s *suite[T, P]) thinned(p piece) [][]byte {
	out := [][]byte{p.enc}
	last, _ := s.pieceField(p.f, p.x, nil, 0)
	for n := 1; ; n++ {
		enc, _ := s.pieceField(p.f, p.x, nil, n)
		if bytes.Equal(enc, last) {
			return out
		}
		out = append(out, s.wrap(enc))
		last = enc
	}
}

// prefixes returns the probes of every prefix of every piece that cuts the
// field or the unknown fields after it, up to the whole piece. A cut inside
// the unknown fields before the field is the same for every piece, and the
// cuts of the unknown fields after the field produce every error of skipping
// an unknown field.
func (s *suite[T, P]) prefixes() []probe {
	var out []probe
	for _, x := range s.pieces {
		for k := s.margin(); k <= len(x.enc); k++ {
			out = append(out, probe{name: x.name + " cut at byte " + strconv.Itoa(k), data: x.enc[:k]})
		}
	}
	return out
}

// faults returns the probes of every piece with one value written wrong, as
// a fault writes it: every value with a length at every length below its
// own and with one zero byte more, and every varint of an integer as
// wideVarint. The lengths around the value follow it, so that the decode
// of the value runs out of bytes inside it, finds a byte after it, or
// reads a value outside its type.
func (s *suite[T, P]) faults() []probe {
	var out []probe
	for _, p := range s.pieces {
		all := &fault{}
		_, _ = s.pieceField(p.f, p.x, all, 0)
		for t, n := range all.lengths {
			if n < 0 {
				out = append(out, s.faulty(p, t, 0, "widened"))
				continue
			}
			for at := range n {
				out = append(out, s.faulty(p, t, at, "cut to "+strconv.Itoa(at)+" bytes"))
			}
			out = append(out, s.faulty(p, t, n+1, "lengthened by a byte"))
		}
	}
	return out
}

// faulty returns the probe of the piece p whose value t, counted from 0 in
// the order of a fault, a fault writes with at bytes, named after change.
func (s *suite[T, P]) faulty(p piece, t, at int, change string) probe {
	enc, _ := s.pieceField(p.f, p.x, &fault{target: t + 1, at: at}, 0)
	return probe{name: p.name + " with value " + strconv.Itoa(t) + " " + change, data: s.wrap(enc)}
}

// rejections returns the probes of every piece with one value of a
// kanon.Validator written as a value that its ValidateKanon rejects, as
// [suite.pieceRejections] returns them, so that a decode, a view and an
// index meet a value that the encode never writes.
func (s *suite[T, P]) rejections() []probe {
	each := make([][]probe, len(s.pieces))
	for k, p := range s.pieces {
		each[k] = s.pieceRejections(p)
	}
	return slices.Concat(each...)
}

// pieceRejections returns the probes of the piece p with one value of a
// kanon.Validator written as a value that its ValidateKanon rejects, as a
// rejection writes it: one probe per such value of the piece.
func (s *suite[T, P]) pieceRejections(p piece) []probe {
	all := &rejection{}
	s.rejectedField(p, all)
	out := make([]probe, all.met)
	for t := range out {
		out[t] = probe{
			name: p.name + " with value " + strconv.Itoa(t) + " rejected",
			data: s.wrap(s.rejectedField(p, &rejection{target: t + 1})),
		}
	}
	return out
}

// rejectedField returns the field of the piece p as the encoder of the
// pieces writes it, with the value that rj targets rejected, as a rejection
// writes it.
func (s *suite[T, P]) rejectedField(p piece, rj *rejection) []byte {
	e := &encoder{r: s.r, mode: s.pieceMode(), reject: rj}
	return e.appendField(nil, s.l.loc(p.f), p.f, p.x, true)
}

// changes returns the probes of every piece with one byte of its field
// changed by one of edits, for every byte of the field and every edit. The
// probes leave the unknown fields around the field unchanged, since a change
// to them is the same for every piece.
func (s *suite[T, P]) changes() []probe {
	var out []probe
	for _, x := range s.pieces {
		for k := s.margin(); k < len(x.enc)-s.margin(); k++ {
			for _, ed := range edits {
				data := bytes.Clone(x.enc)
				data[k] = ed.change(data[k])
				out = append(out, probe{name: x.name + " with byte " + strconv.Itoa(k) + " " + ed.name, data: data})
			}
		}
	}
	return out
}

// formats returns the probes of every piece with the tag of its field in
// each wire format of wireFormats other than its own, and the bytes of its
// value after the tag unchanged, so that a decode, a view and an index meet
// the field in a wire format that its value does not have.
func (s *suite[T, P]) formats() []probe {
	var out []probe
	for _, p := range s.pieces {
		tag, n := binary.Uvarint(p.field)
		for _, format := range wireFormats {
			if format == tag&formatBits {
				continue
			}
			data := s.wrap(slices.Concat(binary.AppendUvarint(nil, tag&^formatBits|format), p.field[n:]))
			out = append(out, probe{name: p.name + " in wire format " + strconv.FormatUint(format, 10), data: data})
		}
	}
	return out
}

// repeats returns the probes of every piece with its field written twice,
// so that a decode meets a second occurrence of the field, and a view and
// an index meet the second occurrence of a struct field before any other
// field that repeats.
func (s *suite[T, P]) repeats() []probe {
	out := make([]probe, len(s.pieces))
	for k, p := range s.pieces {
		out[k] = probe{name: p.name + " twice", data: s.wrap(slices.Concat(p.field, p.field))}
	}
	return out
}

// limits returns the probes of every piece of every sample, the bulk
// samples included and the bound samples left out, in each encoding that
// thinned returns, under each depth limit: the smallest int, which rejects
// every nested value, and 1 to 4, one more than nesting, the levels of the
// deepest value of a sample below T. A decode fails at the first value that
// is too deep, and in one of the encodings each value comes first.
func (s *suite[T, P]) limits() []probe {
	var out []probe
	for _, x := range s.piecesOf(slices.Concat(s.samples, s.bulk)) {
		for n, enc := range s.thinned(x) {
			for _, d := range []int{math.MinInt, 1, 2, 3, 4} {
				name := x.name + " without the first " + strconv.Itoa(n) + " values under depth " + strconv.Itoa(d)
				out = append(out, probe{name: name, data: enc, opts: kanon.Options{Depth: d}})
			}
		}
	}
	return out
}

// redundancies returns the probes of the redundant encoding of every sample
// that encodes, which writes every field of every inline struct twice,
// whatever its presence and its union.
func (s *suite[T, P]) redundancies() []probe {
	xs := s.encodable()
	out := make([]probe, len(xs))
	for k, x := range xs {
		data := s.r.redundant(s.l, reflect.ValueOf(&x.value).Elem())
		out[k] = probe{name: x.name + " with every field twice", data: data}
	}
	return out
}

// concatenations returns the probes of the encodings of every pair of table
// samples, one after the other, in which every field of both repeats.
func (s *suite[T, P]) concatenations() []probe {
	n := len(s.tables)
	out := make([]probe, n*n)
	for i, a := range s.tables {
		for j, b := range s.tables {
			out[i*n+j] = probe{name: a.name + " then " + b.name, data: slices.Concat(a.enc, b.enc)}
		}
	}
	return out
}

// slabs returns the probes of the encoding of every sample that encodes,
// whose options put it in a slab between two slabGuard, at its offset.
func (s *suite[T, P]) slabs() []probe {
	xs := s.encodable()
	out := make([]probe, len(xs))
	for k, x := range xs {
		opts := kanon.Options{Slab: slabGuard + string(x.enc) + slabGuard, Offset: len(slabGuard)}
		out[k] = probe{name: x.name + " in a slab", data: x.enc, opts: opts}
	}
	return out
}

// longForms returns the probes of every piece with one varint one byte longer
// than its shortest form, one probe per varint that a fault of breachLong
// counts. A piece writes its field whatever its presence, so that the probes
// reach the varints of a field that no sample writes, such as an array of no
// elements.
func (s *suite[T, P]) longForms() []probe {
	each := make([][]probe, len(s.pieces))
	for k, p := range s.pieces {
		each[k] = s.pieceLongForms(p)
	}
	return slices.Concat(each...)
}

// pieceLongForms returns the probes of the piece p with one varint one byte
// longer than its shortest form, one probe per varint.
func (s *suite[T, P]) pieceLongForms(p piece) []probe {
	all := &fault{breach: breachLong}
	_, _ = s.pieceField(p.f, p.x, all, 0)
	out := make([]probe, len(all.lengths))
	for t := range out {
		enc, _ := s.pieceField(p.f, p.x, &fault{target: t + 1, breach: breachLong}, 0)
		out[t] = probe{name: p.name + " with varint " + strconv.Itoa(t) + " one byte longer", data: enc}
	}
	return out
}

// breaching returns the function that returns the probes of the encoding of
// every sample that encodes with the rule of the breach b broken at one of
// its sites, as [suite.breaches] returns them.
func (s *suite[T, P]) breaching(b breach) func() []probe {
	return func() []probe {
		xs := encodes(s.samples)
		each := make([][]probe, len(xs))
		for k, x := range xs {
			each[k] = s.breaches(x, b)
		}
		return slices.Concat(each...)
	}
}

// breaches returns the probes of the encoding of the sample x with the rule
// of the breach b broken at one of its sites, one probe per site.
func (s *suite[T, P]) breaches(x sample[T], b breach) []probe {
	v := reflect.ValueOf(&x.value).Elem()
	all := &fault{breach: b}
	s.r.breached(s.l, v, all)
	out := make([]probe, len(all.lengths))
	for t := range out {
		data := s.r.breached(s.l, v, &fault{target: t + 1, breach: b})
		out[t] = probe{name: x.name + " broken at site " + strconv.Itoa(t), data: data}
	}
	return out
}

// digest returns the number of the probes and the 64-bit FNV-1a hash of
// their bytes and options, which pin a family of probes in the golden file.
// It hashes every negative depth as -1, so that the digest of the depth
// limits, whose smallest is math.MinInt, is the same on every platform.
func digest(probes []probe) string {
	h := fnv.New64a()
	for _, p := range probes {
		fmt.Fprintf(h, "%d %d %d %d\n", len(p.data), len(p.opts.Slab), p.opts.Offset, max(p.opts.Depth, -1))
		// The Write method of a hash.Hash never returns an error.
		_, _ = h.Write(p.data)
		_, _ = h.Write([]byte(p.opts.Slab))
	}
	return strconv.Itoa(len(probes)) + " probes, digest " + strconv.FormatUint(h.Sum64(), 16)
}
