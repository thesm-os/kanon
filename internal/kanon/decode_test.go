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
	})
}
