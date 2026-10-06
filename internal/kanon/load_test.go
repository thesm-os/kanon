// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"testing"

	"go.dokimi.dev/assert"
	"go.dokimi.dev/assert/files"
)

// Environment of the fake go command, which TestMain runs in place of the
// tests: fakeGoMode selects what it fakes, and realGo names the go command
// that it runs for everything else.
const (
	fakeGoMode = "KANON_TEST_FAKE_GO"
	realGo     = "KANON_TEST_REAL_GO"
)

// Modes of the fake go command.
const (
	// fakeGarbage writes output that is not JSON for every command.
	fakeGarbage = "garbage"
	// fakeDeps fails the go list of the dependencies, which lists their
	// export data.
	fakeDeps = "deps"
	// fakeFile adds depFile to the Go files of every dependency.
	fakeFile = "file"
)

// Names of the fake go command, its flag that lists export data, and the
// file that fakeFile adds to the dependencies, which a test writes.
const (
	goCommand  = "go"
	exportFlag = "-export"
	depFile    = "broken.txt"
	windows    = "windows"
	exeSuffix  = ".exe"
)

// TestMain runs the tests, or runs as the fake go command when the
// environment selects a mode of it.
func TestMain(m *testing.M) {
	if mode, ok := os.LookupEnv(fakeGoMode); ok {
		os.Exit(fakeGo(mode, os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeGo runs as the go command with the arguments args in the mode mode,
// and returns its exit status. It runs the go command that realGo names,
// and changes its output as mode states.
func fakeGo(mode string, args []string) int {
	if mode == fakeGarbage {
		_, _ = os.Stdout.WriteString("{")
		return 0
	}
	deps := slices.Contains(args, exportFlag)
	if deps && mode == fakeDeps {
		fmt.Fprintln(os.Stderr, "fake go: the dependencies do not list")
		return 1
	}
	cmd := exec.Command(os.Getenv(realGo), args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return 1
	}
	if !deps {
		_, _ = os.Stdout.Write(out)
		return 0
	}
	enc := json.NewEncoder(os.Stdout)
	for dec := json.NewDecoder(bytes.NewReader(out)); dec.More(); {
		var p map[string]any
		if dec.Decode(&p) != nil {
			return 1
		}
		files, _ := p["GoFiles"].([]any)
		p["GoFiles"] = append(files, depFile)
		_ = enc.Encode(p)
	}
	return 0
}

// TestLoadEnv runs Generate with the fake go command on the PATH, which it
// sets for the process, so that it runs serially.
func TestLoadEnv(t *testing.T) {
	goPath, err := exec.LookPath(goCommand)
	assert.NoError(t, err, "the PATH has the go command")
	self, err := os.Executable()
	assert.NoError(t, err, "the test binary has a path")
	name := goCommand
	if runtime.GOOS == windows {
		name += exeSuffix
	}
	dir := files.Workspace(t, files.Tree{name: files.Executable(files.Read(t, self))})
	t.Setenv("PATH", dir)
	t.Setenv(realGo, goPath)
	t.Run("Generate", func(t *testing.T) {
		t.Run("returns an error for go list output that is not JSON", func(t *testing.T) {
			t.Setenv(fakeGoMode, fakeGarbage)
			_, err := generate(t, module(t, map[string]string{source: structXY}), source, "A")
			assert.HasError(t, err, "Generate fails for the output")
			assert.HasPrefix(t, err.Error(), "kanon: decode go list output: ", "Generate states why the output fails")
		})
		t.Run("returns the error of the go list of the dependencies", func(t *testing.T) {
			t.Setenv(fakeGoMode, fakeDeps)
			sources := map[string]string{
				"sub/sub.go": "// T is a struct type.\ntype T struct {\n\tX int32\n}\n",
				source:       subImport + structA("T sub.T"),
			}
			mod := module(t, sources)
			_, err := generate(t, mod, source, "A")
			assert.HasError(t, err, "Generate fails for the dependencies")
			assert.HasPrefix(t, err.Error(), "kanon: go list in "+mod+": ", "Generate states the command that fails")
			assert.HasSuffix(t, err.Error(), ": fake go: the dependencies do not list",
				"Generate states what go list writes to its standard error")
		})
		t.Run("ignores a file of a dependency that does not parse", func(t *testing.T) {
			t.Setenv(fakeGoMode, fakeFile)
			sources := map[string]string{
				"dep/dep.go": "//go:generate go tool kanon -type=T\n\n// T is a struct type with a kanon codec.\n" +
					"type T struct {\n\tX int32\n}\n",
				"dep/" + depFile: "{",
				source:           "import \"example.com/m/dep\"\n\n" + structA("T dep.T"),
			}
			got, err := generate(t, module(t, sources), source, "A")
			assert.NoError(t, err, "Generate reads the directive of the dependency")
			assert.Contains(t, got[codeName], "m.T.SizeKanon()", "the field encodes through the codec of T")
		})
	})
}

func TestLoad(t *testing.T) {
	t.Parallel()
	t.Run("Generate", func(t *testing.T) {
		t.Parallel()
		t.Run("returns the error of go list for a go.mod that does not parse", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{modName: "bogus\n", source: structXY})
			_, err := generate(t, dir, source, "A")
			assert.HasError(t, err, "Generate fails for the go.mod")
			assert.HasPrefix(t, err.Error(), "kanon: go list in "+dir+": ", "Generate states the command that fails")
			assert.Contains(t, err.Error(), "go.mod", "Generate states what go list writes to its standard error")
		})
		t.Run("returns the error of go list for a package without Go files", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{})
			_, err := generate(t, dir, source, "A")
			assert.HasError(t, err, "Generate fails for the package")
			assert.HasPrefix(t, err.Error(), "kanon: package in "+dir+": no Go files in ",
				"Generate states the error of go list")
		})
		t.Run("returns an error for a package of test files alone", func(t *testing.T) {
			t.Parallel()
			dir := module(t, map[string]string{"a_test.go": ""})
			_, err := generate(t, dir, source, "A")
			assert.HasError(t, err, "Generate fails for the package")
			assert.Equal(t, err.Error(), "kanon: package in "+dir+": no Go files", "Generate states why")
		})
		t.Run("returns an error for a file that does not parse", func(t *testing.T) {
			t.Parallel()
			_, err := generate(t, module(t, map[string]string{source: "func {\n"}), source, "A")
			assert.HasError(t, err, "Generate fails for the file")
			assert.HasSuffix(t, err.Error(), "a.go:3:6: expected 'IDENT', found '{'", "Generate states the parse error")
		})
		t.Run("returns an error for a field of a type of a package that does not compile", func(t *testing.T) {
			t.Parallel()
			files := map[string]string{
				"sub/sub.go": "// T is a struct type of a package that does not compile.\ntype T struct {\n\tX Missing\n}\n",
				source:       subImport + structA("T sub.T"),
			}
			assert.Equal(t, generateError(t, files),
				"kanon: a.go:6:2: A.T: the type does not type-check: go build reports why",
				"Generate states why the field fails")
		})
	})
}
