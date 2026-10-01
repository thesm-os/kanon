// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Command kanon generates the binary codecs of Go struct types. Run it from
// a go:generate directive, as stringer runs:
//
//	//go:generate go tool kanon -type=Order,Line
//
// -type names one or more types of the package, separated by commas: struct
// types, and named bools, numbers, strings, slices, arrays and maps. kanon
// writes <base>.kanon.go, the codec methods of the struct types and the
// ValidateKanon method of the other types, and <base>.kanon_test.go, their
// conformance tests, where <base> is the name of the file with the
// directive. go generate names that file in $GOFILE and runs kanon in its
// directory. Outside go generate, name the file as the only argument.
// -views also declares a view type per struct type, whose methods read one
// field of an encoding without decoding it. -validate names a method of
// signature func() error on a value receiver, whose error the ValidateKanon
// method of each type that is not a struct returns. Without -validate, that
// method returns nil, and the code that kanon generates for a struct does
// not call it. -canonical makes the decode of each struct type accept only
// the canonical encoding of a value, the bytes that its encode writes.
//
// kanon writes a file only when its content changes, so that an unchanged
// codec keeps its modification time.
//
// Usage:
//
//	kanon -type=T[,T...] [-views] [-validate=method] [-canonical] [file]
//	kanon inspect [-frames | -batch] [-hex] [-json] [file]
//	kanon -version
//
// The exit status is 0 on success, 1 when the generation or the check of
// the field numbers fails, and 2 for a usage error.
//
// # Inspect
//
// kanon inspect prints the fields of a kanon encoding without its Go types:
// the number, the wire format and every reading of each value, in the text
// notation of protoscope or as JSON with -json. It reads the file, or the
// standard input without one, as one struct encoding, as a stream of frames
// with -frames, or as a batch with -batch, and reads hexadecimal digits with
// -hex:
//
//	echo '0a 01 78 10 0e' | go tool kanon inspect -hex
//
// Its exit status is 0 when the whole input parses, 1 when a part of it
// does not, and 2 for a usage error.
//
// # Field numbers
//
// Every field encodes under a number. A field without a kanon tag takes
// the smallest free number in declaration order, and <base>.kanon.go
// records the numbers of each struct in a //kanon:numbers line.
// Regeneration keeps the recorded numbers, so that data encoded before an
// edit of a struct still decodes:
//
//   - A field keeps its number when fields are added, removed or
//     reordered.
//   - A new field takes the smallest number that is neither taken nor
//     reserved.
//   - The number of a removed field is reserved. kanon gives it to no other
//     field, and a tag may name it.
//   - A tag that gives a recorded field another number fails the
//     generation.
//
// Commit <base>.kanon.go with its source. Deleting it loses the record. A
// renamed field is a new field: tag it with the number of its old name to
// keep decoding the data that the old name encoded.
//
// Set KANON_CHECK to a git revision, such as the branch that a change
// merges into, to check the numbers against the ones that the revision
// records, without writing any file:
//
//	KANON_CHECK=origin/main go generate ./...
//
// The check fails when a field has another number than at the revision,
// and when a number that the revision gives a removed field or reserves is
// neither reserved nor taken by a field whose tag names it. The same rules
// apply to the concrete types of interfaces, without the exception for
// tags, and to a struct of another package that the revision records as an
// inline struct and that now has a kanon codec of its own, against the
// numbers that the code file of its package records.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.thesmos.sh/kanon/internal/inspect"
	"go.thesmos.sh/kanon/internal/kanon"
	"go.thesmos.sh/kanon/internal/version"
)

// Exit statuses.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// commandName names the command in its messages.
const commandName = "kanon"

// inspectCommand is the first argument that runs the inspector with the
// arguments after it. A source file ends in .go, so it is not a file name.
const inspectCommand = "inspect"

// fileEnv is the environment variable in which go generate names the file
// that contains the directive.
const fileEnv = "GOFILE"

// checkEnv is the environment variable whose value is a git revision. When
// it is set, kanon checks the field numbers against the numbers that the
// code files of the package record at the revision, and writes no file.
const checkEnv = "KANON_CHECK"

// The git commands that read the code files of a package at a revision:
// ls-tree lists the names of the entries of the working directory, each
// ended by a NUL byte, and show writes a file of the working directory.
// The revision follows --end-of-options, so that git does not read it as
// an option.
const (
	gitTool      = "git"
	listTree     = "ls-tree"
	nulFlag      = "-z"
	nameOnlyFlag = "--name-only"
	optionsEnd   = "--end-of-options"
	pathsMark    = "--"
	workingDir   = "."
	showCommand  = "show"
	// revisionPath joins a revision and a name of the working directory in
	// the name of an object: REV:./name.
	revisionPath = ":./"
	// nameEnd ends each name that ls-tree -z writes.
	nameEnd = "\x00"
)

// fileMode is the permission of a file that kanon creates, before the
// umask.
const fileMode = 0o666

// main runs kanon with the arguments, the environment and the standard
// streams of the process, and exits with the status that run returns.
func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr))
}

// run executes kanon with the command-line arguments args, reads the
// environment through getenv, and writes the version to stdout and the
// diagnostics to stderr. With inspect as the first argument, it runs the
// inspector on the arguments after it, which reads stdin when they name no
// file. It returns the exit status.
func run(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == inspectCommand {
		return inspect.Command(args[1:], stdin, stdout, stderr)
	}
	opts, err := kanon.ParseOptions(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return exitOK
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	if opts.Version {
		fmt.Fprintln(stdout, commandName, version.Full())
		return exitOK
	}
	if len(opts.Args) > 1 {
		fmt.Fprintln(stderr, "kanon: more than one file:", strings.Join(opts.Args, " "))
		return exitUsage
	}
	file := getenv(fileEnv)
	if len(opts.Args) == 1 {
		file = opts.Args[0]
	}
	if file == "" {
		fmt.Fprintln(stderr, "kanon: no source file: run kanon from a go:generate directive or name the file")
		return exitUsage
	}
	if rev := getenv(checkEnv); rev != "" {
		err = check(file, opts, rev)
	} else {
		err = generate(file, opts)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFail
	}
	return exitOK
}

// check checks the field numbers of the struct types that opts names, from
// the source file at path, against the numbers that the code files of its
// package record at the git revision rev.
func check(path string, opts kanon.Options, rev string) error {
	dir := filepath.Dir(path)
	base, err := codeFiles(dir, rev)
	if err != nil {
		return err
	}
	return kanon.Check(dir, filepath.Base(path), opts, base)
}

// codeFiles returns the code files of the package in dir at the git
// revision rev, by name. A revision without the directory has none.
func codeFiles(dir, rev string) (map[string][]byte, error) {
	names, err := git(dir, listTree, nulFlag, nameOnlyFlag, optionsEnd, rev, pathsMark, workingDir)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte)
	for name := range strings.SplitSeq(string(names), nameEnd) {
		if !strings.HasSuffix(name, kanon.CodeSuffix) {
			continue
		}
		src, err := git(dir, showCommand, optionsEnd, rev+revisionPath+name)
		if err != nil {
			return nil, err
		}
		out[name] = src
	}
	return out, nil
}

// git runs the git command args in dir and returns what it writes to its
// standard output. The error of a failed command names the command and ends
// with what git writes to its standard error.
func git(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(gitTool, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kanon: git %s: %w: %s", args[0], err, bytes.TrimSpace(stderr.Bytes()))
	}
	return out, nil
}

// generate generates the files of the struct types that opts names from the
// source file at path, and writes each one whose content differs from the
// file on disk.
func generate(path string, opts kanon.Options) error {
	dir := filepath.Dir(path)
	files, err := kanon.Generate(dir, filepath.Base(path), opts)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := write(filepath.Join(dir, f.Name), f.Src); err != nil {
			return err
		}
	}
	return nil
}

// write writes src to the file at path unless the file contains src
// already.
func write(path string, src []byte) error {
	old, err := os.ReadFile(path)
	if err == nil && bytes.Equal(old, src) {
		return nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("kanon: read %s: %w", path, err)
	}
	if err := os.WriteFile(path, src, fileMode); err != nil {
		return fmt.Errorf("kanon: write %s: %w", path, err)
	}
	return nil
}
