// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// Names of the checks of [RunExact]: the method whose contract a check
// states, a slash, and the behaviour. ExactKanon marks the guarantees of
// kanon.Exact, which the append method, SizeKanon and the decode method of
// the type keep.
const (
	// exactTypeCheck names the one check of a type that does not meet the
	// requirements of kanon.Exact.
	exactTypeCheck = "ExactKanon/marks a type with an append method whose == compares every bit"
	// exactAppendCheck names the check of the first guarantee.
	exactAppendCheck = "ExactKanon/appends SizeKanon bytes without an error for a value other than the zero value"
	// exactDecodeCheck names the check of the second guarantee.
	exactDecodeCheck = "ExactKanon/decodes only the bytes that the append method writes for the decoded value"
)

// exactSuite is the conformance suite of a type that declares kanon.Exact:
// the values of the type that the value tables build, and the inputs that
// its decode method accepts, with the value that it decodes from each.
type exactSuite struct {
	// values are the values that entry 0 to 7 of the value tables build, as
	// [resolver.self] builds them.
	values []reflect.Value
	// decoded are the inputs that the decode method accepts among those that
	// [newExactSuite] derives, in their order.
	decoded []decoding
}

// decoding is an input that the decode method of a type accepts, and the
// value that it decodes from the input.
type decoding struct {
	input []byte
	value reflect.Value
}

// newExactSuite returns the suite of t. Its inputs are the encoding of each
// value of the value tables that encodes, as [variants] derives inputs from
// it, and the bytes that [counted] returns for every length from 0 to the
// size of a value of t in memory. It fails for a type that kanon does not
// encode, and for a type that does not meet the requirements of kanon.Exact
// that the generator checks: t encodes itself through the append method of
// its family, as [appends] reports, and == compares every bit of it, as
// [bitwiseType] reports.
func newExactSuite(t reflect.Type) (*exactSuite, error) {
	r := resolverFor(t.PkgPath())
	s, err := r.shapeOf(t, opts{seen: make(map[seenKey]*shape), loose: true})
	if err != nil {
		return nil, fmt.Errorf("kanontest: %v: %w", t, err)
	}
	if s.kind != kindBinary || !appends(t) {
		return nil, fmt.Errorf("kanontest: %v declares ExactKanon, and does not encode itself through AppendBinary or "+
			"AppendText", t)
	}
	if !s.zeroAbsent {
		return nil, fmt.Errorf("kanontest: %v declares ExactKanon, and == does not compare every bit of it", t)
	}
	es := &exactSuite{}
	var inputs [][]byte
	for i := range drawCount {
		v := reflect.New(t).Elem()
		r.self(table(i), v, 0)
		es.values = append(es.values, v)
		if enc, err := marshal(v); err == nil {
			inputs = append(inputs, variants(enc)...)
		}
	}
	for n := range int(t.Size()) + 1 {
		inputs = append(inputs, counted(n))
	}
	for _, b := range inputs {
		v := reflect.New(t).Elem()
		if unmarshal(v, b) == nil {
			es.decoded = append(es.decoded, decoding{input: b, value: v})
		}
	}
	return es, nil
}

// append checks the first guarantee of kanon.Exact, as [appendBreach] states
// it, for the values of the value tables and for the decoded values.
func (es *exactSuite) append(tb assert.TB) {
	tb.Helper()
	for i, v := range es.values {
		if msg := appendBreach(v); msg != "" {
			tb.Fatalf("value %d of the value tables: %s", i, msg)
		}
	}
	for _, d := range es.decoded {
		if msg := appendBreach(d.value); msg != "" {
			tb.Fatalf("the value that the decode method decodes from %x: %s", d.input, msg)
		}
	}
}

// decode checks the second guarantee of kanon.Exact: for each input that
// the decode method accepts, the append method writes the input for the
// value that the decode method sets.
func (es *exactSuite) decode(tb assert.TB) {
	tb.Helper()
	for _, d := range es.decoded {
		enc, err := marshal(d.value)
		if err != nil || !bytes.Equal(enc, d.input) {
			tb.Fatalf("input %x: the append method writes the input for the value that the decode method decodes "+
				"from it\ngot:  %x, error %v", d.input, enc, err)
		}
	}
}

// ExactChecks returns the checks of T, a type that declares kanon.Exact,
// over the values that entry 0 to 7 of the value tables build for it and the
// values that its decode method decodes:
//
//   - For every such value other than the zero value, the append method
//     returns no error and appends as many bytes to its buffer as SizeKanon
//     returns.
//   - The decode method accepts a byte string only when the append method
//     writes that byte string for the value that the decode method sets. The
//     check decodes the encoding of every value of the value tables, every
//     prefix of it, every change of one of its bytes and the encoding with
//     one more byte, and a byte string of every length up to the size of T
//     in memory.
//
// For a type that kanon does not encode, and for a type that does not meet
// the requirements of kanon.Exact that the generator checks, ExactChecks
// returns one check, which fails with the reason: T encodes itself through
// AppendBinary or AppendText, and == compares every bit of it.
func ExactChecks[T kanon.Exact]() []Check {
	es, err := newExactSuite(reflect.TypeFor[T]())
	if err != nil {
		return []Check{{Name: exactTypeCheck, Run: func(tb assert.TB) {
			tb.Helper()
			assert.NoError(tb, err, "the type meets the requirements of kanon.Exact")
		}}}
	}
	return []Check{{Name: exactAppendCheck, Run: es.append}, {Name: exactDecodeCheck, Run: es.decode}}
}

// RunExact runs the checks of T that [ExactChecks] returns, each as a
// parallel subtest of t.
func RunExact[T kanon.Exact](t *testing.T) {
	t.Helper()
	run(t, ExactChecks[T]())
}

// appendBreach returns how the append method of v, a value of a kanon.Exact
// type, breaks the first guarantee of kanon.Exact, and "" when it keeps it:
// for a value other than the zero value, the method returns no error and
// appends as many bytes as SizeKanon returns to a buffer of two guard bytes,
// which it keeps.
func appendBreach(v reflect.Value) string {
	if v.IsZero() {
		return ""
	}
	prefix := []byte{guard, guard}
	n := sizeKanon(v)
	got, err := appendTo(v, []byte{guard, guard})
	if err == nil && bytes.HasPrefix(got, prefix) && len(got)-len(prefix) == n {
		return ""
	}
	return fmt.Sprintf("the append method appends SizeKanon bytes to its buffer without an error\n"+
		"got:  %x, error %v\nwant: %x and %d bytes after it, error <nil>", got, err, prefix, n)
}

// variants returns the inputs that the decode check derives from enc, the
// encoding of a value: enc, every prefix of it, enc with each of its bytes
// changed to each other value of a byte, which adds 1 to 255 to the byte
// modulo 256, and enc with one more byte, 0. Each input is a copy.
func variants(enc []byte) [][]byte {
	out := [][]byte{bytes.Clone(enc)}
	for n := range len(enc) {
		out = append(out, bytes.Clone(enc[:n]))
	}
	for k := range enc {
		for d := byte(1); d != 0; d++ {
			b := bytes.Clone(enc)
			b[k] += d
			out = append(out, b)
		}
	}
	return append(out, append(bytes.Clone(enc), 0))
}
