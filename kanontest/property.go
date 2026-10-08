// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"reflect"
	"slices"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/prop"
)

// Names of the property of [Checks].
const (
	// propertyCheck names the check that runs the property.
	propertyCheck = "Message/encodes and decodes generated values as the reference"
	// propertyContract is the contract of the record of the property, which
	// names its cases in the store of the test.
	propertyContract = "the codec encodes and decodes generated values as the reference"
)

// Labels of the draws of a case of the property that no part of a value
// names.
const (
	// firstValue and secondValue label the two values of T that a case
	// draws, and begin the paths of their parts.
	firstValue  = "a"
	secondValue = "b"
	// familyDraw labels the draw of a family of probes, and probeDraw the
	// draw of a probe of the family.
	familyDraw = "family"
	probeDraw  = "probe"
	// inputDraw labels the draw of the bytes of the probe of arbitrary
	// bytes.
	inputDraw = "input"
)

// Names of the samples and the probes of a case of the property.
const (
	// generatedName begins the name of the sample of a value that a case
	// draws, which the label of the value ends.
	generatedName = "the generated value "
	// arbitraryName names the family of the probe of arbitrary bytes, and
	// the probe.
	arbitraryName = "arbitrary bytes"
)

// property checks the codec of T on the cases of a run of prop.ForAll, each
// as [suite.generated] checks it. prop runs the cases of the store of the
// test before the cases of the seed of the run. It shrinks a failing case to
// the smallest case that fails the same assertion, and the record of the
// failure states the values of that case and its replay token.
func (s *suite[T, P]) property(tb assert.TB) {
	tb.Helper()
	prop.ForAll(tb, propertyContract, s.generated)
}

// generated checks the codec of T on the case c as the table checks check
// the samples. The case draws two values of T, as [suite.drawSample] draws
// them, and a probe derived from them, as [suite.drawProbe] draws it, before
// any check runs:
//
//   - the encode checks and the check of UnmarshalBinary on both values;
//   - the check of a decode into a receiver that decoded each bulk sample
//     and each value, the merge of each value into each value that decodes,
//     Reset, and CloneKanon for a codec that implements kanon.Cloner, which
//     the clone check requires;
//   - DecodeKanon on the probe;
//   - for a canonical Spec, the round trip of the probe and of the
//     encoding of each value that encodes;
//   - for a Spec with a stream decoder, the stream decoder on the probe and
//     on the encoding of each value that encodes, as [suite.streams] runs
//     it;
//   - the view methods and IndexKanon on the probe and on the encoding of
//     each value that encodes.
//
// Each check states the value by the name of its sample. The notes of the
// case, which prop reports for a failing case, state the encodings of the
// values and the probe.
func (s *suite[T, P]) generated(c *prop.Case) {
	xs := []sample[T]{s.drawSample(c, firstValue), s.drawSample(c, secondValue)}
	p := s.drawProbe(c, xs)
	for _, x := range xs {
		c.Logf("%s: %x, error %v", x.name, x.enc, x.err)
	}
	c.Logf("%s: %x", p.name, p.data)
	s.size(c, xs)
	s.marshal(c, xs)
	s.marshalError(c, xs)
	s.append(c, xs)
	s.encode(c, xs)
	s.short(c, xs)
	s.unmarshal(c, xs)
	s.reuse(c, xs, slices.Concat(s.bulk, xs))
	s.merge(c, xs)
	s.reset(c, xs)
	if s.cloner {
		s.clones(c, xs)
	}
	s.decode(c, p)
	encoded := encodes(xs)
	probes := make([]probe, len(encoded)+1)
	probes[0] = p
	for k, x := range encoded {
		probes[k+1] = probe{name: x.name, data: x.enc}
	}
	inputs := make([][]byte, len(probes))
	for k, q := range probes {
		inputs[k] = q.data
		if s.r.canonical {
			s.roundTrip(c, q)
		}
		if s.stream != nil {
			s.streams(c, q)
		}
	}
	if s.view != nil {
		s.views(c, inputs)
	}
	if s.index {
		s.indexes(c, inputs)
	}
}

// drawSample draws a value of T from c under the label root and returns it
// as the sample named after root. The builder builds the value from a drawn
// source whose paths begin at root, nesting levels deep at most, as it
// builds the samples of the value tables.
func (s *suite[T, P]) drawSample(c *prop.Case, root string) sample[T] {
	v := c.Draw(prop.Composite(func(c *prop.Case) T {
		x, _ := reflect.TypeAssert[T](s.fill(drawn{c: c, path: root}, nesting))
		return x
	}), root)
	return s.sample(generatedName+root, reflect.ValueOf(&v).Elem())
}

// drawProbe draws one probe of xs from c. It first draws a family: one of
// the families of the probes that the decode checks derive from xs, as
// [suite.over] derives them, or the family of the probe of arbitrary bytes.
// It then draws a probe of that family, or the encoding of the first sample
// of xs, which gives a family without a probe of xs a probe too.
func (s *suite[T, P]) drawProbe(c *prop.Case, xs []sample[T]) probe {
	arbitrary := probeFamily{check: arbitraryName, probes: func() []probe {
		return []probe{{name: arbitraryName, data: c.Draw(prop.Bytes(), inputDraw)}}
	}}
	families := append(s.over(xs).families(), arbitrary)
	f := families[c.Draw(prop.Integer(0, len(families)-1), familyDraw)]
	probes := append([]probe{{name: xs[0].name, data: xs[0].enc}}, f.probes()...)
	return probes[c.Draw(prop.Integer(0, len(probes)-1), probeDraw)]
}

// over returns a copy of s whose samples and table samples are xs. The copy
// has the pieces of xs and no bulk samples, so that the families of the
// decode checks derive their probes from xs alone.
func (s *suite[T, P]) over(xs []sample[T]) *suite[T, P] {
	d := *s
	d.samples, d.tables, d.bulk = xs, xs, nil
	d.pieces = d.piecesOf(xs)
	return &d
}
