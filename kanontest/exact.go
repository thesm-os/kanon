// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"slices"
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
	// value tables build, as [tableValues] builds them.
	values []markedValue
	// decoded are the values that the decode method decodes from the inputs
	// that [decodedFrom] derives from the values of the value tables.
	decoded []markedValue
}

// newExactSuite returns the suite of t, whose decoded values [decodedFrom]
// derives from the values of the value tables. It fails for a type that kanon
// does not encode, and for a type that does not meet the requirements of
// kanon.Exact that the generator checks: t encodes itself through the append
// method of its family, as [methodsOf] reports, and == compares every bit of
// it, as [bitwiseType] reports.
func newExactSuite(t reflect.Type) (*exactSuite, error) {
	r := resolverFor(t.PkgPath())
	s, err := r.shapeOf(t, opts{seen: make(map[seenKey]*shape), loose: true})
	if err != nil {
		return nil, fmt.Errorf("kanontest: %v: %w", t, err)
	}
	if s.kind != kindBinary || !methodsOf(t).appends {
		return nil, fmt.Errorf("kanontest: %v declares ExactKanon, and does not encode itself through AppendBinary or "+
			"AppendText", t)
	}
	if !s.zeroAbsent {
		return nil, fmt.Errorf("kanontest: %v declares ExactKanon, and == does not compare every bit of it", t)
	}
	values := tableValues(r, t)
	return &exactSuite{
		values:  append([]markedValue{{v: reflect.New(t).Elem(), name: "the zero value"}}, values...),
		decoded: decodedFrom(t, values),
	}, nil
}

// all returns the values that the checks of the first guarantee of
// kanon.Exact run on: the values of es, then the decoded values.
func (es *exactSuite) all() []markedValue {
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
			assert.That(tb, got).
				HasPrefix(string(prefix), contract).
				Length(len(prefix)+n, contract)
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

// decode checks the second guarantee of kanon.Exact, as [rewrites] checks it
// with the append method of the family.
func (es *exactSuite) decode(tb assert.TB) {
	tb.Helper()
	rewrites(tb, es.decoded, "the append method")
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
	if methodsOf(typ).appender {
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
