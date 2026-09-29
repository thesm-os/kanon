// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"fmt"
	"go/token"
	"go/types"
)

// Names of the methods that decide how a named type encodes: the methods of
// a kanon codec, and the families of methods through which a type encodes
// itself.
const (
	sizeKanonName       = "SizeKanon"
	encodeKanonName     = "EncodeKanon"
	decodeKanonName     = "DecodeKanon"
	mergeKanonName      = "MergeKanon"
	cloneKanonName      = "CloneKanon"
	resetName           = "Reset"
	appendBinaryName    = "AppendBinary"
	marshalBinaryName   = "MarshalBinary"
	unmarshalBinaryName = "UnmarshalBinary"
	gobEncodeName       = "GobEncode"
	gobDecodeName       = "GobDecode"
	appendTextName      = "AppendText"
	marshalTextName     = "MarshalText"
	unmarshalTextName   = "UnmarshalText"
	// validateKanonName names the method of kanon.Validator, through which
	// a named type that is not a struct encodes as its underlying type.
	validateKanonName = "ValidateKanon"
)

// Import path of the runtime package, and the names of its declarations
// that the signatures of kanon codecs and the generated code name.
const (
	runtimePath = "go.thesmos.sh/kanon"
	runtimeName = "kanon"
	optionsName = "Options"
)

// errorName is the name of the predeclared type error.
const errorName = "error"

// selfCodec names the methods through which a type encodes itself: the
// method that appends the encoding to a buffer, which is empty when the
// type has none, the method that returns the encoding, and the method that
// decodes it into the receiver.
type selfCodec struct {
	appender    string
	marshaler   string
	unmarshaler string
	// sizer reports that the type is a kanon.Sizer: its SizeKanon returns
	// the length of the encoding, which the generated code sizes the values
	// of the type with, and appends the encoding into.
	sizer bool
	// zeroer reports that the type declares IsZero() bool, which the
	// generated code tests the presence of a value with in place of ==.
	zeroer bool
}

// selfCodecs returns the families of methods through which a type encodes
// itself, in the order that kanon tries them: the binary methods of package
// encoding; GobEncode and GobDecode of package encoding/gob, which have no
// append method; and the text methods of package encoding. encoding/gob
// tries the same families, with its own first.
func selfCodecs() []selfCodec {
	return []selfCodec{
		{appender: appendBinaryName, marshaler: marshalBinaryName, unmarshaler: unmarshalBinaryName},
		{marshaler: gobEncodeName, unmarshaler: gobDecodeName},
		{appender: appendTextName, marshaler: marshalTextName, unmarshaler: unmarshalTextName},
	}
}

// selfCodecOf returns the methods through which the named type t encodes
// itself, and whether it does: the first family of [selfCodecs] whose
// decode method, and whose append method or encode method, the pointer to t
// has, with the signatures of packages encoding and encoding/gob. The
// appender of the result is empty when t has the encode method of the
// family alone. The result records whether the pointer to t has SizeKanon()
// int and IsZero() bool.
func selfCodecOf(t *types.Named) (selfCodec, bool) {
	s := methodsOf(t)
	encoded := []types.Type{byteSlice(), errorType()}
	for _, c := range selfCodecs() {
		appends := c.appender != "" && s.has(c.appender, []types.Type{byteSlice()}, encoded)
		marshals := s.has(c.marshaler, nil, encoded)
		if !s.has(c.unmarshaler, []types.Type{byteSlice()}, []types.Type{errorType()}) || !appends && !marshals {
			continue
		}
		if !appends {
			c.appender = ""
		}
		c.sizer = s.has(sizeKanonName, nil, []types.Type{types.Typ[types.Int]})
		c.zeroer = s.has(isZeroName, nil, []types.Type{types.Typ[types.Bool]})
		return c, true
	}
	return selfCodec{}, false
}

// methodSet is the method set of the pointer to a named type.
type methodSet struct {
	ms  *types.MethodSet
	pkg *types.Package
}

// methodsOf returns the method set of the pointer to t, which contains the
// methods of t and of *t.
func methodsOf(t *types.Named) methodSet {
	return methodSet{ms: types.NewMethodSet(types.NewPointer(t)), pkg: t.Obj().Pkg()}
}

// signatureOf returns the signature of the method name, or nil when the set
// has no such method.
func (s methodSet) signatureOf(name string) *types.Signature {
	sel := s.ms.Lookup(s.pkg, name)
	if sel == nil {
		return nil
	}
	sig, _ := sel.Type().(*types.Signature)
	return sig
}

// has reports whether the set has the method name with the parameter types
// params and the result types results.
func (s methodSet) has(name string, params, results []types.Type) bool {
	sig := s.signatureOf(name)
	return sig != nil && types.Identical(sig, signature(params, results))
}

// decodes reports whether the set has the method name with the signature of
// DecodeKanon: func([]byte, kanon.Options) error.
func (s methodSet) decodes(name string) bool {
	sig := s.signatureOf(name)
	if sig == nil || sig.Params().Len() != 2 || sig.Results().Len() != 1 {
		return false
	}
	opts, _ := types.Unalias(sig.Params().At(1).Type()).(*types.Named)
	return types.Identical(sig.Params().At(0).Type(), byteSlice()) &&
		opts != nil && opts.Obj().Pkg() != nil && opts.Obj().Pkg().Path() == runtimePath &&
		opts.Obj().Name() == optionsName &&
		types.Identical(sig.Results().At(0).Type(), errorType())
}

// hasKanonMethods reports whether the pointer to t has the methods that the
// generated code of an enclosing struct calls: SizeKanon, EncodeKanon,
// DecodeKanon, MergeKanon and Reset of kanon.Message, and CloneKanon, which
// returns a *t.
func hasKanonMethods(t *types.Named) bool {
	s := methodsOf(t)
	integer := types.Typ[types.Int]
	return s.has(sizeKanonName, nil, []types.Type{integer}) &&
		s.has(encodeKanonName, []types.Type{byteSlice()}, []types.Type{integer, errorType()}) &&
		s.decodes(decodeKanonName) && s.decodes(mergeKanonName) &&
		s.has(resetName, nil, nil) &&
		s.has(cloneKanonName, nil, []types.Type{types.NewPointer(t)})
}

// validatorOf reports whether the named type t is a kanon.Validator, which
// kanon encodes as its underlying type ahead of its binary, gob and text
// methods: t is neither a struct nor an interface, and has ValidateKanon of
// signature func() error on a value receiver. An interface type is no
// Validator whatever its methods, since kanon encodes an interface value by
// its concrete type. validatorOf fails for a struct type with ValidateKanon,
// which applies to types that are not structs, and for a ValidateKanon with
// another signature or with a pointer receiver, which the encode cannot call
// on a map key.
func validatorOf(t *types.Named) (bool, error) {
	if types.IsInterface(t) {
		return false, nil
	}
	sig := methodsOf(t).signatureOf(validateKanonName)
	if sig == nil {
		return false, nil
	}
	name := types.TypeString(t, nil)
	if isStruct(t) {
		return false, fmt.Errorf("kanon: struct type %s declares %s, which applies to types that are not structs",
			name, validateKanonName)
	}
	if !types.Identical(sig, signature(nil, []types.Type{errorType()})) {
		return false, fmt.Errorf("kanon: %s.%s has the signature %s: declare it as func() error",
			name, validateKanonName, types.TypeString(sig, nil))
	}
	if types.NewMethodSet(t).Lookup(t.Obj().Pkg(), validateKanonName) == nil {
		return false, fmt.Errorf("kanon: %s.%s has a pointer receiver: declare it on a value receiver",
			name, validateKanonName)
	}
	return true, nil
}

// signature returns the signature of a function without a receiver, with
// the parameter types params and the result types results.
func signature(params, results []types.Type) *types.Signature {
	vars := func(ts []types.Type) *types.Tuple {
		vs := make([]*types.Var, len(ts))
		for i, t := range ts {
			vs[i] = types.NewParam(token.NoPos, nil, "", t)
		}
		return types.NewTuple(vs...)
	}
	return types.NewSignatureType(nil, nil, nil, vars(params), vars(results), false)
}

// byteSlice returns the type []byte.
func byteSlice() types.Type {
	return types.NewSlice(types.Typ[types.Byte])
}

// errorType returns the predeclared type error.
func errorType() types.Type {
	return types.Universe.Lookup(errorName).Type()
}
