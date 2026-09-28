// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package imports

import "go.thesmos.sh/kanon/internal/fixture/math"

//go:generate go tool kanon -type=Angles

// Angles has fields of a type of the fixture package math, whose name the
// standard library's math takes in the generated code.
type Angles struct {
	Turn   math.Angle
	Angles []math.Angle
}
