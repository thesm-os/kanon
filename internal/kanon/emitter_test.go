// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"unicode"

	"go.dokimi.dev/assert"
)

// Suffixes of the two files that kanon generates.
const (
	codeSuffix = ".kanon.go"
	testSuffix = ".kanon_test.go"
)

// Names of the methods that a code file declares on each struct type.
const (
	sizeKanon       = "SizeKanon"
	encodeKanon     = "EncodeKanon"
	encodeInner     = "encodeKanon"
	appendBinary    = "AppendBinary"
	marshalBinary   = "MarshalBinary"
	unmarshalBinary = "UnmarshalBinary"
	decodeKanon     = "DecodeKanon"
	mergeKanon      = "MergeKanon"
	decodeInner     = "decodeKanon"
	mergeInner      = "mergeKanon"
	fieldsInner     = "fieldsKanon"
	resetMethod     = "Reset"
	cloneKanon      = "CloneKanon"
	cloneInner      = "cloneKanon"
)

// Operations of the helpers of a code file, the word after its helper
// prefix.
const (
	opSize     = "size"
	opPresent  = "present"
	opPut      = "put"
	opRead     = "read"
	opMerge    = "merge"
	opFields   = "fields"
	opDeselect = "deselect"
	opReset    = "reset"
	opClone    = "clone"
	opCompare  = "compare"
	// The operations of the helpers of map keys, which write their
	// projections and check them.
	opKeySize    = "keysize"
	opKeyPut     = "keyput"
	opKeyPresent = "keypresent"
	opNaN        = "nan"
	opCanon      = "canon"
)

// viewSuffix ends the name of the view type of a struct type.
const viewSuffix = "View"

// declaration is a top-level declaration of a generated file.
type declaration struct {
	// key names the declaration in its file: the receiver type and the name
	// of a method, the name of a function, the first name of a type, a
	// constant or a variable declaration, and the path of an import.
	key string
	// method is the name of a method, and empty for any other declaration.
	method string
	// recv is the receiver type of a method.
	recv string
	// op is the operation of a helper, the lower-case word after the helper
	// prefix of its file, such as size or read, and empty for any other
	// declaration.
	op string
	// doc reports that the declaration has a doc comment.
	doc bool
	// imports reports an import declaration.
	imports bool
	// text is the source of the declaration with its doc comment.
	text string
}

// prefixOf returns the helper prefix of the source file named file: its
// base name between underscores, with an underscore for every character
// that cannot occur in an identifier.
func prefixOf(file string) string {
	base := strings.TrimSuffix(file, goSuffix)
	return "_" + strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '_'
	}, base) + "_"
}

// declarations returns the top-level declarations of the Go source src of
// a file generated from the source file named file, in source order.
func declarations(t *testing.T, file, src string) []declaration {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	assert.NoError(t, err, file+": the generated file parses")
	prefix := prefixOf(file)
	out := make([]declaration, 0, len(f.Decls))
	for _, d := range f.Decls {
		var dc declaration
		var doc *ast.CommentGroup
		switch d := d.(type) {
		case *ast.FuncDecl:
			doc, dc.key = d.Doc, d.Name.Name
			if d.Recv != nil {
				recv := d.Recv.List[0].Type
				if star, ok := recv.(*ast.StarExpr); ok {
					recv = star.X
				}
				ident, _ := recv.(*ast.Ident)
				dc.recv, dc.method = ident.Name, d.Name.Name
				dc.key = dc.recv + "." + dc.method
			} else if rest, ok := strings.CutPrefix(d.Name.Name, prefix); ok {
				if i := strings.IndexFunc(rest, unicode.IsUpper); i > 0 {
					dc.op = rest[:i]
				}
			}
		case *ast.GenDecl:
			doc, dc.imports = d.Doc, d.Tok == token.IMPORT
			switch s := d.Specs[0].(type) {
			case *ast.TypeSpec:
				dc.key = s.Name.Name
			case *ast.ValueSpec:
				dc.key = d.Tok.String() + " " + s.Names[0].Name
			default:
				dc.key = d.Tok.String()
			}
		}
		start := d.Pos()
		if doc != nil {
			start, dc.doc = doc.Pos(), true
		}
		dc.text = src[fset.Position(start).Offset:fset.Position(d.End()).Offset]
		out = append(out, dc)
	}
	return out
}

// pickDeclarations returns the text of each declaration of the file that
// pick reports true for, by key.
func pickDeclarations(t *testing.T, file, src string, pick func(declaration) bool) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, d := range declarations(t, file, src) {
		if pick(d) {
			out[d.key] = d.text
		}
	}
	return out
}

// goldenDeclarations checks that the declarations that pick reports true
// for, in the file of every fixture that ends with suffix, are those that
// go generate wrote, and that pick reports true for at least one of them.
func goldenDeclarations(t *testing.T, suffix string, pick func(declaration) bool) {
	t.Helper()
	picked := 0
	for _, g := range generations(t) {
		for _, f := range g.files {
			if !strings.HasSuffix(f.Name, suffix) {
				continue
			}
			got := pickDeclarations(t, g.file, string(f.Src), pick)
			assert.Equal(t, got, pickDeclarations(t, g.file, golden(t, g.dir, f.Name), pick),
				g.dir+f.Name+": Generate writes the declarations that go generate wrote")
			picked += len(got)
		}
	}
	assert.True(t, picked > 0, "the fixtures have such declarations")
}

func TestEmitter(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes a doc comment on every declaration of every generated file", func(t *testing.T) {
			t.Parallel()
			for _, g := range generations(t) {
				for _, f := range g.files {
					for _, d := range declarations(t, g.file, string(f.Src)) {
						assert.True(t, d.doc || d.imports, f.Name+": "+d.key+" has a doc comment")
					}
				}
			}
		})
	})
}
