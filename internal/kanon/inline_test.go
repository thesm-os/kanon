// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import "testing"

func TestInline(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run(
			"writes the functions of the inline structs that go generate wrote for every fixture",
			func(t *testing.T) {
				t.Parallel()
				goldenDeclarations(
					t,
					codeSuffix,
					func(d declaration) bool { return d.op == opFields || d.op == opMerge },
				)
			},
		)
	})
}
