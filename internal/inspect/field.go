// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect

import (
	"bytes"
	"unicode"
	"unicode/utf8"

	"go.thesmos.sh/kanon/wire"
)

// inputLoc names the input of [Parse] in its errors.
const inputLoc = "input"

// Field is one field of a struct encoding, with the struct reading of a
// bytes value.
type Field struct {
	// Number is the field number of the tag, and Wire its wire format:
	// wire.Varint, wire.Fixed64, wire.Bytes or wire.Fixed32.
	Number uint64
	Wire   uint64
	// Offset is the offset of the tag in the input of Parse.
	Offset int
	// Value is the bytes of the value after the tag, without the length of
	// a bytes value. It aliases the input of Parse.
	Value []byte
	// Fields is the struct reading of a bytes value: the fields that Value
	// parses to, with offsets in Value. It is nil for any other value, and
	// for a bytes value that is empty, that does not parse to its end, or
	// that lies deeper than the depth of Parse.
	Fields []Field
}

// Parse returns the fields of the struct encoding in data, in the order of
// the input, with the struct reading of every bytes value down to depth
// levels below data. It reads each field as a decode skips an unknown field,
// and fails as that skip fails. On malformed input it returns the fields
// before the malformed one and a *kanon.DecodeError at the offset of its
// tag, which wraps io.ErrUnexpectedEOF for data that ends inside a tag or a
// value, and kanon.ErrMalformed for a varint of more than 64 bits and a tag
// with field number 0 or wire format 3, 4, 6 or 7.
func Parse(data []byte, depth int) ([]Field, error) {
	return parse(data, depth, inputLoc)
}

// parse is [Parse] with loc naming the input in the errors. A field takes
// two bytes at least, so the loop ends within len(data) iterations.
func parse(data []byte, depth int, loc string) ([]Field, error) {
	var fields []Field
	i := 0
	for range len(data) {
		if i == len(data) {
			break
		}
		tag, n := wire.Uvarint(data[i:])
		if n <= 0 {
			return fields, wire.ReadError(n, loc, 0, i)
		}
		size, err := wire.Skip(data[i+n:], tag, loc, 0, i)
		if err != nil {
			return fields, err
		}
		f := Field{Number: tag >> 3, Wire: tag & 7, Offset: i, Value: data[i+n : i+n+size]}
		if f.Wire == wire.Bytes {
			_, m := wire.Uvarint(f.Value)
			f.Value = f.Value[m:]
			f.Fields = structReading(f.Value, depth)
		}
		fields = append(fields, f)
		i += n + size
	}
	return fields, nil
}

// structReading returns the fields of value, a bytes value, when it is not
// empty, parses to its end and depth is above 0, and nil otherwise.
func structReading(value []byte, depth int) []Field {
	if depth <= 0 || len(value) == 0 {
		return nil
	}
	fields, err := parse(value, depth-1, inputLoc)
	if err != nil {
		return nil
	}
	return fields
}

// isText reports whether value reads as text: valid UTF-8 without control
// characters other than tab and newline.
func isText(value []byte) bool {
	return utf8.Valid(value) && !bytes.ContainsFunc(value, isControl)
}

// isControl reports whether r is a control character other than tab and
// newline.
func isControl(r rune) bool {
	return unicode.IsControl(r) && r != '\t' && r != '\n'
}

// varintReading returns the values of value read as a run of varints of 64
// bits at most, and nil when value does not end at the end of such a
// varint. An empty value is a run of no varints. The run has one varint per
// byte of value without the continuation bit.
func varintReading(value []byte) []uint64 {
	out := make([]uint64, wire.CountVarints(value))
	for k := range out {
		u, n := wire.Uvarint(value)
		if n < 1 {
			return nil
		}
		out[k], value = u, value[n:]
	}
	if len(value) > 0 {
		return nil
	}
	return out
}
