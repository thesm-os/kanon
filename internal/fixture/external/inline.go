// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package external

// Remote is a struct of another package without a kanon codec, which the
// code file of a fixture encodes as an inline struct. Its anonymous struct
// field records its numbers under the key of Remote.
type Remote struct {
	Name  string
	Count int32
	Meta  struct {
		Tag  string
		Rank Rank
	}
}

// Sealed is a struct of another package without a kanon codec with an
// unexported field, which the code file of a fixture cannot name. Its reset
// and its decode zero the whole struct, and keep the memory of the encoded
// fields that refer to memory and of the field that keeps unknown fields.
type Sealed struct {
	Items []string
	Next  *int32
	Count int32
	Rest  []byte `kanon:",unknown"`
	state string
}

// Badge is a comparable struct of another package without a kanon codec
// with an unexported field, which a fixture uses as a map key, whose field
// the code file of the fixture cannot read.
type Badge struct {
	Name   string
	serial int32
}

// Pair is a generic struct of another package without a kanon codec. Its
// instantiations share the field numbers that the code file of a fixture
// records for Pair.
type Pair[K comparable, V any] struct {
	Key K
	Val V
}

// Variant is a struct of another package without a kanon codec that has a
// union.
type Variant struct {
	Kind   VariantKind
	Text   string `kanon:",union=Kind"`
	Number int64  `kanon:",union=Kind"`
}

// VariantKind selects the member of the union of Variant.
type VariantKind uint8

// The members of the union of Variant.
const (
	VariantKindText   VariantKind = 1
	VariantKindNumber VariantKind = 2
)

// Choice is the discriminator type of a union that a fixture package
// declares, so that its code file qualifies the constants.
type Choice uint8

// The members of the union of a fixture that Choice selects.
const (
	ChoiceText   Choice = 1
	ChoiceNumber Choice = 2
)
