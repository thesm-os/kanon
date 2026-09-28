// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

//go:generate go tool kanon -type=Circle,Square

// Shape is the interface of the shapes.
type Shape interface {
	shape()
}

// Circle is a struct with a kanon codec that implements Shape with a value
// receiver, so that a *Circle implements it too.
type Circle struct {
	Radius float64
}

// shape marks Circle as a Shape.
func (Circle) shape() {}

// Square is a struct with a kanon codec whose pointer implements Shape.
type Square struct {
	Side  int32
	Label string
}

// shape marks *Square as a Shape.
func (*Square) shape() {}

// Patch is a struct without a kanon codec that implements Shape with a
// value receiver, which the code files encode as an inline struct.
type Patch struct {
	Cells []int32
}

// shape marks Patch as a Shape.
func (Patch) shape() {}
