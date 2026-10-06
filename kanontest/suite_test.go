// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest_test

import (
	"errors"
	"flag"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/internal/fixture/view"
	"go.thesmos.sh/kanon/kanontest"
)

// Words of the kanon struct tag and the name of a blank field, which decide
// the fields that kanon encodes, as the completeness check of a Spec reads
// them.
const (
	kanonTag   = "kanon"
	skipTag    = "-"
	unknownTag = "unknown"
	blankField = "_"
)

// Assertions and detail keys of the failure records that the tests read:
//
//   - noErrorAssertion, the record of NoError, whose got is the error;
//   - goldenAssertion, the record of golden.MatchAt, whose want is the
//     content of the golden file, nil for a file that does not exist;
//   - gotDetail, wantDetail and prefixDetail, the keys of the values of an
//     assertion that compares them, which a table of cases names.
const (
	noErrorAssertion = "err-absent"
	goldenAssertion  = "golden-match-at"
	gotDetail        = "got"
	wantDetail       = "want"
	prefixDetail     = "prefix"
)

// itemSpec describes view.Item, a struct of a string and a uint32.
var itemSpec = kanontest.Spec[view.Item]{
	Fields: []kanontest.Field{
		{Name: "Name", Number: 1},
		{Name: "Count", Number: 2},
	},
}

// Names of the flags of the test binary that the benchmarks of the tests
// read.
const (
	benchFlag     = "test.bench"
	benchTimeFlag = "test.benchtime"
	// oneRun makes testing.Benchmark run a benchmark function once.
	oneRun = "1x"
)

// TestMain runs the tests with a benchtime of one run when the test binary
// runs no benchmark, so that the tests of Bench, which call
// testing.Benchmark, take one iteration per sub-benchmark instead of a
// second.
func TestMain(m *testing.M) {
	flag.Parse()
	if flag.Lookup(benchFlag).Value.String() == "" {
		// The testing package declares the flag, and it takes the value.
		_ = flag.Set(benchTimeFlag, oneRun)
	}
	os.Exit(m.Run())
}

// passes checks that spec lists every field that kanon encodes, as
// [complete] states, and runs the checks of spec that count no
// allocations, each as a parallel subtest of t, so that t fails when the
// codec of T breaks one.
func passes[T any, P kanontest.Codec[T]](t *testing.T, spec kanontest.Spec[T]) {
	t.Helper()
	complete(t, spec)
	for _, c := range kanontest.Checks[T, P](spec) {
		if c.Serial {
			continue
		}
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			c.Run(t)
		})
	}
}

// complete fails t unless spec lists every field of T, and of the type of
// each of its Structs and Keys, that kanon encodes, as the Spec that kanon
// generates lists them, so that a Spec that a test writes by hand cannot
// fall behind its struct type.
func complete[T any](t *testing.T, spec kanontest.Spec[T]) {
	t.Helper()
	assert.Equal(t, unlisted(reflect.TypeFor[T](), spec.Fields, spec.Unknown), nil,
		"the Spec lists every field of the struct type that kanon encodes")
	for _, s := range slices.Concat(spec.Structs, spec.Keys) {
		msg := "the Spec lists every field of " + s.Type.String() + " that kanon encodes"
		assert.Equal(t, unlisted(s.Type, s.Fields, s.Unknown), nil, msg)
	}
}

// unlisted returns the names of the fields of the struct type typ that kanon
// encodes and that fields, the union discriminators that fields name, and
// unknown do not list. kanon encodes an exported field and an unexported
// field with a kanon tag, but for a blank field, a field of a function or a
// channel type or a pointer to one, a field tagged "-" and the field that
// keeps unknown fields.
func unlisted(typ reflect.Type, fields []kanontest.Field, unknown string) []string {
	listed := map[string]bool{unknown: true}
	for _, f := range fields {
		listed[f.Name], listed[f.Union] = true, true
	}
	var out []string
	for sf := range typ.Fields() {
		tag, tagged := sf.Tag.Lookup(kanonTag)
		left := sf.Name == blankField || !sf.IsExported() && !tagged || unsent(sf.Type) || tag == skipTag ||
			slices.Contains(strings.Split(tag, ","), unknownTag)
		if !left && !listed[sf.Name] {
			out = append(out, sf.Name)
		}
	}
	return out
}

// unsent reports whether t is a function or a channel, or a pointer to one
// at any depth, which kanon leaves out of the encoding as encoding/gob does.
func unsent(t reflect.Type) bool {
	seen := make(map[reflect.Type]bool)
	for t.Kind() == reflect.Pointer && !seen[t] {
		seen[t] = true
		t = t.Elem()
	}
	return t.Kind() == reflect.Func || t.Kind() == reflect.Chan
}

// rejects runs the check of spec named name, and fails t unless the check
// fails with a first failure whose reason contains want.
func rejects[T any, P kanontest.Codec[T]](t *testing.T, spec kanontest.Spec[T], name, want string) {
	t.Helper()
	rejectedFor(t, name, named[T, P](t, spec, name).Run, want)
}

// rejectedFor runs check, which name names, and fails t unless the check
// fails with a first failure whose reason, as reason states it, contains
// want. It returns that failure.
func rejectedFor(t *testing.T, name string, check func(assert.TB), want string) assert.Failure {
	t.Helper()
	got := assert.Rejects(t, name, check)
	assert.NotEmpty(t, got, name+" fails with a failure record")
	assert.Contains(t, reason(got[0]), want, name+" fails for the reason that it states")
	return got[0]
}

// reason returns the reason that the failure f states, from the fields of
// its record: its contract, and after it the text of the error of a NoError
// failure. The reason of the record of a property is the reason of the
// failure of its counterexample.
func reason(f assert.Failure) string {
	if inner, ok := f.CaseFailure(); ok {
		return reason(inner)
	}
	got, _ := f.Got()
	if err, ok := got.(error); ok && f.Assertion == noErrorAssertion {
		return f.Contract + ": " + err.Error()
	}
	return f.Contract
}

// renamed returns err, and names the struct type wrapper in a
// *kanon.DecodeError or a *kanon.EncodeError that names the struct type
// embedded, as the reference decoder and encoder of wrapper name it. A
// wrapper of a generated codec decodes and encodes through the code of the
// type that it embeds, which names that type, so that its errors differ
// from the reference in nothing else than the change that it makes.
func renamed(err error, embedded, wrapper string) error {
	if e, ok := errors.AsType[*kanon.DecodeError](err); ok && e.Type == embedded {
		e.Type = wrapper
	}
	if e, ok := errors.AsType[*kanon.EncodeError](err); ok && e.Type == embedded {
		e.Type = wrapper
	}
	return err
}

// holds runs the check of spec named name on t, which fails when the codec
// of T breaks it.
func holds[T any, P kanontest.Codec[T]](t *testing.T, spec kanontest.Spec[T], name string) {
	t.Helper()
	named[T, P](t, spec, name).Run(t)
}

// named returns the check of spec named name, and fails t when no check has
// that name.
func named[T any, P kanontest.Codec[T]](t *testing.T, spec kanontest.Spec[T], name string) kanontest.Check {
	t.Helper()
	for _, c := range kanontest.Checks[T, P](spec) {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no check is named %q", name)
	return kanontest.Check{}
}

func TestSuite(t *testing.T) {
	t.Parallel()
	t.Run("Checks", func(t *testing.T) {
		t.Parallel()
		t.Run("returns checks that a generated codec passes", func(t *testing.T) {
			t.Parallel()
			passes(t, itemSpec)
		})
		t.Run("returns one check for a Spec that does not describe the struct type", func(t *testing.T) {
			t.Parallel()
			checks := kanontest.Checks(
				kanontest.Spec[view.Item]{Fields: []kanontest.Field{{Name: "Missing", Number: 1}}},
			)
			assert.Length(t, checks, 1, "Checks returns the check of the Spec alone")
			assert.Equal(t, checks[0].Name, "Spec/describes the fields of the struct type",
				"Checks names the check of the Spec")
		})
	})
	t.Run("Bench", func(t *testing.T) {
		t.Parallel()
		t.Run("measures a generated codec", func(t *testing.T) {
			t.Parallel()
			r := testing.Benchmark(func(b *testing.B) {
				b.Helper()
				kanontest.Bench(b, itemSpec)
			})
			assert.True(t, r.N > 0, "Bench runs its sub-benchmarks")
		})
		t.Run("skips a Spec that does not describe the struct type", func(t *testing.T) {
			t.Parallel()
			bad := kanontest.Spec[view.Item]{Fields: []kanontest.Field{{Name: "Missing", Number: 1}}}
			r := testing.Benchmark(func(b *testing.B) {
				b.Helper()
				kanontest.Bench(b, bad)
			})
			assert.Equal(t, r.N, 0, "Bench runs no benchmark for a Spec that does not describe the struct type")
		})
	})
}

func FuzzSuite(f *testing.F) {
	kanontest.Fuzz(f, itemSpec)
}

func FuzzSuiteSpec(f *testing.F) {
	kanontest.Fuzz(f, kanontest.Spec[view.Item]{Fields: []kanontest.Field{{Name: "Missing", Number: 1}}})
}

func FuzzSuiteCanonical(f *testing.F) {
	kanontest.Fuzz(f, mapsSpec)
}
