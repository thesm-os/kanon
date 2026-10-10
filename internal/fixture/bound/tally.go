// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package bound

import (
	"encoding/binary"
	"errors"
	"maps"
	"sync"
)

//go:generate go tool kanon -type=Tallies,TalliesWide

// tallyLength is the length of the encoding of a Tally in bytes.
const tallyLength = 4

// Errors of the methods of Tally.
var (
	// ErrTallyVoid is the error of AppendBinary for TallyVoid.
	ErrTallyVoid = errors.New("bound: the void tally has no encoding")
	// ErrTallyLength is the error of UnmarshalBinary for data that is not
	// four bytes long.
	ErrTallyLength = errors.New("bound: tally encoding is not four bytes")
)

// tallied counts the calls of AppendBinary of Tally per value, and talliedMu
// is the mutex of tallied.
var (
	talliedMu sync.Mutex
	tallied   = make(map[Tally]int)
)

// Tally is a number that encodes itself as its four little-endian bytes.
// AppendBinary and UnmarshalBinary encode and decode it. AppendBinary counts
// its calls per value, and Tallied returns the counts. A test compares the
// counts of two conformance suites with them. TallyVoid has no encoding, and
// the encode of a Tally field fails for it.
type Tally uint32

// TallyVoid is the Tally 4294967295. It has no encoding.
const TallyVoid Tally = 4294967295

// AppendBinary appends the four little-endian bytes of t to b, and counts the
// call for t. It fails with ErrTallyVoid for TallyVoid. It is safe for
// concurrent use.
func (t Tally) AppendBinary(b []byte) ([]byte, error) {
	talliedMu.Lock()
	tallied[t]++
	talliedMu.Unlock()
	if t == TallyVoid {
		return b, ErrTallyVoid
	}
	return binary.LittleEndian.AppendUint32(b, uint32(t)), nil
}

// UnmarshalBinary sets t to the number in the four little-endian bytes of
// data. It fails with ErrTallyLength when data is not four bytes long.
func (t *Tally) UnmarshalBinary(data []byte) error {
	if len(data) != tallyLength {
		return ErrTallyLength
	}
	*t = Tally(binary.LittleEndian.Uint32(data))
	return nil
}

// Tallies is a slice of Tally with the bound 2.
type Tallies struct {
	List []Tally `kanon:",max=2"`
}

// TalliesWide is the slice of Tallies with the bound 64.
type TalliesWide struct {
	List []Tally `kanon:",max=64"`
}

// Tallied returns a copy of the counts that AppendBinary keeps per value of
// Tally. The counts start with the process. It is safe for concurrent use.
func Tallied() map[Tally]int {
	talliedMu.Lock()
	defer talliedMu.Unlock()
	return maps.Clone(tallied)
}
