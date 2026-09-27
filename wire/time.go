// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"math"
	"strconv"
	"time"

	"go.thesmos.sh/kanon"
)

// The fields of the encoding of a time: its Unix seconds, its nanoseconds
// within the second, and the offset of its zone in seconds east of UTC.
// All three are varints, and the seconds and the offset are zigzag
// encoded.
const (
	timeSeconds = 1
	timeNanos   = 2
	timeZone    = 3
)

// maxNanos is the largest nanosecond count of a time.
const maxNanos = 999999999

// SizeTime returns the length of the encoding of t without its length
// prefix: the seconds and the nanoseconds of t, each left out when zero,
// and the offset of its zone when its location is not UTC.
func SizeTime(t time.Time) int {
	n := 0
	if s := t.Unix(); s != 0 {
		n += 1 + SizeUvarint(Zigzag(s))
	}
	if ns := t.Nanosecond(); ns != 0 {
		n += 1 + SizeUvarint(uint64(ns))
	}
	if t.Location() != time.UTC {
		_, zone := t.Zone()
		n += 1 + SizeUvarint(Zigzag(int64(zone)))
	}
	return n
}

// PutTime writes the encoding of t without its length prefix, as
// [SizeTime] measures it, into the bytes of buf that end at i, and returns
// the offset of the first. The fields ascend.
func PutTime(buf []byte, i int, t time.Time) int {
	if t.Location() != time.UTC {
		_, zone := t.Zone()
		i = PutUvarint(buf, i, Zigzag(int64(zone)))
		i = PutTag(buf, i, timeZone<<3|Varint)
	}
	if ns := t.Nanosecond(); ns != 0 {
		i = PutUvarint(buf, i, uint64(ns))
		i = PutTag(buf, i, timeNanos<<3|Varint)
	}
	if s := t.Unix(); s != 0 {
		i = PutUvarint(buf, i, Zigzag(s))
		i = PutTag(buf, i, timeSeconds<<3|Varint)
	}
	return i
}

// Time decodes data, the encoding of a time without its length prefix,
// which starts at offset off of the slab. A time without a zone is in UTC.
// A time with one is in time.Local when the local zone has that offset at
// that instant, and in time.FixedZone("", offset) otherwise, as
// encoding/gob decodes a time. Empty data is the Unix epoch in UTC. A field
// that repeats takes its last value, and an unknown field is skipped.
//
// Time fails for malformed input as the decode of a struct fails, for
// nanoseconds above 999999999, and for a zone offset outside the range of
// an int32, with errors that loc and num locate.
func Time(data []byte, loc string, num, off int) (time.Time, error) {
	var secs, zone int64
	var nanos uint64
	zoned := false
	for i := 0; i < len(data); {
		at := i
		tag, n := Uvarint(data[i:])
		if n <= 0 {
			return time.Time{}, ReadError(n, loc, num, off+i)
		}
		i += n
		field := tag >> 3
		if field < timeSeconds || field > timeZone {
			skipped, err := Skip(data[i:], tag, loc, num, off+at)
			if err != nil {
				return time.Time{}, err
			}
			i += skipped
			continue
		}
		if tag&7 != Varint {
			return time.Time{}, decodeError(kanon.ErrMalformed, loc, num, off+at,
				"time field "+strconv.FormatUint(field, 10)+" has "+wireName(tag&7)+", want varint")
		}
		v, n := Uvarint(data[i:])
		if n <= 0 {
			return time.Time{}, ReadError(n, loc, num, off+i)
		}
		i += n
		switch field {
		case timeSeconds:
			secs = Unzigzag(v)
		case timeNanos:
			nanos = v
		default:
			zone, zoned = Unzigzag(v), true
		}
	}
	if nanos > maxNanos {
		return time.Time{}, decodeError(kanon.ErrRange, loc, num, off,
			"time nanoseconds "+strconv.FormatUint(nanos, 10)+" outside 0 to 999999999")
	}
	if zone < math.MinInt32 || zone > math.MaxInt32 {
		return time.Time{}, decodeError(kanon.ErrRange, loc, num, off,
			"time zone offset "+strconv.FormatInt(zone, 10)+" outside the range of an int32")
	}
	t := time.Unix(secs, int64(nanos))
	if !zoned {
		return t.UTC(), nil
	}
	local := t.In(time.Local)
	if _, lo := local.Zone(); lo == int(zone) {
		return local, nil
	}
	return t.In(time.FixedZone("", int(zone))), nil
}
