// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package wire_test

import (
	"io"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon"
	"go.thesmos.sh/kanon/wire"
)

// The struct and the offset at which the Skip cases locate their errors.
const (
	skipType = "Order"
	skipOff  = 40
)

// Tags of field 1 with each wire format, valid and invalid, and the tag of
// field 0 with the wire format varint.
const (
	varintTag  = 1 << 3
	fixed64Tag = 1<<3 | 1
	bytesTag   = 1<<3 | 2
	groupTag   = 1<<3 | 3
	endTag     = 1<<3 | 4
	fixed32Tag = 1<<3 | 5
	wire6Tag   = 1<<3 | 6
	wire7Tag   = 1<<3 | 7
	field0Tag  = 0
)

// sinkErr keeps the compiler from dropping the results of benchmarked
// calls.
var sinkErr error

func BenchmarkField(b *testing.B) {
	cases := []struct {
		name string
		tag  uint64
		data []byte
	}{
		{name: "Skip/varint", tag: varintTag, data: []byte{0xac, 0x02}},
		{name: "Skip/fixed64", tag: fixed64Tag, data: make([]byte, 8)},
		{name: "Skip/bytes", tag: bytesTag, data: []byte{0x02, 'a', 'b'}},
		{name: "Skip/fixed32", tag: fixed32Tag, data: make([]byte, 4)},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sinkInt, sinkErr = wire.Skip(c.data, c.tag, skipType, 0, skipOff)
			}
		})
	}
}

func TestField(t *testing.T) {
	t.Parallel()
	t.Run("constants", func(t *testing.T) {
		t.Parallel()
		t.Run("number the wire formats as the format defines them", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, []int{wire.Varint, wire.Fixed64, wire.Bytes, wire.Fixed32}, []int{0, 1, 2, 5},
				"the wire formats are 0, 1, 2 and 5")
		})
	})
	t.Run("Skip", func(t *testing.T) {
		t.Parallel()
		lengths := []struct {
			name string
			tag  uint64
			data []byte
			want int
		}{
			{name: "skips a varint", tag: varintTag, data: []byte{0xac, 0x02, 0xff}, want: 2},
			{name: "skips the last eight bytes", tag: fixed64Tag, data: make([]byte, 8), want: 8},
			{name: "skips a length and its bytes", tag: bytesTag, data: []byte{0x02, 'a', 'b', 'c'}, want: 3},
			{name: "skips an empty value", tag: bytesTag, data: []byte{0x00}, want: 1},
			{name: "skips the last four bytes", tag: fixed32Tag, data: make([]byte, 4), want: 4},
		}
		for _, c := range lengths {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				n, err := wire.Skip(c.data, c.tag, skipType, 0, skipOff)
				assert.NoError(t, err, "Skip reads a complete value")
				assert.Equal(t, n, c.want, "Skip returns the length of the value")
			})
		}
		failures := []struct {
			name   string
			tag    uint64
			data   []byte
			cause  error
			detail string
		}{
			{name: "rejects a varint that ends early", tag: varintTag, data: []byte{0x80}, cause: io.ErrUnexpectedEOF},
			{
				name: "rejects a varint that overflows", tag: varintTag, data: ones(9, 0x02),
				cause: kanon.ErrMalformed, detail: "varint overflows 64 bits",
			},
			{name: "rejects seven of eight bytes", tag: fixed64Tag, data: make([]byte, 7), cause: io.ErrUnexpectedEOF},
			{name: "rejects a length that ends early", tag: bytesTag, data: []byte{0x80}, cause: io.ErrUnexpectedEOF},
			{
				name: "rejects a length that overflows", tag: bytesTag, data: ones(9, 0x02),
				cause: kanon.ErrMalformed, detail: "varint overflows 64 bits",
			},
			{
				name: "rejects a value past the end", tag: bytesTag, data: []byte{0x03, 'a', 'b'},
				cause: io.ErrUnexpectedEOF,
			},
			{name: "rejects three of four bytes", tag: fixed32Tag, data: make([]byte, 3), cause: io.ErrUnexpectedEOF},
			{
				name: "rejects wire format 3", tag: groupTag, data: []byte{0}, cause: kanon.ErrMalformed,
				detail: "field 1 has invalid wire format 3",
			},
			{
				name: "rejects wire format 4", tag: endTag, data: []byte{0}, cause: kanon.ErrMalformed,
				detail: "field 1 has invalid wire format 4",
			},
			{
				name: "rejects wire format 6", tag: wire6Tag, data: []byte{0}, cause: kanon.ErrMalformed,
				detail: "field 1 has invalid wire format 6",
			},
			{
				name: "rejects wire format 7", tag: wire7Tag, data: []byte{0}, cause: kanon.ErrMalformed,
				detail: "field 1 has invalid wire format 7",
			},
			{
				name: "rejects field number 0", tag: field0Tag, data: []byte{0}, cause: kanon.ErrMalformed,
				detail: "field number 0",
			},
		}
		for _, c := range failures {
			t.Run(c.name, func(t *testing.T) {
				t.Parallel()
				n, err := wire.Skip(c.data, c.tag, skipType, 0, skipOff)
				assert.Equal(t, n, 0, "Skip returns no length with an error")
				assert.ErrorIs(t, err, c.cause, "Skip reports the cause")
				want := &kanon.DecodeError{Type: skipType, Offset: skipOff, Detail: c.detail, Err: c.cause}
				assert.Equal(t, assert.ErrorAs[*kanon.DecodeError](t, err, "Skip returns a *kanon.DecodeError"), want,
					"Skip locates the error at the struct and the offset of the tag")
			})
		}
	})
}
