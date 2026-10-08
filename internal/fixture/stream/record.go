// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package stream

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Part,Record,Letter

// Caption is a named string, which streams as a string.
type Caption string

// Blob is a named byte slice, which streams as a byte slice.
type Blob []byte

// RecordKind selects the member of the union of Record.
type RecordKind uint8

// The members of the union of Record.
const (
	RecordKindText   RecordKind = 1
	RecordKindNumber RecordKind = 2
)

// Part is the element type of the streamed slice of Record of this package.
type Part struct {
	Name string
	Size int64
}

// Record is a type that accepts every encoding of a value, with a streamed
// field of each kind: a named string, a named byte slice, and slices of a
// struct with a kanon codec of this package and of another. Its decode
// tracks Head and Meta, its union has a member on each side of the streamed
// fields, and it keeps the unknown fields of a decode.
type Record struct {
	ID     int64
	Note   Caption `kanon:",stream"`
	Head   *Part
	Body   Blob `kanon:",stream"`
	Kind   RecordKind
	Text   string           `kanon:",union=Kind"`
	Parts  []Part           `kanon:",stream"`
	Labels []external.Label `kanon:",stream"`
	Number int64            `kanon:",union=Kind"`
	Meta   map[string]string
	Rest   []byte `kanon:"unknown"`
}

// Letter is a type that accepts every encoding of a value, with a streamed
// string alone and no field that the decode tracks.
type Letter struct {
	Subject string
	Body    string `kanon:",stream"`
}
