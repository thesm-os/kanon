// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import "errors"

// portDynamic is the first port of the dynamic range, which IANA leaves
// unregistered.
const portDynamic = 49152

// Errors of Port.valid.
var (
	// ErrPortZero is the error of Port.valid for port 0, which TCP reserves.
	ErrPortZero = errors.New("validate: port 0 is reserved")
	// ErrPortDynamic is the error of Port.valid for a port of the dynamic
	// range, from 49152.
	ErrPortDynamic = errors.New("validate: port in the dynamic range")
)

// Port is a registered TCP port number, from 1 to 49151. Its ValidateKanon
// rejects the zero value, which the encoding of a field leaves out, and the
// ports of the dynamic range, which a field encodes.
type Port uint16

// valid returns ErrPortZero for port 0, and ErrPortDynamic for a port from
// 49152.
func (p Port) valid() error {
	if p == 0 {
		return ErrPortZero
	}
	if p >= portDynamic {
		return ErrPortDynamic
	}
	return nil
}
