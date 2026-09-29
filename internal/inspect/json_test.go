// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect_test

import (
	"bytes"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.thesmos.sh/kanon/internal/inspect"
)

// readings is a struct encoding with a field of every reading: a bytes value
// with a struct reading that is also text, a varint, a fixed64 whose float
// is NaN, a fixed32, an empty bytes value, a run of varints, bytes that are
// neither text nor a run of varints, and a fixed64 whose float is 1.5.
const readings = "0a 02 68 69 10 0e 21 ffffffffffffffff 2d 00 00 c0 3f 1a 00 12 03 02 04 06 22 01 ff " +
	"31 000000000000f83f"

// readingsGolden is the golden file of the JSON of readings.
const readingsGolden = "readings.json"

func TestJSON(t *testing.T) {
	t.Parallel()
	t.Run("WriteJSON", func(t *testing.T) {
		t.Parallel()
		t.Run("writes every reading of each field", func(t *testing.T) {
			t.Parallel()
			fields, err := inspect.Parse(unhex(t, readings), depth)
			assert.NoError(t, err, "Parse reads the input")
			var b bytes.Buffer
			assert.NoError(t, inspect.WriteJSON(&b, fields), "WriteJSON writes the fields")
			golden.Match(t, readingsGolden, b.Bytes(), golden.ShouldUpdate())
		})
		t.Run("writes an empty array for no fields", func(t *testing.T) {
			t.Parallel()
			var b bytes.Buffer
			assert.NoError(t, inspect.WriteJSON(&b, nil), "WriteJSON writes no fields")
			assert.Equal(t, b.String(), "{\n  \"fields\": []\n}\n", "WriteJSON writes an empty array")
		})
		t.Run("returns the error of the writer", func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, inspect.WriteJSON(failingWriter{}, nil), errWrite, "WriteJSON fails as the write")
		})
	})
}
