// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"
	"go.dokimi.dev/assert/prop"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// Names of the checks of [RunValue]: the method whose contract a check
// states, a slash, and the behaviour.
const (
	// valueTypeCheck names the one check of a type that RunValue does not
	// validate.
	valueTypeCheck = "ValidateKanon/validates a named type that is not a struct"
	// valueAllocsCheck names the check of the allocation contract.
	valueAllocsCheck = "ValidateKanon/allocates nothing for a value that it accepts"
	// valueDomainCheck names the check of the values that the method
	// accepts against those that the encode method of the type accepts.
	valueDomainCheck = "ValidateKanon/accepts a value exactly when the encode method of the type accepts it"
	// valuePropertyCheck names the check of the domain on generated values.
	valuePropertyCheck = "ValidateKanon/accepts a generated value exactly when the encode method of the type accepts it"
	// valueGoldenCheck names the check of the golden file.
	valueGoldenCheck = "ValidateKanon/matches the golden file of the values"
)

// valuePropertyContract is the contract of the record of the property of
// [RunValue], which names its cases in the store of the test.
const valuePropertyContract = "ValidateKanon accepts a generated value exactly when the encode method of the type " +
	"accepts it"

// valueField names the one field of the struct in which the golden file of
// [RunValue] encodes each value.
const valueField = "Value"

// valueSuite is the conformance suite of a kanon.Validator: the shape of
// the type and the values that the value tables build for it.
type valueSuite struct {
	r *resolver
	s *shape
	// values are the values that entry i of the value tables builds, as
	// [resolver.shaped] builds them.
	values []reflect.Value
}

// newValueSuite returns the suite of t. It fails for a struct and an
// interface type, which kanon does not validate, and for a type that kanon
// does not encode.
func newValueSuite(t reflect.Type) (*valueSuite, error) {
	if t.Kind() == reflect.Struct || t.Kind() == reflect.Interface {
		return nil, fmt.Errorf("kanontest: %v is a struct or an interface type, which kanon does not validate", t)
	}
	r := resolverFor(t.PkgPath())
	s, err := r.shapeOf(t, opts{seen: make(map[seenKey]*shape), loose: true})
	if err != nil {
		return nil, fmt.Errorf("kanontest: %v: %w", t, err)
	}
	r.prepare(s)
	vs := &valueSuite{r: r, s: s}
	for i := range drawCount {
		vs.values = append(vs.values, r.shaped(table(i), s, nesting))
	}
	return vs, nil
}

// domain checks each value of vs as [valueSuite.accepts] checks it.
func (vs *valueSuite) domain(tb assert.TB) {
	tb.Helper()
	for i, v := range vs.values {
		vs.accepts(tb, "value "+strconv.Itoa(i), v)
	}
}

// property checks, on the cases of a run of prop.ForAll, the value of the
// type of vs that each case draws, as [valueSuite.accepts] checks it. The
// builder builds the value from a drawn source, as [resolver.shaped] builds
// the values of the value tables, so that ValidateKanon meets the values
// that it rejects as well.
func (vs *valueSuite) property(tb assert.TB) {
	tb.Helper()
	prop.ForAll(tb, valuePropertyContract, func(c *prop.Case) {
		x := c.Draw(prop.Composite(func(c *prop.Case) any {
			return vs.r.shaped(drawn{c: c, path: firstValue}, vs.s, nesting).Interface()
		}), firstValue)
		v := reflect.New(vs.s.typ).Elem()
		v.Set(reflect.ValueOf(x))
		vs.accepts(c, generatedName+firstValue, v)
	})
}

// accepts checks that ValidateKanon accepts v, a value of the type of vs
// that name names, exactly when the encode method of the family of its
// type, which [marshal] calls, accepts it, and that the decode method of the
// family decodes the encoding of an accepted v back to v, as their
// fingerprints compare.
func (vs *valueSuite) accepts(tb assert.TB, name string, v reflect.Value) {
	tb.Helper()
	enc, err := marshal(v)
	assert.Equal(tb, validate(v) == nil, err == nil,
		name+": ValidateKanon accepts it exactly when the encode method of the type does")
	if err != nil {
		return
	}
	back := reflect.New(vs.s.typ).Elem()
	assert.NoError(tb, unmarshal(back, enc), name+": the decode method of the type decodes its encoding")
	assert.Equal(tb, vs.print(back), vs.print(v), name+": the decode method of the type returns the value")
}

// print returns the fingerprint of x, a value of the type of vs, as
// modeFingerprint writes it.
func (vs *valueSuite) print(x reflect.Value) []byte {
	e := &encoder{r: vs.r, mode: modeFingerprint}
	return e.appendValue(nil, vs.s, x, vs.s.typ.Name()+"."+valueField, 1)
}

// golden checks that the golden file of the type of vs matches one line per
// value, which the -update flag of the test binary writes: the hexadecimal
// encoding of the value as field 1 of a struct, or the error of
// ValidateKanon for a value that it rejects.
func (vs *valueSuite) golden(tb assert.TB) {
	tb.Helper()
	typ := vs.s.typ
	f := &field{Name: valueField, Number: 1, exported: true, shape: vs.s}
	var b strings.Builder
	for i, v := range vs.values {
		b.WriteString("value " + strconv.Itoa(i) + ": ")
		if err := validate(v); err != nil {
			b.WriteString(goldenFails + err.Error())
		} else {
			e := &encoder{r: vs.r, mode: modeReference}
			b.WriteString(hex.EncodeToString(e.appendField(nil, typ.Name()+"."+valueField, f, v, false)))
		}
		b.WriteByte('\n')
	}
	path := filepath.Join(filepath.FromSlash(goldenDir), typ.String()+goldenSuffix)
	golden.MatchAt(tb, path, []byte(b.String()), golden.ShouldUpdate())
}

// ValueChecks returns the checks of T, a named type that is not a struct and
// that implements kanon.Validator, over the values that entry 0 to 7 of the
// value tables of its underlying type build:
//
//   - ValidateKanon allocates nothing for a value that it accepts, which
//     the allocation contract of kanon.Message relies on.
//   - For a type with binary, gob or text methods, ValidateKanon accepts a
//     value exactly when the encode method of the type accepts it, so that
//     a directive without the -validate that the type needs fails. A
//     property of go.dokimi.dev/assert/prop checks the same on generated
//     values of the underlying type.
//   - The golden file of T pins the encoding of each value that
//     ValidateKanon accepts, as field 1 of a struct, and the error of each
//     value that it rejects.
//
// For a struct or an interface type, which kanon does not validate,
// ValueChecks returns one check, which fails with the reason.
func ValueChecks[T kanon.Validator]() []Check {
	typ := reflect.TypeFor[T]()
	vs, err := newValueSuite(typ)
	if err != nil {
		return []Check{{Name: valueTypeCheck, Run: func(tb assert.TB) {
			tb.Helper()
			assert.NoError(tb, err, "the type is a named type that is not a struct")
		}}}
	}
	checks := []Check{{Name: valueAllocsCheck, Serial: true, Run: func(tb assert.TB) {
		tb.Helper()
		valueAllocs[T](tb, vs)
	}}}
	if methodsOf(typ).family != 0 {
		checks = append(checks,
			Check{Name: valueDomainCheck, Run: vs.domain},
			Check{Name: valuePropertyCheck, Run: vs.property},
		)
	}
	return append(checks, Check{Name: valueGoldenCheck, Run: vs.golden})
}

// RunValue runs the checks of T that [ValueChecks] returns, each as a
// subtest of t. The check that counts allocations runs first and serially,
// and the others in parallel, so the test that calls RunValue does not call
// t.Parallel.
func RunValue[T kanon.Validator](t *testing.T) {
	t.Helper()
	run(t, ValueChecks[T]())
}

// valueAllocs checks that ValidateKanon of T allocates nothing for each
// value of vs that it accepts. It calls the method on a value of T, so that
// the count contains no conversion to an interface.
func valueAllocs[T kanon.Validator](tb assert.TB, vs *valueSuite) {
	tb.Helper()
	for i, v := range vs.values {
		x, _ := reflect.TypeAssert[T](v)
		if x.ValidateKanon() != nil {
			continue
		}
		msg := "value " + strconv.Itoa(i) + ": ValidateKanon allocates nothing"
		assert.MaxAllocs(tb, func() { _ = x.ValidateKanon() }, 0, msg)
	}
}

// validates reports whether t is a kanon.Validator that the generated code
// encodes as its underlying type: t is neither a struct, nor an interface,
// nor a pointer, whose method set contains the value methods of the type
// that it points at, and has ValidateKanon on a value receiver.
func validates(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Struct, reflect.Interface, reflect.Pointer:
		return false
	default:
		return t.Implements(reflect.TypeFor[kanon.Validator]())
	}
}

// validate returns the error of the ValidateKanon of v, a value of a
// kanon.Validator.
func validate(v reflect.Value) error {
	m, _ := reflect.TypeAssert[kanon.Validator](v)
	return m.ValidateKanon()
}

// validated returns the error of the decode of x, a value of s at offset
// off that loc and num locate, when s is a kanon.Validator whose
// ValidateKanon rejects x, and nil otherwise.
func validated(s *shape, x reflect.Value, loc string, num, off int) error {
	if !s.validate {
		return nil
	}
	if err := validate(x); err != nil {
		return wire.UnmarshalError(err, loc, num, off)
	}
	return nil
}
