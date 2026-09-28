// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

package iface

import "go.thesmos.sh/kanon/internal/fixture/external"

//go:generate go tool kanon -type=Remote

// Remote has an interface of another package, whose concrete type another
// package declares too.
type Remote struct {
	Tagged external.Tagged `kanon:",types=external.Label"`
}
