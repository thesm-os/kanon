// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import (
	"cmp"
	"time"
)

// CompareBool returns -1, 0 or +1 as the map key a sorts before, with or
// after b: false before true.
func CompareBool(a, b bool) int {
	if a == b {
		return 0
	}
	if a {
		return 1
	}
	return -1
}

// CompareComplex returns -1, 0 or +1 as the map key a sorts before, with or
// after b: by the real parts, then by the imaginary parts, each as
// cmp.Compare orders floats, NaN first. A complex64 key converts without
// loss.
func CompareComplex(a, b complex128) int {
	if c := cmp.Compare(real(a), real(b)); c != 0 {
		return c
	}
	return cmp.Compare(imag(a), imag(b))
}

// CompareTime returns -1, 0 or +1 as the map key a sorts before, with or
// after b: by Unix seconds, then by nanoseconds, then a time in UTC before
// a time in a zone, then by the offsets of the zones. The order leaves out
// the monotonic clock reading and the name of the zone, as the encoding
// does, so two times compare equal exactly when they encode alike.
func CompareTime(a, b time.Time) int {
	if c := cmp.Compare(a.Unix(), b.Unix()); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Nanosecond(), b.Nanosecond()); c != 0 {
		return c
	}
	au, bu := a.Location() == time.UTC, b.Location() == time.UTC
	if au || bu {
		return CompareBool(!au, !bu)
	}
	_, ao := a.Zone()
	_, bo := b.Zone()
	return cmp.Compare(ao, bo)
}
