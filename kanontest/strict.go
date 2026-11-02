// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
)

// Names of the checks of [RunStrict]: the method whose contract a check
// states, a slash, and the behaviour. StrictKanon marks the guarantee of
// kanon.Strict, which the decode method of the type keeps.
const (
	// strictTypeCheck names the one check of a type that does not meet the
	// requirements of kanon.Strict.
	strictTypeCheck = "StrictKanon/marks a type that encodes itself through the methods of a family"
	// strictDecodeCheck names the check of the guarantee.
	strictDecodeCheck = "StrictKanon/decodes only the bytes that the encode method writes for the decoded value"
)

// markedValue is a value of a type that declares kanon.Exact or kanon.Strict,
// which the checks of its marker run on.
type markedValue struct {
	v reflect.Value
	// name names the value in failures, and is empty for a decoded value,
	// whose input names it.
	name string
	// input is the input that the decode method decodes a decoded value
	// from.
	input []byte
}

// String returns the name of x in failures.
func (x markedValue) String() string {
	if x.name != "" {
		return x.name
	}
	return fmt.Sprintf("the value that the decode method decodes from %x", x.input)
}

// StrictChecks returns the check of T, a type that declares kanon.Strict:
// for every input that the decode method of T accepts, the encode method of
// its family writes the input for the value that the decode method sets. The
// check decodes the encoding of every value of the value tables, every prefix
// of it, every change of one of its bytes and the encoding with one more
// byte, and a byte string of every length up to the size of T in memory.
//
// For a type that kanon does not encode, and for a type that does not encode
// itself through the methods of a family, StrictChecks returns one check,
// which fails with the reason. kanon encodes a kanon.Validator as its
// underlying type, so a Validator is such a type.
func StrictChecks[T kanon.Strict]() []Check {
	decoded, err := strictDecoded(reflect.TypeFor[T]())
	if err != nil {
		return []Check{{Name: strictTypeCheck, Run: func(tb assert.TB) {
			tb.Helper()
			assert.NoError(tb, err, "the type meets the requirements of kanon.Strict")
		}}}
	}
	return []Check{{Name: strictDecodeCheck, Run: func(tb assert.TB) {
		tb.Helper()
		rewrites(tb, decoded, "the encode method")
	}}}
}

// RunStrict runs the checks of T that [StrictChecks] returns, each as a
// parallel subtest of t.
func RunStrict[T kanon.Strict](t *testing.T) {
	t.Helper()
	run(t, StrictChecks[T]())
}

// strictDecoded returns the values that the decode method of t decodes from
// the inputs that [decodedFrom] derives from the values of the value tables.
// It fails for a type that kanon does not encode, and for a type that does
// not encode itself through the methods of a family.
func strictDecoded(t reflect.Type) ([]markedValue, error) {
	r := resolverFor(t.PkgPath())
	s, err := r.shapeOf(t, opts{seen: make(map[seenKey]*shape), loose: true})
	if err != nil {
		return nil, fmt.Errorf("kanontest: %v: %w", t, err)
	}
	if s.kind != kindBinary {
		return nil, fmt.Errorf("kanontest: %v declares StrictKanon, and does not encode itself through binary, gob or "+
			"text methods", t)
	}
	return decodedFrom(t, tableValues(r, t)), nil
}

// tableValues returns the values that entry 0 to 7 of the value tables build
// for t, a type that encodes itself, as [resolver.self] builds them.
func tableValues(r *resolver, t reflect.Type) []markedValue {
	out := make([]markedValue, drawCount)
	for i := range out {
		v := reflect.New(t).Elem()
		r.self(table(i), v, 0)
		out[i] = markedValue{v: v, name: "value " + strconv.Itoa(i) + " of the value tables"}
	}
	return out
}

// decodedFrom returns the values that the decode method of t decodes from
// the inputs of the checks of the decode method, for each input that the
// method accepts, in their order. The inputs are the encoding of each of
// values that encodes, as [variants] derives inputs from it, and the bytes
// that [counted] returns for every length from 0 to the size of a value of t
// in memory.
func decodedFrom(t reflect.Type, values []markedValue) []markedValue {
	var inputs [][]byte
	for _, x := range values {
		if enc, err := marshal(x.v); err == nil {
			inputs = append(inputs, variants(enc)...)
		}
	}
	for n := range int(t.Size()) + 1 {
		inputs = append(inputs, counted(n))
	}
	var out []markedValue
	for _, b := range inputs {
		v := reflect.New(t).Elem()
		if unmarshal(v, b) == nil {
			out = append(out, markedValue{v: v, input: b})
		}
	}
	return out
}

// rewrites checks the guarantee of kanon.Strict on decoded, which is the
// second guarantee of kanon.Exact. For each input that the decode method
// accepts, the encode method of the family must write the input for the value
// that the decode method sets. [marshal] calls the encode method, and encoder
// names it in the contract of a failure. rewrites fails with an err-absent
// record of the error of the method, and with an equal record of the bytes.
func rewrites(tb assert.TB, decoded []markedValue, encoder string) {
	tb.Helper()
	for _, x := range decoded {
		enc, err := marshal(x.v)
		if err != nil || !bytes.Equal(enc, x.input) {
			contract := "input " + hex.EncodeToString(x.input) + ": " + encoder +
				" writes the input for the value that the decode method decodes from it"
			assert.NoError(tb, err, contract)
			assert.Equal(tb, enc, x.input, contract)
		}
	}
}

// variants returns the inputs that the decode checks derive from enc, the
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
