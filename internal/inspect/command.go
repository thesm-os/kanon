// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package inspect

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"go.thesmos.sh/kanon"
)

// Exit statuses of [Command].
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// commandName names the subcommand in its usage.
const commandName = "kanon inspect"

// errPrefix begins every error that the subcommand writes to its standard
// error.
const errPrefix = "inspect: "

// Names of the flags of the subcommand.
const (
	framesFlag = "frames"
	batchFlag  = "batch"
	hexFlag    = "hex"
	jsonFlag   = "json"
)

// usage is the first line of the usage of the subcommand.
const usage = "usage: kanon inspect [-frames | -batch] [-hex] [-json] [file]"

// format is the notation of the output of the subcommand.
type format uint8

// Formats of the output.
const (
	// formatText is the text notation of protoscope.
	formatText format = 0
	// formatJSON is JSON.
	formatJSON format = 1
)

// Command runs the inspect subcommand of the kanon command with args, the
// arguments after the word inspect:
//
//	kanon inspect [-frames | -batch] [-hex] [-json] [file]
//
// It reads the file that args name, or stdin without a file, as one struct
// encoding, as a stream of frames with -frames, or as one batch with -batch,
// and the input as hexadecimal digits with whitespace between them with
// -hex. It writes the fields of the input to stdout in the text notation of
// [WriteText], or as JSON with -json, and an error that ends the input to
// stderr. It returns 0 when the whole input parses, 1 when the input does not
// read or parse, and 2 for a usage error.
func Command(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintln(stderr, usage)
		fs.PrintDefaults()
	}
	frames := fs.Bool(framesFlag, false, "read the input as a stream of frames")
	batch := fs.Bool(batchFlag, false, "read the input as a batch")
	hexInput := fs.Bool(hexFlag, false, "read the input as hexadecimal digits, with whitespace between them")
	asJSON := fs.Bool(jsonFlag, false, "write JSON instead of text")
	if err := fs.Parse(args); errors.Is(err, flag.ErrHelp) {
		return exitOK
	} else if err != nil {
		return exitUsage
	}
	if *frames && *batch {
		fmt.Fprintln(stderr, errPrefix+"-frames and -batch exclude each other")
		return exitUsage
	}
	if fs.NArg() > 1 {
		fmt.Fprintln(stderr, errPrefix+"more than one file: "+strings.Join(fs.Args(), " "))
		return exitUsage
	}
	data, err := readInput(fs.Arg(0), stdin, *hexInput)
	if err != nil {
		fmt.Fprintln(stderr, errPrefix+err.Error())
		return exitFail
	}
	out := formatText
	if *asJSON {
		out = formatJSON
	}
	switch {
	case *frames:
		return inspectFrames(data, out, stdout, stderr)
	case *batch:
		return inspectBatch(data, out, stdout, stderr)
	default:
		return inspectStruct(data, out, stdout, stderr)
	}
}

// readInput returns the input of the subcommand: the content of the file at
// path, or of stdin for an empty path, and the bytes that its hexadecimal
// digits spell with hexInput set.
func readInput(path string, stdin io.Reader, hexInput bool) ([]byte, error) {
	var data []byte
	var err error
	if path != "" {
		data, err = os.ReadFile(path)
	} else {
		data, err = io.ReadAll(stdin)
	}
	if err != nil || !hexInput {
		return data, err
	}
	digits := bytes.Join(bytes.Fields(data), nil)
	out := make([]byte, hex.DecodedLen(len(digits)))
	_, err = hex.Decode(out, digits)
	return out, err
}

// inspectStruct writes the fields of data, one struct encoding, to stdout in
// the format out, and the error of the write or else of the parse to
// stderr, and returns the exit status of [Command].
func inspectStruct(data []byte, out format, stdout, stderr io.Writer) int {
	fields, err := Parse(data, kanon.DefaultDepth)
	var werr error
	if out == formatJSON {
		werr = WriteJSON(stdout, fields)
	} else {
		werr = WriteText(stdout, fields)
	}
	if werr != nil {
		return report(stderr, werr)
	}
	return report(stderr, err)
}

// report writes err, when it is not nil, to stderr, and returns the exit
// status for it: exitFail for an error and exitOK for none.
func report(stderr io.Writer, err error) int {
	if err == nil {
		return exitOK
	}
	fmt.Fprintln(stderr, errPrefix+err.Error())
	return exitFail
}
