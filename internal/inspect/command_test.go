// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/inspect"
)

// Exit statuses of Command.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// Flags of the subcommand in the tests.
const (
	framesFlag  = "-frames"
	batchFlag   = "-batch"
	hexFlag     = "-hex"
	jsonFlag    = "-json"
	helpFlag    = "-h"
	unknownFlag = "-bogus"
)

// The input of the tests of a struct encoding: {Label: "x", Count: 7}, and
// the text that Command writes for it.
const (
	structHex  = "0a 01 78 10 0e"
	structText = "1: {\"x\"}\n2: 14  # zigzag 7\n"
)

// fileMode is the permission of the input files of the tests.
const fileMode = 0o644

// errRead is the error of a standard input that fails.
var errRead = errors.New("inspect_test: the read fails")

// command runs Command with args and the standard input stdin, and returns
// the exit status and what Command writes to its standard output and its
// standard error.
func command(args []string, stdin io.Reader) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := inspect.Command(args, stdin, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// inputFile writes data to a new file and returns its path.
func inputFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input")
	assert.NoError(t, os.WriteFile(path, data, fileMode), "the input file writes")
	return path
}

func TestCommand(t *testing.T) {
	t.Parallel()
	t.Run("Command", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "missing")
		tests := []struct {
			name       string
			args       []string
			stdin      io.Reader
			wantCode   int
			wantStdout string
			// wantStderr is the beginning of what Command writes to its
			// standard error.
			wantStderr string
		}{
			{
				name:       "writes the fields of the struct encoding on its standard input",
				stdin:      bytes.NewReader([]byte{0x0a, 0x01, 0x78, 0x10, 0x0e}),
				wantCode:   exitOK,
				wantStdout: structText,
			},
			{
				name:       "reads hexadecimal digits with whitespace between them for -hex",
				args:       []string{hexFlag},
				stdin:      strings.NewReader("0a 01\n78\t10 0e\n"),
				wantCode:   exitOK,
				wantStdout: structText,
			},
			{
				name:     "writes JSON for -json",
				args:     []string{hexFlag, jsonFlag},
				stdin:    strings.NewReader("08 00"),
				wantCode: exitOK,
				wantStdout: "{\n  \"fields\": [\n    {\n      \"field\": 1,\n      \"wire\": \"varint\",\n" +
					"      \"offset\": 0,\n      \"unsigned\": 0,\n      \"zigzag\": 0\n    }\n  ]\n}\n",
			},
			{
				name:       "returns exitFail for a struct encoding that does not parse",
				args:       []string{hexFlag},
				stdin:      strings.NewReader("08 01 10"),
				wantCode:   exitFail,
				wantStdout: "1: 1  # zigzag -1\n",
				wantStderr: "inspect: kanon: input at offset 2: unexpected EOF\n",
			},
			{
				name:       "returns exitFail for hexadecimal digits of odd length",
				args:       []string{hexFlag},
				stdin:      strings.NewReader("0a 0"),
				wantCode:   exitFail,
				wantStderr: "inspect: encoding/hex: odd length hex string\n",
			},
			{
				name:       "returns exitFail for a byte that is not a hexadecimal digit",
				args:       []string{hexFlag},
				stdin:      strings.NewReader("zz"),
				wantCode:   exitFail,
				wantStderr: "inspect: encoding/hex: invalid byte: U+007A 'z'\n",
			},
			{
				name:       "returns exitFail for a standard input that fails",
				stdin:      iotest.ErrReader(errRead),
				wantCode:   exitFail,
				wantStderr: "inspect: inspect_test: the read fails\n",
			},
			{
				name:       "returns exitFail for a file that does not exist",
				args:       []string{missing},
				wantCode:   exitFail,
				wantStderr: "inspect: open " + missing + ": ",
			},
			{
				name:       "returns exitFail for a file of hexadecimal digits that does not exist",
				args:       []string{hexFlag, missing},
				wantCode:   exitFail,
				wantStderr: "inspect: open " + missing + ": ",
			},
			{
				name:       "returns exitUsage for -frames with -batch",
				args:       []string{framesFlag, batchFlag},
				wantCode:   exitUsage,
				wantStderr: "inspect: -frames and -batch exclude each other\n",
			},
			{
				name:       "returns exitUsage for two files",
				args:       []string{"a", "b"},
				wantCode:   exitUsage,
				wantStderr: "inspect: more than one file: a b\n",
			},
			{
				name:       "returns exitUsage for a flag that it does not define",
				args:       []string{unknownFlag},
				wantCode:   exitUsage,
				wantStderr: "flag provided but not defined: -bogus\n",
			},
			{
				name:       "returns exitOK for -h",
				args:       []string{helpFlag},
				wantCode:   exitOK,
				wantStderr: "usage: kanon inspect [-frames | -batch] [-hex] [-json] [file]\n",
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				stdin := tt.stdin
				if stdin == nil {
					stdin = strings.NewReader("")
				}
				code, stdout, stderr := command(tt.args, stdin)
				assert.Equal(t, code, tt.wantCode, "Command returns the exit status")
				assert.Equal(t, stdout, tt.wantStdout, "Command writes its standard output")
				assert.HasPrefix(t, stderr, tt.wantStderr, "Command writes its standard error")
			})
		}
		t.Run("reads the file that its argument names", func(t *testing.T) {
			t.Parallel()
			path := inputFile(t, unhex(t, structHex))
			code, stdout, stderr := command([]string{path}, strings.NewReader(""))
			assert.Equal(t, code, exitOK, "Command returns exitOK: "+stderr)
			assert.Equal(t, stdout, structText, "Command writes the fields of the file")
		})
		writes := []struct {
			name string
			args []string
		}{
			{name: "returns exitFail for text on a standard output that fails", args: []string{hexFlag}},
			{name: "returns exitFail for JSON on a standard output that fails", args: []string{hexFlag, jsonFlag}},
		}
		for _, tt := range writes {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var stderr bytes.Buffer
				code := inspect.Command(tt.args, strings.NewReader("08 01 10"), failingWriter{}, &stderr)
				assert.Equal(t, code, exitFail, "Command returns exitFail")
				assert.Equal(t, stderr.String(), "inspect: inspect_test: the write fails\n",
					"Command writes the error of the write and not the error of the parse")
			})
		}
	})
}
