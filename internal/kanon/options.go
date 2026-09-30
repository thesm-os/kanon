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
	// flagType names the types to generate code for.
	flagType = "type"
	// flagViews requests a view type per struct type.
	flagViews = "views"
	// flagValidate names the method that the ValidateKanon method of each
	// named type that is not a struct calls.
	flagValidate = "validate"
	// flagCanonical makes the decode of each struct type accept only its
	// canonical encoding.
	flagCanonical = "canonical"
	// flagVersion requests the version of the command.
	flagVersion = "version"
	// typeSeparator separates the names in the value of -type.
	typeSeparator = ","
)

// usageLine is the synopsis of the kanon command.
const usageLine = "usage: kanon -type=T[,T...] [-views] [-validate=method] [-canonical] [file]"

// Options are the command-line options of kanon. A go:generate directive
// passes them in the same form.
type Options struct {
	// Types lists the types that -type names, in flag order: struct types,
	// and named types that are not structs.
	Types []string
	// Args lists the arguments after the flags.
	Args []string
	// Validate names the method, of signature func() error, that the
	// ValidateKanon method of each named type of Types that is not a struct
	// calls. It is empty when ValidateKanon accepts every value.
	Validate string
	// Views reports that -views requests a view type per struct type.
	Views bool
	// Canonical reports that -canonical makes the decode of each struct type
	// of Types, and of the inline structs of their fields, accept only the
	// canonical encoding of a value.
	Canonical bool
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
	flags.StringVar(&types, flagType, "", "comma-separated `names` of the types to generate code for")
	flags.BoolVar(&opts.Views, flagViews, false, "generate a view type per struct type")
	flags.StringVar(&opts.Validate, flagValidate, "",
		"the `method` that ValidateKanon of each named type that is not a struct calls")
	flags.BoolVar(&opts.Canonical, flagCanonical, false,
		"make the decode of each struct type accept only the canonical encoding of a value")
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
