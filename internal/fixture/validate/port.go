// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package validate

import "errors"

// ErrPortZero is the error of Port.valid for port 0, which TCP reserves.
var ErrPortZero = errors.New("validate: port 0 is reserved")

// Port is a TCP port number other than 0. Its ValidateKanon rejects the zero
// value, which the encoding of a field leaves out.
type Port uint16

// valid returns ErrPortZero for port 0.
func (p Port) valid() error {
	if p == 0 {
		return ErrPortZero
	}
	return nil
}
