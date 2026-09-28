// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package kanon

import (
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// errPrefix begins every error message the package produces.
const errPrefix = "kanon: "

// sourceError is a failure tied to a declaration in the source file. Its
// message names the position and the struct or field, then the cause:
// "kanon: order.go:12:2: Order.Shape: interface type Shape needs the tag
// option types, which lists its concrete types".
type sourceError struct {
	// err is the cause. Its message starts with errPrefix, which Error
	// drops so the prefix appears once.
	err     error
	pos     token.Position
	subject string
}

// Error renders the position, the subject and the cause after a single
// "kanon: " prefix. The file is named by its base name.
func (e *sourceError) Error() string {
	var sb strings.Builder
	sb.WriteString(errPrefix)
	sb.WriteString(filepath.Base(e.pos.Filename))
	sb.WriteByte(':')
	sb.WriteString(strconv.Itoa(e.pos.Line))
	sb.WriteByte(':')
	sb.WriteString(strconv.Itoa(e.pos.Column))
	sb.WriteString(": ")
	sb.WriteString(e.subject)
	sb.WriteString(": ")
	sb.WriteString(strings.TrimPrefix(e.err.Error(), errPrefix))
	return sb.String()
}

// Unwrap returns the cause.
func (e *sourceError) Unwrap() error { return e.err }
