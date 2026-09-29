// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package view

//go:generate go tool kanon -views -type=Page

// Page has fields of kinds that a view does not read, a slice of integers
// and a slice of byte slices, so that its view type has no method that
// reads a field, and its IndexKanon returns the index of the view.
type Page struct {
	Items []uint64
	Path  [][]byte
}
