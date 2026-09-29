// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect_test

import (
	"bytes"
	"strings"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/golden"

	"go.thesmos.sh/kanon/internal/inspect"
)

// Frames of the tests: the test vectors of the frame format for type 1, and
// frames that fail.
const (
	// emptyFrame has an empty payload and no checksum.
	emptyFrame = "03 01 00 01"
	// itemFrame has the payload 0a 01 61 and no checksum.
	itemFrame = "06 01 00 01 0a 01 61"
	// checkedFrame has an empty payload and a checksum.
	checkedFrame = "07 01 01 01 70 2a ec 24"
	// badChecksum is checkedFrame with the last byte of its checksum changed.
	badChecksum = "07 01 01 01 70 2a ec 25"
	// badVersion is emptyFrame with version 2.
	badVersion = "03 02 00 01"
	// badFlags is emptyFrame with flag bit 1 set.
	badFlags = "03 01 02 01"
	// cutPayload is a frame whose payload ends inside the varint of its
	// second field.
	cutPayload = "06 01 00 01 08 01 10"
	// cutFrame declares 3 bytes after its length and has 2.
	cutFrame = "03 01 00"
	// malformedLength is a length of ten bytes with the continuation bit.
	malformedLength = "80 80 80 80 80 80 80 80 80 80"
	// longFrame declares a length of 5 bytes in an input of 4.
	longFrame = "05 01 00 01"
)

// Batches of the tests: the test vectors of the batch format, and batches
// that fail.
const (
	// emptyBatch has no records.
	emptyBatch = "01 00 00 00 00 00"
	// oneEmpty has one empty record.
	oneEmpty = "01 00 00 00 00 00 01 00 00 00"
	// twoEmpty has two empty records.
	twoEmpty = "01 00 00 00 00 00 00 00 00 00 02 00 00 00"
	// oneRecord has one record, 08 02.
	oneRecord = "01 00 08 02 00 00 00 00 01 00 00 00"
	// wideEmpty has one empty record and 8-byte offsets.
	wideEmpty = "01 01 00 00 00 00 00 00 00 00 01 00 00 00"
	// firstOffsetOne has a first offset of 1.
	firstOffsetOne = "01 00 01 00 00 00 01 00 00 00"
	// cutRecord has one record of one byte, 08, which ends inside a field.
	cutRecord = "01 00 08 00 00 00 00 01 00 00 00"
)

// Golden files of the JSON of a stream of frames and of a batch.
const (
	framesGolden = "frames.json"
	batchGolden  = "batch.json"
)

// hexArgs are the arguments of Command for the flag of a layout, in
// hexadecimal digits.
func hexArgs(layout string) []string {
	return []string{layout, hexFlag}
}

func TestContainer(t *testing.T) {
	t.Parallel()
	t.Run("Command", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name       string
			args       []string
			give       []string
			wantCode   int
			wantStdout string
			wantStderr string
		}{
			{
				name:     "writes each frame of a stream with the fields of its payload",
				args:     hexArgs(framesFlag),
				give:     []string{emptyFrame, itemFrame, checkedFrame},
				wantCode: exitOK,
				wantStdout: "# frame 0 at 0: 4 bytes, no checksum, type 1\n# frame 1 at 4: 7 bytes, no checksum, type 1\n" +
					"1: {\"a\"}\n# frame 2 at 11: 8 bytes, checksum, type 1\n",
			},
			{name: "writes nothing for a stream without frames", args: hexArgs(framesFlag), wantCode: exitOK},
			{
				name:     "writes a frame that fails its checksum with its error",
				args:     hexArgs(framesFlag),
				give:     []string{badChecksum, itemFrame},
				wantCode: exitFail,
				wantStdout: "# frame 0 at 0: 8 bytes: frame: checksum mismatch\n" +
					"# frame 1 at 8: 7 bytes, no checksum, type 1\n1: {\"a\"}\n",
			},
			{
				name:     "writes a frame that fails its version with its error",
				args:     hexArgs(framesFlag),
				give:     []string{badVersion, itemFrame},
				wantCode: exitFail,
				wantStdout: "# frame 0 at 0: 4 bytes: frame: unsupported version\n" +
					"# frame 1 at 4: 7 bytes, no checksum, type 1\n1: {\"a\"}\n",
			},
			{
				name:     "writes a frame that fails its flags with its error",
				args:     hexArgs(framesFlag),
				give:     []string{badFlags, itemFrame},
				wantCode: exitFail,
				wantStdout: "# frame 0 at 0: 4 bytes: frame: unknown flag\n" +
					"# frame 1 at 4: 7 bytes, no checksum, type 1\n1: {\"a\"}\n",
			},
			{
				name:     "writes the fields of a payload before its error",
				args:     hexArgs(framesFlag),
				give:     []string{cutPayload, emptyFrame},
				wantCode: exitFail,
				wantStdout: "# frame 0 at 0: 7 bytes, no checksum, type 1\n1: 1  # zigzag -1\n" +
					"# kanon: frame 0 at offset 2: unexpected EOF\n# frame 1 at 7: 4 bytes, no checksum, type 1\n",
			},
			{
				name:       "ends the output at a frame that the stream cuts short",
				args:       hexArgs(framesFlag),
				give:       []string{itemFrame, cutFrame},
				wantCode:   exitFail,
				wantStdout: "# frame 0 at 0: 7 bytes, no checksum, type 1\n1: {\"a\"}\n",
				wantStderr: "inspect: frame 1 at 7: unexpected EOF\n",
			},
			{
				name:       "ends the output at a malformed length",
				args:       hexArgs(framesFlag),
				give:       []string{malformedLength},
				wantCode:   exitFail,
				wantStderr: "inspect: frame 0 at 0: frame: malformed length or frame\n",
			},
			{
				name:       "ends the output at a length above the length of the input",
				args:       hexArgs(framesFlag),
				give:       []string{longFrame},
				wantCode:   exitFail,
				wantStderr: "inspect: frame 0 at 0: frame: longer than the limit\n",
			},
			{
				name:       "writes a batch without records",
				args:       hexArgs(batchFlag),
				give:       []string{emptyBatch},
				wantCode:   exitOK,
				wantStdout: "# batch: 0 records, 4-byte offsets\n",
			},
			{
				name:       "writes a batch of one empty record",
				args:       hexArgs(batchFlag),
				give:       []string{oneEmpty},
				wantCode:   exitOK,
				wantStdout: "# batch: 1 record, 4-byte offsets\n# record 0 at 2: 0 bytes\n",
			},
			{
				name:       "writes a batch of two empty records",
				args:       hexArgs(batchFlag),
				give:       []string{twoEmpty},
				wantCode:   exitOK,
				wantStdout: "# batch: 2 records, 4-byte offsets\n# record 0 at 2: 0 bytes\n# record 1 at 2: 0 bytes\n",
			},
			{
				name:       "writes the fields of each record",
				args:       hexArgs(batchFlag),
				give:       []string{oneRecord},
				wantCode:   exitOK,
				wantStdout: "# batch: 1 record, 4-byte offsets\n# record 0 at 2: 2 bytes\n1: 2  # zigzag 1\n",
			},
			{
				name:       "writes a batch with 8-byte offsets",
				args:       hexArgs(batchFlag),
				give:       []string{wideEmpty},
				wantCode:   exitOK,
				wantStdout: "# batch: 1 record, 8-byte offsets\n# record 0 at 2: 0 bytes\n",
			},
			{
				name:     "writes a record that does not parse with its error",
				args:     hexArgs(batchFlag),
				give:     []string{cutRecord},
				wantCode: exitFail,
				wantStdout: "# batch: 1 record, 4-byte offsets\n# record 0 at 2: 1 byte\n" +
					"# kanon: record 0 at offset 0: unexpected EOF\n",
			},
			{
				name:       "returns exitFail for a batch whose layout does not check",
				args:       hexArgs(batchFlag),
				give:       []string{firstOffsetOne},
				wantCode:   exitFail,
				wantStderr: "inspect: batch: index does not match the data\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				code, stdout, stderr := command(tt.args, strings.NewReader(strings.Join(tt.give, " ")))
				assert.Equal(t, code, tt.wantCode, "Command returns the exit status")
				assert.Equal(t, stdout, tt.wantStdout, "Command writes its standard output")
				assert.Equal(t, stderr, tt.wantStderr, "Command writes its standard error")
			})
		}
		goldens := []struct {
			name   string
			layout string
			give   []string
			golden string
		}{
			{
				name:   "writes the frames of a stream as JSON for -json",
				layout: framesFlag,
				give:   []string{itemFrame, checkedFrame, badChecksum, cutPayload},
				golden: framesGolden,
			},
			{
				name:   "writes the records of a batch as JSON for -json",
				layout: batchFlag,
				give:   []string{"01 00 08 02 08 00 00 00 00 02 00 00 00 02 00 00 00"},
				golden: batchGolden,
			},
		}
		for _, tt := range goldens {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				args := []string{tt.layout, hexFlag, jsonFlag}
				code, stdout, stderr := command(args, strings.NewReader(strings.Join(tt.give, " ")))
				assert.Equal(t, code, exitFail, "Command returns exitFail for the part that fails: "+stderr)
				golden.Match(t, tt.golden, []byte(stdout), golden.ShouldUpdate())
			})
		}
		t.Run("writes an empty array for a stream without frames as JSON", func(t *testing.T) {
			t.Parallel()
			code, stdout, _ := command([]string{framesFlag, hexFlag, jsonFlag}, strings.NewReader(""))
			assert.Equal(t, code, exitOK, "Command returns exitOK")
			assert.Equal(t, stdout, "{\n  \"frames\": []\n}\n", "Command writes an empty array")
		})
		writes := []struct {
			name string
			args []string
			give string
		}{
			{
				name: "returns exitFail for frames on a standard output that fails",
				args: hexArgs(framesFlag), give: itemFrame,
			},
			{
				name: "returns exitFail for frames as JSON on a standard output that fails",
				args: []string{framesFlag, hexFlag, jsonFlag}, give: itemFrame,
			},
			{
				name: "returns exitFail for a batch on a standard output that fails",
				args: hexArgs(batchFlag), give: oneRecord,
			},
			{
				name: "returns exitFail for a batch as JSON on a standard output that fails",
				args: []string{batchFlag, hexFlag, jsonFlag}, give: oneRecord,
			},
		}
		for _, tt := range writes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var stderr bytes.Buffer
				code := inspect.Command(tt.args, strings.NewReader(tt.give), failingWriter{}, &stderr)
				assert.Equal(t, code, exitFail, "Command returns exitFail")
				assert.Equal(t, stderr.String(), "inspect: inspect_test: the write fails\n", "Command writes the error")
			})
		}
	})
}
