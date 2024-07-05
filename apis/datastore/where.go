//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

// Where represents a condition in a query.
type Where struct {
	Logical    WhereLogical     // can be "all" or "any".
	Conditions []WhereCondition // cannot be empty.
}

// WhereLogical represents the logical operator of a where.
// It can be "all" or "any".
type WhereLogical string

// WhereCondition represents the condition of a where.
type WhereCondition struct {
	Property string // A property identifier or selector (e.g. "street1" or "traits.address.street1").
	Operator string // "is", "is not".
	Value    string // "Track", "Page", ...
}
