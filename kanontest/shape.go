// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// kind is the encoding of a Go value, as the kanon generator classifies its
// type. A value of a kind encodes alike wherever it occurs.
type kind uint8

// Kinds. The zero kind is invalid.
const (
	// kindBool is a varint of 0 or 1.
	kindBool kind = 1
	// kindInt is the zigzag varint of a signed integer.
	kindInt kind = 2
	// kindUint is the varint of an unsigned integer.
	kindUint kind = 3
	// kindFixed is the four or eight little-endian bytes of an integer of 32
	// or 64 bits under the tag option fixed.
	kindFixed kind = 4
	// kindFloat is the four or eight little-endian IEEE 754 bytes of a
	// float.
	kindFloat kind = 5
	// kindComplex64 is the eight bytes of the two float32 parts.
	kindComplex64 kind = 6
	// kindComplex128 is the length 16 and the two float64 parts.
	kindComplex128 kind = 7
	// kindString is a length and the bytes of a string.
	kindString kind = 8
	// kindBytes is a length and the bytes of a byte slice.
	kindBytes kind = 9
	// kindByteArray is a length and the bytes of a byte array.
	kindByteArray kind = 10
	// kindStruct is a length and the encoding of a struct with a kanon
	// codec, which the methods of its codec write and read.
	kindStruct kind = 11
	// kindInline is a length and the fields of an inline struct, which the
	// layout of the struct numbers.
	kindInline kind = 12
	// kindTime is a length and the three fields of a time.Time.
	kindTime kind = 13
	// kindBinary is a length and the encoding of a type that encodes
	// itself.
	kindBinary kind = 14
	// kindSlice is a length and the elements of a slice.
	kindSlice kind = 15
	// kindArray is a length and the elements of an array.
	kindArray kind = 16
	// kindMap is a length and the keys and values of a map, in key order.
	kindMap kind = 17
	// kindPointer is a presence byte and the value that a pointer points
	// at.
	kindPointer kind = 18
	// kindInterface is a type number and the value of that concrete type.
	kindInterface kind = 19
)

// Words of the kanon struct tag that change which fields of a struct type
// that no Spec describes are encoded.
const (
	// tagKey is the struct tag key that kanon reads.
	tagKey = "kanon"
	// tagSkip leaves a field out of the encoding.
	tagSkip = "-"
	// tagUnknown marks the field that keeps unknown fields.
	tagUnknown = "unknown"
)

// blankName is the name of a blank struct field, which no code reads or
// writes.
const blankName = "_"

// shape is the encoding of the values of one Go type.
type shape struct {
	typ  reflect.Type
	kind kind
	// elem is the shape of the elements of a slice or an array, of the
	// values of a map, and of the value that a pointer points at.
	elem *shape
	// key is the shape of the keys of a map.
	key *shape
	// layout is the layout of an inline struct.
	layout *layout
	// variants lists the concrete types that an interface stores, in the
	// order of the Types of its field.
	variants []variant
	// zeroAbsent reports that a value of a type that encodes itself is
	// absent exactly when it is the zero value of its type, since == compares
	// every bit of the type, as [bitwiseType] reports.
	zeroAbsent bool
	// sizer reports that a type that encodes itself is a kanon.Sizer, as
	// [sizes] reports, whose encoding the generated code writes into the
	// room that its SizeKanon sizes.
	sizer bool
	// validate reports that the type is a kanon.Validator, as [validates]
	// reports: it encodes as its underlying type, and every value that the
	// codec writes or reads passes its ValidateKanon.
	validate bool
}

// variant is a concrete type that an interface stores, and its number.
type variant struct {
	num   int
	shape *shape
}

// wire returns the wire format of a value of s after the tag of a field: a
// varint for a bool and an integer, four or eight bytes for a fixed-size
// value, and a length and bytes otherwise.
func (s *shape) wire() int {
	switch s.kind {
	case kindBool, kindInt, kindUint:
		return wire.Varint
	case kindFixed, kindFloat:
		if s.typ.Size() == fixed32Width {
			return wire.Fixed32
		}
		return wire.Fixed64
	case kindComplex64:
		return wire.Fixed64
	default:
		return wire.Bytes
	}
}

// unframed reports whether the encoding of a value of s begins with a
// presence byte or a type number instead of a length or a fixed width: a
// pointer and an interface. A field puts a length before such a value.
func (s *shape) unframed() bool {
	return s.kind == kindPointer || s.kind == kindInterface
}

// level reports whether a value of s is a level of the depth of a decode:
// a struct, a slice, a map, a pointer or an interface. An array is no
// level, and neither is a time.
func (s *shape) level() bool {
	switch s.kind {
	case kindStruct, kindInline, kindSlice, kindMap, kindPointer, kindInterface:
		return true
	default:
		return false
	}
}

// variant returns the variant of the interface shape s that stores values
// of type t, or nil when the Types of the field do not list t.
func (s *shape) variant(t reflect.Type) *variant {
	for k := range s.variants {
		if s.variants[k].shape.typ == t {
			return &s.variants[k]
		}
	}
	return nil
}

// layout is the encoding of a struct type: T, an inline struct that a
// Spec describes, or a struct type with a kanon codec, whose fields a
// sample fills.
type layout struct {
	typ reflect.Type
	// name names the struct in the errors of the codec.
	name string
	// fields lists the encoded fields in ascending field number, the order
	// of the encoding. A layout that no Spec describes lists its fields in
	// the order of their names, with number 0, so that a reorder of the
	// field declarations leaves the samples that fill the struct unchanged.
	fields []*field
	// unknown is the index of the field that keeps unknown fields, and nil
	// when the struct keeps none.
	unknown []int
}

// field is an encoded field of a layout.
type field struct {
	Field
	// index is the index path of the field in its struct type.
	index []int
	// exported reports that the field is exported, which reflection reads
	// and sets.
	exported bool
	// shape is the encoding of the type of the field.
	shape *shape
	// disc is the index path of the discriminator of a union member.
	disc []int
}

// loc returns the location of f of l in the errors of the codec:
// "Type.Field".
func (l *layout) loc(f *field) string {
	return l.name + "." + f.Name
}

// selected reports whether the struct v of l selects its union member f:
// the discriminator of f has the value of Case.
func (*layout) selected(v reflect.Value, f *field) bool {
	return v.FieldByIndex(f.disc).Equal(reflect.ValueOf(f.Case))
}

// of returns the field f of the struct v, which the returned value can set
// when v is addressable: the field at its index for an exported field, and
// for an unexported field the value that reflect.NewAt makes at the field's
// address, since reflection neither sets an unexported field nor returns its
// value as an interface. The unexported field of a struct that is not
// addressable, such as a map key, is the field of a copy of v.
func (f *field) of(v reflect.Value) reflect.Value {
	if f.exported {
		return v.FieldByIndex(f.index)
	}
	if !v.CanAddr() {
		c := reflect.New(v.Type()).Elem()
		c.Set(v)
		v = c
	}
	x := v.FieldByIndex(f.index)
	return reflect.NewAt(x.Type(), x.Addr().UnsafePointer()).Elem()
}

// resolver resolves the shapes and the layouts of the Go types of one Spec.
type resolver struct {
	// pkg is the import path of the package of T.
	pkg string
	// layouts maps each struct type that the Spec describes to its layout.
	layouts map[reflect.Type]*layout
	// loose maps each other struct type whose fields a sample fills to its
	// layout.
	loose map[reflect.Type]*layout
	// concretes lists the concrete types that the Spec lists for any field,
	// once each, among which a sample finds a type that an interface of
	// another field does not list.
	concretes []reflect.Type
	// bad caches the ways to make the keys of a map of each key shape fail
	// its encode, as [resolver.badKeys] finds them.
	bad map[*shape][]badKey
	// taints maps the type of each field that the encoding leaves out of a
	// struct that a value of T can contain to the value that
	// [resolver.taintField] sets it to, and to the invalid value for a type of
	// which [resolver.taintOf] returns none. [resolver.prepare] fills it.
	taints map[reflect.Type]reflect.Value
	// opaques maps each type that encodes itself that a sample builds to the
	// lengths at which [resolver.opaque] found that its decode method decodes
	// a value, and to nil for a type that reflection fills.
	opaques map[reflect.Type][]int
	// canonical reports that the Spec is canonical, so that the reference
	// decode applies the rules of a canonical decode.
	canonical bool
}

// resolverFor returns the resolver of the types of package pkg without a
// Spec: every collection empty, and the reference decode not canonical.
func resolverFor(pkg string) *resolver {
	return &resolver{
		pkg:     pkg,
		layouts: make(map[reflect.Type]*layout),
		loose:   make(map[reflect.Type]*layout),
		bad:     make(map[*shape][]badKey),
		taints:  make(map[reflect.Type]reflect.Value),
		opaques: make(map[reflect.Type][]int),
	}
}

// opts are the options of a field that apply to the shapes of its type.
type opts struct {
	// types lists the concrete types of the interfaces of the field.
	types []ConcreteType
	// seen maps each type of the field that the resolution met, with the
	// options that change its shape, to its shape, so that a type that
	// contains itself ends the resolution.
	seen map[seenKey]*shape
	// fixed selects the fixed-size encoding of the 32- and 64-bit integers.
	fixed bool
	// key reports a value inside a map key, whose interfaces store the
	// comparable concrete types alone.
	key bool
	// loose resolves the fields of a struct type that no Spec describes: an
	// interface stores nothing, and a struct type without a layout takes a
	// loose one.
	loose bool
}

// seenKey identifies a shape among the shapes of one field.
type seenKey struct {
	typ   reflect.Type
	fixed bool
	key   bool
}

// newResolver returns the resolver of spec and the layout of T. It fails
// when spec does not describe T, its inline structs and its key structs: a
// field, a discriminator or an unknown field that the struct type does not
// declare, a Struct whose type is not a struct, a field of a type that
// kanon does not encode, a struct type without a kanon codec that no
// Struct describes, and an interface that no concrete type of the field
// fits.
func newResolver[T any](spec Spec[T]) (*resolver, *layout, error) {
	typ := reflect.TypeFor[T]()
	r := resolverFor(typ.PkgPath())
	r.canonical = spec.Canonical
	root := &layout{typ: typ, name: typ.Name()}
	r.layouts[typ] = root
	structs := slices.Concat(spec.Structs, spec.Keys)
	for _, st := range structs {
		if st.Type == nil || st.Type.Kind() != reflect.Struct {
			return nil, nil, fmt.Errorf("kanontest: the Spec describes %v, which is not a struct type", st.Type)
		}
		// A key struct has no Name, and names itself by its type name in
		// the errors of the projection that its fields write.
		name := st.Name
		if name == "" {
			name = st.Type.Name()
		}
		r.layouts[st.Type] = &layout{typ: st.Type, name: name}
	}
	if err := r.describe(root, spec.Fields, spec.Unknown); err != nil {
		return nil, nil, err
	}
	for _, st := range structs {
		if err := r.describe(r.layouts[st.Type], st.Fields, st.Unknown); err != nil {
			return nil, nil, err
		}
	}
	return r, root, nil
}

// describe sets the fields of l, the layout of a struct type that the Spec
// describes with fields and unknown, in ascending field number. It fails
// as [newResolver] states, and for an unexported discriminator or unknown
// field, which the generator rejects.
func (r *resolver) describe(l *layout, fields []Field, unknown string) error {
	for _, f := range fields {
		sf, ok := l.typ.FieldByName(f.Name)
		if !ok {
			return fmt.Errorf("kanontest: the Spec names field %s, which %s does not declare", f.Name, l.typ)
		}
		s, err := r.shapeOf(sf.Type, opts{types: f.Types, seen: make(map[seenKey]*shape), fixed: f.Fixed})
		if err != nil {
			return fmt.Errorf("kanontest: field %s of %s: %w", f.Name, l.typ, err)
		}
		for _, c := range f.Types {
			if !slices.Contains(r.concretes, c.Type) {
				r.concretes = append(r.concretes, c.Type)
			}
		}
		fi := &field{Field: f, index: sf.Index, exported: sf.IsExported(), shape: s}
		if f.Union != "" {
			d, ok := l.typ.FieldByName(f.Union)
			if !ok {
				return fmt.Errorf(
					"kanontest: the Spec names discriminator %s, which %s does not declare",
					f.Union,
					l.typ,
				)
			}
			if !d.IsExported() {
				return fmt.Errorf("kanontest: the Spec names discriminator %s, which %s does not export",
					f.Union, l.typ)
			}
			fi.disc = d.Index
		}
		l.fields = append(l.fields, fi)
	}
	slices.SortFunc(l.fields, func(a, b *field) int { return cmp.Compare(a.Number, b.Number) })
	if unknown != "" {
		u, ok := l.typ.FieldByName(unknown)
		if !ok {
			return fmt.Errorf("kanontest: the Spec names unknown field %s, which %s does not declare", unknown, l.typ)
		}
		if !u.IsExported() {
			return fmt.Errorf("kanontest: the Spec names unknown field %s, which %s does not export", unknown, l.typ)
		}
		l.unknown = u.Index
	}
	return nil
}

// shapeOf returns the shape of the Go type t under o. A type resolves in the
// order of the generator: time.Time, a struct with a kanon codec, a
// kanon.Validator, which takes the shape of its kind, a type that encodes
// itself, and last its kind, so that a time.Duration encodes as the int64
// that it is and any other struct is an inline struct. It fails for a type
// that kanon does not encode, and as [resolver.inline] and [resolver.iface]
// fail.
func (r *resolver) shapeOf(t reflect.Type, o opts) (*shape, error) {
	k := seenKey{typ: t, fixed: o.fixed, key: o.key}
	if s := o.seen[k]; s != nil {
		return s, nil
	}
	s := &shape{typ: t}
	o.seen[k] = s
	if t == reflect.TypeFor[time.Time]() {
		s.kind = kindTime
		return s, nil
	}
	if reflect.PointerTo(t).Implements(reflect.TypeFor[kanon.Message]()) {
		s.kind = kindStruct
		return s, nil
	}
	s.validate = validates(t)
	if !s.validate && familyOf(t) != 0 {
		s.kind, s.zeroAbsent, s.sizer = kindBinary, bitwiseType(t), sizes(t)
		return s, nil
	}
	var err error
	switch t.Kind() {
	case reflect.Bool:
		s.kind = kindBool
	case reflect.Int, reflect.Int8, reflect.Int16:
		s.kind = kindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uintptr:
		s.kind = kindUint
	case reflect.Int32, reflect.Int64:
		s.kind = pick(o.fixed, kindFixed, kindInt)
	case reflect.Uint32, reflect.Uint64:
		s.kind = pick(o.fixed, kindFixed, kindUint)
	case reflect.Float32, reflect.Float64:
		s.kind = kindFloat
	case reflect.Complex64:
		s.kind = kindComplex64
	case reflect.Complex128:
		s.kind = kindComplex128
	case reflect.String:
		s.kind = kindString
	case reflect.Struct:
		s.kind = kindInline
		s.layout, err = r.inline(t, o.loose)
	case reflect.Slice, reflect.Array:
		err = r.sequence(s, o)
	case reflect.Map:
		s.kind = kindMap
		keyOpts := o
		keyOpts.fixed, keyOpts.key = false, true
		if s.key, err = r.shapeOf(t.Key(), keyOpts); err == nil {
			s.elem, err = r.shapeOf(t.Elem(), o)
		}
	case reflect.Pointer:
		s.kind = kindPointer
		s.elem, err = r.shapeOf(t.Elem(), o)
	case reflect.Interface:
		err = r.iface(s, o)
	default:
		err = fmt.Errorf("type %s is not encoded", t)
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// sequence completes s, the shape of a slice or an array: bytes for a
// sequence of byte itself, whose elements have no shape, and a sequence of
// elements of their own shape otherwise.
func (r *resolver) sequence(s *shape, o opts) error {
	bytes := s.typ.Elem() == reflect.TypeFor[byte]()
	if s.typ.Kind() == reflect.Slice {
		s.kind = pick(bytes, kindBytes, kindSlice)
	} else {
		s.kind = pick(bytes, kindByteArray, kindArray)
	}
	if bytes {
		return nil
	}
	var err error
	s.elem, err = r.shapeOf(s.typ.Elem(), o)
	return err
}

// inline returns the layout of the inline struct type t: the layout of its
// Struct in the Spec, or with loose set a layout of the encoded fields of t
// in the order of their names. It fails for a struct type that the Spec does
// not describe unless loose is set.
func (r *resolver) inline(t reflect.Type, loose bool) (*layout, error) {
	if l := r.layouts[t]; l != nil {
		return l, nil
	}
	if !loose {
		return nil, fmt.Errorf("struct type %s has no kanon codec and no Struct in the Spec", t)
	}
	return r.looseLayout(t), nil
}

// looseLayout returns the layout of the struct type t that no Spec
// describes: its fields that kanon encodes, as [encoded] reports, in the
// order of their names, whose interfaces store nothing. A reorder of the
// declarations of t, which keeps the field numbers of a struct with a kanon
// codec, leaves the layout unchanged. looseLayout records the layout before
// it resolves the fields, so that a struct type that contains itself ends
// the resolution.
func (r *resolver) looseLayout(t reflect.Type) *layout {
	if l := r.loose[t]; l != nil {
		return l
	}
	l := &layout{typ: t, name: t.String()}
	r.loose[t] = l
	for sf := range t.Fields() {
		if !encoded(sf) {
			continue
		}
		// A loose resolution fails for no type that a generated codec
		// contains, since the generator rejects every other type.
		s, _ := r.shapeOf(sf.Type, opts{seen: make(map[seenKey]*shape), loose: true})
		l.fields = append(l.fields, &field{Name: sf.Name, index: sf.Index, exported: sf.IsExported(), shape: s})
	}
	slices.SortFunc(l.fields, func(a, b *field) int { return cmp.Compare(a.Name, b.Name) })
	return l
}

// iface completes s, the shape of an interface: its variants are the
// concrete types of o.types that implement it, and inside a map key the
// comparable ones alone. It fails when none of them fits, unless o.loose
// is set, under which an interface stores nothing.
func (r *resolver) iface(s *shape, o opts) error {
	s.kind = kindInterface
	for _, c := range o.types {
		if !c.Type.Implements(s.typ) || o.key && !c.Type.Comparable() {
			continue
		}
		cs, err := r.shapeOf(c.Type, o)
		if err != nil {
			return err
		}
		s.variants = append(s.variants, variant{num: c.Number, shape: cs})
	}
	if len(s.variants) == 0 && !o.loose {
		return fmt.Errorf("the Types of the field list no concrete type of interface type %s", s.typ)
	}
	return nil
}

// encoded reports whether kanon encodes the struct field sf: sf is exported
// or has a kanon tag, which opts an unexported field in, it is neither a
// blank field nor a field that [unsent] reports, and its kanon tag neither
// skips it nor keeps unknown fields.
func encoded(sf reflect.StructField) bool {
	tag, tagged := sf.Tag.Lookup(tagKey)
	if !sf.IsExported() && !tagged || sf.Name == blankName || unsent(sf.Type) {
		return false
	}
	return tag != tagSkip && !keepsUnknown(tag)
}

// unsent reports whether kanon leaves a field of type t out of the
// encoding, as encoding/gob leaves it out: t is a function or a channel, or
// a pointer to one at any depth. A named pointer type that points at itself
// is sent.
func unsent(t reflect.Type) bool {
	seen := make(map[reflect.Type]bool)
	for t.Kind() == reflect.Pointer && !seen[t] {
		seen[t] = true
		t = t.Elem()
	}
	return t.Kind() == reflect.Func || t.Kind() == reflect.Chan
}

// bitwiseType reports whether == compares every bit of a value of t, as the
// generator decides it for a type that encodes itself: t consists of bools,
// integers, strings, pointers and channels, in arrays and structs. A float
// compares equal at -0.0 and +0.0, and an interface can store a value that
// == panics on, so a type that contains either does not. A slice, a map and
// a function have no ==.
func bitwiseType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Interface,
		reflect.Slice, reflect.Map, reflect.Func:
		return false
	case reflect.Array:
		return bitwiseType(t.Elem())
	case reflect.Struct:
		for f := range t.Fields() {
			if !bitwiseType(f.Type) {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// leftOut reports whether the encoding leaves out the struct field sf, whose
// value Reset and a decode clear and a merge keeps: a field that kanon does
// not encode, as [encoded] reports, other than a blank field, which no code
// reads or writes, and the field that keeps unknown fields.
func leftOut(sf reflect.StructField) bool {
	return sf.Name != blankName && !encoded(sf) && !keepsUnknown(sf.Tag.Get(tagKey))
}

// keepsUnknown reports whether the kanon tag value tag marks the field that
// keeps unknown fields.
func keepsUnknown(tag string) bool {
	return slices.Contains(strings.Split(tag, ","), tagUnknown)
}

// pick returns a when cond is true and b otherwise.
func pick(cond bool, a, b kind) kind {
	if cond {
		return a
	}
	return b
}
