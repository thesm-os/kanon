// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.thesmos.sh/kanon"
)

// guard fills the bytes of a buffer that an encoding must leave alone.
const guard = 0xaa

// Location of the golden file of a struct type,
// testdata/golden/<package>.<T>.kanon.golden in the directory of the package
// whose test runs the checks.
const (
	goldenDir    = "testdata/golden"
	goldenSuffix = ".kanon.golden"
	// goldenFails precedes the error of a sample that fails to encode in
	// the golden file.
	goldenFails = "fails: "
)

// golden checks that the golden file of T matches these lines, which the
// -update flag of the test binary writes:
//
//   - per sample, its name and the encoding that MarshalBinary returns in
//     hex, or its error when it fails to encode, which are the test vectors
//     of the codec;
//   - per family of probes, its check and the digest of its probes;
//   - per allocation measure, the number of the samples that it measures and
//     the digest of their names.
func (s *suite[T, P]) golden(tb assert.TB) {
	tb.Helper()
	var b strings.Builder
	for _, x := range s.all() {
		v := x.value
		b.WriteString(x.name + ": ")
		if enc, err := P(&v).MarshalBinary(); err != nil {
			b.WriteString(goldenFails + err.Error())
		} else {
			b.WriteString(hex.EncodeToString(enc))
		}
		b.WriteByte('\n')
	}
	for _, f := range s.families() {
		b.WriteString(f.check + ": " + digest(f.probes()) + "\n")
	}
	b.WriteString("the samples that the encode allocation checks measure: " +
		names(s.measured(encodeAllocates)) + "\n")
	b.WriteString("the samples that the decode allocation check measures: " +
		names(s.measured(s.r.decodeAllocates)) + "\n")
	path := filepath.Join(filepath.FromSlash(goldenDir), s.l.typ.String()+goldenSuffix)
	golden.MatchAt(tb, path, []byte(b.String()), golden.ShouldUpdate())
}

// size checks that SizeKanon returns the length of the reference encoding
// of every sample that encodes.
func (s *suite[T, P]) size(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		v := x.value
		assert.Equal(
			tb,
			P(&v).SizeKanon(),
			len(x.enc),
			x.name+": SizeKanon returns the length of the reference encoding",
		)
	}
}

// marshal checks that MarshalBinary returns the reference encoding of every
// sample that encodes.
func (s *suite[T, P]) marshal(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		v := x.value
		got, err := P(&v).MarshalBinary()
		assert.NoError(tb, err, x.name+": MarshalBinary encodes a value whose fields encode")
		assert.Equal(tb, got, x.enc, x.name+": MarshalBinary returns the reference encoding")
	}
}

// marshalError checks that MarshalBinary returns the error of the reference
// encoding for every sample that fails to encode: the error of the value
// that the generated code meets first.
func (s *suite[T, P]) marshalError(tb assert.TB) {
	tb.Helper()
	for _, x := range s.failing() {
		v := x.value
		_, err := P(&v).MarshalBinary()
		assert.Equal(tb, err, x.err, x.name+": MarshalBinary returns the error of the value that fails to encode")
	}
}

// append checks that AppendBinary appends the reference encoding of every
// sample that encodes to the bytes of its buffer.
func (s *suite[T, P]) append(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		v := x.value
		got, err := P(&v).AppendBinary([]byte{guard, guard})
		assert.NoError(tb, err, x.name+": AppendBinary encodes a value whose fields encode")
		assert.Equal(tb, got, append([]byte{guard, guard}, x.enc...),
			x.name+": AppendBinary appends the reference encoding to the buffer")
	}
}

// encode checks that EncodeKanon writes the reference encoding of every
// sample that encodes into the end of a longer buffer, and returns its
// length.
func (s *suite[T, P]) encode(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		v := x.value
		buf := bytes.Repeat([]byte{guard}, len(x.enc)+2)
		n, err := P(&v).EncodeKanon(buf)
		assert.NoError(tb, err, x.name+": EncodeKanon encodes a value whose fields encode")
		assert.Equal(tb, n, len(x.enc), x.name+": EncodeKanon returns the length of the encoding")
		assert.Equal(tb, buf, append([]byte{guard, guard}, x.enc...),
			x.name+": EncodeKanon writes the reference encoding into the end of the buffer")
	}
}

// short checks that EncodeKanon writes nothing into a buffer one byte
// shorter than the reference encoding of a sample, and returns
// io.ErrShortBuffer.
func (s *suite[T, P]) short(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		if len(x.enc) == 0 {
			continue
		}
		v := x.value
		buf := bytes.Repeat([]byte{guard}, len(x.enc)-1)
		n, err := P(&v).EncodeKanon(buf)
		assert.ErrorIs(tb, err, io.ErrShortBuffer, x.name+": EncodeKanon returns io.ErrShortBuffer for a short buffer")
		assert.Equal(tb, n, 0, x.name+": EncodeKanon returns length 0 for a short buffer")
		assert.Equal(tb, buf, bytes.Repeat([]byte{guard}, len(x.enc)-1),
			x.name+": EncodeKanon writes nothing into a short buffer")
	}
}

// nilSize checks that SizeKanon returns 0 for a nil receiver.
func (*suite[T, P]) nilSize(tb assert.TB) {
	tb.Helper()
	var p P
	assert.Equal(tb, p.SizeKanon(), 0, "SizeKanon returns 0 for a nil receiver")
}

// nilEncode checks that EncodeKanon writes nothing for a nil receiver and
// returns 0 and no error.
func (*suite[T, P]) nilEncode(tb assert.TB) {
	tb.Helper()
	var p P
	buf := []byte{guard}
	n, err := p.EncodeKanon(buf)
	assert.NoError(tb, err, "EncodeKanon returns no error for a nil receiver")
	assert.Equal(tb, n, 0, "EncodeKanon returns length 0 for a nil receiver")
	assert.Equal(tb, buf, []byte{guard}, "EncodeKanon writes nothing for a nil receiver")
}

// nilAppend checks that AppendBinary returns its buffer unchanged for a nil
// receiver.
func (*suite[T, P]) nilAppend(tb assert.TB) {
	tb.Helper()
	var p P
	got, err := p.AppendBinary([]byte{guard})
	assert.NoError(tb, err, "AppendBinary returns no error for a nil receiver")
	assert.Equal(tb, got, []byte{guard}, "AppendBinary returns the buffer for a nil receiver")
}

// unmarshal checks that UnmarshalBinary decodes the reference encoding of
// every sample that encodes as the reference decode, and that the reference
// encoding of a value that it decodes is that encoding.
func (s *suite[T, P]) unmarshal(tb assert.TB) {
	tb.Helper()
	for _, x := range s.encodable() {
		var got T
		err := P(&got).UnmarshalBinary(x.enc)
		want, wantErr := s.reference(probe{data: x.enc})
		s.same(tb, x.name+": UnmarshalBinary", got, err, want, wantErr)
		if err == nil {
			enc, _ := s.r.encode(s.l, reflect.ValueOf(&got).Elem())
			assert.Equal(tb, enc, x.enc, x.name+": the reference encoding of the decoded value is the encoding")
		}
	}
}

// probing returns the check that decodes the probes of a family, which
// probes returns, as the reference decode.
func (s *suite[T, P]) probing(probes func() []probe) func(assert.TB) {
	return func(tb assert.TB) {
		tb.Helper()
		for _, p := range probes() {
			s.decode(tb, p)
		}
	}
}

// decode checks that DecodeKanon decodes the probe p into the zero value of
// T as the reference decode.
func (s *suite[T, P]) decode(tb assert.TB, p probe) {
	tb.Helper()
	var got T
	err := P(&got).DecodeKanon(p.data, p.opts)
	want, wantErr := s.reference(p)
	s.same(tb, p.name+": DecodeKanon", got, err, want, wantErr)
}

// same checks that the decode that op names, which returned got and err,
// decodes as the reference decode, which returned want and wantErr: it
// returns the same error, as sameError compares them, and without an error
// a value with the same fingerprint.
func (s *suite[T, P]) same(tb assert.TB, op string, got T, err error, want T, wantErr error) {
	tb.Helper()
	if !sameError(err, wantErr) {
		tb.Fatalf("%s returns the error of the reference decode\ngot:  %v\nwant: %v", op, err, wantErr)
	}
	if wantErr != nil || reflect.DeepEqual(got, want) {
		return
	}
	assert.Equal(tb, s.r.fingerprint(s.l, reflect.ValueOf(&got).Elem()),
		s.r.fingerprint(s.l, reflect.ValueOf(&want).Elem()),
		op+" decodes the value of the reference decode, which their fingerprints compare")
}

// causes lists the causes of a decode error that kanon defines.
var causes = [...]error{
	io.ErrUnexpectedEOF, kanon.ErrMalformed, kanon.ErrRange, kanon.ErrDepth, kanon.ErrUnknownType,
	kanon.ErrRepeatedView, kanon.ErrInvalidKey, kanon.ErrAmbiguousKey,
}

// sameError reports whether a and b are the same error of a decode: both
// nil, or two errors with the same message, which names the location, the
// offset and the cause, that wrap the same causes of causes. The error of
// a type that decodes itself can be a new value on every call, so that two
// such causes are the same when their messages are.
func sameError(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.Error() != b.Error() {
		return false
	}
	for _, c := range causes {
		if errors.Is(a, c) != errors.Is(b, c) {
			return false
		}
	}
	return true
}

// encodable returns the samples that encode, the bulk samples included.
func (s *suite[T, P]) encodable() []sample[T] {
	return encodes(s.all())
}

// failing returns the samples that fail to encode.
func (s *suite[T, P]) failing() []sample[T] {
	return slices.DeleteFunc(s.all(), func(x sample[T]) bool { return x.err == nil })
}

// encodes returns the samples of samples that encode, in their order.
func encodes[T any](samples []sample[T]) []sample[T] {
	return slices.DeleteFunc(slices.Clone(samples), func(x sample[T]) bool { return x.err != nil })
}
