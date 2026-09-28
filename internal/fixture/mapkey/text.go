// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package mapkey

//go:generate go tool kanon -type=Text

// Name is a named string.
type Name string

// Text has a map with keys of each string type. A byte slice has no ==
// and keys no map.
type Text struct {
	String map[string]string
	Name   map[Name]string
}
