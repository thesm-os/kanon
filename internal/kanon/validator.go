// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"fmt"
	"go/types"
)

// Words that name the families of methods of [selfCodecs] in the docblock of
// a generated ValidateKanon method.
const (
	binaryFamily = "binary"
	gobFamily    = "gob"
	textFamily   = "text"
)

// fromName names the local variable of the generated decode that records
// the offset of a value of a kanon.Validator, which the check of its
// ValidateKanon reports.
const fromName = "from"

// valueType is a named type that is not a struct and that a -type flag
// names. The code file declares its ValidateKanon method, through which the
// type is a kanon.Validator, and the test file runs kanontest.RunValue on
// it.
type valueType struct {
	typ *types.Named
	// check names the method of the type that ValidateKanon calls, and is
	// empty when ValidateKanon accepts every value.
	check string
	// family names the family of methods through which the type encodes
	// itself without ValidateKanon, one of the words binaryFamily,
	// gobFamily and textFamily, and is empty for a type with none.
	family string
}

// familyWord returns the word that names the family of methods self in the
// docblock of a generated ValidateKanon method.
func familyWord(self selfCodec) string {
	switch self.marshaler {
	case marshalBinaryName:
		return binaryFamily
	case gobEncodeName:
		return gobFamily
	default:
		return textFamily
	}
}

// validateMethod writes the ValidateKanon method of vt: a call of the method
// that vt.check names, or nil when vt accepts every value. Its docblock
// states the encoding that the method selects.
func (e *emitter) validateMethod(vt *valueType) {
	typ := e.p.typ(vt.typ)
	under := types.TypeString(vt.typ.Underlying(), e.cls.relative)
	doc := validateKanonName + " returns nil, since every value of " + typ + " is valid."
	if vt.check != "" {
		doc = validateKanonName + " returns the error of x." + vt.check + "."
	}
	doc += " kanon encodes a value of " + typ + " as a value of its underlying type " + under
	if vt.family != "" {
		doc += ", ahead of the " + vt.family + " methods of " + typ + ","
	}
	e.doc(doc + " and calls " + validateKanonName + " on every value of " + typ + " that it encodes or decodes.")
	if vt.check == "" {
		e.line("func (%s) %s() error {", typ, validateKanonName)
		e.line("return nil")
	} else {
		e.line("func (x %s) %s() error {", typ, validateKanonName)
		e.line("return x.%s()", vt.check)
	}
	e.line("}")
	e.line("")
}

// validPut writes the statements that return the error of ValidateKanon of
// x, a value of v that the encode writes, for the field that loc and num
// locate, when v is a kanon.Validator.
func (e *emitter) validPut(v *value, x, loc, num string) {
	if !v.validate {
		return
	}
	e.line("if err := %s(); err != nil {", method(x, validateKanonName))
	e.fail(e.wire() + "MarshalError(err, " + loc + ", " + num + ")")
	e.line("}")
}

// validFrom writes the statement that records the offset of a value of v
// at p, which [emitter.validRead] reports, when v is a kanon.Validator.
func (e *emitter) validFrom(v *value, p place) {
	if !v.validate {
		return
	}
	e.line("%s := %s", fromName, p.offset())
}

// validRead writes the statements that return the error of ValidateKanon of
// dst, the value of v that the decode read at p, at the offset that
// [emitter.validFrom] recorded, when v is a kanon.Validator.
func (e *emitter) validRead(v *value, dst string, p place) {
	if !v.validate {
		return
	}
	e.line("if err := %s(); err != nil {", method(dst, validateKanonName))
	e.fail(e.wire() + "UnmarshalError(err, " + p.loc + ", " + p.num + ", " + fromName + ")")
	e.line("}")
}

// viewReturn writes the statement of a view method that returns x, the
// value of v that the method read, and for a kanon.Validator first the
// statements that return the error of its ValidateKanon at offset i, the
// offset of the value after its tag. The value takes the local variable x.
func (e *emitter) viewReturn(v *value, x, loc, num string) {
	if v.validate {
		if x != "x" {
			e.line("x := %s", x)
		}
		e.line("if err := x.%s(); err != nil {", validateKanonName)
		e.fail(e.wire() + "UnmarshalError(err, " + loc + ", " + num + ", i)")
		e.line("}")
		x = "x"
	}
	e.line("return %s, nil", x)
}

// valueTypes returns the value types of the named types that are not
// structs and that a -type flag names, in that order, with check, the
// method that -validate names. It fails when check is set and named is
// empty, and for a type that declares ValidateKanon, which kanon generates,
// or that lacks the method check of signature func() error on a value
// receiver.
func (p *pkg) valueTypes(named []*types.Named, check string) ([]*valueType, error) {
	if check != "" && len(named) == 0 {
		return nil, fmt.Errorf("kanon: -validate names %s, and -type names no type that is not a struct", check)
	}
	out := make([]*valueType, 0, len(named))
	for _, t := range named {
		obj := t.Obj()
		fail := func(err error) error {
			return &sourceError{err: err, pos: p.fset.Position(obj.Pos()), subject: obj.Name()}
		}
		if methodsOf(t).signatureOf(validateKanonName) != nil {
			return nil, fail(fmt.Errorf("kanon: %s declares %s, which kanon generates", obj.Name(),
				validateKanonName))
		}
		if check != "" {
			sel := types.NewMethodSet(t).Lookup(obj.Pkg(), check)
			if sel == nil || !types.Identical(sel.Type(), signature(nil, []types.Type{errorType()})) {
				return nil, fail(fmt.Errorf(
					"kanon: %s has no method %s() error on a value receiver, which -validate names",
					obj.Name(), check))
			}
		}
		vt := &valueType{typ: t, check: check}
		if self, ok := selfCodecOf(t); ok {
			vt.family = familyWord(self)
		}
		out = append(out, vt)
	}
	return out, nil
}
