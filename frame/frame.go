// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package frame

import (
	"encoding/binary"
	"hash/crc32"
	"reflect"

	"go.thesmos.sh/kanon"
)

// These constants give the values and the lengths of the frame layout, in
// which the version byte, the flags byte and the type ID precede the
// payload and the optional checksum.
const (
	// version is the value of the version byte of this layout.
	version = 1
	// flagChecksum is the bit of the flags byte that marks a checksum.
	flagChecksum = 1
	// fixedHeader is the length in bytes of the version and the flags.
	fixedHeader = 2
	// checksumSize is the length in bytes of the checksum.
	checksumSize = 4
	// maxVarint is the length in bytes of the longest uvarint.
	maxVarint = binary.MaxVarintLen64
)

// castagnoli is the table of CRC-32C, the checksum of the Castagnoli
// polynomial.
var castagnoli = crc32.MakeTable(crc32.Castagnoli)

// checksum returns the CRC-32C of b.
func checksum(b []byte) uint32 {
	return crc32.Checksum(b, castagnoli)
}

// isNil reports whether m is nil or stores a nil pointer, which a frame
// cannot encode and a decode cannot fill.
func isNil(m kanon.Message) bool {
	if m == nil {
		return true
	}
	v := reflect.ValueOf(m)
	return v.Kind() == reflect.Pointer && v.IsNil()
}
