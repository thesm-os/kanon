// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package number

//go:generate go tool kanon -type=Private,PrivateKey -views

// Private has unexported fields that a kanon tag opts into the encoding, in
// every role of an encoded field: a number, a fixed-size number, a slice, a
// map whose key struct has an unexported field of its own, a pointer to a
// struct with a kanon codec, an inline struct with an unexported field of
// its own, and the members of a union. The field secret has no tag, so the
// encoding leaves it out. The view types have a method for the exported
// field Name alone.
type Private struct {
	Name   string
	count  int32                 `kanon:""`
	total  int64                 `kanon:",fixed"`
	tags   []string              `kanon:""`
	byKey  map[PrivateKey]string `kanon:""`
	next   *Private              `kanon:""`
	detail privateDetail         `kanon:""`
	Kind   PrivateKind
	text   string `kanon:",union=Kind"`
	number int64  `kanon:",union=Kind"`
	secret string
}

// PrivateKey is a struct with a kanon codec whose unexported field a tag
// opts into the encoding, which a map orders its keys by.
type PrivateKey struct {
	id   int32 `kanon:""`
	Name string
}

// privateDetail is an unexported struct type without a kanon codec, which
// the code file encodes as an inline struct, with an unexported field that
// a tag opts into the encoding.
type privateDetail struct {
	label string `kanon:""`
	Rank  int32
}

// PrivateKind selects the member of the union of Private. The constant of
// an unexported member takes its name with the first letter in upper case.
type PrivateKind uint8

// The members of the union of Private.
const (
	PrivateKindText   PrivateKind = 1
	PrivateKindNumber PrivateKind = 2
)
