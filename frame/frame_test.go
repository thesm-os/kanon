// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame_test

import (
	"bytes"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/frame"
	"go.thesmos.sh/kanon/internal/fixture/view"
)

// typeID is the type ID of the frames of the tests.
const typeID = 1

// Vectors of the frame layout for type ID 1. The checksums are the CRC-32C
// of the version through the payload, little-endian, from an implementation
// of CRC-32C outside Go that returns the check value 0xe3069283 for
// "123456789".
var (
	// emptyFrame is an empty payload without a checksum.
	emptyFrame = []byte{0x03, 0x01, 0x00, 0x01}
	// itemFrame is the payload 0a 01 61, view.Item{Name: "a"}, without a
	// checksum.
	itemFrame = []byte{0x06, 0x01, 0x00, 0x01, 0x0a, 0x01, 0x61}
	// checkedFrame is an empty payload with its checksum, the CRC-32C of
	// 01 01 01, 0x24ec2a70.
	checkedFrame = []byte{0x07, 0x01, 0x01, 0x01, 0x70, 0x2a, 0xec, 0x24}
	// checkedItemFrame is the payload of itemFrame with its checksum, the
	// CRC-32C of 01 01 01 0a 01 61, 0xaa7f3659.
	checkedItemFrame = []byte{0x0a, 0x01, 0x01, 0x01, 0x0a, 0x01, 0x61, 0x59, 0x36, 0x7f, 0xaa}
)

func TestFrame(t *testing.T) {
	t.Parallel()
	vectors := []struct {
		name     string
		item     view.Item
		checksum bool
		want     []byte
	}{
		{name: "an empty payload without a checksum", want: emptyFrame},
		{name: "a payload without a checksum", item: view.Item{Name: "a"}, want: itemFrame},
		{name: "an empty payload with a checksum", checksum: true, want: checkedFrame},
		{name: "a payload with a checksum", item: view.Item{Name: "a"}, checksum: true, want: checkedItemFrame},
	}
	t.Run("Write", func(t *testing.T) {
		t.Parallel()
		for _, tt := range vectors {
			t.Run("writes the vector of "+tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				w := frame.NewWriter(&out)
				w.Checksum = tt.checksum
				assert.NoError(t, w.Write(typeID, &tt.item), "Write writes the frame")
				assert.Equal(t, out.Bytes(), tt.want, "the frame has the bytes of the vector")
			})
		}
	})
	t.Run("Next", func(t *testing.T) {
		t.Parallel()
		for _, tt := range vectors {
			t.Run("reads the vector of "+tt.name, func(t *testing.T) {
				t.Parallel()
				r := frame.NewReader(bytes.NewReader(tt.want))
				id, _, err := r.Next()
				assert.NoError(t, err, "Next reads the frame")
				assert.Equal(t, id, uint64(typeID), "the frame has the type ID of the vector")
				var got view.Item
				assert.NoError(t, r.Decode(&got), "Decode decodes the payload")
				assert.Equal(t, got, tt.item, "the payload decodes to the message of the vector")
			})
		}
	})
}
