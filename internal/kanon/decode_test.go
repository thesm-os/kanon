// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import "testing"

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
	})
}
