// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/version"
)

// mainEnv makes the test binary run main in place of its tests when it has
// the value runMain. A test then runs kanon as a process, with arguments, an
// environment and an exit status. go test sets GOCOVERDIR for a test binary
// built with coverage. The process writes its coverage counters there when
// it exits, and the test binary merges them into its coverage profile.
const (
	mainEnv = "KANON_TEST_MAIN"
	runMain = "1"
)

// graceShare is the share of the time left before the deadline of a test
// that a process of the test does not get: the test kills the process at
// the deadline less the time left divided by graceShare.
const graceShare = 20

// The module of one package that a test writes: its go.mod, its source
// file, the two files that kanon generates for the source file, and the
// directory of a second package.
const (
	modName    = "go.mod"
	goMod      = "module example.com/m\n\ngo 1.27\n"
	sourceName = "a.go"
	codeName   = "a.kanon.go"
	testName   = "a.kanon_test.go"
	subName    = "sub"
	// source declares the struct type A, whose fields X and Y take the
	// numbers 1 and 2.
	source = "package m\n\ntype A struct {\n\tX int32\n\tY int32\n}\n"
	// renumbered gives Y the number 5 with a tag.
	renumbered = "package m\n\ntype A struct {\n\tX int32\n\tY int32 `kanon:\"5\"`\n}\n"
	// subSource declares A in the package sub.
	subSource = "package sub\n\ntype A struct {\n\tX int32\n}\n"
)

// Permissions of the files and the directories that a test writes, and of a
// generated file that kanon cannot write.
const (
	sourceMode   = 0o644
	dirMode      = 0o755
	readOnlyMode = 0o444
)

// Arguments of kanon in the tests.
const (
	typeFlag    = "-type=A"
	missingFlag = "-type=Missing"
	helpFlag    = "-h"
	unknownFlag = "-bogus"
	versionFlag = "-version"
	otherName   = "b.go"
	// inspectHex runs the inspector on hexadecimal digits.
	inspectHex = "-hex"
)

// The input of the inspector in the tests, the encoding of a varint field 1
// of value 2, and the text that the inspector writes for it.
const (
	inspectInput = "08 02"
	inspectText  = "1: 2  # zigzag 1\n"
)

// noSourceFile is what run writes to its standard error when neither GOFILE
// nor an argument names the source file.
const noSourceFile = "kanon: no source file: run kanon from a go:generate directive or name the file\n"

// The git commands that commit runs, with the configuration of a test: a
// fixed author, no signature, and a hooks directory that does not exist, so
// that no hook of the machine runs.
const (
	initCommand   = "init"
	addCommand    = "add"
	commitCommand = "commit"
	quietFlag     = "--quiet"
	allFlag       = "--all"
	messageFlag   = "--message=base"
	configFlag    = "-c"
	authorName    = "user.name=kanon"
	authorEmail   = "user.email=kanon@example.com"
	noSignature   = "commit.gpgsign=false"
	noHooks       = "core.hooksPath=.git/no-hooks"
)

// gitEnvPrefix begins the names of the environment variables that locate
// the repository of a git command.
const gitEnvPrefix = "GIT_"

// The revisions that KANON_CHECK names in the tests: the last commit, a name
// that git does not know, and the tree of the last commit, in which git show
// does not find a file under REV:./name.
const (
	headRevision    = "HEAD"
	unknownRevision = "missing"
	treeRevision    = "HEAD:"
)

// past is a modification time before the time of every file that a test
// writes.
var past = time.Unix(0, 0)

// TestMain runs main in place of the tests when mainEnv has the value
// runMain, and the tests otherwise.
func TestMain(m *testing.M) {
	if os.Getenv(mainEnv) == runMain {
		main()
	}
	os.Exit(m.Run())
}

// module writes a module of one package, whose source file declares A, into
// a new directory, and returns the path of the source file.
func module(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, modName), []byte(goMod), sourceMode), "the go.mod writes")
	path := filepath.Join(dir, sourceName)
	assert.NoError(t, os.WriteFile(path, []byte(source), sourceMode), "the source file writes")
	return path
}

// invoke runs run with the arguments args, the environment vars, a map from
// the name of each variable to its value, and the standard input stdin, and
// returns the exit status and what run writes to its standard output and its
// standard error.
func invoke(args []string, vars map[string]string, stdin string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, func(name string) string { return vars[name] }, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// generated runs run on the source file at path, and stops t unless run
// writes the generated files of A.
func generated(t *testing.T, path string) {
	t.Helper()
	code, _, stderr := invoke([]string{typeFlag, path}, nil, "")
	assert.Equal(t, code, exitOK, "run generates the files: "+stderr)
}

// command returns the command that runs name with the arguments args, and
// that t kills shortly before its deadline. A process that hangs then fails
// t, and does not keep running after the test binary exits.
func command(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()
	ctx := t.Context()
	if deadline, ok := t.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline.Add(-time.Until(deadline)/graceShare))
		t.Cleanup(cancel)
	}
	return exec.CommandContext(ctx, name, args...)
}

// commit creates a git repository in dir and commits every file of dir, with
// the configuration of a test on every git command. The git commands inherit
// the environment of the process, which TestKanonEnv clears of the GIT_
// variables.
func commit(t *testing.T, dir string) {
	t.Helper()
	for _, step := range [][]string{
		{initCommand, quietFlag},
		{addCommand, allFlag},
		{commitCommand, quietFlag, messageFlag},
	} {
		args := append([]string{
			configFlag, authorName,
			configFlag, authorEmail,
			configFlag, noSignature,
			configFlag, noHooks,
		}, step...)
		cmd := command(t, gitTool, args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		assert.NoError(t, err, "git "+step[0]+" runs: "+string(out))
	}
}

// TestKanonProcess runs kanon as a process of the test binary, one process
// at a time. Every process writes the coverage meta-data file of the binary
// into the one GOCOVERDIR under a temporary name that Go 1.27 makes from the
// time in nanoseconds alone. Two processes that exit in the same nanosecond
// write one temporary file, the rename of one of them fails, and that
// process writes the failure to its standard error.
func TestKanonProcess(t *testing.T) {
	t.Run("main", func(t *testing.T) {
		self, err := os.Executable()
		assert.NoError(t, err, "the test binary has a path")
		tests := []struct {
			name       string
			args       []string
			stdin      string
			wantCode   int
			wantStdout string
			wantStderr string
		}{
			{
				name:       "writes the version to os.Stdout",
				args:       []string{versionFlag},
				wantCode:   exitOK,
				wantStdout: commandName + " " + version.Full() + "\n",
			},
			{
				name:       "exits with the status that run returns",
				args:       []string{typeFlag},
				wantCode:   exitUsage,
				wantStderr: noSourceFile,
			},
			{
				name:       "reads os.Stdin for the inspector",
				args:       []string{inspectCommand, inspectHex},
				stdin:      inspectInput,
				wantCode:   exitOK,
				wantStdout: inspectText,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := command(t, self, tt.args...)
				cmd.Env = append(os.Environ(), mainEnv+"="+runMain, fileEnv+"=")
				var stdout, stderr bytes.Buffer
				cmd.Stdin, cmd.Stdout, cmd.Stderr = strings.NewReader(tt.stdin), &stdout, &stderr
				if err := cmd.Run(); err != nil {
					_, exited := errors.AsType[*exec.ExitError](err)
					assert.True(t, exited, "kanon runs: "+err.Error())
				}
				assert.Equal(t, cmd.ProcessState.ExitCode(), tt.wantCode, "kanon exits with the status of run")
				assert.Equal(t, stdout.String(), tt.wantStdout, "kanon writes the standard output of run")
				assert.Equal(t, stderr.String(), tt.wantStderr, "kanon writes the standard error of run")
			})
		}
	})
}

func TestKanon(t *testing.T) {
	t.Parallel()
	t.Run("run", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name       string
			args       []string
			stdin      string
			wantCode   int
			wantStdout string
			// wantStderr is the beginning of what run writes to its standard
			// error.
			wantStderr string
		}{
			{
				name:       "runs the inspector on the arguments after inspect",
				args:       []string{inspectCommand, inspectHex},
				stdin:      inspectInput,
				wantCode:   exitOK,
				wantStdout: inspectText,
			},
			{
				name:       "returns exitOK for -h",
				args:       []string{helpFlag},
				wantCode:   exitOK,
				wantStderr: "usage: kanon ",
			},
			{
				name:       "returns exitUsage for a flag that kanon does not define",
				args:       []string{unknownFlag},
				wantCode:   exitUsage,
				wantStderr: "flag provided but not defined: -bogus\n",
			},
			{
				name:       "writes the version for -version",
				args:       []string{versionFlag},
				wantCode:   exitOK,
				wantStdout: commandName + " " + version.Full() + "\n",
			},
			{
				name:       "returns exitUsage for two files",
				args:       []string{typeFlag, sourceName, otherName},
				wantCode:   exitUsage,
				wantStderr: "kanon: more than one file: a.go b.go\n",
			},
			{
				name:       "returns exitUsage without a source file",
				args:       []string{typeFlag},
				wantCode:   exitUsage,
				wantStderr: noSourceFile,
			},
			{
				name:       "returns exitUsage without arguments",
				wantCode:   exitUsage,
				wantStderr: noSourceFile,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				code, stdout, stderr := invoke(tt.args, nil, tt.stdin)
				assert.Equal(t, code, tt.wantCode, "run returns the exit status")
				assert.Equal(t, stdout, tt.wantStdout, "run writes its standard output")
				assert.HasPrefix(t, stderr, tt.wantStderr, "run writes its standard error")
			})
		}
		t.Run("returns exitFail for a type that the package does not declare", func(t *testing.T) {
			t.Parallel()
			code, _, stderr := invoke([]string{missingFlag, module(t)}, nil, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.Equal(t, stderr, "kanon: -type names Missing, which package m does not declare\n",
				"run writes the error of the generation")
		})
		t.Run("writes the generated files beside the file that GOFILE names", func(t *testing.T) {
			t.Parallel()
			path := module(t)
			code, _, stderr := invoke([]string{typeFlag}, map[string]string{fileEnv: path}, "")
			assert.Equal(t, code, exitOK, "run returns exitOK: "+stderr)
			for _, name := range []string{codeName, testName} {
				_, err := os.Stat(filepath.Join(filepath.Dir(path), name))
				assert.NoError(t, err, "run writes "+name)
			}
		})
		t.Run("leaves a generated file whose content does not change", func(t *testing.T) {
			t.Parallel()
			path := module(t)
			generated(t, path)
			codePath := filepath.Join(filepath.Dir(path), codeName)
			assert.NoError(t, os.Chtimes(codePath, past, past), "the code file takes an earlier modification time")
			generated(t, path)
			info, err := os.Stat(codePath)
			assert.NoError(t, err, "the code file stats")
			assert.True(t, info.ModTime().Equal(past), "run leaves the modification time of the code file")
		})
		t.Run("returns exitFail when a generated file is a directory", func(t *testing.T) {
			t.Parallel()
			path := module(t)
			test := filepath.Join(filepath.Dir(path), testName)
			assert.NoError(t, os.Mkdir(test, dirMode), "a directory takes the name of the test file")
			code, _, stderr := invoke([]string{typeFlag, path}, nil, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.HasPrefix(t, stderr, "kanon: read "+test+": ", "run writes the error of the read")
		})
		t.Run("returns exitFail when a generated file is read-only", func(t *testing.T) {
			t.Parallel()
			if os.Geteuid() == 0 {
				t.Skip("the superuser writes a read-only file")
			}
			path := module(t)
			test := filepath.Join(filepath.Dir(path), testName)
			assert.NoError(t, os.WriteFile(test, nil, readOnlyMode), "a read-only test file writes")
			code, _, stderr := invoke([]string{typeFlag, path}, nil, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.HasPrefix(t, stderr, "kanon: write "+test+": ", "run writes the error of the write")
		})
	})
}

// TestKanonEnv runs the check of the field numbers. Its git commands inherit
// the environment of the process, in which a git hook exports GIT_DIR and
// GIT_INDEX_FILE for the repository that runs the hook. Because TestKanonEnv
// clears the GIT_ variables of the process, it runs serially.
func TestKanonEnv(t *testing.T) {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, gitEnvPrefix) {
			t.Setenv(name, "")
			assert.NoError(t, os.Unsetenv(name), "the process clears "+name)
		}
	}
	t.Run("run", func(t *testing.T) {
		t.Run("returns exitOK for the numbers that the base revision records", func(t *testing.T) {
			path := module(t)
			generated(t, path)
			commit(t, filepath.Dir(path))
			code, _, stderr := invoke([]string{typeFlag, path}, map[string]string{checkEnv: headRevision}, "")
			assert.Equal(t, code, exitOK, "run returns exitOK: "+stderr)
		})
		t.Run("returns exitFail for a field that the source renumbers", func(t *testing.T) {
			path := module(t)
			generated(t, path)
			commit(t, filepath.Dir(path))
			assert.NoError(t, os.Remove(filepath.Join(filepath.Dir(path), codeName)), "the code file removes")
			assert.NoError(t, os.WriteFile(path, []byte(renumbered), sourceMode), "the source file renumbers Y")
			code, _, stderr := invoke([]string{typeFlag, path}, map[string]string{checkEnv: headRevision}, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.Equal(t, stderr, "kanon: field Y of A has number 5, and the base revision gives it number 2\n",
				"run writes the rule that the number breaks")
		})
		t.Run("returns exitOK for a directory that the revision does not have", func(t *testing.T) {
			path := module(t)
			dir := filepath.Dir(path)
			commit(t, dir)
			sub := filepath.Join(dir, subName)
			assert.NoError(t, os.Mkdir(sub, dirMode), "the directory of a new package writes")
			other := filepath.Join(sub, sourceName)
			assert.NoError(t, os.WriteFile(other, []byte(subSource), sourceMode),
				"the source file of the package writes")
			code, _, stderr := invoke([]string{typeFlag, other}, map[string]string{checkEnv: headRevision}, "")
			assert.Equal(t, code, exitOK, "run returns exitOK: "+stderr)
		})
		t.Run("returns exitFail for a revision that git does not know", func(t *testing.T) {
			path := module(t)
			commit(t, filepath.Dir(path))
			code, _, stderr := invoke([]string{typeFlag, path}, map[string]string{checkEnv: unknownRevision}, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.HasPrefix(t, stderr, "kanon: git ls-tree: ", "run writes the git command that fails")
		})
		t.Run("returns exitFail for a code file that git show does not find", func(t *testing.T) {
			path := module(t)
			generated(t, path)
			commit(t, filepath.Dir(path))
			code, _, stderr := invoke([]string{typeFlag, path}, map[string]string{checkEnv: treeRevision}, "")
			assert.Equal(t, code, exitFail, "run returns exitFail")
			assert.HasPrefix(t, stderr, "kanon: git show: ", "run writes the git command that fails")
		})
	})
}
