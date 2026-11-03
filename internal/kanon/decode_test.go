// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"testing"

	"go.dokimi.dev/assert"
)

// oneMember is the source of the struct type A with a union of one member,
// Only, whose discriminator is Kind.
const oneMember = "type Choice uint8\n\nconst ChoiceOnly Choice = 1\n\n" +
	"type A struct {\n\tKind Choice\n\tOnly int32 `kanon:\",union=Kind\"`\n}\n"

// stringFields is the source of the struct type A with a string, a named
// string type and an integer, and of the struct type B with an integer
// alone.
const stringFields = "type Label string\n\n" +
	"type A struct {\n\tName string\n\tTag  Label\n\tSize int32\n}\n\n" +
	"type B struct {\n\tSize int32\n}\n"

func TestDecode(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("writes the decode methods and the deselect functions that go generate wrote for every fixture",
			func(t *testing.T) {
				t.Parallel()
				goldenDeclarations(t, codeSuffix, func(d declaration) bool {
					switch d.method {
					case unmarshalBinary, decodeKanon, mergeKanon, decodeInner, mergeInner:
						return true
					default:
						return d.op == opDeselect
					}
				})
			})
		t.Run("sets the discriminator of a union of one member without a deselect function", func(t *testing.T) {
			t.Parallel()
			files, err := generate(t, module(t, map[string]string{source: oneMember}), source, "A")
			assert.NoError(t, err, "Generate generates a struct with a union of one member")
			assert.Contains(t, files[codeName], "m.Kind = ChoiceOnly",
				"the decode of the member sets the discriminator")
			assert.NotContains(t, files[codeName], "if m.Kind != ChoiceOnly",
				"the decode sets the discriminator without a check of its value")
			assert.NotContains(t, files[codeName], "deselect",
				"the code file declares no deselect function, since the union has no other member to zero")
		})
		files, err := generate(t, module(t, map[string]string{source: stringFields}), source, "A,B")
		assert.NoError(t, err, "Generate generates struct types with string fields and without them")
		tests := []struct {
			name string
			want string
		}{
			{
				name: "sets the slab of a decode of a struct with strings with wire.Slab.Reset",
				want: "\tvar slab wire.Slab\n\toff := slab.Reset(data, opts)\n" +
					"\treturn m.decodeKanon(data, &slab, off, opts.Limit())",
			},
			{
				name: "takes the zero wire.Slab for a decode of a struct without strings",
				want: "\tvar slab wire.Slab\n\toff := opts.Offset\n\treturn m.decodeKanon(data, &slab, off, opts.Limit())",
			},
			{
				name: "reads a string through the slab with the string of the field",
				want: "m.Name = slab.String(m.Name, data[i:i+int(l)], off+i)",
			},
			{
				name: "converts a named string type for the slab and back",
				want: "m.Tag = Label(slab.String(string(m.Tag), data[i:i+int(l)], off+i))",
			},
			{
				name: "keeps a string field in the prologue and clears it when the input leaves it out",
				want: "\tm.Size = 0\n\tvar seen [1]uint64\n\terr := m.fieldsKanon(data, slab, off, depth, &seen)\n" +
					"\tif seen[0]&(1<<0) == 0 {\n\t\tm.Name = \"\"\n\t}\n",
			},
			{
				name: "passes the seen bitmap to the loop function by pointer",
				want: "func (m *A) fieldsKanon(data []byte, slab *wire.Slab, off, depth int, seen *[1]uint64) error {",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, files[codeName], tt.want, "the generated file decodes the strings through the slab")
			})
		}
	})
}
