// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// Flags of the kanon command.
const (
	// flagType names the struct types to generate codecs for.
	flagType = "type"
	// flagViews requests a view type per struct type.
	flagViews = "views"
	// flagVersion requests the version of the command.
	flagVersion = "version"
	// typeSeparator separates the names in the value of -type.
	typeSeparator = ","
)

// usageLine is the synopsis of the kanon command.
const usageLine = "usage: kanon -type=T[,T...] [-views] [file]"

// Options are the command-line options of kanon. A go:generate directive
// passes them in the same form.
type Options struct {
	// Types lists the struct types that -type names, in flag order.
	Types []string
	// Args lists the arguments after the flags.
	Args []string
	// Views reports that -views requests a view type per struct type.
	Views bool
	// Version reports that -version requests the version of the command.
	Version bool
}

// ParseOptions parses the command-line arguments args of kanon and writes
// the usage and any flag error to output. It fails for an unknown flag, for
// -h and -help with [flag.ErrHelp], and for a -type value that names an
// empty type or one type twice. It does not require -type, which
// [Generate] and [Check] require.
func ParseOptions(args []string, output io.Writer) (Options, error) {
	var opts Options
	var types string
	flags := flag.NewFlagSet(commandName, flag.ContinueOnError)
	flags.SetOutput(output)
	flags.StringVar(&types, flagType, "", "comma-separated `names` of the struct types to generate codecs for")
	flags.BoolVar(&opts.Views, flagViews, false, "generate a view type per struct type")
	flags.BoolVar(&opts.Version, flagVersion, false, "print the version and exit")
	flags.Usage = func() {
		fmt.Fprintln(output, usageLine)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return Options{}, err
	}
	opts.Args = flags.Args()
	if types == "" {
		return opts, nil
	}
	seen := make(map[string]bool)
	for name := range strings.SplitSeq(types, typeSeparator) {
		if name == "" {
			return Options{}, fmt.Errorf("kanon: -type %q names an empty type", types)
		}
		if seen[name] {
			return Options{}, fmt.Errorf("kanon: -type %q names %s twice", types, name)
		}
		seen[name] = true
		opts.Types = append(opts.Types, name)
	}
	return opts, nil
}
