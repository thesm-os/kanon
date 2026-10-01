// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

// canonicalStructs is the source of the struct types A and B, whose fields
// cover every check that a canonical decode makes: every kind of field, a
// field number above 15, a union of two members, maps with every kind of key
// that a float or a pointer makes, a key type of one value, and types that
// encode themselves through an append method, through an encode method alone,
// and with an IsZero method. E is a struct without fields.
const canonicalStructs = "import \"time\"\n\n" +
	"type Mode uint8\n\nconst (\n\tModeWord Mode = 1\n\tModeCode Mode = 2\n)\n\n" +
	"type Tok [2]byte\n\n" +
	"func (t Tok) AppendBinary(b []byte) ([]byte, error) { return append(b, t[:]...), nil }\n\n" +
	"func (t *Tok) UnmarshalBinary(d []byte) error { copy(t[:], d); return nil }\n\n" +
	"type Note [1]byte\n\n" +
	"func (n Note) MarshalBinary() ([]byte, error) { return n[:], nil }\n\n" +
	"func (n *Note) UnmarshalBinary(d []byte) error { copy(n[:], d); return nil }\n\n" +
	"type Seal struct{ n uint8 }\n\n" +
	"func (s Seal) AppendBinary(b []byte) ([]byte, error) { return append(b, s.n), nil }\n\n" +
	"func (s *Seal) UnmarshalBinary(d []byte) error { s.n = d[0]; return nil }\n\n" +
	"func (s Seal) IsZero() bool { return s.n == 0 }\n\n" +
	"type Point struct {\n\tX float64\n\tY int32\n}\n\n" +
	"type Temp float64\n\n" +
	"type A struct {\n\tFlag bool\n\tCount int64\n\tSize int\n\tName string\n\tAt time.Time\n\tKind Mode\n" +
	"\tWord string `kanon:\",union=Kind\"`\n\tCode int32 `kanon:\",union=Kind\"`\n\tTags []string\n" +
	"\tMarks map[float64]string\n\tPoints map[Point]int32\n\tRefs map[*int32]string\n\tGrid [2]int8\n" +
	"\tAny any `kanon:\",types=string\"`\n\tToken Tok\n\tNext *A\n\tZs map[*complex128]bool\n}\n\n" +
	"type B struct {\n\tF32 map[float32]bool\n\tC64 map[complex64]bool\n\tC128 map[complex128]bool\n" +
	"\tNamed map[Temp]bool\n\tPairs map[[2]float64]bool\n\tOne map[struct{}]int32\n\tMemo Note\n\tMark Seal\n}\n\n" +
	"type E struct{}\n"

// generateCanonical returns the files that Generate returns with -canonical
// for the types that types names, from a.go in dir, by name, and its error.
func generateCanonical(t *testing.T, dir, types string) (map[string]string, error) {
	t.Helper()
	files, err := kanon.Generate(dir, source, kanon.Options{Types: strings.Split(types, ","), Canonical: true})
	out := make(map[string]string, len(files))
	for _, f := range files {
		out[f.Name] = string(f.Src)
	}
	return out, err
}

func TestCanonical(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		files, err := generateCanonical(t, module(t, map[string]string{source: canonicalStructs}), "A,B,E")
		assert.NoError(t, err, "Generate generates the canonical struct types")
		checks := []struct {
			name string
			want string
		}{
			{name: "declares the number of the last field that the decode read", want: "var prior uint64"},
			{name: "declares the bitmap of the unions", want: "var members [1]uint64"},
			{
				name: "matches the tag byte of a field at the start of the remaining data",
				want: "if i < len(data) && data[i] == 1<<3|wire.Varint {",
			},
			{
				name: "matches the tag bytes of a field number above 15",
				want: `if len(data)-i >= 2 && string(data[i:i+2]) == "\x82\x01" {`,
			},
			{name: "records the number of a field whose tag matches", want: "prior = 16"},
			{
				name: "returns the error of the tag function for a byte that begins no field after the fields before it",
				want: "return seen, _a_tagA(data, i, prior, off)",
			},
			{
				name: "returns ErrNotCanonical for a tag that is not in its shortest form",
				want: `return wire.LongFormError(n, "A", 0, off+i)`,
			},
			{
				name: "returns ErrNotCanonical for a field number that is not above the one before it",
				want: `return wire.OrderError(tag>>3, prior, "A", 0, off+i)`,
			},
			{
				name: "returns ErrMalformed for field number 0",
				want: `return wire.TagError(tag, "A", 0, off+i)`,
			},
			{
				name: "returns ErrMalformed for a field in another wire format",
				want: `return wire.FormatError(tag, wire.Varint, "A.Flag", off+i)`,
			},
			{
				name: "returns ErrNotCanonical for an unknown field",
				want: `return wire.UnknownFieldError(tag>>3, "A", 0, off+i)`,
			},
			{
				name: "returns ErrNotCanonical for a second member of a union",
				want: `return seen, wire.MemberError("Kind", "A.Code", 7, off+at)`,
			},
			{name: "records the member of a union", want: "members[0] |= 1 << 0"},
			{
				name: "declares the tag function of a struct without fields without a field before the tag",
				want: "func _a_tagE(data []byte, i, off int) error {",
			},
			{
				name: "returns ErrNotCanonical for a bool above 1",
				want: `return seen, wire.BoolError(u, "A.Flag", 1, off+i)`,
			},
			{name: "tests the presence of a bool field", want: "if !m.Flag {"},
			{
				name: "returns ErrNotCanonical for a bool field of false",
				want: `return seen, wire.AbsentError("A.Flag", 1, off+at)`,
			},
			{name: "tests the presence of an integer field", want: "if m.Count == 0 {"},
			{
				name: "returns ErrNotCanonical for a varint of an int that is not in its shortest form",
				want: `return seen, wire.LongFormError(n, "A.Size", 3, off+i)`,
			},
			{
				name: "decodes a time through wire.CanonicalTime",
				want: `t, err := wire.CanonicalTime(data[i:i+int(l)], "A.At", 5, off+i)`,
			},
			{
				name: "tests the presence of a time field",
				want: "if m.At.IsZero() && m.At.Location() == time.UTC {",
			},
			{
				name: "returns ErrNotCanonical for a slice field of no elements",
				want: `return seen, wire.AbsentError("A.Tags", 8, off+at)`,
			},
			{
				name: "returns ErrNotCanonical for a length that is not in its shortest form",
				want: `return seen, wire.LongFormError(n, "A.Tags", 8, off+i)`,
			},
			{name: "tests the presence of an array field", want: "if m.Grid == ([2]int8{}) {"},
			{name: "tests the presence of an interface field", want: "if m.Any == nil {"},
			{
				name: "returns ErrNotCanonical for an interface type number that is not in its shortest form",
				want: "return 0, wire.LongFormError(i, loc, num, off)",
			},
			{
				name: "encodes a value of a type with an append method again",
				want: "enc, err := m.Token.AppendBinary(scratch[:0])",
			},
			{
				name: "encodes a value of a type with an encode method alone again",
				want: "enc, err := m.Memo.MarshalBinary()",
			},
			{
				name: "returns ErrNotCanonical for bytes that the encode method does not write",
				want: `return seen, wire.EncodingError("A.Token", 14, off+i)`,
			},
			{name: "tests the presence of a value of a type that encodes itself", want: "if m.Token == (Tok{}) {"},
			{name: "tests the presence of a value of a type with an IsZero method", want: "if m.Mark.IsZero() {"},
			{name: "declares the key before the current one", want: "var pk float64"},
			{name: "finds a float64 key of -0.0", want: "if mk == 0 && math.Signbit(mk) {"},
			{name: "finds a float32 key of -0.0", want: "if mk == 0 && math.Signbit(float64(mk)) {"},
			{
				name: "finds a complex64 key with a part of -0.0",
				want: "if real(mk) == 0 && math.Signbit(float64(real(mk))) || imag(mk) == 0 && " +
					"math.Signbit(float64(imag(mk))) {",
			},
			{
				name: "finds a complex128 key with a part of -0.0",
				want: "if real(mk) == 0 && math.Signbit(real(mk)) || imag(mk) == 0 && math.Signbit(imag(mk)) {",
			},
			{
				name: "returns ErrNotCanonical for a map key of -0.0",
				want: "return wire.NegativeZeroError(loc, num, off+at)",
			},
			{name: "finds a struct key with a field of -0.0", want: "return x.X == 0 && math.Signbit(x.X)"},
			{name: "finds an array key with an element of -0.0", want: "if x[k] == 0 && math.Signbit(x[k]) {"},
			{
				name: "finds a pointer key to a complex number with a part of -0.0",
				want: "return x != nil && (real(*x) == 0 && math.Signbit(real(*x)) || imag(*x) == 0 && " +
					"math.Signbit(imag(*x)))",
			},
			{name: "orders a key after the key before it", want: "if at > 0 && cmp.Compare(pk, mk) >= 0 {"},
			{name: "records the key", want: "pk = mk"},
			{
				name: "returns ErrNotCanonical for a key that is not above the key before it",
				want: "return wire.KeyOrderError(loc, num, off+at)",
			},
			{
				name: "returns ErrNotCanonical for the second key of a key type of one value",
				want: "if at > 0 {\n\t\t\treturn wire.KeyOrderError(loc, num, off+at)",
			},
			{
				name: "collects the keys of one projection in a merge alone",
				want: "if !collect && (all || !_a_canonPtrInt32(mk)) {",
			},
			{
				name: "states the canonical rule in the docblock of DecodeKanon",
				want: "kanon.ErrNotCanonical for any",
			},
		}
		for _, tt := range checks {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, files[codeName], tt.want, "the code file writes the check")
			})
		}
		t.Run("checks no member before the first member of a union", func(t *testing.T) {
			t.Parallel()
			assert.False(t, strings.Contains(files[codeName], `MemberError("Kind", "A.Word"`),
				"Word, the first member of the union of Kind, follows no member of it")
		})
		t.Run("records no member after the last member of a union", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, strings.Count(files[codeName], "members[0] |= 1 << 0"), 1,
				"Word records the union of Kind, and Code, its last member, records nothing")
		})
		t.Run("checks no order in the tag function of a struct without fields", func(t *testing.T) {
			t.Parallel()
			start := strings.Index(files[codeName], "func _a_tagE(")
			end := start + strings.Index(files[codeName][start:], "\n}\n")
			assert.False(t, strings.Contains(files[codeName][start:end], "OrderError"),
				"a struct without fields reads no field before a tag")
		})
		t.Run("keeps no key before the current one for a key type of one value", func(t *testing.T) {
			t.Parallel()
			assert.False(t, strings.Contains(files[codeName], "var pk struct{}"),
				"the read function of map[struct{}]int32 declares no pk, which it would never read")
		})
		t.Run("writes no negated presence condition", func(t *testing.T) {
			t.Parallel()
			assert.False(t, strings.Contains(files[codeName], "if !("),
				"the code file tests the absence of every field with its absence condition")
			for _, g := range generations(t) {
				if !g.opts.Canonical {
					continue
				}
				for _, f := range g.files {
					assert.False(t, strings.Contains(string(f.Src), "if !("),
						filepath.Join(g.dir, f.Name)+": the file writes no negated presence condition")
				}
			}
		})
		t.Run("marks the Spec of every canonical struct type", func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, strings.Count(files["a.kanon_test.go"], "Canonical: true,"), 3,
				"the Specs of A, B and E are canonical")
		})
		t.Run("writes no canonical check into a code file without the flag", func(t *testing.T) {
			t.Parallel()
			for _, g := range generations(t) {
				if g.opts.Canonical {
					continue
				}
				for _, f := range g.files {
					assert.False(t, strings.Contains(string(f.Src), "LongFormError"),
						filepath.Join(g.dir, f.Name)+": the file checks no shortest form")
					assert.False(t, strings.Contains(string(f.Src), "Canonical: true"),
						filepath.Join(g.dir, f.Name)+": the Spec is not canonical")
				}
			}
		})
		passes := []struct {
			name  string
			files map[string]string
		}{
			{
				name: "generates a canonical struct type that contains a canonical struct type of the package",
				files: map[string]string{
					"b.go": "//go:generate go tool kanon -type=B -canonical\n\ntype B struct {\n\tX int32\n}\n",
					source: structA("F B"),
				},
			},
			{
				name: "generates a canonical struct type that contains a canonical struct type of a dependency",
				files: map[string]string{
					"dep/dep.go": "//go:generate go tool kanon -type=B -canonical\n\ntype B struct {\n\tX int32\n}\n",
					source:       "import \"example.com/m/dep\"\n\n" + structA("F dep.B"),
				},
			},
			{
				name:  "generates a canonical struct type that contains itself without a directive in its file",
				files: map[string]string{source: structA("Next *A")},
			},
		}
		for _, tt := range passes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := generateCanonical(t, module(t, tt.files), "A")
				assert.NoError(t, err, "Generate generates the canonical struct type")
			})
		}
		failures := []struct {
			name  string
			files map[string]string
			types string
			want  string
		}{
			{
				name:  "returns an error for -canonical when -type names no struct type",
				files: map[string]string{source: "type N int32\n"},
				types: "N",
				want:  "kanon: -canonical is set, and -type names no struct type",
			},
			{
				name:  "returns an error for a canonical struct type with a field that keeps unknown fields",
				files: map[string]string{source: structA("X int32", "Rest []byte `kanon:\",unknown\"`")},
				types: "A",
				want: "kanon: a.go:5:2: A.Rest: -canonical rejects every unknown field, and the field tagged " +
					"unknown would always be empty: remove the tag or the flag",
			},
			{
				name: "returns an error for an inline struct of a canonical struct type with a field that keeps " +
					"unknown fields",
				files: map[string]string{
					source: "type L struct {\n\tRest []byte `kanon:\",unknown\"`\n}\n\n" + structA("F L"),
				},
				types: "A",
				want: "kanon: a.go:4:2: L.Rest: -canonical rejects every unknown field, and the field tagged " +
					"unknown would always be empty: remove the tag or the flag",
			},
			{
				name: "returns an error for a struct type of the package whose directive does not set -canonical",
				files: map[string]string{
					"b.go": "//go:generate go tool kanon -type=B\n\ntype B struct {\n\tX int32\n}\n",
					source: structA("F B"),
				},
				types: "A",
				want: "kanon: a.go:4:2: A.F: example.com/m.B decodes without the -canonical flag, which a canonical " +
					"type cannot contain: add the flag to the directive that names B",
			},
			{
				name: "returns an error for a struct type of a dependency whose directive does not set -canonical",
				files: map[string]string{
					"dep/dep.go": "//go:generate go tool kanon -type=B\n\ntype B struct {\n\tX int32\n}\n",
					source:       "import \"example.com/m/dep\"\n\n" + structA("F dep.B"),
				},
				types: "A",
				want: "kanon: a.go:6:2: A.F: example.com/m/dep.B decodes without the -canonical flag, which a " +
					"canonical type cannot contain: add the flag to the directive that names B",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := generateCanonical(t, module(t, tt.files), tt.types)
				assert.HasError(t, err, "Generate fails")
				assert.Equal(t, err.Error(), tt.want, "Generate states why it fails")
			})
		}
		exact, exactErr := generateCanonical(t, module(t, map[string]string{source: exactStructs}), "A,B,C")
		assert.NoError(t, exactErr, "Generate generates the canonical struct types with values of a kanon.Exact type")
		exactChecks := []struct {
			name string
			want string
		}{
			{
				name: "decodes a field of a kanon.Exact type with its presence in one statement",
				want: "if err := m.Mark.UnmarshalBinary(data[i : i+int(l)]); err != nil || m.Mark == (Tok{}) {",
			},
			{
				name: "returns the error of wire.ExactError for a field of a kanon.Exact type",
				want: `return wire.ExactError(err, "A.Mark", 1, off+at, off+i)`,
			},
			{
				name: "decodes an element of a kanon.Exact type with the error of its decode method",
				want: "if err := x[last].UnmarshalBinary(data[i : i+int(l)]); err != nil {\n" +
					"\t\t\treturn wire.UnmarshalError(err, loc, num, off+i)\n\t\t}\n\t\ti += int(l)",
			},
			{
				name: "decodes a union member of a kanon.Exact type with the error of its decode method",
				want: "if err := m.One.UnmarshalBinary(data[i : i+int(l)]); err != nil {\n" +
					"\t\t\treturn wire.UnmarshalError(err, \"C.One\", 1, off+i)\n\t\t}\n\t\ti += int(l)",
			},
		}
		for _, tt := range exactChecks {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, exact[codeName], tt.want, "the code file writes the check")
			})
		}
		t.Run("encodes no value of a kanon.Exact type again", func(t *testing.T) {
			t.Parallel()
			assert.NotContains(t, exact[codeName], "EncodingError",
				"kanon.Exact guarantees that the decode method accepts only the bytes of the append method")
		})
		t.Run("returns an error for a struct type whose kanon codec is written by hand", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{
				"options.go": "// Options takes the place of the options of the runtime.\ntype Options struct{}\n",
				"sub/a.go": "import kanon \"go.thesmos.sh/kanon\"\n\ntype T struct {\n\tX int32\n}\n" +
					kanonMethods("T") + structA("T T"),
			})
			assert.NoError(t, os.WriteFile(filepath.Join(dir, modName), []byte(runtimeMod), fileMode),
				"the go.mod of the runtime path writes")
			_, err := generateCanonical(t, filepath.Join(dir, "sub"), "A")
			assert.HasError(t, err, "Generate fails")
			assert.Contains(t, err.Error(), "go.thesmos.sh/kanon/sub.T has a kanon codec written by hand, which a "+
				"canonical type cannot contain", "Generate states why it fails")
		})
	})
}
