// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
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
	// exactSizeCheck names the check that SizeKanon returns no negative
	// value.
	exactSizeCheck = "ExactKanon/sizes every value at 0 bytes or more"
	// exactAppendCheck names the check of the length of an encoding.
	exactAppendCheck = "ExactKanon/appends SizeKanon bytes whenever the append method returns no error"
	// exactErrorCheck names the check that the append method fails for no
	// value but the zero value.
	exactErrorCheck = "ExactKanon/appends every value but the zero value without an error"
	// exactDecodeCheck names the check of the second guarantee.
	exactDecodeCheck = "ExactKanon/decodes only the bytes that the append method writes for the decoded value"
	// exactAppenderCheck names the check of a kanon.Appender that AppendKanon
	// appends the bytes of the append method of its family.
	exactAppenderCheck = "AppendKanon/appends the bytes that the append method appends for every value"
)

// exactSuite is the conformance suite of a type that declares kanon.Exact:
// the values of the type that the checks of the first guarantee run on, and
// the inputs that its decode method accepts, with the value that it decodes
// from each.
type exactSuite struct {
	// values are the zero value and the values that entry 0 to 7 of the
	// value tables build, as [resolver.self] builds them.
	values []exactValue
	// decoded are the values that the decode method decodes from the inputs
	// that [newExactSuite] derives, for each input that it accepts, in their
	// order.
	decoded []exactValue
}

// exactValue is a value of a kanon.Exact type that the checks run on.
type exactValue struct {
	v reflect.Value
	// name names the value in failures, and is empty for a decoded value,
	// whose input names it.
	name string
	// input is the input that the decode method decodes a decoded value
	// from.
	input []byte
}

// String returns the name of x in failures.
func (x exactValue) String() string {
	if x.name != "" {
		return x.name
	}
	return fmt.Sprintf("the value that the decode method decodes from %x", x.input)
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
	es := &exactSuite{values: []exactValue{{v: reflect.New(t).Elem(), name: "the zero value"}}}
	var inputs [][]byte
	for i := range drawCount {
		v := reflect.New(t).Elem()
		r.self(table(i), v, 0)
		es.values = append(es.values, exactValue{v: v, name: "value " + strconv.Itoa(i) + " of the value tables"})
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
			es.decoded = append(es.decoded, exactValue{v: v, input: b})
		}
	}
	return es, nil
}

// all returns the values that the checks of the first guarantee of
// kanon.Exact run on: the values of es, then the decoded values.
func (es *exactSuite) all() []exactValue {
	return slices.Concat(es.values, es.decoded)
}

// size checks that SizeKanon returns no negative value for any value of es.
// It fails with an in-range record of the size.
func (es *exactSuite) size(tb assert.TB) {
	tb.Helper()
	for _, x := range es.all() {
		if n := sizeKanon(x.v); n < 0 {
			assert.InRange(tb, n, 0, math.MaxInt, x.String()+": SizeKanon returns no negative value")
		}
	}
}

// append checks that the append method appends as many bytes as SizeKanon
// returns to a buffer of two guard bytes, which it keeps, for every value of
// es for which it returns no error. It fails with a has-prefix record of a
// buffer that lost its guard bytes, and with a length record of an encoding
// of another length than SizeKanon.
func (es *exactSuite) append(tb assert.TB) {
	tb.Helper()
	prefix := []byte{guard, guard}
	for _, x := range es.all() {
		n := sizeKanon(x.v)
		got, err := appendTo(x.v, []byte{guard, guard})
		if err == nil && (!bytes.HasPrefix(got, prefix) || len(got)-len(prefix) != n) {
			contract := x.String() + ": the append method appends SizeKanon bytes to its buffer"
			assert.HasPrefix(tb, got, string(prefix), contract)
			assert.Length(tb, got, len(prefix)+n, contract)
		}
	}
}

// succeed checks that the append method returns no error for every value of
// es other than the zero value. It fails with an err-absent record of the
// error.
func (es *exactSuite) succeed(tb assert.TB) {
	tb.Helper()
	for _, x := range es.all() {
		if _, err := appendTo(x.v, nil); err != nil && !x.v.IsZero() {
			assert.NoError(tb, err,
				x.String()+": the append method encodes a value other than the zero value without an error")
		}
	}
}

// appender checks that AppendKanon, of a kanon.Appender, appends to a buffer
// of two guard bytes what the append method of its family appends to it
// without an error, for every value of es, the zero value included. It fails
// with an err-absent record of the error of the append method, and with an
// equal record of the bytes.
func (es *exactSuite) appender(tb assert.TB) {
	tb.Helper()
	for _, x := range es.all() {
		want, err := appendTo(x.v, []byte{guard, guard})
		if got := appendKanon(x.v, []byte{guard, guard}); err != nil || !bytes.Equal(got, want) {
			contract := x.String() + ": AppendKanon appends the bytes that the append method appends"
			assert.NoError(tb, err, contract)
			assert.Equal(tb, got, want, contract)
		}
	}
}

// decode checks the second guarantee of kanon.Exact: for each input that
// the decode method accepts, the append method writes the input for the
// value that the decode method sets. It fails with an err-absent record of
// the error of the append method, and with an equal record of the bytes.
func (es *exactSuite) decode(tb assert.TB) {
	tb.Helper()
	for _, x := range es.decoded {
		enc, err := marshal(x.v)
		if err != nil || !bytes.Equal(enc, x.input) {
			contract := "input " + hex.EncodeToString(x.input) +
				": the append method writes the input for the value that the decode method decodes from it"
			assert.NoError(tb, err, contract)
			assert.Equal(tb, enc, x.input, contract)
		}
	}
}

// ExactChecks returns the checks of T, a type that declares kanon.Exact,
// over its zero value, the values that entry 0 to 7 of the value tables
// build for it and the values that its decode method decodes:
//
//   - SizeKanon returns no negative value.
//   - The append method appends as many bytes to its buffer as SizeKanon
//     returns whenever it returns no error.
//   - The append method returns no error for a value other than the zero
//     value.
//   - The decode method accepts a byte string only when the append method
//     writes that byte string for the value that the decode method sets. The
//     check decodes the encoding of every value of the value tables, every
//     prefix of it, every change of one of its bytes and the encoding with
//     one more byte, and a byte string of every length up to the size of T
//     in memory.
//   - For a kanon.Appender, AppendKanon appends the bytes that the append
//     method appends, which returns no error, for every value, the zero
//     value included.
//
// For a type that kanon does not encode, and for a type that does not meet
// the requirements of kanon.Exact that the generator checks, ExactChecks
// returns one check, which fails with the reason: T encodes itself through
// AppendBinary or AppendText, and == compares every bit of it.
func ExactChecks[T kanon.Exact]() []Check {
	typ := reflect.TypeFor[T]()
	es, err := newExactSuite(typ)
	if err != nil {
		return []Check{{Name: exactTypeCheck, Run: func(tb assert.TB) {
			tb.Helper()
			assert.NoError(tb, err, "the type meets the requirements of kanon.Exact")
		}}}
	}
	checks := []Check{
		{Name: exactSizeCheck, Run: es.size},
		{Name: exactAppendCheck, Run: es.append},
		{Name: exactErrorCheck, Run: es.succeed},
		{Name: exactDecodeCheck, Run: es.decode},
	}
	if appendsKanon(typ) {
		checks = append(checks, Check{Name: exactAppenderCheck, Run: es.appender})
	}
	return checks
}

// RunExact runs the checks of T that [ExactChecks] returns, each as a
// parallel subtest of t.
func RunExact[T kanon.Exact](t *testing.T) {
	t.Helper()
	run(t, ExactChecks[T]())
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
