// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"encoding/binary"
	"errors"
	"math"
)

// ticketLength is the length of the encoding of a Ticket other than the
// zero one.
const ticketLength = 4

// TicketVoid is the Ticket whose encoding fails: the largest one.
const TicketVoid Ticket = math.MaxUint32

// Errors of the methods of Ticket.
var (
	// ErrTicketVoid is the error of GobEncode for TicketVoid.
	ErrTicketVoid = errors.New("codec: ticket is void")
	// ErrTicketLength is the error of GobDecode for data that is neither
	// empty nor four bytes long.
	ErrTicketLength = errors.New("codec: ticket encoding is not four bytes")
)

// Ticket encodes itself as four big-endian bytes, through GobEncode and
// GobDecode. The zero Ticket encodes to no bytes.
type Ticket uint32

// GobEncode returns the four big-endian bytes of t, and no bytes for the
// zero Ticket. It fails with ErrTicketVoid for TicketVoid.
func (t Ticket) GobEncode() ([]byte, error) {
	if t == TicketVoid {
		return nil, ErrTicketVoid
	}
	if t == 0 {
		return nil, nil
	}
	return binary.BigEndian.AppendUint32(nil, uint32(t)), nil
}

// GobDecode sets t to the four big-endian bytes in data, and to the zero
// Ticket for empty data. It fails with ErrTicketLength for data of any
// other length.
func (t *Ticket) GobDecode(data []byte) error {
	if len(data) == 0 {
		*t = 0
		return nil
	}
	if len(data) != ticketLength {
		return ErrTicketLength
	}
	*t = Ticket(binary.BigEndian.Uint32(data))
	return nil
}
