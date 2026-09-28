// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire

import "go.thesmos.sh/kanon"

// repeat selects what a scan of an encoding does with a second occurrence
// of the field that it looks for.
type repeat uint8

// The choices of a scan for a second occurrence of its field.
const (
	// takeLast takes the value of the last occurrence, as a decode takes it.
	takeLast repeat = 0
	// rejectRepeat fails at the tag of the second occurrence.
	rejectRepeat repeat = 1
)

// Find returns the offset in data, the encoding of a struct, of the value
// of the last occurrence of the field whose tag is tag: the offset of the
// varint of a field of the wire format Varint, of the bytes of Fixed32 and
// Fixed64, and of the length of Bytes. It returns -1 when data has no such
// field. A view reads one field of an encoding with it, and a field that
// repeats takes its last value, as a decode takes it.
//
// Find fails for malformed input as the decode of a struct fails, with
// errors at the struct that loc, "Type.Field", names, and for an occurrence
// of the field number of tag with another wire format, with an error at the
// field.
func Find(data []byte, tag uint64, loc string) (int, error) {
	return scan(data, tag, loc, takeLast)
}

// FindOne returns the offset of the value of the one occurrence of the
// field whose tag is tag, as [Find] returns it, for the view of a struct
// field: a decode merges two occurrences of a struct, which one byte slice
// cannot hold. It fails as Find fails, and for a second occurrence of the
// field with a *kanon.DecodeError that wraps kanon.ErrRepeatedView at its
// tag.
func FindOne(data []byte, tag uint64, loc string) (int, error) {
	return scan(data, tag, loc, rejectRepeat)
}

// scan returns the offset of the value of the field whose tag is tag in
// data, for [Find] and [FindOne], and does what r selects with a second
// occurrence of the field.
func scan(data []byte, tag uint64, loc string, r repeat) (int, error) {
	typ, _ := split(loc)
	at := -1
	for i := 0; i < len(data); {
		start := i
		t, n := Uvarint(data[i:])
		if n <= 0 {
			return -1, ReadError(n, typ, 0, i)
		}
		i += n
		skipped, err := Skip(data[i:], t, typ, 0, start)
		if err != nil {
			return -1, err
		}
		if t>>3 == tag>>3 {
			if t != tag {
				return -1, FormatError(t, int(tag&7), loc, start)
			}
			if at != -1 && r == rejectRepeat {
				return -1, decodeError(kanon.ErrRepeatedView, loc, int(tag>>3), start, "")
			}
			at = i
		}
		i += skipped
	}
	return at, nil
}
