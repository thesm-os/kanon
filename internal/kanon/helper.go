// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/types"
	"strconv"
	"strings"
	"unicode"
)

// Words of the Go types in the names of helpers, as [typeWord] joins them.
const (
	sliceWord   = "Slice"
	arrayWord   = "Array"
	mapWord     = "Map"
	pointerWord = "Ptr"
	anyWord     = "Any"
)

// op is an operation of the code file on the values of one encoding: a
// slice, an array, a map, a pointer, an interface, a type that encodes
// itself or an inline struct. Each op on the values of one encoding is one
// helper function.
type op uint8

// Operations. The zero op is invalid.
const (
	// opSize returns the length of an encoding.
	opSize op = 1
	// opPut writes an encoding into the end of a buffer.
	opPut op = 2
	// opRead decodes an encoding into a value.
	opRead op = 3
	// opMerge decodes the encoding of an inline struct into a value without
	// resetting the value first.
	opMerge op = 4
	// opFields decodes the fields of an inline struct for opRead and
	// opMerge.
	opFields op = 5
	// opReset resets a value and keeps its memory.
	opReset op = 6
	// opClone copies a value.
	opClone op = 7
	// opCompare orders two map keys.
	opCompare op = 8
	// opPresent reports whether an array field is present.
	opPresent op = 9
	// opDeselect zeroes the member of a union that its discriminator
	// selects.
	opDeselect op = 10
	// opKeySize, opKeyPut and opKeyPresent are opSize, opPut and opPresent
	// for a map key, whose encoding writes every float component of -0.0 as
	// +0.0, so that a float field of -0.0 is absent.
	opKeySize    op = 11
	opKeyPut     op = 12
	opKeyPresent op = 13
	// opNaN reports whether a map key has a NaN component.
	opNaN op = 14
	// opCanon reports whether a map key is the key that a decode of its
	// projection yields.
	opCanon op = 15
)

// word returns the word of o in the names of its helpers, after the prefix
// of the file.
func (o op) word() string {
	switch o {
	case opSize:
		return "size"
	case opPut:
		return "put"
	case opRead:
		return "read"
	case opMerge:
		return "merge"
	case opFields:
		return "fields"
	case opReset:
		return "reset"
	case opClone:
		return "clone"
	case opCompare:
		return "compare"
	case opPresent:
		return "present"
	case opKeySize:
		return "keysize"
	case opKeyPut:
		return "keyput"
	case opKeyPresent:
		return "keypresent"
	case opNaN:
		return "nan"
	case opCanon:
		return "canon"
	default:
		return "deselect"
	}
}

// helper is a function that the code file declares: an operation on the
// values of one encoding, or the deselector of a union.
type helper struct {
	name string
	op   op
	v    *value
	// m is the struct of a deselector, and disc its discriminator.
	m    *target
	disc *types.Var
}

// helperKey identifies a helper: its operation and the id of its values, or
// the discriminator of a deselector.
type helperKey struct {
	op   op
	id   string
	disc *types.Var
}

// fn returns the name of the helper of o for the values of v, and records
// that the code calls it. The name is the prefix of the file, the word of o
// and the word of v.
func (e *emitter) fn(o op, v *value) string {
	return e.helper(helper{op: o, v: v}, helperKey{op: o, id: v.id}, e.word(v))
}

// deselector returns the name of the helper that zeroes the selected member
// of the union of the discriminator disc of m, and records that the code
// calls it.
func (e *emitter) deselector(m *target, disc *types.Var) string {
	return e.helper(helper{op: opDeselect, m: m, disc: disc}, helperKey{op: opDeselect, disc: disc},
		camel(m.name)+disc.Name())
}

// helper returns the name of h, whose key is k and whose name ends with
// word, and records h as pending on the first call.
func (e *emitter) helper(h helper, k helperKey, word string) string {
	if name, ok := e.names[k]; ok {
		return name
	}
	h.name = e.prefix + h.op.word() + word
	e.names[k] = h.name
	e.pending = append(e.pending, h)
	return h.name
}

// word returns the word that names the helpers of the values of v: the name
// of the site of an anonymous inline struct, and otherwise the words of the
// Go type of v, as [typeWord] joins them. A number from 2 follows the word
// when the values of another id have the same one.
func (e *emitter) word(v *value) string {
	if w, ok := e.words[v.id]; ok {
		return w
	}
	base := typeWord(v.typ, e.p.self)
	if v.inline != nil && v.inline.decl == nil {
		base = camel(v.inline.name)
	}
	w := base
	for k := range len(e.owners) {
		if e.owners[w] == "" {
			break
		}
		w = base + strconv.Itoa(k+2)
	}
	e.words[v.id], e.owners[w] = w, v.id
	return w
}

// helpers writes the declarations of the pending helpers, and of the
// helpers that they call in turn, in the order in which the code first
// calls them.
func (e *emitter) helpers() {
	for len(e.pending) > 0 {
		h := e.pending[0]
		e.pending = e.pending[1:]
		e.writeHelper(h)
	}
}

// writeHelper writes the declaration of h. The helpers of a map key write
// the projection of the key, as [emitter.inKey] states.
func (e *emitter) writeHelper(h helper) {
	e.inKey = h.op == opKeySize || h.op == opKeyPut || h.op == opKeyPresent
	switch h.op {
	case opSize, opKeySize:
		e.sizeHelper(h.name, h.v)
	case opPut, opKeyPut:
		e.putHelper(h.name, h.v)
	case opRead:
		e.readHelper(h.name, h.v)
	case opMerge:
		e.mergeHelper(h.name, h.v)
	case opFields:
		e.fieldsHelper(h.name, h.v)
	case opReset:
		e.resetHelper(h.name, h.v)
	case opClone:
		e.cloneHelper(h.name, h.v)
	case opCompare:
		e.compareHelper(h.name, h.v)
	case opPresent, opKeyPresent:
		e.presentHelper(h.name, h.v)
	case opNaN:
		e.nanHelper(h.name, h.v)
	case opCanon:
		e.canonHelper(h.name, h.v)
	default:
		e.deselectHelper(h.name, h.m, h.disc)
	}
	e.inKey = false
}

// typeWord returns the words of the Go type t in the name of a helper: the
// name of a named type, preceded by the name of its package outside the
// package self and followed by the words of its type arguments; the name of
// a basic type; and Slice, Array with the length, Map, Ptr and Any followed
// by the words of their elements, keys and values.
func typeWord(t types.Type, self string) string {
	switch t := types.Unalias(t).(type) {
	case *types.Named:
		var sb strings.Builder
		if pkg := t.Obj().Pkg(); pkg != nil && pkg.Path() != self {
			sb.WriteString(camel(pkg.Name()))
		}
		sb.WriteString(camel(t.Obj().Name()))
		for arg := range t.TypeArgs().Types() {
			sb.WriteString(typeWord(arg, self))
		}
		return sb.String()
	case *types.Basic:
		return camel(t.Name())
	case *types.Slice:
		return sliceWord + typeWord(t.Elem(), self)
	case *types.Array:
		return arrayWord + strconv.FormatInt(t.Len(), 10) + typeWord(t.Elem(), self)
	case *types.Map:
		return mapWord + typeWord(t.Key(), self) + typeWord(t.Elem(), self)
	case *types.Pointer:
		return pointerWord + typeWord(t.Elem(), self)
	default:
		return anyWord
	}
}

// camel returns s with its first letter in upper case and every character
// that cannot occur in an identifier removed, where the letter after it
// begins a new word in upper case.
func camel(s string) string {
	var sb strings.Builder
	upper := true
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			upper = true
			continue
		}
		if upper {
			r = unicode.ToUpper(r)
			upper = false
		}
		sb.WriteRune(r)
	}
	return sb.String()
}
