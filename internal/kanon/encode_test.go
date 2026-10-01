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
				name: "writes the put function of a kanon.Exact type in another position with the error of the " +
					"append method and the check of wire.MustExact",
				want: "func _a_putTok(buf []byte, x *Tok, loc string, num int) (int, error) {\n\tn := x.SizeKanon()\n" +
					"\ti := len(buf) - n\n\tenc, err := x.AppendBinary(buf[i:i:len(buf)])\n\tif err != nil {\n" +
					"\t\treturn 0, wire.MarshalError(err, loc, num)\n\t}\n\twire.MustExact(enc, nil, n, loc, num)\n" +
					"\treturn n, nil\n}",
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
		t.Run("writes no check of the room or the length for a value of a kanon.Exact type", func(t *testing.T) {
			t.Parallel()
			assert.NotContains(t, files[codeName], "SizeError",
				"kanon.Exact rules out a SizeKanon below 0 and an encoding of another length")
		})
	})
}
