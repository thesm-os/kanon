// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// Check is one check of the codec of a struct type, or of the ValidateKanon
// method of a kanon.Validator.
type Check struct {
	// Name names the check as the subtest that [Run] or [RunValue] runs it
	// in: the method whose contract the check states, a slash, and the
	// behaviour.
	Name string
	// Serial reports that the check counts allocations, which the testing
	// package counts while no parallel test runs.
	Serial bool
	// Run fails tb when the codec breaks the contract that Name states.
	Run func(tb assert.TB)
}

// Names of checks that [Checks] returns for some Specs only.
const (
	// specCheck names the one check of a Spec that does not describe its
	// struct type.
	specCheck = "Spec/describes the fields of the struct type"
	// roundTripCheck names the check of a canonical Spec that its reference
	// decode accepts exactly the inputs that round-trip.
	roundTripCheck = "DecodeKanon/accepts exactly the inputs that the reference encode writes back"
)

// Checks returns the checks of the codec of T that spec describes. When
// spec does not describe T, as a hand-edited Spec can fail to, Checks
// returns one check, which fails with the reason: a field, a discriminator
// or an unknown field that the struct type does not declare, a Struct
// whose type is not a struct, a field of a type that kanon does not encode,
// a struct type without a kanon codec that no Struct describes, an
// interface that no concrete type of its field fits, or a View whose type is
// not a byte slice.
func Checks[T any, P Codec[T]](spec Spec[T]) []Check {
	s, err := newSuite[T, P](spec)
	if err != nil {
		return []Check{{Name: specCheck, Run: func(tb assert.TB) {
			tb.Helper()
			assert.NoError(tb, err, "the Spec describes the fields of the struct type")
		}}}
	}
	checks := []Check{
		{Name: "SizeKanon/allocates nothing", Serial: true, Run: s.sizeAllocs},
		{
			Name:   "EncodeKanon/allocates nothing into a buffer of the length of the encoding",
			Serial: true,
			Run:    s.encodeAllocs,
		},
		{
			Name:   "AppendBinary/allocates nothing into a buffer with room for the encoding",
			Serial: true,
			Run:    s.appendAllocs,
		},
		{
			Name:   "DecodeKanon/allocates nothing into a receiver that decoded the encoding before",
			Serial: true,
			Run:    s.decodeAllocs,
		},
		{Name: "SizeKanon/returns the length of the reference encoding", Run: s.size},
		{Name: "MarshalBinary/returns the reference encoding", Run: s.marshal},
		{Name: "MarshalBinary/returns the error of a value that fails to encode", Run: s.marshalError},
		{Name: "AppendBinary/appends the reference encoding to the buffer", Run: s.append},
		{Name: "EncodeKanon/writes the reference encoding into the end of the buffer", Run: s.encode},
		{Name: "EncodeKanon/returns io.ErrShortBuffer for a buffer shorter than the encoding", Run: s.short},
		{Name: "SizeKanon/returns 0 for a nil receiver", Run: s.nilSize},
		{Name: "EncodeKanon/writes nothing for a nil receiver", Run: s.nilEncode},
		{Name: "AppendBinary/returns the buffer for a nil receiver", Run: s.nilAppend},
		{
			Name: "UnmarshalBinary/decodes the reference encoding of each sample as the reference decode",
			Run:  s.unmarshal,
		},
	}
	for _, f := range s.families() {
		checks = append(checks, Check{Name: f.check, Run: s.probing(f.probes)})
	}
	if s.r.canonical {
		checks = append(checks, Check{Name: roundTripCheck, Run: s.roundTrips})
	}
	if s.view != nil {
		checks = append(checks, Check{Name: viewCheck, Run: s.views})
		if _, ok := s.view.MethodByName(indexName); ok {
			checks = append(checks, Check{Name: indexCheck, Run: s.indexes})
		}
	}
	return append(checks,
		Check{Name: "DecodeKanon/decodes into a receiver that decoded before as into a zero one", Run: s.reuse},
		Check{Name: "MergeKanon/merges an encoding into a decoded value as the reference decode", Run: s.merge},
		Check{
			Name: "MergeKanon/merges an encoding into a value that fails to encode as the reference decode",
			Run:  s.mergeFailing,
		},
		Check{Name: "Reset/sets the receiver to the zero value", Run: s.reset},
		Check{Name: "CloneKanon/returns a copy that shares no memory with the receiver", Run: s.clone},
		Check{Name: "MarshalBinary/matches the golden file of the samples and the probes", Run: s.golden},
	)
}

// Run runs the checks of the codec of T that spec describes, each as a
// subtest of t. The checks that count allocations run first and serially,
// and the others in parallel, so the test that calls Run does not call
// t.Parallel.
func Run[T any, P Codec[T]](t *testing.T, spec Spec[T]) {
	t.Helper()
	run(t, Checks[T, P](spec))
}

// run runs checks, each as a subtest of t: the checks that count
// allocations serially, and the others in parallel.
func run(t *testing.T, checks []Check) {
	t.Helper()
	for _, c := range checks {
		t.Run(c.Name, func(t *testing.T) {
			if !c.Serial {
				t.Parallel()
			}
			c.Run(t)
		})
	}
}

// Bench measures SizeKanon, EncodeKanon and DecodeKanon of the codec of T
// that spec describes, each as a sub-benchmark of b, on the table sample
// of entry 1, whose every field is present: EncodeKanon into a buffer of
// the length of the encoding, and DecodeKanon with a slab into a receiver
// that decoded the encoding before. Each reports the length of the
// encoding as the bytes metric. Bench skips a Spec that does not describe
// T, which [Run] reports.
func Bench[T any, P Codec[T]](b *testing.B, spec Spec[T]) {
	b.Helper()
	s, err := newSuite[T, P](spec)
	if err != nil {
		b.Skip(err)
	}
	v := s.bench
	p := P(&v)
	buf := make([]byte, p.SizeKanon())
	enc, _ := p.MarshalBinary()
	b.Run("SizeKanon", func(b *testing.B) {
		for b.Loop() {
			p.SizeKanon()
		}
		b.ReportMetric(float64(len(buf)), bytesMetric)
	})
	b.Run("EncodeKanon", func(b *testing.B) {
		for b.Loop() {
			_, _ = p.EncodeKanon(buf)
		}
		b.ReportMetric(float64(len(buf)), bytesMetric)
	})
	b.Run("DecodeKanon", func(b *testing.B) {
		var d T
		opts := kanon.Options{Slab: string(enc)}
		_ = P(&d).DecodeKanon(enc, opts)
		for b.Loop() {
			_ = P(&d).DecodeKanon(enc, opts)
		}
		b.ReportMetric(float64(len(enc)), bytesMetric)
	})
}

// Fuzz fuzzes DecodeKanon of the codec of T that spec describes, with the
// reference encodings of the samples of the checks that encode, and those
// encodings without their last byte, as the seeds. DecodeKanon decodes
// every input as the reference decode: with the same error, or to the same
// value. For a canonical Spec the reference decode also succeeds exactly
// when the input round-trips through the reference encode. Fuzz skips a Spec
// that does not describe T, which [Run] reports.
func Fuzz[T any, P Codec[T]](f *testing.F, spec Spec[T]) {
	f.Helper()
	s, err := newSuite[T, P](spec)
	if err != nil {
		f.Skip(err)
	}
	for _, x := range s.encodable() {
		f.Add(x.enc)
		f.Add(x.enc[:max(len(x.enc)-1, 0)])
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		s.fuzz(t, data)
	})
}

// bytesMetric is the unit of the length of an encoding that the benchmarks
// report.
const bytesMetric = "bytes"

// suite is the reference data of the checks of one codec: the layout of T,
// and the samples that the checks run on with their reference encodings.
type suite[T any, P Codec[T]] struct {
	r *resolver
	l *layout
	// samples lists the zero value, the table samples, the samples of the
	// fields and the samples that fail to encode, in that order, which the
	// probes of the decode checks change.
	samples []sample[T]
	// tables lists the table samples, whose concatenations the decode checks
	// decode.
	tables []sample[T]
	// fails counts, per table sample, its values that can fail to encode,
	// each of which one sample fails.
	fails []int
	// bulk lists the wide samples and the key sample, whose maps have many
	// entries, which the probes leave unchanged. The first wide sample, which
	// encodes, is the one that the reuse checks decode into a receiver.
	bulk []sample[T]
	// pieces lists the pieces of the samples, which the probes cut and
	// change, as [suite.piecesOf] returns them.
	pieces []piece
	// bench is the table sample of entry 1, which the benchmarks measure.
	bench T
	// view is the view type of T, and nil when the Spec names none.
	view reflect.Type
}

// sample is a value of T that the checks run on.
type sample[T any] struct {
	// name names the sample in failures.
	name  string
	value T
	// enc is the reference encoding of value, and err the error of the
	// codec for it.
	enc []byte
	err error
}

// probe is an input of the decode checks: the bytes that a decode reads,
// and its options.
type probe struct {
	// name names the probe in failures: its sample and how it changes the
	// encoding of the sample.
	name string
	data []byte
	opts kanon.Options
}

// newSuite returns the suite of spec, with these samples:
//
//   - the zero value;
//   - a table sample per table entry, and at least one per sliceLengths
//     entries per choice of the widest pick, so that every choice meets
//     every length of a slice;
//   - per field, the sample of entry 1 that sets that field alone, and per
//     union member its zero value and a sample per table entry;
//   - per table sample and per value in it that can fail to encode, the
//     sample in which that value alone fails, once per way to fail it;
//   - the wide samples, whose maps have more entries than a decode reuses,
//     one per choice of the widest pick and two at least;
//   - the key sample, whose maps order keys that differ in one part each;
//   - per side of the entries of a map that can fail to encode, as
//     [resolver.failingSides] finds them, the first wide sample that has
//     such a map, with one entry of the first such map failing at that side.
//
// It fails as [Checks] states.
func newSuite[T any, P Codec[T]](spec Spec[T]) (*suite[T, P], error) {
	r, l, err := newResolver(spec)
	if err != nil {
		return nil, err
	}
	r.prepare(l)
	s := &suite[T, P]{r: r, l: l}
	if spec.View != nil {
		s.view = reflect.TypeOf(spec.View)
		if s.view.Kind() != reflect.Slice || s.view.Elem() != reflect.TypeFor[byte]() {
			return nil, fmt.Errorf("kanontest: the Spec names view type %v, which is not a byte slice", s.view)
		}
	}
	tables := max(drawCount, choices(l)*sliceLengths)
	for i := range tables {
		s.tables = append(s.tables, s.sample("table sample "+strconv.Itoa(i), s.fill(table(i), nesting)))
	}
	s.bench = s.tables[fieldEntry].value
	s.samples = append([]sample[T]{s.sample("the zero value", reflect.New(l.typ).Elem())}, s.tables...)
	for _, f := range l.fields {
		s.samples = append(s.samples,
			s.sample("the sample of field "+f.Name, s.with(f, r.build(table(fieldEntry), f.shape, nesting))))
		if f.Union == "" {
			continue
		}
		s.samples = append(s.samples, s.sample("the zero member "+f.Name, s.with(f, reflect.Zero(f.shape.typ))))
		for i := range tables {
			s.samples = append(s.samples, s.sample("table sample "+strconv.Itoa(i)+" of member "+f.Name,
				s.with(f, r.build(table(i), f.shape, nesting))))
		}
	}
	for i := range tables {
		// A source whose count of the values left never reaches -1 fails none
		// of them, and counts the values that can fail.
		unfailed := math.MaxInt
		s.fill(failing{table(i), &unfailed}, nesting)
		s.fails = append(s.fails, math.MaxInt-unfailed)
		for j := range s.fails[i] {
			s.samples = append(s.samples, s.sample(failingName(i, j), s.failed(i, j)))
		}
	}
	wide := max(wideSamples, choices(l))
	for n := range wide {
		s.bulk = append(s.bulk, s.sample("wide sample "+strconv.Itoa(n), s.fill(counting(n), wideNesting)))
	}
	s.bulk = append(s.bulk, s.sample("the key sample", s.fill(keyed{table(fieldEntry)}, nesting)))
	for _, side := range r.failingSides(l) {
		if x, ok := s.entryFailed(side, wide); ok {
			s.samples = append(s.samples, x)
		}
	}
	s.pieces = s.piecesOf(s.samples)
	return s, nil
}

// entryFailed returns the first of the wide samples, which number wide,
// that has a map of the shape of side, with one entry of the first such map
// failing at the side of side, as [entryFailing] fails it. It reports false
// when none of them fails that way, such as for a map that no wide sample
// nests deep enough to fill.
func (s *suite[T, P]) entryFailed(side mapSide, wide int) (x sample[T], ok bool) {
	for n := range wide {
		done := false
		x = s.sample(entryFailingName(n, side), s.fill(entryFailing{counting(n), side, &done}, wideNesting))
		if ok = done && x.err != nil; ok {
			break
		}
	}
	return x, ok
}

// failed returns table sample i with its value j that can fail to encode
// failing, built anew.
func (s *suite[T, P]) failed(i, j int) reflect.Value {
	left := j
	return s.fill(failing{table(i), &left}, nesting)
}

// fill returns the value of T that src fills, nesting depth levels deep.
func (s *suite[T, P]) fill(src source, depth int) reflect.Value {
	v := reflect.New(s.l.typ).Elem()
	s.r.fill(src, s.l, v, depth)
	return v
}

// with returns the value of T whose field f alone is x, and whose
// discriminator selects f when f is a union member.
func (s *suite[T, P]) with(f *field, x reflect.Value) reflect.Value {
	v := reflect.New(s.l.typ).Elem()
	f.of(v).Set(x)
	if f.Union != "" {
		v.FieldByIndex(f.disc).Set(reflect.ValueOf(f.Case))
	}
	return v
}

// sample returns the sample named name of v, a value of T, with its
// reference encoding.
func (s *suite[T, P]) sample(name string, v reflect.Value) sample[T] {
	enc, err := s.r.encode(s.l, v)
	value, _ := reflect.TypeAssert[T](v)
	return sample[T]{name: name, value: value, enc: enc, err: err}
}

// all returns the samples and the bulk samples, in that order.
func (s *suite[T, P]) all() []sample[T] {
	return slices.Concat(s.samples, s.bulk)
}

// reference returns the reference decode of the probe p into the zero value
// of T, and its error.
func (s *suite[T, P]) reference(p probe) (T, error) {
	slab, off := p.opts.Source(p.data)
	v := reflect.New(s.l.typ).Elem()
	err := s.r.decode(s.l, v, p.data, slab, off, p.opts.Limit())
	x, _ := reflect.TypeAssert[T](v)
	return x, err
}

// fuzz checks the decode of data: DecodeKanon decodes it as the reference
// decode, and for a canonical Spec the reference decode succeeds exactly
// when data round-trips, as [suite.roundTrip] checks it.
func (s *suite[T, P]) fuzz(t *testing.T, data []byte) {
	t.Helper()
	p := probe{name: "the input", data: data}
	s.decode(t, p)
	if s.r.canonical {
		s.roundTrip(t, p)
	}
}

// failingName returns the name of table sample i with its value j that can
// fail to encode failing.
func failingName(i, j int) string {
	return "table sample " + strconv.Itoa(i) + " that fails at value " + strconv.Itoa(j)
}

// entryFailingName returns the name of wide sample n with one entry of a
// map failing at side.
func entryFailingName(n int, side mapSide) string {
	at := "a value"
	if side.key {
		at = "a key"
	}
	return "wide sample " + strconv.Itoa(n) + " that fails at " + at + " of " + side.s.typ.String()
}
