// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"strings"
	"testing"

	"go.dokimi.dev/assert"
)

// boundStructs is the source of the struct types A, B and E. A has a slice
// with a bound, an unbounded slice of its type, a slice of slices of that
// type, an unbounded slice of strings, maps with the bounds 3, 16 and 17, and
// a map with the bound 2 whose type an element of another field shares. B has
// a slice of A and a streamed slice of E with the bound 2.
const boundStructs = "type A struct {\n" +
	"\tX []int32 `kanon:\",max=3\"`\n" +
	"\tY []int32\n" +
	"\tZ [][]int32\n" +
	"\tM map[string]int32 `kanon:\",max=3\"`\n" +
	"\tS []string\n" +
	"\tN map[string]int64 `kanon:\",max=16\"`\n" +
	"\tO map[string]uint8 `kanon:\",max=17\"`\n" +
	"\tQ map[int32]int32 `kanon:\",max=2\"`\n" +
	"\tR []map[int32]int32\n" +
	"}\n\n" +
	"type B struct {\n\tAs []A\n\tData []E `kanon:\",stream,max=2\"`\n}\n\n" +
	"type E struct {\n\tName string\n}\n"

// selfSlice is the source of T, a named slice type that encodes itself,
// before the struct type A of a field of it with the tag option max on line
// 10 of a.go.
const selfSlice = "type T []int32\n\n" +
	"func (t T) MarshalBinary() ([]byte, error) { return nil, nil }\n\n" +
	"func (t *T) UnmarshalBinary([]byte) error { return nil }\n\n"

func TestBound(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		files, err := generate(t, module(t, map[string]string{source: boundStructs}), source, "A,B,E")
		assert.NoError(t, err, "Generate generates the struct types with bounded fields")
		writes := []struct {
			name string
			file string
			want string
		}{
			{
				name: "passes the bound of a field to the read function of its type",
				file: codeName,
				want: `if err := _a_readSliceInt32(&m.X, data[i:i+int(l)], slab, off+i, depth-1, 3, "A.X", 1); ` +
					"err != nil {",
			},
			{
				name: "passes math.MaxInt for a field without a bound whose read function takes one",
				file: codeName,
				want: `if err := _a_readSliceInt32(&m.Y, data[i:i+int(l)], slab, off+i, depth-1, math.MaxInt, "A.Y", ` +
					"2); err != nil {",
			},
			{
				name: "passes math.MaxInt for an element whose read function takes a bound",
				file: codeName,
				want: "if err := _a_readSliceInt32(&x[last], data[i:i+int(l)], slab, off+i, depth-1, math.MaxInt, " +
					"loc, num); err != nil {",
			},
			{
				name: "passes the bound of a map field before the collect flag",
				file: codeName,
				want: `if err := _a_readMapStringInt32(&m.M, data[i:i+int(l)], slab, off+i, depth-1, 3, ` +
					`seen[0]&(1<<0) == 0, "A.M", 4); err != nil {`,
			},
			{
				name: "writes the bound after the depth among the parameters of a read function that a field " +
					"with a bound calls",
				file: codeName,
				want: "func _a_readSliceInt32(dst *[]int32, data []byte, slab string, off, depth, bound int, " +
					"loc string, num int) error {",
			},
			{
				name: "writes no bound among the parameters of a read function that no field with a bound calls",
				file: codeName,
				want: "func _a_readSliceString(dst *[]string, data []byte, slab string, off, depth int, loc string, " +
					"num int) error {",
			},
			{
				name: "caps the first allocation of a slice with a bound at the bound",
				file: codeName,
				want: "x = make([]int32, 0, min(wire.SliceCap[int32](wire.CountVarints(data)), bound))",
			},
			{
				name: "caps the first allocation of a slice without a bound at 10 MiB",
				file: codeName,
				want: "x = make([]string, 0, wire.SliceCap[string](wire.CountValues(data)))",
			},
			{
				name: "fails at an element of a slice that has its bound of elements",
				file: codeName,
				want: "\tfor i := 0; i < len(data); {\n\t\tif len(x) >= bound {\n" +
					"\t\t\treturn wire.MaxError(loc, num, off+i, bound)\n\t\t}\n",
			},
			{
				name: "fails at a new key of a map that has its bound of entries",
				file: codeName,
				want: "\t\tif len(x) >= bound {\n\t\t\tif _, ok := x[mk]; !ok {\n" +
					"\t\t\t\treturn wire.MaxError(loc, num, off+at, bound)\n\t\t\t}\n\t\t}\n",
			},
			{
				name: "fails the encode of a slice with more elements than its bound before it writes the slice",
				file: codeName,
				want: "\tif len(m.X) > 0 {\n\t\tif len(m.X) > 3 {\n" +
					"\t\t\treturn 0, wire.MarshalError(kanon.ErrMax, \"A.X\", 1)\n\t\t}\n",
			},
			{
				name: "returns the error of the encode of a slice of a struct with a field with a bound",
				file: codeName,
				want: "func _a_putSliceA(buf []byte, x []A, loc string, num int) (int, error) {",
			},
			{
				name: "states the bound of a streamed slice in the schema of the stream decoder",
				file: codeName,
				want: `{Tag: 2<<3 | wire.Bytes, Loc: "B.Data", Elem: "E", Max: 2},`,
			},
			{
				name: "states the bound of a field in the Spec of the test file",
				file: testName,
				want: `{Name: "X", Number: 1, Max: 3},`,
			},
		}
		for _, tt := range writes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Contains(t, files[tt.file], tt.want, "the generated file writes the bound")
			})
		}
		puts := pickDeclarations(t, source, files[codeName], func(d declaration) bool { return d.op == opPut })
		sorts := []struct {
			name string
			put  string
			want bool
		}{
			{name: "sorts a map with a bound of 16 in a stack array alone", put: "_a_putMapStringInt64"},
			{
				name: "sorts a map with a bound of 17 in a slice past 16 entries",
				put:  "_a_putMapStringUint8",
				want: true,
			},
			{
				name: "sorts a map with a bound whose type a field without a bound shares in a slice past 16 entries",
				put:  "_a_putMapInt32Int32",
				want: true,
			},
		}
		for _, tt := range sorts {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				put, ok := puts[tt.put]
				assert.True(t, ok, "the code file declares the put function of the map")
				assert.Equal(t, strings.Contains(put, "if len(x) <= 16 {"), tt.want,
					"the put function sorts a map of more than 16 entries in a slice")
			})
		}
		failures := []struct {
			name  string
			files map[string]string
			want  string
		}{
			{
				name:  "returns an error for the tag option max on an integer",
				files: map[string]string{source: structA("X int32 `kanon:\",max=2\"`")},
				want:  "kanon: a.go:4:2: A.X: " + boundKind,
			},
			{
				name:  "returns an error for the tag option max on a string",
				files: map[string]string{source: structA("X string `kanon:\",max=2\"`")},
				want:  "kanon: a.go:4:2: A.X: " + boundKind,
			},
			{
				name:  "returns an error for the tag option max on a byte slice",
				files: map[string]string{source: structA("X []byte `kanon:\",max=2\"`")},
				want:  "kanon: a.go:4:2: A.X: " + boundKind,
			},
			{
				name:  "returns an error for the tag option max on an array",
				files: map[string]string{source: structA("X [2]int32 `kanon:\",max=2\"`")},
				want:  "kanon: a.go:4:2: A.X: " + boundKind,
			},
			{
				name:  "returns an error for the tag option max on a pointer to a slice",
				files: map[string]string{source: structA("X *[]int32 `kanon:\",max=2\"`")},
				want:  "kanon: a.go:4:2: A.X: " + boundKind,
			},
			{
				name:  "returns an error for the tag option max on a slice type that encodes itself",
				files: map[string]string{source: selfSlice + structA("X T `kanon:\",max=2\"`")},
				want:  "kanon: a.go:10:2: A.X: " + boundKind,
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				assert.Equal(t, generateError(t, tt.files), tt.want, "Generate states why the bound fails")
			})
		}
	})
}

// boundKind is the reason of the error of the tag option max on a field that
// is not a slice or a map.
const boundKind = "the tag option max applies to a slice or a map, other than a byte slice and a type that " +
	"encodes itself"
