// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"cmp"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Import paths and names of the packages that the test file imports: the
// conformance suite, testing, and reflect, whose TypeFor names the types of
// a spec.
const (
	kanontestPath = "go.thesmos.sh/kanon/kanontest"
	kanontestName = "kanontest"
	testingPath   = "testing"
	reflectPath   = "reflect"
)

// Prefixes of the declarations that the test file makes per -type struct,
// which the name of the struct completes.
const (
	specPrefix      = "kanonSpec"
	testPrefix      = "TestKanon"
	benchmarkPrefix = "BenchmarkKanon"
	fuzzPrefix      = "FuzzKanon"
	// adapterPrefix begins the name of the type that adapts the stream
	// decoder of a struct to kanontest.Stream.
	adapterPrefix = "kanonStream"
)

// testFile returns the test file of u: for each target, the kanontest.Spec
// that describes its fields, the inline structs of the types of its fields,
// the key structs of the code file, the view type of the target when the
// code file declares one, whether its decode is canonical and the
// constructor of its stream decoder, and the test, benchmark and fuzz
// functions that run the conformance suite on it, and the adapter of its
// stream decoder to kanontest.Stream, as [streamAdapter] writes it. A field
// of a spec states what its Go type does not: its name, its number, the
// fixed option, the union it belongs to and the constant that selects it,
// the concrete types of its interfaces with their numbers, the stream option
// and the bound of the max option. A struct states the field that keeps its
// unknown fields. Each value type gets a test function that runs
// kanontest.RunValue on it.
func testFile(u *unit) ([]byte, error) {
	p := newPrinter(u.pkg.types.Path(), u.pkg.types.Scope().Names())
	kt := p.use(kanontestPath, kanontestName)
	tst := p.std(testingPath)
	for _, m := range u.targets {
		name := p.typ(m.typ)
		spec := specPrefix + m.name
		p.line("")
		p.line("// %s describes %s to the conformance suite.", spec, name)
		p.line("var %s = %s.Spec[%s]{", spec, kt, name)
		specFields(p, m)
		if structs := reached(m); len(structs) > 0 {
			rt := p.std(reflectPath)
			p.line("Structs: []%s.Struct{", kt)
			for _, s := range structs {
				p.line("{")
				p.line("Type: %s.TypeFor[%s](),", rt, p.typ(s.typ))
				p.line("Name: %s,", strconv.Quote(s.name))
				specFields(p, s)
				p.line("},")
			}
			p.line("},")
		}
		specKeys(p, u.keyStructs)
		if u.views {
			p.line("View: %s%s(nil),", m.name, viewSuffix)
		}
		if u.canonical {
			p.line("Canonical: true,")
		}
		fields := streamed(m)
		if len(fields) > 0 {
			p.line("Stream: func(r %s.Reader, size int64, m *%s, opts %s.StreamOptions) %s.Stream[%s] {",
				p.std(ioPath), name, p.use(runtimePath, runtimeName), kt, name)
			p.line("return %s%s{%s%s%s(r, size, m, opts)}", adapterPrefix, m.name, newPrefix, m.name, streamSuffix)
			p.line("},")
		}
		p.line("}")
		p.line("")
		p.line("// %s%s runs the conformance suite on %s.", testPrefix, m.name, name)
		p.line("func %s%s(t *%s.T) {", testPrefix, m.name, tst)
		p.line("%s.Run(t, %s)", kt, spec)
		p.line("}")
		p.line("")
		p.line("// %s%s measures the codec of %s.", benchmarkPrefix, m.name, name)
		p.line("func %s%s(b *%s.B) {", benchmarkPrefix, m.name, tst)
		p.line("%s.Bench(b, %s)", kt, spec)
		p.line("}")
		p.line("")
		p.line("// %s%s fuzzes the decode of %s.", fuzzPrefix, m.name, name)
		p.line("func %s%s(f *%s.F) {", fuzzPrefix, m.name, tst)
		p.line("%s.Fuzz(f, %s)", kt, spec)
		p.line("}")
		if len(fields) > 0 {
			streamAdapter(p, m, fields)
		}
	}
	for _, vt := range u.values {
		name := p.typ(vt.typ)
		p.line("")
		p.line("// %s%s runs the conformance suite on %s.", testPrefix, vt.typ.Obj().Name(), name)
		p.line("func %s%s(t *%s.T) {", testPrefix, vt.typ.Obj().Name(), tst)
		p.line("%s.RunValue[%s](t)", kt, name)
		p.line("}")
	}
	return p.source(u.header(), u.pkg.types.Name())
}

// specFields writes the fields of the spec of m: one kanontest.Field
// literal per encoded field, and the name of the field that keeps the
// unknown fields of m.
func specFields(p *printer, m *target) {
	kt := p.use(kanontestPath, kanontestName)
	p.line("Fields: []%s.Field{", kt)
	for _, f := range m.fields {
		parts := []string{"Name: " + strconv.Quote(f.name), "Number: " + strconv.Itoa(f.num)}
		if f.tag.fixed {
			parts = append(parts, "Fixed: true")
		}
		if f.member != nil {
			parts = append(parts, "Union: "+strconv.Quote(f.member.disc.Name()), "Case: "+p.object(f.member.value))
		}
		if f.list != nil {
			rt := p.std(reflectPath)
			types := make([]string, 0, len(f.list.types))
			for _, c := range f.list.types {
				types = append(types, "{Type: "+rt+".TypeFor["+p.typ(c.typ)+"](), Number: "+strconv.Itoa(c.num)+"}")
			}
			parts = append(parts, "Types: []"+kt+".ConcreteType{"+strings.Join(types, ", ")+"}")
		}
		if f.tag.stream {
			parts = append(parts, "Stream: true")
		}
		if f.tag.max != 0 {
			parts = append(parts, "Max: "+strconv.Itoa(f.tag.max))
		}
		p.line("{%s},", strings.Join(parts, ", "))
	}
	p.line("},")
	if m.unknown != nil {
		p.line("Unknown: %s,", strconv.Quote(m.unknown.Name()))
	}
}

// specKeys writes the key structs of a spec: one kanontest.Struct literal
// per struct type with a kanon codec in the map keys of the code file, in
// the order of their type strings, with its fields in ascending field
// number. It writes nothing for a code file without such keys.
func specKeys(p *printer, keys map[string]keyStruct) {
	if len(keys) == 0 {
		return
	}
	kt, rt := p.use(kanontestPath, kanontestName), p.std(reflectPath)
	p.line("Keys: []%s.Struct{", kt)
	for _, name := range slices.Sorted(maps.Keys(keys)) {
		k := keys[name]
		p.line("{")
		p.line("Type: %s.TypeFor[%s](),", rt, p.typ(k.typ))
		p.line("Fields: []%s.Field{", kt)
		fields := slices.SortedFunc(
			maps.Keys(k.nums),
			func(a, b string) int { return cmp.Compare(k.nums[a], k.nums[b]) },
		)
		for _, f := range fields {
			p.line("{Name: %s, Number: %d},", strconv.Quote(f), k.nums[f])
		}
		p.line("},")
		p.line("},")
	}
	p.line("},")
}

// streamAdapter writes the type that adapts the stream decoder of m, whose
// streamed fields are fields, to kanontest.Stream: it embeds the stream
// decoder, whose Reset and Len it promotes, and its Read and Element when
// they exist, and adds Next, which returns the number of the field, and,
// when a slice streams, Decode, which calls the decode method of the slice
// num on the element e. Decode returns io.EOF for a number of no streamed
// slice, as the decode method of another field does.
func streamAdapter(p *printer, m *target, fields []*field) {
	kt := p.use(kanontestPath, kanontestName)
	adapter, stream := adapterPrefix+m.name, m.name+streamSuffix
	p.line("")
	p.line("// %s adapts a %s to %s.Stream.", adapter, stream, kt)
	p.line("type %s struct {", adapter)
	p.line("*%s", stream)
	p.line("}")
	p.line("")
	p.line("// Next returns the number of the next streamed field of %s.", m.name)
	p.line("func (s %s) Next() (int, error) {", adapter)
	p.line("f, err := s.%s.Next()", stream)
	p.line("return int(f), err")
	p.line("}")
	var slices []*field
	for _, f := range fields {
		if f.val.kind == kindSlice {
			slices = append(slices, f)
		}
	}
	if len(slices) == 0 {
		return
	}
	p.line("")
	for _, l := range wrap("Decode decodes the next element of the streamed slice of " + m.name +
		" with the number num into e, a pointer to an element.") {
		p.line("%s", l)
	}
	p.line("func (s %s) Decode(num int, e any) error {", adapter)
	p.line("switch num {")
	for _, f := range slices {
		p.line("case %d:", f.num)
		p.line("return s.%s%s(e.(*%s))", decodePrefix, f.name, p.typ(f.val.elem.typ))
	}
	p.line("}")
	p.line("return %s.EOF", p.std(ioPath))
	p.line("}")
}

// reached returns the inline structs of the types of the fields of m: in the
// fields of inline structs, the elements of slices and arrays, the keys and
// values of maps, the values that pointers point at, and the concrete types
// of interfaces, in the order in which a walk of the fields in declaration
// order first visits them.
func reached(m *target) []*target {
	var out []*target
	seen := make(map[*value]bool)
	var walk func(v *value)
	walk = func(v *value) {
		if v == nil || seen[v] {
			return
		}
		seen[v] = true
		if s := v.inline; s != nil && !slices.Contains(out, s) {
			out = append(out, s)
			for _, f := range s.fields {
				walk(f.val)
			}
		}
		for _, w := range v.variants {
			walk(w.val)
		}
		walk(v.key)
		walk(v.elem)
	}
	for _, f := range m.fields {
		walk(f.val)
	}
	return out
}
