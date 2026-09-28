// Copyright ThesmOS B.V. 2026
// SPDX-License-Identifier: Apache-2.0

// Package iface declares fixtures with interfaces: in every place that a
// value can be, and with concrete types of every Go type that kanon
// encodes. The tag option types of a field lists the concrete types of its
// interfaces, and an interface encodes as the number of its concrete type,
// 0 for nil, and the value of that type. An interface field is present
// when it is not nil, and a length precedes its encoding.
//
// # Dependency position
//
// iface imports time from the standard library and the fixture packages
// codec and external, and its generated code the kanon runtime.
package iface
