// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect

import (
	"encoding/hex"
	"io"
	"math"
	"strconv"
	"strings"

	"go.thesmos.sh/kanon/wire"
)

// Tokens of the text notation of protoscope that the text writes.
const (
	// indentStep indents the fields of a struct reading by one level.
	indentStep = "  "
	// commentStart separates a value from the comment after it.
	commentStart = "  # "
	// fixed64Suffix and fixed32Suffix follow an integer that encodes as
	// eight or four bytes.
	fixed64Suffix = "i64"
	fixed32Suffix = "i32"
	// hexQuote delimits hexadecimal digits, and textQuote a string.
	hexQuote  = "`"
	textQuote = `"`
)

// Bit sizes of the float readings of a fixed64 and a fixed32 value.
const (
	float64Bits = 64
	float32Bits = 32
)

// quoteEscapes escapes the bytes of text that protoscope reads as the syntax
// of a string: a backslash and a double quote, and a newline and a tab,
// which it would keep as they are and which would break or hide the line.
var quoteEscapes = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\x09`)

// WriteText writes fields to w in the text notation of protoscope, one line
// per field of the form "number: value":
//
//   - a varint as its unsigned value, and in a comment its zigzag value, when
//     it is not 0;
//   - a fixed64 and a fixed32 as their unsigned value with the suffix i64 or
//     i32, and in a comment their float value, after their signed value when
//     that is negative;
//   - a bytes value with a struct reading as its fields between braces, one
//     level deeper, and in a comment its text when it is also text;
//   - any other bytes value between braces: {} when empty, a quoted string
//     when it is text, and backquoted hexadecimal digits otherwise, with its
//     values in a comment when it is a run of varints.
//
// protoscope assembles the text into the input of Parse when every varint
// of the input has its shortest form, as an encoder writes it. WriteText
// returns the error of the write.
func WriteText(w io.Writer, fields []Field) error {
	var b strings.Builder
	appendText(&b, fields, "")
	_, err := io.WriteString(w, b.String())
	return err
}

// appendText appends the lines of fields to b, each line after indent.
func appendText(b *strings.Builder, fields []Field, indent string) {
	for _, f := range fields {
		b.WriteString(indent)
		b.WriteString(strconv.FormatUint(f.Number, 10))
		b.WriteString(": ")
		switch f.Wire {
		case wire.Varint:
			u, _ := wire.Uvarint(f.Value)
			b.WriteString(strconv.FormatUint(u, 10))
			if u != 0 {
				appendComment(b, "zigzag "+strconv.FormatInt(wire.Unzigzag(u), 10))
			}
		case wire.Fixed64:
			u, _ := wire.Uint64(f.Value)
			b.WriteString(strconv.FormatUint(u, 10) + fixed64Suffix)
			appendComment(b, fixedComment(int64(u), math.Float64frombits(u), float64Bits))
		case wire.Fixed32:
			u, _ := wire.Uint32(f.Value)
			b.WriteString(strconv.FormatUint(uint64(u), 10) + fixed32Suffix)
			appendComment(b, fixedComment(int64(int32(u)), float64(math.Float32frombits(u)), float32Bits))
		default:
			appendBytes(b, f, indent)
		}
		b.WriteByte('\n')
	}
}

// appendBytes appends the value of f, a bytes value, to b: its struct
// reading, whose lines follow indent one level deeper, or {} for an empty
// value, a quoted string for text and backquoted hexadecimal digits
// otherwise.
func appendBytes(b *strings.Builder, f Field, indent string) {
	if f.Fields != nil {
		b.WriteByte('{')
		if isText(f.Value) {
			appendComment(b, textQuote+quoteEscapes.Replace(string(f.Value))+textQuote)
		}
		b.WriteByte('\n')
		appendText(b, f.Fields, indent+indentStep)
		b.WriteString(indent)
		b.WriteByte('}')
	} else if len(f.Value) == 0 {
		b.WriteString("{}")
	} else if isText(f.Value) {
		b.WriteString("{" + textQuote + quoteEscapes.Replace(string(f.Value)) + textQuote + "}")
	} else {
		b.WriteString("{" + hexQuote + hex.EncodeToString(f.Value) + hexQuote + "}")
		if values := varintReading(f.Value); values != nil {
			appendComment(b, varintComment(values))
		}
	}
}

// appendComment appends a comment of text to b, after the value on its
// line.
func appendComment(b *strings.Builder, text string) {
	b.WriteString(commentStart)
	b.WriteString(text)
}

// fixedComment returns the comment of a fixed-size value whose signed
// reading is signed and whose float reading is float, a float of the bit
// size bits: the signed reading when it is negative, and the float reading.
func fixedComment(signed int64, float float64, bits int) string {
	text := "float " + strconv.FormatFloat(float, 'g', -1, bits)
	if signed < 0 {
		return "signed " + strconv.FormatInt(signed, 10) + ", " + text
	}
	return text
}

// varintComment returns the comment of a bytes value that is a run of the
// varints values: the values and their zigzag values.
func varintComment(values []uint64) string {
	var unsigned, zigzag strings.Builder
	for k, u := range values {
		if k > 0 {
			unsigned.WriteByte(' ')
			zigzag.WriteByte(' ')
		}
		unsigned.WriteString(strconv.FormatUint(u, 10))
		zigzag.WriteString(strconv.FormatInt(wire.Unzigzag(u), 10))
	}
	return "varints " + unsigned.String() + ", zigzag " + zigzag.String()
}
