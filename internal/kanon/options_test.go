// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon_test

import (
	"bytes"
	"flag"
	"testing"

	"go.dokimi.dev/assert"

	"go.thesmos.sh/kanon/internal/kanon"
)

func TestOptions(t *testing.T) {
	t.Parallel()
	t.Run("ParseOptions", func(t *testing.T) {
		t.Parallel()
		parses := []struct {
			name string
			args []string
			want kanon.Options
		}{
			{
				name: "returns the struct types of -type in flag order",
				args: []string{"-type=B,A"},
				want: kanon.Options{Types: []string{"B", "A"}, Args: []string{}},
			},
			{
				name: "returns the arguments after the flags",
				args: []string{"-type=A", "a.go"},
				want: kanon.Options{Types: []string{"A"}, Args: []string{"a.go"}},
			},
			{
				name: "reports -views",
				args: []string{"-type=A", "-views"},
				want: kanon.Options{Types: []string{"A"}, Args: []string{}, Views: true},
			},
			{
				name: "reports -version without -type",
				args: []string{"-version"},
				want: kanon.Options{Args: []string{}, Version: true},
			},
		}
		for _, tt := range parses {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				var out bytes.Buffer
				got, err := kanon.ParseOptions(tt.args, &out)
				assert.NoError(t, err, "ParseOptions parses the arguments")
				assert.Equal(t, got, tt.want, "ParseOptions returns the options")
				assert.Equal(t, out.String(), "", "ParseOptions writes nothing for arguments that parse")
			})
		}
		failures := []struct {
			name string
			args []string
			want string
		}{
			{
				name: "returns an error for an empty type", args: []string{"-type=A,,B"},
				want: "kanon: -type \"A,,B\" names an empty type",
			},
			{
				name: "returns an error for a type that -type names twice", args: []string{"-type=A,B,A"},
				want: "kanon: -type \"A,B,A\" names A twice",
			},
			{
				name: "returns an error for an unknown flag", args: []string{"-bogus"},
				want: "flag provided but not defined: -bogus",
			},
		}
		for _, tt := range failures {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				_, err := kanon.ParseOptions(tt.args, &bytes.Buffer{})
				assert.HasError(t, err, "ParseOptions fails for the arguments")
				assert.Equal(t, err.Error(), tt.want, "ParseOptions states why the arguments fail")
			})
		}
		t.Run("returns flag.ErrHelp for -h", func(t *testing.T) {
			t.Parallel()
			_, err := kanon.ParseOptions([]string{"-h"}, &bytes.Buffer{})
			assert.ErrorIs(t, err, flag.ErrHelp, "ParseOptions returns flag.ErrHelp")
		})
		t.Run("writes the usage for -h", func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			_, _ = kanon.ParseOptions([]string{"-h"}, &out)
			assert.HasPrefix(t, out.String(), "usage: kanon -type=T[,T...] [-views] [file]\n",
				"ParseOptions writes the synopsis first")
			assert.Contains(t, out.String(), "-views", "ParseOptions writes the flags")
		})
	})
}
