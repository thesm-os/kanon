// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

//go:generate go tool kanon -type=Trees

// Trees has an any whose list names []any and map[string]any, which
// contain the interface again, so that a value is a tree as deep as its
// data, as a document of encoding/json is.
type Trees struct {
	Tree any `kanon:",types=string|int64|float64|bool|[]any|map[string]any"`
}
