// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
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
// Time fails for malformed input as the decode of a struct fails, and for
// each occurrence of nanoseconds above 999999999 and of a zone offset
// outside the range of an int32, at the offset of its value, with errors
// that loc and num locate.
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
		switch field {
		case timeSeconds:
			secs = Unzigzag(v)
		case timeNanos:
			if v > maxNanos {
				return time.Time{}, decodeError(kanon.ErrRange, loc, num, off+i,
					"time nanoseconds "+strconv.FormatUint(v, 10)+" outside 0 to 999999999")
			}
			nanos = v
		default:
			z := Unzigzag(v)
			if z != int64(int32(z)) {
				return time.Time{}, decodeError(kanon.ErrRange, loc, num, off+i,
					"time zone offset "+strconv.FormatInt(z, 10)+" outside the range of an int32")
			}
			zone, zoned = z, true
		}
		i += n
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

// CanonicalTime decodes data as [Time] does, and accepts only the encoding
// that [PutTime] writes: fields 1 to 3 in ascending order, each at most once,
// each varint in its shortest form, and fields 1 and 2 not 0. It matches the
// tag bytes of the fields in their order, and checks the value of a field in
// this order: that its varint reads, that the varint is in its shortest form,
// the range of the nanoseconds and of the zone offset, and that field 1 or 2
// is not 0. A byte that is not the tag of a field after the fields before it
// fails at its offset: a tag that does not read or is not in its shortest
// form, field number 0, a field number that is not above the one before it,
// a field other than 1 to 3, and a field in another wire format than a
// varint. The errors of the canonical rules wrap kanon.ErrNotCanonical.
func CanonicalTime(data []byte, loc string, num, off int) (time.Time, error) {
	var secs, zone int64
	var nanos, prior uint64
	zoned := false
	i := 0
	if i < len(data) && data[i] == timeSeconds<<3|Varint {
		v, n := Uvarint(data[i+1:])
		if n <= 0 || n > 1 && data[i+n] == 0 || v == 0 {
			return time.Time{}, timeFieldError(data, i, v, n, timeSeconds, loc, num, off)
		}
		secs, prior, i = Unzigzag(v), timeSeconds, i+1+n
	}
	if i < len(data) && data[i] == timeNanos<<3|Varint {
		v, n := Uvarint(data[i+1:])
		if n <= 0 || n > 1 && data[i+n] == 0 || v-1 >= maxNanos {
			return time.Time{}, timeFieldError(data, i, v, n, timeNanos, loc, num, off)
		}
		nanos, prior, i = v, timeNanos, i+1+n
	}
	if i < len(data) && data[i] == timeZone<<3|Varint {
		v, n := Uvarint(data[i+1:])
		zone = Unzigzag(v)
		if n <= 0 || n > 1 && data[i+n] == 0 || zone != int64(int32(zone)) {
			return time.Time{}, timeFieldError(data, i, v, n, timeZone, loc, num, off)
		}
		zoned, prior, i = true, timeZone, i+1+n
	}
	if i != len(data) {
		return time.Time{}, timeTagError(data, i, prior, loc, num, off)
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

// timeFieldError returns the error of a canonical time for the field whose
// tag is at data[i] and whose varint, of the value v and the length n that
// [Uvarint] returns, breaks a rule, checked in this order: the error of
// [ReadError] when the varint does not read, of [LongFormError] when it is
// not in its shortest form, an error that wraps kanon.ErrRange for a zone
// offset outside an int32 and for nanoseconds above 999999999, and an error
// that wraps kanon.ErrNotCanonical for field 1 or 2 of 0. The errors of the
// varint are at its offset, and the error of a value of 0 at the offset of
// the tag.
func timeFieldError(data []byte, i int, v uint64, n int, field uint64, loc string, num, off int) error {
	if n <= 0 {
		return ReadError(n, loc, num, off+i+1)
	}
	if n > 1 && data[i+n] == 0 {
		return LongFormError(n, loc, num, off+i+1)
	}
	if field == timeZone {
		return zoneError(Unzigzag(v), loc, num, off+i+1)
	}
	if v > maxNanos {
		return nanosError(v, loc, num, off+i+1)
	}
	return decodeError(kanon.ErrNotCanonical, loc, num, off+i, "time field "+strconv.FormatUint(field, 10)+" of 0")
}

// timeTagError returns the error of a canonical time for the tag at data[i],
// which is not the tag of a field after prior, the last field that the time
// decoded: the error of [ReadError] for a tag that does not read, of
// [LongFormError] for a tag that is not in its shortest form, of
// [OrderError] for a field number that is not above prior, of
// [UnknownFieldError] for a field other than 1 to 3, of [TagError] for field
// number 0, and an error that wraps kanon.ErrMalformed for a field of another
// wire format than a varint.
func timeTagError(data []byte, i int, prior uint64, loc string, num, off int) error {
	tag, n := Uvarint(data[i:])
	if n <= 0 {
		return ReadError(n, loc, num, off+i)
	}
	if n > 1 && data[i+n-1] == 0 {
		return LongFormError(n, loc, num, off+i)
	}
	field := tag >> 3
	if field == 0 {
		return TagError(tag, loc, num, off+i)
	}
	if field <= prior {
		return OrderError(field, prior, loc, num, off+i)
	}
	if field > timeZone {
		return UnknownFieldError(field, loc, num, off+i)
	}
	return decodeError(kanon.ErrMalformed, loc, num, off+i,
		"time field "+strconv.FormatUint(field, 10)+" has "+wireName(tag&7)+", want varint")
}

// nanosError returns the error for the nanoseconds v of a time, above
// 999999999, at the offset off of the value. It wraps kanon.ErrRange.
func nanosError(v uint64, loc string, num, off int) error {
	return decodeError(kanon.ErrRange, loc, num, off,
		"time nanoseconds "+strconv.FormatUint(v, 10)+" outside 0 to 999999999")
}

// zoneError returns the error for the zone offset z of a time, outside the
// range of an int32, at the offset off of the value. It wraps kanon.ErrRange.
func zoneError(z int64, loc string, num, off int) error {
	return decodeError(kanon.ErrRange, loc, num, off,
		"time zone offset "+strconv.FormatInt(z, 10)+" outside the range of an int32")
}
