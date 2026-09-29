// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"strconv"

	"go.thesmos.sh/kanon/wire"
)

// jsonIndent indents each level of the JSON that the writers write.
const jsonIndent = "  "

// Names of the wire formats in JSON.
const (
	varintName  = "varint"
	fixed64Name = "fixed64"
	bytesName   = "bytes"
	fixed32Name = "fixed32"
)

// fieldsDoc is the JSON document of the fields of one struct encoding.
type fieldsDoc struct {
	Fields []any `json:"fields"`
}

// varintJSON is the JSON of a varint field, with its unsigned and its
// zigzag reading.
type varintJSON struct {
	Field    uint64 `json:"field"`
	Wire     string `json:"wire"`
	Offset   int    `json:"offset"`
	Unsigned uint64 `json:"unsigned"`
	Zigzag   int64  `json:"zigzag"`
}

// fixedJSON is the JSON of a fixed64 or fixed32 field, with its unsigned,
// its signed and its float reading. The float is a string, since JSON has no
// NaN or infinity.
type fixedJSON struct {
	Field    uint64 `json:"field"`
	Wire     string `json:"wire"`
	Offset   int    `json:"offset"`
	Unsigned uint64 `json:"unsigned"`
	Signed   int64  `json:"signed"`
	Float    string `json:"float"`
}

// bytesJSON is the JSON of a bytes field: the length and the hexadecimal
// digits of its value, and each other reading that the value has.
type bytesJSON struct {
	Field  uint64 `json:"field"`
	Wire   string `json:"wire"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Hex    string `json:"hex"`
	// Text is the value when it is text, and nil otherwise.
	Text *string `json:"text,omitempty"`
	// Fields is the struct reading of the value, and Varints its reading as
	// a run of varints.
	Fields  []any         `json:"fields,omitempty"`
	Varints []varintValue `json:"varints,omitempty"`
}

// varintValue is one varint of a run of varints, with its unsigned and its
// zigzag reading.
type varintValue struct {
	Unsigned uint64 `json:"unsigned"`
	Zigzag   int64  `json:"zigzag"`
}

// WriteJSON writes fields to w as the JSON object {"fields": [...]}, with
// every reading of each value: "unsigned" and "zigzag" for a varint;
// "unsigned", "signed" and "float" for a fixed64 and a fixed32, the float as
// a string; and "length" and "hex" for a bytes value, with "text", "fields"
// and "varints" when the value has those readings. Each field has "field",
// "wire" and "offset". WriteJSON returns the error of the write.
func WriteJSON(w io.Writer, fields []Field) error {
	return writeJSON(w, fieldsDoc{Fields: jsonFields(fields)})
}

// writeJSON writes doc to w as indented JSON without HTML escapes, and
// returns the error of the write.
func writeJSON(w io.Writer, doc any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", jsonIndent)
	return enc.Encode(doc)
}

// jsonFields returns the JSON of fields, as [WriteJSON] states it, and an
// empty array for no fields.
func jsonFields(fields []Field) []any {
	out := make([]any, 0, len(fields))
	for _, f := range fields {
		switch f.Wire {
		case wire.Varint:
			u, _ := wire.Uvarint(f.Value)
			out = append(out, varintJSON{
				Field: f.Number, Wire: varintName, Offset: f.Offset, Unsigned: u, Zigzag: wire.Unzigzag(u),
			})
		case wire.Fixed64:
			u, _ := wire.Uint64(f.Value)
			out = append(out, fixedJSON{
				Field: f.Number, Wire: fixed64Name, Offset: f.Offset, Unsigned: u, Signed: int64(u),
				Float: strconv.FormatFloat(math.Float64frombits(u), 'g', -1, float64Bits),
			})
		case wire.Fixed32:
			u, _ := wire.Uint32(f.Value)
			out = append(out, fixedJSON{
				Field: f.Number, Wire: fixed32Name, Offset: f.Offset, Unsigned: uint64(u), Signed: int64(int32(u)),
				Float: strconv.FormatFloat(float64(math.Float32frombits(u)), 'g', -1, float32Bits),
			})
		default:
			out = append(out, bytesField(f))
		}
	}
	return out
}

// bytesField returns the JSON of f, a bytes field.
func bytesField(f Field) bytesJSON {
	b := bytesJSON{
		Field: f.Number, Wire: bytesName, Offset: f.Offset, Length: len(f.Value), Hex: hex.EncodeToString(f.Value),
	}
	if isText(f.Value) {
		text := string(f.Value)
		b.Text = &text
	}
	if f.Fields != nil {
		b.Fields = jsonFields(f.Fields)
	}
	if values := varintReading(f.Value); values != nil {
		b.Varints = make([]varintValue, len(values))
		for k, u := range values {
			b.Varints[k] = varintValue{Unsigned: u, Zigzag: wire.Unzigzag(u)}
		}
	}
	return b
}
