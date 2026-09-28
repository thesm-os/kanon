// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"io"
	"math"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The field and the offset at which the Time cases locate their errors.
const (
	timeLoc    = "Order.At"
	timeType   = "Order"
	timeField  = "At"
	timeNumber = 4
	timeOff    = 100
)

// hourEast is the offset of a zone one hour east of UTC, in seconds.
const hourEast = 3600

// timeVectors are the encodings of times that the wire format pins.
func timeVectors() []struct {
	name string
	t    time.Time
	enc  []byte
} {
	return []struct {
		name string
		t    time.Time
		enc  []byte
	}{
		{name: "the Unix epoch in UTC as no fields", t: time.Unix(0, 0).UTC(), enc: nil},
		{name: "one second after the epoch in UTC", t: time.Unix(1, 0).UTC(), enc: []byte{0x08, 0x02}},
		{name: "a time with nanoseconds", t: time.Unix(1, 500).UTC(), enc: []byte{0x08, 0x02, 0x10, 0xf4, 0x03}},
		{
			name: "a time in a zone one hour east of UTC",
			t:    time.Unix(1, 0).In(time.FixedZone("", hourEast)),
			enc:  []byte{0x08, 0x02, 0x18, 0xa0, 0x38},
		},
		{name: "one second before the epoch in UTC", t: time.Unix(-1, 0).UTC(), enc: []byte{0x08, 0x01}},
		{
			name: "a time in a zone at offset 0 that is not UTC",
			t:    time.Unix(0, 0).In(time.FixedZone("Z0", 0)),
			enc:  []byte{0x18, 0x00},
		},
	}
}

// sinkTime keeps the compiler from dropping the results of benchmarked
// calls.
var sinkTime time.Time

func BenchmarkTime(b *testing.B) {
	now := time.Unix(1_790_000_000, 123_456_789)
	times := []struct {
		name string
		t    time.Time
	}{
		{name: "UTC", t: now.UTC()},
		{name: "the local zone", t: now.In(time.Local)},
		{name: "a zone a whole number of hours east", t: now.In(time.FixedZone("", 2*hourEast))},
	}
	for _, c := range times {
		buf := make([]byte, wire.SizeTime(c.t))
		enc := buf[wire.PutTime(buf, len(buf), c.t):]
		b.Run("SizeTime/"+c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkInt = wire.SizeTime(c.t)
			}
		})
		b.Run("PutTime/"+c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkInt = wire.PutTime(buf, len(buf), c.t)
			}
		})
		b.Run("Time/"+c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkTime, sinkErr = wire.Time(enc, timeLoc, timeNumber, timeOff)
			}
		})
	}
}

// TestTimeAllocs checks that the time functions allocate nothing for a
// time in UTC, in the local zone, and in a zone a whole number of hours
// east of UTC, whose zone time.FixedZone shares. It runs serially:
// testing.AllocsPerRun panics while a parallel test runs.
func TestTimeAllocs(t *testing.T) {
	now := time.Unix(1_790_000_000, 123_456_789)
	times := []struct {
		name string
		t    time.Time
	}{
		{name: "in UTC", t: now.UTC()},
		{name: "in the local zone", t: now.In(time.Local)},
		{name: "in a zone a whole number of hours east", t: now.In(time.FixedZone("", 2*hourEast))},
	}
	for _, c := range times {
		buf := make([]byte, wire.SizeTime(c.t))
		enc := buf[wire.PutTime(buf, len(buf), c.t):]
		t.Run("SizeTime/allocates nothing for a time "+c.name, func(t *testing.T) {
			assert.MaxAllocs(t, func() { sinkInt = wire.SizeTime(c.t) }, 0, "SizeTime allocates nothing")
		})
		t.Run("PutTime/allocates nothing for a time "+c.name, func(t *testing.T) {
			assert.MaxAllocs(t, func() { sinkInt = wire.PutTime(buf, len(buf), c.t) }, 0, "PutTime allocates nothing")
		})
		t.Run("Time/allocates nothing for a time "+c.name, func(t *testing.T) {
			assert.MaxAllocs(t, func() { sinkTime, sinkErr = wire.Time(enc, timeLoc, timeNumber, timeOff) }, 0,
				"Time allocates nothing for a zone it does not create")
		})
	}
}

func TestTime(t *testing.T) {
	t.Parallel()
	t.Run("SizeTime", func(t *testing.T) {
		t.Parallel()
		for _, v := range timeVectors() {
			t.Run("sizes "+v.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, wire.SizeTime(v.t), len(v.enc), "SizeTime returns the length of the encoding")
			})
		}
	})
	t.Run("PutTime", func(t *testing.T) {
		t.Parallel()
		for _, v := range timeVectors() {
			t.Run("writes "+v.name, func(t *testing.T) {
				t.Parallel()
				buf := make([]byte, 16)
				i := wire.PutTime(buf, len(buf), v.t)
				assert.Equal(t, buf[i:], append([]byte{}, v.enc...), "PutTime writes the fields in ascending order")
			})
		}
	})
	t.Run("Time", func(t *testing.T) {
		t.Parallel()
		for _, v := range timeVectors() {
			t.Run("reads "+v.name, func(t *testing.T) {
				t.Parallel()
				got, err := wire.Time(v.enc, timeLoc, timeNumber, timeOff)
				assert.NoError(t, err, "Time decodes the encoding")
				assert.True(t, got.Equal(v.t), "Time decodes the instant")
				_, gotZone := got.Zone()
				_, wantZone := v.t.Zone()
				assert.Equal(t, gotZone, wantZone, "Time decodes the offset of the zone")
				assert.Equal(t, got.Location() == time.UTC, v.t.Location() == time.UTC, "Time decodes UTC as UTC")
			})
		}
		t.Run("reads the last of repeated fields", func(t *testing.T) {
			t.Parallel()
			got, err := wire.Time([]byte{0x08, 0x02, 0x08, 0x04}, timeLoc, timeNumber, timeOff)
			assert.NoError(t, err, "Time decodes repeated fields")
			assert.Equal(t, got, time.Unix(2, 0).UTC(), "Time decodes the last value of a repeated field")
		})
		t.Run("skips an unknown field", func(t *testing.T) {
			t.Parallel()
			got, err := wire.Time([]byte{0x22, 0x01, 0xff, 0x08, 0x02}, timeLoc, timeNumber, timeOff)
			assert.NoError(t, err, "Time skips field 4")
			assert.Equal(t, got, time.Unix(1, 0).UTC(), "the known fields decode")
		})
		t.Run("returns the local zone for the local offset", func(t *testing.T) {
			t.Parallel()
			_, local := time.Unix(1, 0).In(time.Local).Zone()
			got, err := wire.Time(timeWithZone(int64(local)), timeLoc, timeNumber, timeOff)
			assert.NoError(t, err, "Time decodes a zone")
			assert.True(t, got.Location() == time.Local, "a time at the local offset decodes in time.Local")
		})
		t.Run("returns a fixed zone for another offset", func(t *testing.T) {
			t.Parallel()
			_, local := time.Unix(1, 0).In(time.Local).Zone()
			got, err := wire.Time(timeWithZone(int64(local)+1), timeLoc, timeNumber, timeOff)
			assert.NoError(t, err, "Time decodes a zone")
			name, offset := got.Zone()
			assert.Equal(t, name, "", "the fixed zone has no name")
			assert.Equal(t, offset, local+1, "the fixed zone has the decoded offset")
		})
		t.Run("decodes the offsets at the bounds of an int32", func(t *testing.T) {
			t.Parallel()
			for _, zone := range []int64{math.MinInt32, math.MaxInt32} {
				got, err := wire.Time(timeWithZone(zone), timeLoc, timeNumber, timeOff)
				assert.NoError(t, err, "Time decodes an offset that fits an int32")
				_, offset := got.Zone()
				assert.Equal(t, int64(offset), zone, "the zone has the decoded offset")
			}
		})
		t.Run("decodes 999999999 nanoseconds", func(t *testing.T) {
			t.Parallel()
			data := []byte{0x10, 0xff, 0x93, 0xeb, 0xdc, 0x03}
			got, err := wire.Time(data, timeLoc, timeNumber, timeOff)
			assert.NoError(t, err, "Time decodes the largest nanosecond count")
			assert.Equal(t, got, time.Unix(0, 999999999).UTC(), "the nanoseconds decode")
		})
		located := func(cause error, off int, detail string) *kanon.DecodeError {
			return &kanon.DecodeError{
				Type: timeType, Field: timeField, Number: timeNumber, Offset: off, Detail: detail, Err: cause,
			}
		}
		failures := []struct {
			name string
			data []byte
			want *kanon.DecodeError
		}{
			{
				name: "returns io.ErrUnexpectedEOF for a tag that ends early",
				data: []byte{0x08, 0x02, 0x80},
				want: located(io.ErrUnexpectedEOF, timeOff+2, ""),
			},
			{
				name: "returns io.ErrUnexpectedEOF for a value that ends early",
				data: []byte{0x08, 0x82},
				want: located(io.ErrUnexpectedEOF, timeOff+1, ""),
			},
			{
				name: "returns io.ErrUnexpectedEOF for a tag without its value",
				data: []byte{0x08, 0x02, 0x10},
				want: located(io.ErrUnexpectedEOF, timeOff+3, ""),
			},
			{
				name: "returns ErrMalformed for a value that overflows",
				data: append([]byte{0x10}, ones(9, 0x02)...),
				want: located(kanon.ErrMalformed, timeOff+1, "varint overflows 64 bits"),
			},
			{
				name: "returns ErrMalformed at the tag of a known field with another wire format",
				data: []byte{0x08, 0x02, 0x11, 1, 2, 3, 4, 5, 6, 7, 8},
				want: located(kanon.ErrMalformed, timeOff+2, "time field 2 has fixed64, want varint"),
			},
			{
				name: "returns ErrMalformed for field number 0",
				data: []byte{0x00, 0x00},
				want: located(kanon.ErrMalformed, timeOff, "field number 0"),
			},
			{
				name: "returns io.ErrUnexpectedEOF for an unknown field that ends early",
				data: []byte{0x08, 0x02, 0x22, 0x05},
				want: located(io.ErrUnexpectedEOF, timeOff+2, ""),
			},
			{
				name: "returns ErrRange at the value of 1000000000 nanoseconds",
				data: []byte{0x10, 0x80, 0x94, 0xeb, 0xdc, 0x03},
				want: located(kanon.ErrRange, timeOff+1, "time nanoseconds 1000000000 outside 0 to 999999999"),
			},
			{
				name: "returns ErrRange for nanoseconds out of range that valid nanoseconds follow",
				data: []byte{0x10, 0x80, 0x94, 0xeb, 0xdc, 0x03, 0x10, 0x01},
				want: located(kanon.ErrRange, timeOff+1, "time nanoseconds 1000000000 outside 0 to 999999999"),
			},
			{
				name: "returns ErrRange at the value of an offset below the range of an int32",
				data: timeWithZone(math.MinInt32 - 1),
				want: located(kanon.ErrRange, timeOff+3, "time zone offset -2147483649 outside the range of an int32"),
			},
			{
				name: "returns ErrRange at the value of an offset above the range of an int32",
				data: timeWithZone(math.MaxInt32 + 1),
				want: located(kanon.ErrRange, timeOff+3, "time zone offset 2147483648 outside the range of an int32"),
			},
			{
				name: "returns ErrRange for an offset out of range that a valid offset follows",
				data: append(timeWithZone(math.MaxInt32+1), 0x18, 0x00),
				want: located(kanon.ErrRange, timeOff+3, "time zone offset 2147483648 outside the range of an int32"),
			},
		}
		for _, c := range failures {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				got, err := wire.Time(c.data, timeLoc, timeNumber, timeOff)
				assert.Equal(t, got, time.Time{}, "Time returns the zero time with an error")
				assert.Equal(t, assert.ErrorAs[*kanon.DecodeError](t, err, "Time returns a *kanon.DecodeError"), c.want,
					"Time locates the error at the field of the time")
			})
		}
	})
}

// timeWithZone returns the encoding of the Unix epoch plus one second in
// the zone with the offset zone.
func timeWithZone(zone int64) []byte {
	buf := make([]byte, 13)
	i := wire.PutUvarint(buf, len(buf), wire.Zigzag(zone))
	i = wire.PutTag(buf, i, 3<<3|wire.Varint)
	i = wire.PutUvarint(buf, i, 2)
	return buf[wire.PutTag(buf, i, 1<<3|wire.Varint):]
}

func FuzzTime(f *testing.F) {
	for _, v := range timeVectors() {
		f.Add(v.enc)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := wire.Time(data, timeLoc, timeNumber, timeOff)
		if err != nil {
			return
		}
		buf := make([]byte, wire.SizeTime(got))
		i := wire.PutTime(buf, len(buf), got)
		assert.Equal(t, i, 0, "SizeTime measures what PutTime writes")
		again, err := wire.Time(buf, timeLoc, timeNumber, timeOff)
		assert.NoError(t, err, "the encoding of a decoded time decodes")
		assert.True(t, again.Equal(got), "the instant survives a second round trip")
	})
}
