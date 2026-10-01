// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/kanontest"
)

// Names of the checks of kanontest.ExactChecks.
const (
	exactTypeCheck   = "ExactKanon/marks a type with an append method whose == compares every bit"
	exactSizeCheck   = "ExactKanon/sizes every value at 0 bytes or more"
	exactAppendCheck = "ExactKanon/appends SizeKanon bytes whenever the append method returns no error"
	exactErrorCheck  = "ExactKanon/appends every value but the zero value without an error"
	exactDecodeCheck = "ExactKanon/decodes only the bytes that the append method writes for the decoded value"
)

// Lengths of the encodings of the kanon.Exact types of these tests.
const (
	// wordLength is the length of the encoding of a word.
	wordLength = 2
	// digestLength is the length of the encoding of a digest.
	digestLength = 4
	// stretchLength is the length of a stretch in memory, and the length of
	// the input that its decode method accepts beside wordLength.
	stretchLength = 8
)

// gappedValue is the gapped word whose append method fails: 512, which no
// entry of the value tables gives, and which the decode method decodes from
// 02 00, the encoding of 0 with 2 added to its first byte.
const gappedValue = 512

// Errors of the methods of the kanon.Exact types of these tests.
var (
	// errWordLength is the error of the decode method of a word for data that
	// is not wordLength bytes long.
	errWordLength = errors.New("kanontest_test: a word is two bytes long")
	// errDigestLength is the error of the decode method of a digest for data
	// that is not digestLength bytes long.
	errDigestLength = errors.New("kanontest_test: a digest is four bytes long")
	// errZeroDigest is the error of the append method of the zero digest.
	errZeroDigest = errors.New("kanontest_test: the zero digest has no encoding")
	// errBrittle is the error of the append method of a brittle word of
	// math.MaxUint16.
	errBrittle = errors.New("kanontest_test: the word is math.MaxUint16")
	// errGapped is the error of the append method of the gapped word
	// gappedValue.
	errGapped = errors.New("kanontest_test: the word is 512")
	// errStretchLength is the error of the decode method of a stretch for
	// data that is neither wordLength nor stretchLength bytes long.
	errStretchLength = errors.New("kanontest_test: a stretch is two or eight bytes long")
)

// word is a uint16 that keeps both guarantees of kanon.Exact: it appends its
// two big-endian bytes, and decodes exactly two bytes.
type word uint16

// AppendBinary appends the two big-endian bytes of w to b.
func (w word) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w to the two big-endian bytes in data, as [readWord]
// reads them.
func (w *word) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = word(x)
	return err
}

// SizeKanon returns wordLength.
func (word) SizeKanon() int { return wordLength }

// ExactKanon marks word as a kanon.Exact type.
func (word) ExactKanon() {}

// digest is a digest of four bytes behind unexported fields, whose values
// come from its decode method alone. It keeps both guarantees of
// kanon.Exact, and its append method fails for the zero digest, which no
// input decodes to.
type digest struct {
	b   [digestLength]byte
	set bool
}

// AppendBinary appends the four bytes of d to b. It fails with errZeroDigest
// for the zero digest.
func (d digest) AppendBinary(b []byte) ([]byte, error) {
	if !d.set {
		return b, errZeroDigest
	}
	return append(b, d.b[:]...), nil
}

// UnmarshalBinary sets d to the four bytes in data. It fails with
// errDigestLength for data of any other length.
func (d *digest) UnmarshalBinary(data []byte) error {
	if len(data) != digestLength {
		return errDigestLength
	}
	*d = digest{set: true}
	copy(d.b[:], data)
	return nil
}

// SizeKanon returns digestLength, and 0 for the zero digest.
func (d digest) SizeKanon() int {
	if !d.set {
		return 0
	}
	return digestLength
}

// ExactKanon marks digest as a kanon.Exact type.
func (digest) ExactKanon() {}

// brittle is a word whose append method fails for math.MaxUint16, a value
// other than the zero value, which breaks the first guarantee.
type brittle uint16

// AppendBinary appends the two big-endian bytes of w to b. It fails with
// errBrittle for math.MaxUint16.
func (w brittle) AppendBinary(b []byte) ([]byte, error) {
	if w == math.MaxUint16 {
		return b, errBrittle
	}
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *brittle) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = brittle(x)
	return err
}

// SizeKanon returns wordLength.
func (brittle) SizeKanon() int { return wordLength }

// ExactKanon marks brittle as a kanon.Exact type.
func (brittle) ExactKanon() {}

// gapped is a word whose append method fails for gappedValue, which breaks
// the first guarantee at a value that the decode method alone gives.
type gapped uint16

// AppendBinary appends the two big-endian bytes of w to b. It fails with
// errGapped for gappedValue.
func (w gapped) AppendBinary(b []byte) ([]byte, error) {
	if w == gappedValue {
		return b, errGapped
	}
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *gapped) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = gapped(x)
	return err
}

// SizeKanon returns wordLength.
func (gapped) SizeKanon() int { return wordLength }

// ExactKanon marks gapped as a kanon.Exact type.
func (gapped) ExactKanon() {}

// overwriting is a word whose append method sets the first byte of its
// buffer to 0, which breaks the first guarantee.
type overwriting uint16

// AppendBinary appends the two big-endian bytes of w to b, and sets the
// first byte of the result to 0.
func (w overwriting) AppendBinary(b []byte) ([]byte, error) {
	b = binary.BigEndian.AppendUint16(b, uint16(w))
	b[0] = 0
	return b, nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *overwriting) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = overwriting(x)
	return err
}

// SizeKanon returns wordLength.
func (overwriting) SizeKanon() int { return wordLength }

// ExactKanon marks overwriting as a kanon.Exact type.
func (overwriting) ExactKanon() {}

// oversized is a word whose SizeKanon counts one byte more than its append
// method appends, which breaks the first guarantee.
type oversized uint16

// AppendBinary appends the two big-endian bytes of w to b.
func (w oversized) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *oversized) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = oversized(x)
	return err
}

// SizeKanon returns wordLength+1.
func (oversized) SizeKanon() int { return wordLength + 1 }

// ExactKanon marks oversized as a kanon.Exact type.
func (oversized) ExactKanon() {}

// padded is a word whose decode method also decodes its two bytes followed
// by a zero byte, which the append method does not write, and which breaks
// the second guarantee.
type padded uint16

// AppendBinary appends the two big-endian bytes of w to b.
func (w padded) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads the first two bytes of data,
// which is two bytes long, or three with a last byte of 0.
func (w *padded) UnmarshalBinary(data []byte) error {
	if len(data) == wordLength+1 && data[wordLength] == 0 {
		data = data[:wordLength]
	}
	x, err := readWord(data)
	*w = padded(x)
	return err
}

// SizeKanon returns wordLength.
func (padded) SizeKanon() int { return wordLength }

// ExactKanon marks padded as a kanon.Exact type.
func (padded) ExactKanon() {}

// sunken is a digest whose SizeKanon returns -1 for the zero value, which
// breaks the first guarantee.
type sunken struct {
	d digest
}

// AppendBinary appends the encoding of the digest of s to b, as the digest
// appends it.
func (s sunken) AppendBinary(b []byte) ([]byte, error) {
	return s.d.AppendBinary(b)
}

// UnmarshalBinary sets the digest of s to data, as the digest decodes it.
func (s *sunken) UnmarshalBinary(data []byte) error {
	return s.d.UnmarshalBinary(data)
}

// SizeKanon returns -1 for the zero sunken, and the SizeKanon of its digest
// otherwise.
func (s sunken) SizeKanon() int {
	if s == (sunken{}) {
		return -1
	}
	return s.d.SizeKanon()
}

// ExactKanon marks sunken as a kanon.Exact type.
func (sunken) ExactKanon() {}

// hushed is a word whose SizeKanon returns 0 for the zero word, whose
// append method appends two bytes, which breaks the first guarantee.
type hushed uint16

// AppendBinary appends the two big-endian bytes of w to b.
func (w hushed) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *hushed) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = hushed(x)
	return err
}

// SizeKanon returns 0 for the zero word, and wordLength otherwise.
func (w hushed) SizeKanon() int {
	if w == 0 {
		return 0
	}
	return wordLength
}

// ExactKanon marks hushed as a kanon.Exact type.
func (hushed) ExactKanon() {}

// stretch is eight bytes behind an unexported field, which appends its first
// two. Its decode method decodes two bytes, and eight, which no encoding
// has, which breaks the second guarantee.
type stretch struct {
	b [stretchLength]byte
}

// AppendBinary appends the first two bytes of s to b.
func (s stretch) AppendBinary(b []byte) ([]byte, error) {
	return append(b, s.b[:wordLength]...), nil
}

// UnmarshalBinary sets the first bytes of s to the two or eight bytes in
// data, and the others to 0. It fails with errStretchLength for data of any
// other length.
func (s *stretch) UnmarshalBinary(data []byte) error {
	if len(data) != wordLength && len(data) != stretchLength {
		return errStretchLength
	}
	*s = stretch{}
	copy(s.b[:], data)
	return nil
}

// SizeKanon returns wordLength.
func (stretch) SizeKanon() int { return wordLength }

// ExactKanon marks stretch as a kanon.Exact type.
func (stretch) ExactKanon() {}

// hollow is a digest whose decode method decodes the empty input to the zero
// hollow, which has no encoding, and which breaks the second guarantee.
type hollow struct {
	d digest
}

// AppendBinary appends the encoding of the digest of h to b, as the digest
// appends it.
func (h hollow) AppendBinary(b []byte) ([]byte, error) {
	return h.d.AppendBinary(b)
}

// UnmarshalBinary sets h to the zero hollow for empty data, and its digest
// to data otherwise, as the digest decodes it.
func (h *hollow) UnmarshalBinary(data []byte) error {
	if len(data) == 0 {
		*h = hollow{}
		return nil
	}
	return h.d.UnmarshalBinary(data)
}

// SizeKanon returns digestLength.
func (hollow) SizeKanon() int { return digestLength }

// ExactKanon marks hollow as a kanon.Exact type.
func (hollow) ExactKanon() {}

// marshaled is a word that encodes itself through MarshalBinary alone, and
// has no append method that kanon.Exact requires.
type marshaled uint16

// MarshalBinary returns the two big-endian bytes of w.
func (w marshaled) MarshalBinary() ([]byte, error) {
	return binary.BigEndian.AppendUint16(nil, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *marshaled) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = marshaled(x)
	return err
}

// SizeKanon returns wordLength.
func (marshaled) SizeKanon() int { return wordLength }

// ExactKanon marks marshaled as a kanon.Exact type.
func (marshaled) ExactKanon() {}

// validated is a word that is a kanon.Validator, which kanon encodes as its
// underlying type and not through its append method.
type validated uint16

// ValidateKanon returns nil.
func (validated) ValidateKanon() error { return nil }

// AppendBinary appends the two big-endian bytes of w to b.
func (w validated) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint16(b, uint16(w)), nil
}

// UnmarshalBinary sets w as [readWord] reads it.
func (w *validated) UnmarshalBinary(data []byte) error {
	x, err := readWord(data)
	*w = validated(x)
	return err
}

// SizeKanon returns wordLength.
func (validated) SizeKanon() int { return wordLength }

// ExactKanon marks validated as a kanon.Exact type.
func (validated) ExactKanon() {}

// ratio is a float64 that appends its bits, whose == does not compare every
// bit: it reports -0.0 equal to +0.0.
type ratio float64

// AppendBinary appends the eight big-endian bytes of the bits of r to b.
func (r ratio) AppendBinary(b []byte) ([]byte, error) {
	return binary.BigEndian.AppendUint64(b, math.Float64bits(float64(r))), nil
}

// UnmarshalBinary sets r to the bits in the first eight bytes of data.
func (r *ratio) UnmarshalBinary(data []byte) error {
	*r = ratio(math.Float64frombits(binary.BigEndian.Uint64(data)))
	return nil
}

// SizeKanon returns 8.
func (ratio) SizeKanon() int { return 8 }

// ExactKanon marks ratio as a kanon.Exact type.
func (ratio) ExactKanon() {}

// fn is a function type that declares kanon.Exact, which kanon does not
// encode.
type fn func()

// SizeKanon returns 0.
func (fn) SizeKanon() int { return 0 }

// ExactKanon marks fn as a kanon.Exact type.
func (fn) ExactKanon() {}

// rejectsExact runs the check of kanontest.ExactChecks for T named name, and
// fails t unless the check fails with a message that contains want.
func rejectsExact[T kanon.Exact](t *testing.T, name, want string) {
	t.Helper()
	for _, c := range kanontest.ExactChecks[T]() {
		if c.Name == name {
			got := assert.Rejects(t, name, c.Run)
			assert.Contains(t, got, want, name+" fails for the reason that it states")
			return
		}
	}
	t.Fatalf("no check is named %q", name)
}

func TestExact(t *testing.T) {
	t.Parallel()
	t.Run("ExactChecks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the check of each guarantee for a type that meets the requirements", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.ExactChecks[word]()
			names := make([]string, 0, len(checks))
			for _, c := range checks {
				names = append(names, c.Name)
			}
			assert.Equal(t, names, []string{exactSizeCheck, exactAppendCheck, exactErrorCheck, exactDecodeCheck},
				"ExactChecks returns the size, append, error and decode checks")
		})
		t.Run("returns one check for a type without an append method", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.ExactChecks[marshaled]()
			assert.Length(t, checks, 1, "ExactChecks returns the check of the type alone")
			assert.Equal(t, checks[0].Name, exactTypeCheck, "ExactChecks names the check of the type")
		})
	})
	t.Run("ExactKanon", func(t *testing.T) {
		t.Parallel()
		rejections := []struct {
			name  string
			check string
			run   func(t *testing.T, name, want string)
			want  string
		}{
			{
				name:  "fails for a type without an append method",
				check: exactTypeCheck,
				run:   rejectsExact[marshaled],
				want: "kanontest_test.marshaled declares ExactKanon, and does not encode itself through " +
					"AppendBinary or AppendText",
			},
			{
				name:  "fails for a kanon.Validator",
				check: exactTypeCheck,
				run:   rejectsExact[validated],
				want: "kanontest_test.validated declares ExactKanon, and does not encode itself through " +
					"AppendBinary or AppendText",
			},
			{
				name:  "fails for a type whose == does not compare every bit",
				check: exactTypeCheck,
				run:   rejectsExact[ratio],
				want:  "kanontest_test.ratio declares ExactKanon, and == does not compare every bit of it",
			},
			{
				name:  "fails for a type that kanon does not encode",
				check: exactTypeCheck,
				run:   rejectsExact[fn],
				want:  "kanontest: kanontest_test.fn: ",
			},
			{
				name:  "fails for a SizeKanon below 0 for the zero value",
				check: exactSizeCheck,
				run:   rejectsExact[sunken],
				want:  "the zero value: SizeKanon returns no negative value\ngot:  -1",
			},
			{
				name:  "fails for a zero value that appends another length than its SizeKanon",
				check: exactAppendCheck,
				run:   rejectsExact[hushed],
				want: "the zero value: the append method appends SizeKanon bytes to its buffer\ngot:  aaaa0000\n" +
					"want: aaaa and 0 bytes after it",
			},
			{
				name:  "fails for an append method that changes its buffer",
				check: exactAppendCheck,
				run:   rejectsExact[overwriting],
				want:  "the zero value: the append method appends SizeKanon bytes to its buffer\ngot:  00aa0000",
			},
			{
				name:  "fails for a SizeKanon that differs from the length of the encoding",
				check: exactAppendCheck,
				run:   rejectsExact[oversized],
				want:  "want: aaaa and 3 bytes after it",
			},
			{
				name:  "fails for an append method that fails for a value other than the zero value",
				check: exactErrorCheck,
				run:   rejectsExact[brittle],
				want: "value 6 of the value tables: the append method encodes a value other than the zero value " +
					"without an error\ngot:  error kanontest_test: the word is math.MaxUint16",
			},
			{
				name:  "fails for an append method that fails for a value that the decode method alone gives",
				check: exactErrorCheck,
				run:   rejectsExact[gapped],
				want: "the value that the decode method decodes from 0200: the append method encodes a value other " +
					"than the zero value without an error",
			},
			{
				name:  "fails for a decode method that accepts bytes that the append method does not write",
				check: exactDecodeCheck,
				run:   rejectsExact[padded],
				want:  "input 000000: the append method writes the input",
			},
			{
				name:  "fails for a decode method that accepts a length that no encoding has",
				check: exactDecodeCheck,
				run:   rejectsExact[stretch],
				want:  "input 0102030405060708: the append method writes the input",
			},
			{
				name:  "fails for a decode method that decodes a value that the append method does not encode",
				check: exactDecodeCheck,
				run:   rejectsExact[hollow],
				want:  "got:  , error kanontest_test: the zero digest has no encoding",
			},
		}
		for _, tt := range rejections {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				tt.run(t, tt.check, tt.want)
			})
		}
	})
	t.Run("RunExact", func(t *testing.T) {
		t.Parallel()
		t.Run("passes a type that keeps both guarantees", func(t *testing.T) {
			t.Parallel()
			kanontest.RunExact[word](t)
		})
		t.Run("passes a type whose append method fails for the zero value", func(t *testing.T) {
			t.Parallel()
			kanontest.RunExact[digest](t)
		})
	})
}

// readWord returns the big-endian uint16 in data, which is wordLength bytes
// long, and errWordLength for data of any other length.
func readWord(data []byte) (uint16, error) {
	if len(data) != wordLength {
		return 0, errWordLength
	}
	return binary.BigEndian.Uint16(data), nil
}
