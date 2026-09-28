// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"bytes"
	"errors"
	"fmt"
	"go/types"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

// Syntax of the numbers line that a code file contains for each struct and
// each list of concrete types: `//kanon:numbers Order ID=1 Kind=2 ~5`. A key
// or a name that contains a space, such as the name of a struct type
// literal or of an instantiation with two type arguments, is a Go string
// literal: `//kanon:numbers Holder.Any "Pair[int, string]"=1`.
const (
	// numbersMark is the first word of a numbers line.
	numbersMark = "//kanon:numbers"
	// numberSeparator joins a name and its number.
	numberSeparator = "="
	// reservedMark precedes a reserved number.
	reservedMark = "~"
	// quoteMark begins and ends a quoted key or name.
	quoteMark = `"`
)

// record is the numbering that a numbers line records for one struct or one
// list of concrete types.
type record struct {
	// nums maps the name of each field or concrete type to its number.
	nums map[string]int
	// order lists the names in the order of the numbers line.
	order []string
	// reserved lists the reserved numbers: the numbers of removed fields and
	// types, which the automatic numbering never gives again.
	reserved []int
}

// recordFor returns the record of a struct or a list that the code files
// can record under any of keys: the record that [recordOf] picks for the
// first of keys that a code file records, or an empty record.
func recordFor(keys []string, own records, others map[string][]recordAt) (*record, error) {
	for _, k := range keys {
		if own[k] != nil || len(others[k]) > 0 {
			return recordOf(k, own, others)
		}
	}
	return &record{}, nil
}

// recordOf returns the record under the key name: its record in own, the
// records of the code file that kanon generates, or else the record that
// the other code files of the package keep for it, or else an empty record.
// A struct that moved to another source file keeps its numbers this way.
// recordOf fails when two other code files keep different records for name.
func recordOf(name string, own records, others map[string][]recordAt) (*record, error) {
	if rec := own[name]; rec != nil {
		return rec, nil
	}
	kept := others[name]
	if len(kept) == 0 {
		return &record{}, nil
	}
	for _, k := range kept[1:] {
		if !k.rec.equal(kept[0].rec) {
			return nil, fmt.Errorf(
				"kanon: %s and %s record different numbers for %s: regenerate the one that does not generate %s",
				kept[0].file,
				k.file,
				name,
				name,
			)
		}
	}
	return kept[0].rec, nil
}

// add parses word, one word of the numbers line for owner, into rec:
// `Name=N`, `"Name"=N` or `~N`. It fails for a malformed word and for a
// name that the line repeats.
func (rec *record) add(owner, word string) error {
	if n, ok := strings.CutPrefix(word, reservedMark); ok {
		num, err := parseNumber(owner, n)
		if err != nil {
			return err
		}
		rec.reserved = append(rec.reserved, num)
		return nil
	}
	name, n, ok := strings.Cut(word, numberSeparator)
	if strings.HasPrefix(word, quoteMark) {
		name, n, ok = cutQuoted(word)
	}
	if !ok || name == "" {
		return fmt.Errorf("kanon: the numbers line for %s has the malformed word %q", owner, word)
	}
	num, err := parseNumber(owner, n)
	if err != nil {
		return err
	}
	if _, dup := rec.nums[name]; dup {
		return fmt.Errorf("kanon: the numbers line for %s repeats field %s", owner, name)
	}
	rec.nums[name] = num
	rec.order = append(rec.order, name)
	return nil
}

// equal reports whether rec and other give the same names the same numbers
// and reserve the same numbers in the same order.
func (rec *record) equal(other *record) bool {
	return maps.Equal(rec.nums, other.nums) && slices.Equal(rec.reserved, other.reserved)
}

// line returns the numbers line of rec under the key name, as the line that
// rec was parsed from reads.
func (rec *record) line(name string) string {
	return renderNumbers(name, rec.order, rec.nums, rec.reserved)
}

// records maps the key of each numbers line of a code file to its record.
type records map[string]*record

// readRecords returns the records of the code file at path. A missing file
// records nothing. readRecords fails for a file that does not read or
// parse.
func readRecords(path string) (records, error) {
	src, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return records{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("kanon: read the numbers of %s: %w", path, err)
	}
	return parseRecords(src)
}

// parseRecords parses the numbers lines of the Go source src: the lines whose
// first word is numbersMark. A line without a key, a quote that does not
// end, a malformed word, and a key or a name that the lines repeat fail the
// parse, since they mean a hand-edited code file.
func parseRecords(src []byte) (records, error) {
	recs := records{}
	for line := range bytes.Lines(src) {
		if first := strings.Fields(string(line)); len(first) == 0 || first[0] != numbersMark {
			continue
		}
		words, err := lineWords(string(line))
		if err != nil {
			return nil, fmt.Errorf(
				"kanon: the numbers line %q has a quote that does not end",
				strings.TrimSpace(string(line)),
			)
		}
		if len(words) == 1 {
			return nil, errors.New("kanon: a numbers line names no struct")
		}
		name := words[1]
		if strings.HasPrefix(name, quoteMark) {
			if name, err = strconv.Unquote(name); err != nil {
				return nil, fmt.Errorf("kanon: a numbers line has the malformed key %q", words[1])
			}
		}
		if recs[name] != nil {
			return nil, fmt.Errorf("kanon: the numbers line for %s repeats", name)
		}
		rec := &record{nums: make(map[string]int)}
		for _, w := range words[2:] {
			if err := rec.add(name, w); err != nil {
				return nil, err
			}
		}
		recs[name] = rec
	}
	return recs, nil
}

// baseRecords returns the records of the code file named code in base, and
// the records that the other code files of base keep, by key, in the order
// of their names, as [pkg.codeRecords] returns them for the package
// directory. It fails for a code file that does not parse.
func baseRecords(code string, base map[string][]byte) (records, map[string][]recordAt, error) {
	own := records{}
	others := make(map[string][]recordAt)
	for _, name := range slices.Sorted(maps.Keys(base)) {
		recs, err := parseRecords(base[name])
		if err != nil {
			return nil, nil, err
		}
		if name == code {
			own = recs
			continue
		}
		for key, rec := range recs {
			others[key] = append(others[key], recordAt{file: name, rec: rec})
		}
	}
	return own, others, nil
}

// recordAt is a record of a code file other than the one that kanon
// generates.
type recordAt struct {
	// file is the base name of the code file.
	file string
	rec  *record
}

// lineWords returns the words of a numbers line, split at runs of spaces. A
// word that begins with a quote runs to the end of the Go string literal it
// begins with, spaces included. lineWords fails for a quote that does not
// end.
func lineWords(line string) ([]string, error) {
	var out []string
	rest := strings.TrimLeftFunc(line, unicode.IsSpace)
	for rest != "" {
		start := 0
		if strings.HasPrefix(rest, quoteMark) {
			q, err := strconv.QuotedPrefix(rest)
			if err != nil {
				return nil, err
			}
			start = len(q)
		}
		end := len(rest)
		if k := strings.IndexFunc(rest[start:], unicode.IsSpace); k >= 0 {
			end = start + k
		}
		out = append(out, rest[:end])
		rest = strings.TrimLeftFunc(rest[end:], unicode.IsSpace)
	}
	return out, nil
}

// cutQuoted cuts the word `"Name"=N` of a numbers line after its quoted
// name, and returns the name, the number and whether the word has that
// form. word begins with a Go string literal, as [lineWords] splits it.
func cutQuoted(word string) (string, string, bool) {
	// lineWords split the line at the end of the literal that word begins
	// with, so the prefix is a complete literal, which unquotes.
	q, _ := strconv.QuotedPrefix(word)
	n, ok := strings.CutPrefix(word[len(q):], numberSeparator)
	name, _ := strconv.Unquote(q)
	return name, n, ok
}

// numbersWord returns s as a key or a name of a numbers line: s itself, or
// its Go string literal when s contains a space.
func numbersWord(s string) string {
	if strings.ContainsFunc(s, unicode.IsSpace) {
		return strconv.Quote(s)
	}
	return s
}

// parseNumber parses a number of the numbers line for owner. It accepts 1
// to 2147483647.
func parseNumber(owner, s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > maxFieldNumber {
		return 0, fmt.Errorf("kanon: the numbers line for %s has the malformed field number %q", owner, s)
	}
	return n, nil
}

// carried returns the numbers lines that the code file keeps for the
// structs that it no longer generates: the keys of own that names does not
// list, that the package still declares, and that no other code file
// records, in key order. A struct that moves to another source file keeps
// its numbers whichever of the two files generates first, and its line
// leaves the old code file when the new one records the struct.
func carried(own records, others map[string][]recordAt, names []string, declared func(string) bool) []string {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(own)) {
		if !slices.Contains(names, name) && declared(name) && len(others[name]) == 0 {
			out = append(out, own[name].line(name))
		}
	}
	return out
}

// assign numbers the fields of m. A field takes its tag number first, then
// the number that rec records for it, then the smallest number that is
// neither taken nor reserved, in declaration order. assign sets m.reserved
// to the reservations of rec and the numbers of the recorded fields that m
// no longer declares, minus the numbers that tags take. A tag may take a
// reserved number: a tag renames a field or reuses a number on purpose.
//
// assign fails when a tag gives a recorded field another number and when
// two fields take one number.
func assign(m *target, rec *record) error {
	present := make(map[string]bool, len(m.fields))
	for _, f := range m.fields {
		present[f.name] = true
	}
	reserved := make(map[int]bool, len(rec.reserved))
	for _, n := range rec.reserved {
		reserved[n] = true
	}
	for name, n := range rec.nums {
		if !present[name] {
			reserved[n] = true
		}
	}
	taken := make(map[int]string, len(m.fields))
	take := func(f *field, n int) error {
		if other, ok := taken[n]; ok {
			return m.fail(f.obj, fmt.Errorf("kanon: field number %d is also field %s", n, other))
		}
		taken[n] = f.name
		f.num = n
		return nil
	}
	for _, f := range m.fields {
		n := f.tag.num
		if n == 0 {
			continue
		}
		if was, ok := rec.nums[f.name]; ok && was != n {
			return m.fail(f.obj, fmt.Errorf(
				"kanon: the tag numbers the field %d but it is recorded as %d: renumbering breaks encoded data",
				n,
				was,
			))
		}
		if err := take(f, n); err != nil {
			return err
		}
		delete(reserved, n)
	}
	for _, f := range m.fields {
		if n, ok := rec.nums[f.name]; ok && f.num == 0 {
			if err := take(f, n); err != nil {
				return err
			}
		}
	}
	next := 1
	for _, f := range m.fields {
		if f.num != 0 {
			continue
		}
		next = nextFree(next, taken, reserved)
		taken[next] = f.name
		f.num = next
	}
	m.reserved = slices.Sorted(maps.Keys(reserved))
	return nil
}

// nextFree returns the smallest number from next on that taken does not
// give and reserved does not mark: the number of the next field or concrete
// type without a recorded one. Of the numbers from next on, the first
// len(taken)+len(reserved)+1 contain a free one.
func nextFree(next int, taken map[int]string, reserved map[int]bool) int {
	for range len(taken) + len(reserved) {
		if taken[next] == "" && !reserved[next] {
			return next
		}
		next++
	}
	return next
}

// genericFields sets the fields of g, the target of a generic struct type,
// to the fields that an instantiation of it can encode, which
// [classifier.analyze] keeps: the fields that kanon reads, as [reads]
// reports, without a `kanon:"-"` tag, other than functions, channels, the
// field tagged unknown and the discriminators of unions. A field whose type
// is a type parameter counts, since an instantiation can encode it.
func genericFields(g *target) {
	st, _ := g.typ.Underlying().(*types.Struct)
	discs := make(map[string]bool)
	var fields []*field
	for i := range st.NumFields() {
		obj := st.Field(i)
		// The analysis of an instantiation parsed every tag of the struct.
		t, _ := parseTag(st.Tag(i))
		discs[t.union] = true
		if reads(obj, t) && !unencoded(obj.Type()) && !t.skip && !t.unknown {
			fields = append(fields, &field{obj: obj, name: obj.Name(), tag: t})
		}
	}
	for _, f := range fields {
		if !discs[f.name] {
			g.fields = append(g.fields, f)
		}
	}
}

// shareNumbers gives each field of the instantiation m the number of the
// field of its name in g, the target of its generic type, and gives m the
// reserved numbers of g.
func shareNumbers(m, g *target) {
	nums := make(map[string]int, len(g.fields))
	for _, f := range g.fields {
		nums[f.name] = f.num
	}
	for _, f := range m.fields {
		f.num = nums[f.name]
	}
	m.reserved = g.reserved
}

// renderNumbers returns the numbers line under the key name: the names in
// that order with their numbers in nums, then the reserved numbers.
func renderNumbers(name string, names []string, nums map[string]int, reserved []int) string {
	var sb strings.Builder
	sb.WriteString(numbersMark)
	sb.WriteByte(' ')
	sb.WriteString(numbersWord(name))
	for _, f := range names {
		sb.WriteByte(' ')
		sb.WriteString(numbersWord(f))
		sb.WriteString(numberSeparator)
		sb.WriteString(strconv.Itoa(nums[f]))
	}
	for _, n := range reserved {
		sb.WriteByte(' ')
		sb.WriteString(reservedMark)
		sb.WriteString(strconv.Itoa(n))
	}
	return sb.String()
}

// recordKeys returns the keys under which a code file can record the
// numbers of m: the keys of the sites of an anonymous struct type, in the
// order in which the analysis visits them, and the key of m for a named
// one. An anonymous struct keeps its numbers when the site that the
// analysis visits first changes.
func (m *target) recordKeys() []string {
	if m.decl != nil {
		return []string{m.key}
	}
	keys := make([]string, 0, len(m.sites))
	for _, s := range m.sites {
		keys = append(keys, s.key)
	}
	return keys
}

// numbersLine returns the numbers line of m: its key, the field numbers in
// declaration order, and the reserved numbers.
func (m *target) numbersLine() string {
	names := make([]string, 0, len(m.fields))
	nums := make(map[string]int, len(m.fields))
	for _, f := range m.fields {
		names = append(names, f.name)
		nums[f.name] = f.num
	}
	return renderNumbers(m.key, names, nums, m.reserved)
}

// number numbers the fields of the inline structs of in, in the order of
// the analysis, and returns the targets whose numbers lines record them, one
// per key. A struct takes the record that [recordFor] picks for the keys
// that [target.recordKeys] returns. The instantiations of a generic struct
// type share the numbers of the target of the generic type, which
// [genericFields] fills and whose numbers line records them.
func (in *inlines) number(own records, others map[string][]recordAt) ([]*target, error) {
	var out []*target
	for _, m := range in.list {
		n := m
		if m.generic != nil {
			n = m.generic
		}
		if !slices.Contains(out, n) {
			if n != m {
				genericFields(n)
			}
			rec, err := recordFor(n.recordKeys(), own, others)
			if err != nil {
				return nil, err
			}
			if err := assign(n, rec); err != nil {
				return nil, err
			}
			out = append(out, n)
		}
		if n != m {
			shareNumbers(m, n)
		}
	}
	return out, nil
}

// codeRecords returns the records of the code file named code, and the
// records that the other code files of p keep, by key. It fails for a code
// file that does not read or parse.
func (p *pkg) codeRecords(code string) (records, map[string][]recordAt, error) {
	own, err := readRecords(filepath.Join(p.dir, code))
	if err != nil {
		return nil, nil, err
	}
	others := make(map[string][]recordAt)
	for _, file := range p.codes {
		if file == code {
			continue
		}
		recs, err := readRecords(filepath.Join(p.dir, file))
		if err != nil {
			return nil, nil, err
		}
		for name, rec := range recs {
			others[name] = append(others[name], recordAt{file: file, rec: rec})
		}
	}
	return own, others, nil
}

// depRecords returns the records of the code files of the dependency of p
// with the import path importPath, by key. It fails for a code file that
// does not read or parse.
func (p *pkg) depRecords(importPath string) (records, error) {
	dep := p.deps[importPath]
	out := records{}
	for _, file := range dep.GoFiles {
		if !strings.HasSuffix(file, CodeSuffix) {
			continue
		}
		recs, err := readRecords(filepath.Join(dep.Dir, file))
		if err != nil {
			return nil, err
		}
		maps.Copy(out, recs)
	}
	return out, nil
}

// declares reports whether p declares a type named name at package level.
func (p *pkg) declares(name string) bool {
	_, ok := p.types.Scope().Lookup(name).(*types.TypeName)
	return ok
}
