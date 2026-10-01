// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanontest

import "reflect"

// self sets v, a new value of a type that encodes itself, from src. A type
// of which [resolver.opaque] finds lengths takes the value that its decode
// method decodes from the bytes of src repeated to one of them, which src
// picks, or from the bytes that [counted] returns when the method rejects
// those, and the zero value for a nil byte slice of src. Any other type takes
// the value of its Go kind that [resolver.scalar] sets.
func (r *resolver) self(src source, v reflect.Value, depth int) {
	lengths := r.opaque(v.Type())
	if len(lengths) == 0 {
		r.scalar(src, v, depth)
		return
	}
	b := src.bytes()
	if b == nil {
		return
	}
	n := lengths[src.pick(len(lengths))]
	x, ok := decodes(v.Type(), repeated(b, n))
	if !ok {
		x, _ = decodes(v.Type(), counted(n))
	}
	v.Set(x)
}

// opaque returns the lengths at which the decode method of t, a type that
// encodes itself, decodes the bytes that [counted] returns to a value other
// than the zero value, every length from 0 to the size of a value of t in
// memory, for a type of which [resolver.scalar] sets no value other than the
// zero value from the value tables, such as a struct whose fields are all
// unexported. It returns nil for any other type, and for a type whose decode
// method decodes no such bytes. It records the lengths of each type, so that
// the search runs once per type.
func (r *resolver) opaque(t reflect.Type) []int {
	if lengths, ok := r.opaques[t]; ok {
		return lengths
	}
	var lengths []int
	if !r.fills(t) {
		for n := range int(t.Size()) + 1 {
			if _, ok := decodes(t, counted(n)); ok {
				lengths = append(lengths, n)
			}
		}
	}
	r.opaques[t] = lengths
	return lengths
}

// fills reports whether [resolver.scalar] sets a value of t other than the
// zero value from an entry of the value tables.
func (r *resolver) fills(t reflect.Type) bool {
	for i := range drawCount {
		v := reflect.New(t).Elem()
		r.scalar(table(i), v, 0)
		if !v.IsZero() {
			return true
		}
	}
	return false
}

// decodes returns the value of t, a type that encodes itself, that its decode
// method decodes data to, and reports whether the method accepts data and the
// value is not the zero value.
func decodes(t reflect.Type, data []byte) (reflect.Value, bool) {
	v := reflect.New(t).Elem()
	err := unmarshal(v, data)
	return v, err == nil && !v.IsZero()
}

// counted returns n bytes that count from 1: 1, 2, 3 and so on, modulo 256.
func counted(n int) []byte {
	b := make([]byte, n)
	for k := range b {
		b[k] = byte(k + 1)
	}
	return b
}

// repeated returns n bytes of b repeated from its start, and the bytes that
// [counted] returns for an empty b.
func repeated(b []byte, n int) []byte {
	if len(b) == 0 {
		return counted(n)
	}
	out := make([]byte, n)
	for k := range out {
		out[k] = b[k%len(b)]
	}
	return out
}
