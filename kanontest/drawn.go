// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"math"
	"strconv"
	"time"

	"go.dokimi.dev/assert/prop"
)

// Bounds of the parts of a generated value.
const (
	// drawnLength is the most elements of a slice, and entries of a map, of
	// a generated value.
	drawnLength = 4
	// drawnBytes is the most bytes of a string and of a byte slice of a
	// generated value: more than the 127 bytes that a one-byte length
	// counts, and few enough that the input of a fuzzer states each length
	// in one byte.
	drawnBytes = 255
	// failOdds is the denominator of the odds that a value that can fail to
	// encode fails in a generated value: one in failOdds.
	failOdds = 16
	// maxNanos is the largest nanosecond count of a time.
	maxNanos = 999999999
)

// pathSeparator separates the numbers of the parts in the path of a part
// of a generated value.
const pathSeparator = "."

// Generators of the parts of a generated value that take no argument.
var (
	// booleans generates a bool, and the presence of a pointer.
	booleans = prop.Boolean()
	// failCoin generates whether a value that can fail to encode fails: true
	// with odds of one in failOdds.
	failCoin = prop.Boolean(prop.Odds(1, failOdds))
	// texts generates the bytes of a string.
	texts = prop.Bytes(prop.MaxSize(drawnBytes))
	// byteSlices generates a byte slice: nil, or bytes, which can be empty.
	byteSlices = prop.OneOf(prop.Just[[]byte](nil), prop.Bytes(prop.MaxSize(drawnBytes)))
	// nanoseconds generates the nanoseconds of a time.
	nanoseconds = prop.Integer[int64](0, maxNanos)
	// zones generates the zone of a time: nil for UTC, and otherwise the
	// offset of a fixed zone in seconds east of UTC.
	zones = prop.Optional(prop.Integer[int64](math.MinInt32, math.MaxInt32))
	// signedOf maps a width in bits to the generator of the signed integers
	// of that width.
	signedOf = map[int]prop.Generator[int64]{
		8:  prop.Integer[int64](math.MinInt8, math.MaxInt8),
		16: prop.Integer[int64](math.MinInt16, math.MaxInt16),
		32: prop.Integer[int64](math.MinInt32, math.MaxInt32),
		64: prop.Integer[int64](math.MinInt64, math.MaxInt64),
	}
	// unsignedOf maps a width in bits to the generator of the unsigned
	// integers of that width.
	unsignedOf = map[int]prop.Generator[uint64]{
		8:  prop.Integer[uint64](0, math.MaxUint8),
		16: prop.Integer[uint64](0, math.MaxUint16),
		32: prop.Integer[uint64](0, math.MaxUint32),
		64: prop.Integer[uint64](0, math.MaxUint64),
	}
)

// drawn is the source of a generated value. The case c of a property draws
// each value that the builder takes from the source from a generator of
// prop. prop then shrinks, stores and replays the value through the choices
// of the case. Each draw is labelled with path, the path of the part that
// the source supplies. A path is the name of the value followed by the
// number that the builder gives each part on the way to the part, as in
// a.3.1.
//
// A drawn source generates every value of each Go kind. An integer takes
// the range of its width, and a float the values from -Inf to +Inf with NaN.
// A string and a byte slice take arbitrary bytes. A time takes the range of
// time.Time, in UTC or in a fixed zone. A length ranges from nil to
// drawnLength. A value that can fail to encode fails with odds of one in
// failOdds. The builder decides the structure of the value, as it does for
// every source.
//
// # Concurrency
//
// A drawn source draws from its case, so the body of the case uses it on
// its own goroutine, as prop requires of a draw.
//
// # Allocation contract
//
// Each draw allocates the record of its value in the case. A length, a
// pick, a float and the seconds of a time allocate their generator as well.
type drawn struct {
	c    *prop.Case
	path string
}

var _ source = drawn{}

// part returns the source of part k, whose draws are labelled with the path
// of d followed by k.
func (d drawn) part(k int) source {
	return drawn{c: d.c, path: d.path + pathSeparator + strconv.Itoa(k)}
}

// boolean draws a bool.
func (d drawn) boolean() bool { return d.c.Draw(booleans, d.path) }

// signed draws a signed integer of bits bits, over the whole range of the
// width.
func (d drawn) signed(bits int) int64 { return d.c.Draw(signedOf[bits], d.path) }

// unsigned draws an unsigned integer of bits bits, over the whole range of
// the width.
func (d drawn) unsigned(bits int) uint64 { return d.c.Draw(unsignedOf[bits], d.path) }

// float draws a float64 from -Inf to +Inf, NaN included. A float32 takes the
// value rounded to its width, which keeps the infinities and NaN.
func (d drawn) float(int) float64 {
	return d.c.Draw(prop.Float(math.Inf(-1), math.Inf(1), prop.AllowNaN()), d.path)
}

// text draws a string of up to drawnBytes arbitrary bytes, invalid UTF-8
// and NUL bytes included.
func (d drawn) text() string { return string(d.c.Draw(texts, d.path)) }

// bytes draws a byte slice of up to drawnBytes arbitrary bytes, or nil.
func (d drawn) bytes() []byte { return d.c.Draw(byteSlices, d.path) }

// instant draws a time: its Unix seconds over the range that time.Unix
// represents, from math.MinInt64 to the second whose count from year 1 is
// math.MaxInt64, its nanoseconds, and UTC or a fixed zone at an offset in
// the range of an int32.
func (d drawn) instant() time.Time {
	seconds := prop.Integer(math.MinInt64, math.MaxInt64+time.Time{}.Unix())
	t := time.Unix(d.c.Draw(seconds, d.path), d.c.Draw(nanoseconds, d.path))
	zone := d.c.Draw(zones, d.path)
	if zone == nil {
		return t.UTC()
	}
	return t.In(time.FixedZone("", int(*zone)))
}

// count draws the length of a slice: -1 for a nil one, or 0 to drawnLength.
func (d drawn) count() int { return d.c.Draw(prop.Integer(-1, drawnLength), d.path) }

// entries draws the number of entries of a map as count draws a length.
func (d drawn) entries() int { return d.count() }

// set draws whether a pointer points at a value.
func (d drawn) set() bool { return d.c.Draw(booleans, d.path) }

// pick draws one of n choices, from 0 to n-1.
func (d drawn) pick(n int) int { return d.c.Draw(prop.Integer(0, n-1), d.path) }

// keyed returns false.
func (drawn) keyed() bool { return false }

// fail draws whether the value that can fail to encode, which the builder is
// about to build, fails, with odds of one in failOdds.
func (d drawn) fail() bool { return d.c.Draw(failCoin, d.path) }

// failEntry returns false. A key or a value of a map of a generated value
// fails where one of its own values draws a failure.
func (drawn) failEntry(*shape, bool) bool { return false }
