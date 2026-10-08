// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// tagKey is the struct tag key that kanon reads.
const tagKey = "kanon"

// Words of a kanon tag.
const (
	// tagSkip leaves a field out of the encoding.
	tagSkip = "-"
	// tagFixed selects the fixed-size encoding for the 32- and 64-bit
	// integers of a field.
	tagFixed = "fixed"
	// tagUnknown marks the []byte field that keeps the unknown fields of a
	// decode.
	tagUnknown = "unknown"
	// tagStream marks a streamed field, whose value the stream decoder of its
	// struct returns to its caller instead of decoding it.
	tagStream = "stream"
	// tagUnion names the discriminator field of a union member, as in
	// "union=Kind".
	tagUnion = "union="
	// optionTypes names the option that lists the concrete types that the
	// interfaces of a field store, and tagTypes begins it in a tag, as in
	// "types=Circle|*Square".
	optionTypes = "types"
	tagTypes    = "types="
	// optionMax names the option that bounds the number of the elements of a
	// slice or a map field, and tagMax begins it in a tag, as in "max=1024".
	optionMax = "max"
	tagMax    = "max="
	// tagSeparator separates the words of a tag.
	tagSeparator = ','
	// typesSeparator separates the concrete types in the value of the types
	// option.
	typesSeparator = "|"
)

// Brackets of a Go type expression. A tag separator between an opening and
// its closing bracket belongs to the type expression.
const (
	openBrackets  = "[({"
	closeBrackets = "])}"
)

// maxFieldNumber is the largest field number, the largest int32, since a
// kanon.DecodeError reports the number as an int, which is 32 bits wide on
// some platforms.
const maxFieldNumber = 2147483647

// maxBound is the largest bound of the tag option max, the largest int32,
// since the generated code passes the bound as an int, which is 32 bits wide
// on some platforms.
const maxBound = 2147483647

// tag is a parsed kanon struct tag, whose zero value is the tag of a field
// without a kanon tag: a number in declaration order, the default encoding
// and no union.
type tag struct {
	// union names the discriminator field of a union member.
	union string
	// types is the value of the types option: Go type expressions joined by
	// typesSeparator, or "" for a tag without the option.
	types string
	// num is the field number that the tag sets, or 0.
	num int
	// max is the bound of the tag option max, the most elements of the slice
	// or the map of the field, or 0 for a tag without the option.
	max int
	// tagged reports that the field has a kanon tag, even an empty one,
	// which opts an unexported field into the encoding.
	tagged bool
	// skip leaves the field out of the encoding.
	skip bool
	// fixed selects the fixed-size encoding.
	fixed bool
	// unknown marks the field that keeps unknown fields.
	unknown bool
	// stream marks a streamed field.
	stream bool
}

// parseTag parses the value of the kanon key of the struct tag structTag:
// "-" or a comma-separated list, in any order, of at most one field number
// and the words fixed, unknown, stream, union=Name, types=T1|T2, which lists
// Go type expressions, and max=N, a decimal bound. A comma inside brackets,
// parentheses or braces belongs to the type expression around it, so that
// `types=Pair[int, string]` is one word. parseTag ignores empty words, so
// that `kanon:",fixed"` selects the encoding alone and `kanon:""` states only
// that the field has a tag.
//
// parseTag fails for an unknown word, a second number, a number outside 1
// to 2147483647, an empty union name, a second types option, a types option
// with an empty type, a second max option, a max that is not a decimal from 1
// to 2147483647, and the word unknown beside any other word.
func parseTag(structTag string) (tag, error) {
	value, ok := reflect.StructTag(structTag).Lookup(tagKey)
	if !ok {
		return tag{}, nil
	}
	if value == tagSkip {
		return tag{tagged: true, skip: true}, nil
	}
	t := tag{tagged: true}
	for _, word := range tagWords(value) {
		if err := t.add(value, strings.TrimSpace(word)); err != nil {
			return tag{}, err
		}
	}
	if t.unknown && (t.num != 0 || t.fixed || t.union != "" || t.types != "" || t.stream || t.max != 0) {
		return tag{}, fmt.Errorf("kanon: tag %q: the word %s takes no other word", value, tagUnknown)
	}
	return t, nil
}

// add parses word, one word of the tag value value, into t. It fails for a
// word that [parseTag] rejects, and cites value in the error.
func (t *tag) add(value, word string) error {
	if word == "" {
		return nil
	}
	if word == tagFixed {
		t.fixed = true
		return nil
	}
	if word == tagUnknown {
		t.unknown = true
		return nil
	}
	if word == tagStream {
		t.stream = true
		return nil
	}
	if union, ok := strings.CutPrefix(word, tagUnion); ok {
		if union == "" {
			return fmt.Errorf("kanon: tag %q: union names no discriminator field", value)
		}
		t.union = union
		return nil
	}
	if types, ok := strings.CutPrefix(word, tagTypes); ok {
		if t.types != "" {
			return fmt.Errorf("kanon: tag %q: second types option", value)
		}
		for typ := range strings.SplitSeq(types, typesSeparator) {
			if strings.TrimSpace(typ) == "" {
				return fmt.Errorf("kanon: tag %q: types lists an empty type", value)
			}
		}
		t.types = types
		return nil
	}
	if bound, ok := strings.CutPrefix(word, tagMax); ok {
		if t.max != 0 {
			return fmt.Errorf("kanon: tag %q: second %s option", value, optionMax)
		}
		n, err := strconv.ParseInt(bound, 10, 64)
		if err != nil || n < 1 || n > maxBound {
			return fmt.Errorf("kanon: tag %q: %s %q is not a decimal from 1 to %d", value, optionMax, bound, maxBound)
		}
		t.max = int(n)
		return nil
	}
	n, err := strconv.ParseInt(word, 10, 64)
	if err != nil {
		return fmt.Errorf("kanon: tag %q: unknown word %q", value, word)
	}
	if t.num != 0 {
		return fmt.Errorf("kanon: tag %q: second field number %d", value, n)
	}
	if n < 1 || n > maxFieldNumber {
		return fmt.Errorf("kanon: tag %q: field number %d outside 1 to %d", value, n, maxFieldNumber)
	}
	t.num = int(n)
	return nil
}

// concreteTypes returns the type expressions that the types option of t
// lists, in tag order, without the spaces around them.
func (t tag) concreteTypes() []string {
	var out []string
	for typ := range strings.SplitSeq(t.types, typesSeparator) {
		out = append(out, strings.TrimSpace(typ))
	}
	return out
}

// tagWords returns the words of the kanon tag value, split at the tag
// separators outside brackets, parentheses and braces.
func tagWords(value string) []string {
	var out []string
	depth, start := 0, 0
	for i, r := range value {
		if strings.ContainsRune(openBrackets, r) {
			depth++
		} else if strings.ContainsRune(closeBrackets, r) {
			depth--
		} else if r == tagSeparator && depth == 0 {
			out = append(out, value[start:i])
			start = i + 1
		}
	}
	return append(out, value[start:])
}
