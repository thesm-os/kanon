// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"cmp"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Sizes of the samples.
const (
	// drawCount is the length of every value table, and the number of
	// table samples.
	drawCount = 8
	// fieldEntry is the table entry of a field sample. Entry 1 of every
	// table is present.
	fieldEntry = 1
	// nesting is the number of levels of slices, maps, pointers,
	// interfaces and structs that a sample nests at most, so that a type
	// that contains itself ends.
	nesting = 3
	// sliceLengths is the cycle of the lengths of the slices and maps of
	// the table samples.
	sliceLengths = 3
	// pointerCycle is the cycle of the pointers of the table samples: one
	// in every pointerCycle entries is nil.
	pointerCycle = 4
	// wideEntries is the number of entries of every map of a wide sample:
	// more than twice the 16 values that the free list of a map decode
	// keeps.
	wideEntries = 33
	// wideSamples is the fewest wide samples. A type with a wider pick has a
	// wide sample per choice of it, so that a decode reuses a wide map in
	// every choice.
	wideSamples = 2
	// wideNesting is the number of levels that a wide sample nests at
	// most. A map of maps of a wide sample has wideEntries squared entries
	// at most.
	wideNesting = 2
	// intBits is the width of the values of an int, a uint and a uintptr
	// in a sample, their width on the narrowest platform, so that the
	// samples and their encodings are the same on every platform.
	intBits = 32
)

// Zones of the time table: three and a half hours west of UTC, a zone that
// time.FixedZone allocates on every call and a decode shares, so that two
// builds of one key differ under ==, and one hour east of UTC, a zone that
// both share.
const (
	westOffset = 12600
	eastOffset = 3600
)

// widePrefix begins every string of a wide sample.
const widePrefix = "w"

// keyLoc is the location of the errors of the encodings of map keys that
// the builder compares, which it discards.
const keyLoc = "map.key"

// source supplies the values of a sample. The builder decides the
// structure of the sample, and the source decides every value in it.
type source interface {
	// part returns the source of part k of a composite value: element k of
	// a sequence, entry k of a map, field k of a struct, or the value of a
	// pointer or an interface.
	part(k int) source
	boolean() bool
	signed(bits int) int64
	unsigned(bits int) uint64
	float(bits int) float64
	text() string
	// bytes returns a byte slice, and nil for a nil one.
	bytes() []byte
	instant() time.Time
	// count returns the number of elements of a slice, and -1 for a nil
	// one. A byte array is zero for -1.
	count() int
	// entries returns the number of entries of a map, and -1 for a nil one.
	entries() int
	// set reports whether a pointer is not nil.
	set() bool
	// pick returns one of n choices, from 0 to n-1.
	pick(n int) int
	// keyed reports that every map takes the keys of the key sample, as
	// [resolver.buildMap] builds them.
	keyed() bool
	// fail reports whether the value that can fail to encode, which the
	// builder is about to build, fails. The builder calls it once per way to
	// fail such a value: a value of a type that encodes itself once per way
	// that [resolver.failures] finds, a value of a kanon.Validator and an
	// interface once each, and a map once per way to fail its keys that
	// [resolver.badKeys] finds.
	fail() bool
	// failEntry reports whether the map of the shape s, which the builder has
	// filled with its entries, takes an entry that fails to encode: at its
	// key with key set, and at its value otherwise. The builder calls it once
	// per side of the entries of a map that is not nil.
	failEntry(s *shape, key bool) bool
}

// table is the source of table sample i: every value takes entry i of its
// table, and part k of a value takes entry i+k.
type table int

// part returns table i+k.
func (i table) part(k int) source { return i + table(k) }

// boolean returns true for an odd entry.
func (i table) boolean() bool { return i%2 == 1 }

// signed returns entry i of the table of signed integers of bits bits.
func (i table) signed(bits int) int64 { return intTable(bits)[i%drawCount] }

// unsigned returns entry i of the table of unsigned integers of bits bits.
func (i table) unsigned(bits int) uint64 { return uintTable(bits)[i%drawCount] }

// float returns entry i of the table of floats of bits bits.
func (i table) float(bits int) float64 { return floatTable(bits)[i%drawCount] }

// text returns entry i of the table of strings.
func (i table) text() string { return stringTable()[i%drawCount] }

// bytes returns entry i of the table of byte slices.
func (i table) bytes() []byte { return bytesTable()[i%drawCount] }

// instant returns entry i of the table of times.
func (i table) instant() time.Time { return timeTable()[i%drawCount] }

// count returns i modulo sliceLengths, and -1 for a nil value when that is
// 0 and i is even.
func (i table) count() int {
	n := int(i % sliceLengths)
	if n == 0 && i%2 == 0 {
		return -1
	}
	return n
}

// entries returns the count of i, as for a slice.
func (i table) entries() int { return i.count() }

// set reports whether i is not a multiple of pointerCycle, so that the
// pointers of a chain of three point at values.
func (i table) set() bool { return i%pointerCycle != 0 }

// pick returns one more than i divided by sliceLengths, modulo n. The pick
// changes once per sliceLengths entries, so that every choice meets every
// length of a slice, and entry 1 picks the first concrete type of an
// interface.
func (i table) pick(n int) int { return (int(i)/sliceLengths + 1) % n }

// keyed returns false.
func (table) keyed() bool { return false }

// fail returns false.
func (table) fail() bool { return false }

// failEntry returns false.
func (table) failEntry(*shape, bool) bool { return false }

// counting is the source of a wide sample: every map has wideEntries
// entries, every slice one element, every pointer a value and every bool
// is true, and every other value derives from n. Part k of counting n is
// counting n+k, so the keys of a map, which take the parts 0 to
// wideEntries-1, differ unless they are bools.
type counting int

// part returns counting n+k.
func (n counting) part(k int) source { return n + counting(k) }

// boolean returns true.
func (counting) boolean() bool { return true }

// signed returns n, which a narrower integer type truncates.
func (n counting) signed(int) int64 { return int64(n) }

// unsigned returns n, which a narrower integer type truncates.
func (n counting) unsigned(int) uint64 { return uint64(n) }

// float returns n, which a float32 represents exactly.
func (n counting) float(int) float64 { return float64(n) }

// text returns the name of n.
func (n counting) text() string { return widePrefix + strconv.Itoa(int(n)) }

// bytes returns the bytes of the name of n.
func (n counting) bytes() []byte { return []byte(n.text()) }

// instant returns the time n seconds after the Unix epoch, in the zone
// east of UTC.
func (n counting) instant() time.Time {
	return time.Unix(int64(n), 0).In(time.FixedZone("", eastOffset))
}

// count returns 1.
func (counting) count() int { return 1 }

// entries returns wideEntries.
func (counting) entries() int { return wideEntries }

// set returns true.
func (counting) set() bool { return true }

// pick returns n modulo m.
func (n counting) pick(m int) int { return int(n) % m }

// keyed returns false.
func (counting) keyed() bool { return false }

// fail returns false.
func (counting) fail() bool { return false }

// failEntry returns false.
func (counting) failEntry(*shape, bool) bool { return false }

// keyed is the source of the key sample: the values of src, with the keys
// of the key sample in every map, as [resolver.buildMap] builds them.
type keyed struct {
	source
}

// part returns part k of src, keyed.
func (k keyed) part(n int) source { return keyed{k.source.part(n)} }

// keyed returns true.
func (keyed) keyed() bool { return true }

// keyTable is the source of the keys of the key sample: the values of src,
// a table entry, with the float of the next entry in place of NaN. A map of
// a sample does not keep a key with a NaN component, so the key sample
// keeps every key that [resolver.keyAlts] derives from its base.
type keyTable struct {
	source
}

// part returns part k of src, as a keyTable.
func (k keyTable) part(n int) source { return keyTable{k.source.part(n)} }

// float returns the float of src, or the float of its next part for NaN.
func (k keyTable) float(bits int) float64 {
	if f := k.source.float(bits); !math.IsNaN(f) {
		return f
	}
	return k.source.part(1).float(bits)
}

// failing is the source of a sample that fails to encode: the values of
// src, and a failure in the value that can fail whose number, counted from
// 0 in the order of the builder, left starts at.
type failing struct {
	source
	// left is the number of values that can fail before the one that
	// fails. The parts of a source share it.
	left *int
}

// part returns part k of src, with the count of the values left.
func (f failing) part(k int) source { return failing{f.source.part(k), f.left} }

// fail counts one value that can fail, and reports whether it is the one
// that fails.
func (f failing) fail() bool {
	*f.left--
	return *f.left == -1
}

// mapSide is the keys, with key set, or the values of the maps of the shape
// s.
type mapSide struct {
	s   *shape
	key bool
}

// shape returns the shape of the keys or the values of side.
func (side mapSide) shape() *shape {
	if side.key {
		return side.s.key
	}
	return side.s.elem
}

// entryFailing is the source of a wide sample that fails at one entry of a
// map: the values of src, and in the first map of the shape of side that
// the builder fills, an entry that fails to encode at the side of side, as
// [resolver.addFailingEntry] adds it.
type entryFailing struct {
	source
	side mapSide
	// done records that a map took the entry that fails. The parts of a
	// source share it.
	done *bool
}

// part returns part k of src, which fails at the same side.
func (f entryFailing) part(k int) source { return entryFailing{f.source.part(k), f.side, f.done} }

// failEntry reports whether s and key are the shape and the side of f, for
// the first map that meets them.
func (f entryFailing) failEntry(s *shape, key bool) bool {
	if *f.done || s != f.side.s || key != f.side.key {
		return false
	}
	*f.done = true
	return true
}

// intTable returns the table of signed integers of bits bits: zero, a
// present value, the varint length boundaries, the sign extension of
// negative values and the extremes of the type.
func intTable(bits int) [drawCount]int64 {
	switch bits {
	case 8:
		return [drawCount]int64{0, 1, -1, 64, -65, math.MaxInt8, math.MinInt8, -64}
	case 16:
		return [drawCount]int64{0, 1, -1, 128, -129, 16383, math.MaxInt16, math.MinInt16}
	case 32:
		return [drawCount]int64{0, 1, -1, 128, 16384, -129, math.MaxInt32, math.MinInt32}
	default:
		return [drawCount]int64{0, 1, -1, 128, 1 << 35, -(1 << 35), math.MaxInt64, math.MinInt64}
	}
}

// uintTable is intTable for unsigned integers.
func uintTable(bits int) [drawCount]uint64 {
	switch bits {
	case 8:
		return [drawCount]uint64{0, 1, 127, 128, math.MaxUint8, 2, 64, 200}
	case 16:
		return [drawCount]uint64{0, 1, 127, 128, 16383, 16384, math.MaxUint16, 300}
	case 32:
		return [drawCount]uint64{0, 1, 127, 128, 16384, 1 << 21, 1 << 28, math.MaxUint32}
	default:
		return [drawCount]uint64{0, 1, 127, 128, 1 << 35, 1 << 56, 1 << 63, math.MaxUint64}
	}
}

// floatTable returns the table of floats of bits bits: zero, a present
// value, negative zero, NaN, the infinities, the largest value and the
// smallest positive one.
func floatTable(bits int) [drawCount]float64 {
	largest, smallest := math.MaxFloat64, math.SmallestNonzeroFloat64
	if bits == 32 {
		largest, smallest = math.MaxFloat32, math.SmallestNonzeroFloat32
	}
	return [drawCount]float64{0, 1.5, math.Copysign(0, -1), math.NaN(), math.Inf(1), math.Inf(-1), largest, smallest}
}

// stringTable returns the table of strings: lengths on both sides of a
// one-byte length, multi-byte runes, a NUL byte and invalid UTF-8.
func stringTable() [drawCount]string {
	return [drawCount]string{
		"", "kanon", "é✓ω", strings.Repeat("a", 127), strings.Repeat("b", 128), "\x00", strings.Repeat("c", 300),
		"\xff\xfe",
	}
}

// bytesTable returns the table of byte slices. Entry 5 is empty and not
// nil.
func bytesTable() [drawCount][]byte {
	s := stringTable()
	return [drawCount][]byte{nil, {0xde, 0xad, 0xbe, 0xef}, {0}, []byte(s[3]), []byte(s[4]), {}, []byte(s[6]), {0xff}}
}

// timeTable returns the table of times: the zero time, a time with
// nanoseconds, the Unix epoch, a time before the epoch in the zone west of
// UTC, the last nanosecond of year 9999, a time in the zone east of UTC,
// the zero instant in the zone east of UTC, which is present, and year
// 20000.
func timeTable() [drawCount]time.Time {
	return [drawCount]time.Time{
		{},
		time.Unix(1713400000, 500000000).UTC(),
		time.Unix(0, 0).UTC(),
		time.Unix(-1, 999999999).In(time.FixedZone("", -westOffset)),
		time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC),
		time.Unix(1, 1).In(time.FixedZone("", eastOffset)),
		time.Time{}.In(time.FixedZone("", eastOffset)),
		time.Date(20000, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// build returns a value of the shape s from src, nesting depth levels
// below it at most: a slice, a map, a pointer, an interface and a struct
// at depth 0 take their zero value. A map leaves out a key that orders
// equal to one it has, which has no order in the encoding. A value of a
// kanon.Validator counts as one value that can fail: it takes a value that
// its ValidateKanon rejects when src fails it, and a value that it accepts
// in place of a rejected one otherwise, so that a sample fails to encode
// only where a failing source fails it.
func (r *resolver) build(src source, s *shape, depth int) reflect.Value {
	v := r.shaped(src, s, depth)
	if !s.validate {
		return v
	}
	if x, ok := r.rejected(s); ok && src.fail() {
		v.Set(x)
	} else if validate(v) != nil {
		v.Set(r.accepted(s))
	}
	return v
}

// shaped returns a value of the shape s from src, as [resolver.build]
// builds it, before the ValidateKanon of a kanon.Validator decides it.
func (r *resolver) shaped(src source, s *shape, depth int) reflect.Value {
	v := reflect.New(s.typ).Elem()
	if depth == 0 && s.level() {
		return v
	}
	switch s.kind {
	case kindTime:
		v.Set(reflect.ValueOf(src.instant()))
	case kindStruct:
		r.fill(src, r.looseLayout(s.typ), v, depth-1)
	case kindInline:
		r.fill(src, s.layout, v, depth-1)
	case kindSlice:
		if n := src.count(); n >= 0 {
			v.Set(reflect.MakeSlice(s.typ, n, n))
			for j := range n {
				v.Index(j).Set(r.build(src.part(j), s.elem, depth-1))
			}
		}
	case kindArray:
		for j := range v.Len() {
			v.Index(j).Set(r.build(src.part(j), s.elem, depth))
		}
	case kindMap:
		r.buildMap(src, s, v, depth-1)
	case kindPointer:
		if src.set() {
			v.Set(newTarget(s.elem.typ))
			v.Elem().Set(r.build(src.part(1), s.elem, depth-1))
		}
	case kindInterface:
		r.buildInterface(src, s, v, depth)
	case kindBinary:
		r.self(src, v, depth)
		if x, ok := failure(src, r.failures(s.typ)); ok {
			v.Set(x)
		} else if _, err := encodeSelf(v); err != nil {
			v.Set(r.success(s.typ))
		}
	default:
		r.scalar(src, v, depth)
	}
	return v
}

// buildInterface sets v, an interface of the shape s, to a value that fails
// to encode when src fails it, and otherwise to nil or to a value of the
// concrete type that src picks.
func (r *resolver) buildInterface(src source, s *shape, v reflect.Value, depth int) {
	if x, ok := r.unlisted(s); ok && src.fail() {
		v.Set(x)
		return
	}
	if k := src.pick(len(s.variants) + 1); k > 0 {
		v.Set(r.build(src.part(1), s.variants[k-1].shape, depth-1))
	}
}

// buildMap sets the map v of the shape s to the entries that src counts.
// Entry j takes its key from part j and its value from part j+1, so that
// keys and values of one type differ. [resolver.taintField] sets each field
// of a key that the encoding leaves out to a value other than the zero
// value that a decode yields. A keyed source takes the keys of the key
// sample instead: the key that [keyTable] builds from entry 1 and the keys
// that [resolver.keyAlts] derives from it.
//
// After its entries, a map takes the entry that fails at a side of them
// when src fails that side, as [resolver.addFailingEntry] adds it. The map
// then counts as one value that can fail for each way to fail its encode
// that [resolver.badKeys] finds: a key with a NaN component, and two keys
// of one projection. A source that fails such a value adds its key, or its
// two keys, to the map with the zero value, on which no check depends.
func (r *resolver) buildMap(src source, s *shape, v reflect.Value, depth int) {
	if src.keyed() {
		base := r.build(keyTable{table(fieldEntry)}, s.key, nesting)
		keys := append([]reflect.Value{base}, r.keyAlts(s.key, base, nesting)...)
		v.Set(reflect.MakeMapWithSize(s.typ, len(keys)))
		for j, k := range keys {
			r.insert(s, v, k, r.build(src.part(j+1), s.elem, depth))
		}
		return
	}
	n := src.entries()
	if n < 0 {
		return
	}
	v.Set(reflect.MakeMapWithSize(s.typ, n))
	for j := range n {
		k := r.build(src.part(j), s.key, depth)
		r.leftOutIn(s.key, k, r.taintField)
		r.insert(s, v, k, r.build(src.part(j+1), s.elem, depth))
	}
	r.addFailingEntry(src, s, v, true, depth)
	r.addFailingEntry(src, s, v, false, depth)
	for _, b := range r.badKeys(s.key) {
		if src.fail() {
			for _, k := range b.keys() {
				v.SetMapIndex(k, reflect.Zero(s.elem.typ))
			}
		}
	}
}

// insert sets the key k of the map m of the shape s to x, unless k has a
// NaN component or the projection of a key of m, which [resolver.compareKeys]
// orders equal to it, so that the map encodes.
func (r *resolver) insert(s *shape, m, k, x reflect.Value) {
	if r.hasNaN(s.key, k) {
		return
	}
	for it := m.MapRange(); it.Next(); {
		if r.compareKeys(s.key, k, it.Key()) == 0 {
			return
		}
	}
	m.SetMapIndex(k, x)
}

// addFailingEntry sets an entry of the map v of the shape s, whose entries
// nest depth levels deep, that fails to encode when src fails the side of
// the entries of s that key selects, as [source.failEntry] reports it: with
// key set, a key that fails, with the value of entry 0, and otherwise the
// key of entry 0 with a value that fails. A key or a value that fails is
// the one that table entry fieldEntry builds, nesting levels deep, with its
// first value that can fail failing, as [resolver.fails] builds it, since a
// value at depth 0 is the zero value, which cannot fail.
func (r *resolver) addFailingEntry(src source, s *shape, v reflect.Value, key bool, depth int) {
	if !src.failEntry(s, key) {
		return
	}
	left := 0
	bad := failing{table(fieldEntry), &left}
	k, x := r.build(src.part(0), s.key, depth), r.build(src.part(1), s.elem, depth)
	if key {
		k = r.build(bad, s.key, nesting)
	} else {
		x = r.build(bad, s.elem, nesting)
	}
	r.leftOutIn(s.key, k, r.taintField)
	v.SetMapIndex(k, x)
}

// fails reports whether a value of s can fail to encode: the value that
// table entry fieldEntry builds has a value that a failing source fails.
func (r *resolver) fails(s *shape) bool {
	unfailed := math.MaxInt
	r.build(failing{table(fieldEntry), &unfailed}, s, nesting)
	return unfailed < math.MaxInt
}

// failingSides returns the sides of the entries of the maps that a value of
// l contains outside the structs with a kanon codec, whose own checks cover
// their maps, that can fail to encode, as [resolver.fails] reports them: per
// map shape in the order of a walk of the fields of l, its keys, then its
// values.
func (r *resolver) failingSides(l *layout) []mapSide {
	var out []mapSide
	seen := make(map[*shape]bool)
	var walk func(s *shape)
	walk = func(s *shape) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		switch s.kind {
		case kindMap:
			for _, side := range []mapSide{{s, true}, {s, false}} {
				if r.fails(side.shape()) {
					out = append(out, side)
				}
			}
		case kindInline:
			for _, f := range s.layout.fields {
				walk(f.shape)
			}
		case kindInterface:
			for _, w := range s.variants {
				walk(w.shape)
			}
		default:
			// A slice, an array and a pointer contain maps in their elements,
			// which the walk enters next. A struct with a kanon codec has none.
		}
		walk(s.key)
		walk(s.elem)
	}
	for _, f := range l.fields {
		walk(f.shape)
	}
	return out
}

// keySearch is the number of table entries among which [resolver.badKeys]
// looks for a key with a NaN component and for two keys of one projection:
// drawCount squared, every entry of every table for each of eight parts.
const keySearch = 64

// badKey is a way to make the keys of a map of one key type fail its encode,
// which [resolver.badKeys] finds: the key that table entry i builds, and
// with pair set, a second key of its projection that differs under ==, which
// entry j builds, and which [resolver.taint] changes when tainted is set.
type badKey struct {
	r *resolver
	s *shape
	i int
	j int
	// pair reports two keys of one projection, and a key with a NaN
	// component otherwise.
	pair    bool
	tainted bool
}

// keys returns the keys of b, each built anew, so that no two maps share
// the memory of a key.
func (b badKey) keys() []reflect.Value {
	a := b.r.build(table(b.i), b.s, nesting)
	if !b.pair {
		return []reflect.Value{a}
	}
	c := b.r.build(table(b.j), b.s, nesting)
	if b.tainted {
		b.r.leftOutIn(b.s, c, b.r.taintField)
	}
	return []reflect.Value{a, c}
}

// badKeys returns the ways to make the keys of a map with keys of the shape
// s fail its encode, among the first keySearch table entries: a key with a
// NaN component, and two keys of one projection that differ under ==, which
// two builds of one entry make with pointers or with times in a zone that
// time.FixedZone allocates, a key whose fields that the encoding leaves out
// [resolver.taint] sets, or two entries of a type that encodes itself to
// the same bytes. The result for a shape is cached.
func (r *resolver) badKeys(s *shape) []badKey {
	if bad, ok := r.bad[s]; ok {
		return bad
	}
	var bad []badKey
	keys := make([]reflect.Value, keySearch)
	for i := range keySearch {
		keys[i] = r.build(table(i), s, nesting)
	}
	nan := slices.IndexFunc(keys, func(k reflect.Value) bool { return r.hasNaN(s, k) })
	if nan >= 0 {
		bad = append(bad, badKey{r: r, s: s, i: nan})
	}
	if pair, ok := r.ambiguousPair(s, keys); ok {
		bad = append(bad, pair)
	}
	r.bad[s] = bad
	return bad
}

// ambiguousPair returns two keys of the shape s with one projection that
// differ under ==, as a badKey, and reports whether it finds them among
// keys, the keys that the first table entries build, and their copies.
func (r *resolver) ambiguousPair(s *shape, keys []reflect.Value) (badKey, bool) {
	for i, k := range keys {
		if r.hasNaN(s, k) {
			continue
		}
		again := r.build(table(i), s, nesting)
		if k.Interface() != again.Interface() {
			return badKey{r: r, s: s, i: i, j: i, pair: true}, true
		}
		r.leftOutIn(s, again, r.taintField)
		if k.Interface() != again.Interface() {
			return badKey{r: r, s: s, i: i, j: i, pair: true, tainted: true}, true
		}
		for j := range i {
			if !r.hasNaN(s, keys[j]) && r.compareKeys(s, keys[j], k) == 0 && keys[j].Interface() != k.Interface() {
				return badKey{r: r, s: s, i: j, j: i, pair: true}, true
			}
		}
	}
	return badKey{}, false
}

// keyAlts returns keys of the shape s that differ from the key base in one
// part each, so that the order of a map with these keys compares each part
// of two keys whose earlier parts are equal. The parts of a struct, a
// pointer and an interface are depth levels deep at most. By kind of key:
//
//   - a struct varies each field, and an array each element;
//   - a pointer takes nil, the pointer that [keyTable] builds from entry 1,
//     and pointers to the keys that vary its target;
//   - an interface takes nil, and per concrete type the value that
//     [keyTable] builds from entry 1 and the keys that vary it;
//   - a time varies its instant, then UTC against a zone, then the offset
//     of the zone;
//   - a complex number varies each part;
//   - any other key takes the first value of its tables that differs.
func (r *resolver) keyAlts(s *shape, base reflect.Value, depth int) []reflect.Value {
	var out []reflect.Value
	if depth == 0 && s.level() {
		return out
	}
	with := func(set func(reflect.Value)) {
		k := reflect.New(s.typ).Elem()
		k.Set(base)
		set(k)
		out = append(out, k)
	}
	switch s.kind {
	case kindStruct, kindInline:
		for _, f := range r.keyLayout(s.typ).fields {
			for _, alt := range r.keyAlts(f.shape, f.of(base), depth-1) {
				with(func(k reflect.Value) { f.of(k).Set(alt) })
			}
		}
	case kindArray:
		for j := range base.Len() {
			for _, alt := range r.keyAlts(s.elem, base.Index(j), depth) {
				with(func(k reflect.Value) { k.Index(j).Set(alt) })
			}
		}
	case kindPointer:
		x := r.build(keyTable{table(fieldEntry)}, s, depth)
		out = append(out, reflect.Zero(s.typ), x)
		for _, alt := range r.keyAlts(s.elem, x.Elem(), depth-1) {
			p := reflect.New(s.elem.typ)
			p.Elem().Set(alt)
			out = append(out, p)
		}
	case kindInterface:
		out = append(out, reflect.Zero(s.typ))
		for _, w := range s.variants {
			x := r.build(keyTable{table(fieldEntry)}, w.shape, depth-1)
			for _, alt := range append([]reflect.Value{x}, r.keyAlts(w.shape, x, depth-1)...) {
				with(func(k reflect.Value) { k.Set(alt) })
			}
		}
	case kindTime:
		t, _ := reflect.TypeAssert[time.Time](base)
		for _, alt := range []time.Time{
			t.Add(time.Second), t.UTC(), t.In(time.FixedZone("", eastOffset)),
			t.In(time.FixedZone("", -westOffset)),
		} {
			out = append(out, reflect.ValueOf(alt))
		}
	case kindComplex64, kindComplex128:
		c := base.Complex()
		out = append(out, reflect.ValueOf(complex(other(real(c)), imag(c))).Convert(s.typ),
			reflect.ValueOf(complex(real(c), other(imag(c)))).Convert(s.typ))
	default:
		for i := range drawCount {
			if x := r.build(table(i), s, depth); r.compareKeys(s, x, base) != 0 {
				return []reflect.Value{x}
			}
		}
	}
	return out
}

// other returns a float that orders apart from f: entry 0 of the table of
// float64s, or entry 1 when entry 0 orders equal to f.
func other(f float64) float64 {
	table := floatTable(64)
	if cmp.Compare(table[0], f) == 0 {
		return table[1]
	}
	return table[0]
}

// failures returns values of t, a type that encodes itself, that fail to
// encode: per way to fail, as [failModeOf] tells them apart, the first value
// that [resolver.self] builds from the value tables that fails that way, in
// the order of the ways. A value other than the zero value takes the place of
// the zero value, when the tables give one that fails the same way: a field
// leaves out the zero value of a type whose == compares every bit, so that it
// fails in no field.
func (r *resolver) failures(t reflect.Type) []reflect.Value {
	var first [failsLength + 1]reflect.Value
	for i := range drawCount {
		v := reflect.New(t).Elem()
		r.self(table(i), v, 0)
		m := failModeOf(v)
		if m != 0 && (!first[m].IsValid() || first[m].IsZero() && !v.IsZero()) {
			first[m] = v
		}
	}
	return slices.DeleteFunc(first[:], func(v reflect.Value) bool { return !v.IsValid() })
}

// failure calls fail of src once per value of fails, and returns the value
// whose call reports that it fails, and false when no call does.
func failure(src source, fails []reflect.Value) (reflect.Value, bool) {
	var out reflect.Value
	for _, x := range fails {
		if src.fail() {
			out = x
		}
	}
	return out, out.IsValid()
}

// success returns a value of t, a type that encodes itself, that encodes,
// as [encodeSelf] reports it: the first value that [resolver.self] builds
// from the value tables that does, or the zero value when none does, so that
// a sample fails to encode only where a failing source fails it.
func (r *resolver) success(t reflect.Type) reflect.Value {
	ok := reflect.Zero(t)
	found := false
	for i := range drawCount {
		v := reflect.New(t).Elem()
		r.self(table(i), v, 0)
		if _, err := encodeSelf(v); !found && err == nil {
			ok, found = v, true
		}
	}
	return ok
}

// rejected returns a value of s, the shape of a kanon.Validator, that its
// ValidateKanon rejects: the first value of the value tables that it
// rejects, as [resolver.shaped] builds them. A value other than the zero
// value takes the place of the zero value, when the method rejects one: a
// field leaves out the zero value, so that its encode meets no rejection.
// It reports false when it rejects none.
func (r *resolver) rejected(s *shape) (reflect.Value, bool) {
	var fail reflect.Value
	found := false
	for i := range drawCount {
		v := r.shaped(table(i), s, nesting)
		if validate(v) != nil && (!found || fail.IsZero() && !v.IsZero()) {
			fail, found = v, true
		}
	}
	return fail, found
}

// accepted returns a value of s, the shape of a kanon.Validator, that its
// ValidateKanon accepts: the first value of the value tables that it
// accepts, as [resolver.shaped] builds them, or the zero value when it
// accepts none.
func (r *resolver) accepted(s *shape) reflect.Value {
	ok := reflect.Zero(s.typ)
	found := false
	for i := range drawCount {
		v := r.shaped(table(i), s, nesting)
		if !found && validate(v) == nil {
			ok, found = v, true
		}
	}
	return ok
}

// outsider is a type that the Types of no field list, which an empty
// interface stores in a sample that fails to encode.
type outsider struct{}

// unlisted returns an interface value of the shape s whose concrete type
// the Types of its field do not list, which fails to encode: an outsider,
// or a concrete type that the Spec lists for any field, a pointer to one,
// or the type that one points at, the first of them that implements the
// interface and is comparable, as a map key requires. It reports false when
// there is none.
func (r *resolver) unlisted(s *shape) (reflect.Value, bool) {
	candidates := []reflect.Type{reflect.TypeFor[outsider]()}
	for _, t := range r.concretes {
		candidates = append(candidates, t, reflect.PointerTo(t))
		if t.Kind() == reflect.Pointer {
			candidates = append(candidates, t.Elem())
		}
	}
	var v reflect.Value
	found := false
	for _, t := range candidates {
		if found || !t.Implements(s.typ) || s.variant(t) != nil || !t.Comparable() {
			continue
		}
		v, found = reflect.New(s.typ).Elem(), true
		if t.Kind() == reflect.Pointer {
			v.Set(reflect.New(t.Elem()))
		} else {
			v.Set(reflect.New(t).Elem())
		}
	}
	return v, found
}

// choices returns the most choices that one pick of a sample of l makes, at
// any depth: nil and the concrete types of an interface, or the members of
// a union and the zero discriminator.
func choices(l *layout) int {
	most := 0
	seen := make(map[*shape]bool)
	var walk func(l *layout)
	var shapes func(s *shape)
	walk = func(l *layout) {
		for _, u := range l.unions() {
			most = max(most, len(u)+1)
		}
		for _, f := range l.fields {
			shapes(f.shape)
		}
	}
	shapes = func(s *shape) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		most = max(most, len(s.variants)+1)
		for _, w := range s.variants {
			shapes(w.shape)
		}
		shapes(s.elem)
		shapes(s.key)
		if s.layout != nil {
			walk(s.layout)
		}
	}
	walk(l)
	return most
}

// scalar sets v, a bool, a number, a string, a byte slice or a byte array,
// or a value of a type that encodes itself, to the value of its Go kind
// that src supplies. A struct that encodes itself takes the values of its
// exported fields. Any other value of such a type keeps its zero value.
func (r *resolver) scalar(src source, v reflect.Value, depth int) {
	t := v.Type()
	switch t.Kind() {
	case reflect.Bool:
		v.SetBool(src.boolean())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(src.signed(bitsOf(t)))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		v.SetUint(src.unsigned(bitsOf(t)))
	case reflect.Float32, reflect.Float64:
		v.SetFloat(src.float(t.Bits()))
	case reflect.Complex64, reflect.Complex128:
		v.SetComplex(complex(src.float(t.Bits()/2), src.part(1).float(t.Bits()/2)))
	case reflect.String:
		v.SetString(src.text())
	case reflect.Slice:
		if b := src.bytes(); b != nil && t.Elem() == reflect.TypeFor[byte]() {
			v.SetBytes(b)
		}
	case reflect.Array:
		if src.count() >= 0 && t.Elem() == reflect.TypeFor[byte]() {
			reflect.Copy(v, reflect.ValueOf(src.bytes()))
		}
	case reflect.Struct:
		r.fill(src, r.looseLayout(t), v, depth)
	default:
		// A type that encodes itself as a map, a pointer or an interface
		// takes its zero value.
	}
}

// fill sets the fields of v, a struct of the layout l, from src: field k of
// the layout takes part k+1. A union takes the member that the part of the
// number of its first member picks, or the zero discriminator, and every
// member has a value.
func (r *resolver) fill(src source, l *layout, v reflect.Value, depth int) {
	for k, f := range l.fields {
		f.of(v).Set(r.build(src.part(k+1), f.shape, depth))
	}
	for _, members := range l.unions() {
		d := v.FieldByIndex(members[0].disc)
		choices := make([]reflect.Value, len(members)+1)
		for k, m := range members {
			choices[k] = reflect.ValueOf(m.Case)
		}
		choices[len(members)] = reflect.Zero(d.Type())
		d.Set(choices[src.part(members[0].Number).pick(len(choices))])
	}
}

// unions returns the members of each union of l, in the order of their
// field numbers, one union after another in the order of their first
// members.
func (l *layout) unions() [][]*field {
	var out [][]*field
	index := make(map[string]int)
	for _, f := range l.fields {
		if f.Union == "" {
			continue
		}
		k, ok := index[f.Union]
		if !ok {
			k = len(out)
			index[f.Union] = k
			out = append(out, nil)
		}
		out[k] = append(out[k], f)
	}
	return out
}

// bitsOf returns the width of the values of the integer type t in a
// sample: intBits for an int, a uint and a uintptr, and the width of t
// otherwise.
func bitsOf(t reflect.Type) int {
	switch t.Kind() {
	case reflect.Int, reflect.Uint, reflect.Uintptr:
		return intBits
	default:
		return t.Bits()
	}
}

// newTarget returns a pointer to a new zero value of t at an address that
// no other call returns. reflect.New returns one address for every value
// of a zero-size type, so a value of such a type is the field of a new
// struct after a byte. Distinct zero-size variables of a program can have
// distinct addresses, and pointers to them are distinct map keys of one
// projection.
func newTarget(t reflect.Type) reflect.Value {
	if t.Size() == 0 {
		holder := reflect.StructOf([]reflect.StructField{
			{Name: "Byte", Type: reflect.TypeFor[byte]()},
			{Name: "Value", Type: t},
		})
		return reflect.New(holder).Elem().Field(1).Addr()
	}
	return reflect.New(t)
}
