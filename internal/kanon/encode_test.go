// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

func TestEncode(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the encode methods and functions that go generate wrote for every fixture", func(t *testing.T) {
			t.Parallel()
			goldenDeclarations(t, codeSuffix, func(d declaration) bool {
				switch d.method {
				case encodeKanon, encodeInner, appendBinary, marshalBinary:
					return true
				default:
					return d.op == opPut || d.op == opExactPut
				}
			})
		})
		files, err := generate(t, module(t, map[string]string{source: exactStructs}), source, "A,B,C")
		assert.NoError(t, err, "Generate generates the struct types with values of a kanon.Exact type")
		writes := []struct {
			name string
			want string
		}{
			{
				name: "writes a field of a kanon.Exact type with a put function without an error",
				want: `w := _a_exactputTok(buf[:i], &m.Mark, "A.Mark", 1)`,
			},
			{
				name: "writes the put function of a kanon.Exact type with the check of wire.MustExact",
				want: "func _a_exactputTok(buf []byte, x *Tok, loc string, num int) int {\n\tn := x.SizeKanon()\n" +
					"\ti := len(buf) - n\n\tenc, err := x.AppendBinary(buf[i:i:len(buf)])\n" +
					"\twire.MustExact(enc, err, n, loc, num)\n\treturn n\n}",
			},
			{
				name: "writes an element of a kanon.Exact type with the error of its put function",
				want: "w, err := _a_putTok(buf[:i], &x[k], loc, num)",
			},
			{
				name: "writes a union member of a kanon.Exact type with the error of its put function",
				want: `w, err := _a_putTok(buf[:i], &m.One, "C.One", 1)`,
			},
		}
		for _, tt := range writes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, files[codeName], tt.want, "the code file writes the value")
			})
		}
	})
}
