// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package canonical

import "errors"

// digestLength is the length of a Digest and of its encoding.
const digestLength = 2

// DigestVoid is the Digest ff 00, which has no encoding.
var DigestVoid = Digest{0xff}

// Errors of the methods of Digest.
var (
	// ErrDigestZero is the error of AppendBinary for the zero Digest.
	ErrDigestZero = errors.New("canonical: the zero digest has no encoding")
	// ErrDigestVoid is the error of AppendBinary for DigestVoid.
	ErrDigestVoid = errors.New("canonical: the void digest has no encoding")
	// ErrDigestLength is the error of UnmarshalBinary for data that is neither
	// two bytes long nor three with a last byte of 0.
	ErrDigestLength = errors.New("canonical: digest encoding is not two bytes")
)

// Digest is an array of two bytes that encodes itself as its bytes, through
// AppendBinary and UnmarshalBinary. The zero Digest has no encoding, and its
// decode method decodes two zero bytes to it. The decode method also decodes
// the two bytes followed by a zero byte, which the encode method does not
// write, so that a canonical decode meets bytes that the encode method does
// not write. DigestVoid has no encoding either, so that the encode of a
// field fails, which leaves out the zero Digest. == compares every bit of a
// Digest, so a codec leaves out a field of the zero Digest.
type Digest [digestLength]byte

// AppendBinary appends the bytes of d to b. It fails with ErrDigestZero for
// the zero Digest, and with ErrDigestVoid for DigestVoid.
func (d Digest) AppendBinary(b []byte) ([]byte, error) {
	if d == (Digest{}) {
		return b, ErrDigestZero
	}
	if d == DigestVoid {
		return b, ErrDigestVoid
	}
	return append(b, d[:]...), nil
}

// UnmarshalBinary sets d to the first two bytes of data, which is two bytes
// long, or three with a last byte of 0. It fails with ErrDigestLength for any
// other data.
func (d *Digest) UnmarshalBinary(data []byte) error {
	if len(data) == digestLength+1 && data[digestLength] == 0 {
		data = data[:digestLength]
	}
	if len(data) != digestLength {
		return ErrDigestLength
	}
	*d = Digest(data)
	return nil
}
