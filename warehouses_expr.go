//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package meergo

// Expr represents a subset of SQL expressions.
type Expr interface {
	expr()
}

// MultiExpr represents an SQL expression with a logical operator, which can be
// both And or Or, and a list of SQL expressions on which the operator is
// applied.
type MultiExpr struct {
	Operator LogicalOperator
	Operands []Expr
}

func (*MultiExpr) expr() {}

// NewMultiExpr returns a new MultiExpr expression with the given operator and
// operands.
func NewMultiExpr(operator LogicalOperator, operands []Expr) *MultiExpr {
	return &MultiExpr{Operator: operator, Operands: operands}
}

// LogicalOperator represents the logical operator of a MultiExpr.
type LogicalOperator int

const (
	OpAnd LogicalOperator = iota
	OpOr
)

// BaseExpr represents an SQL expression that refers to a property, on which an
// operator is applied, an eventually an operand, if the operator is binary.
type BaseExpr struct {
	Column   Column
	Operator Operator
	Values   []any // may be nil for unary expressions.
}

func (*BaseExpr) expr() {}

// NewBaseExpr returns a new BaseExpr expression that applies to the given
// column with the given operator and values.
// If the operator is unary, value should be nil.
func NewBaseExpr(column Column, operator Operator, values ...any) *BaseExpr {
	return &BaseExpr{Column: column, Operator: operator, Values: values}
}

// Operator presents a unary or binary operator of a BaseExpr.
type Operator int

const (
	OpIs                     Operator = iota // is
	OpIsNot                                  // is not
	OpIsLessThan                             // is less than
	OpIsLessThanOrEqualTo                    // is less than or equal to
	OpIsGreaterThan                          // is greater than
	OpIsGreaterThanOrEqualTo                 // is greater than or equal to
	OpIsBetween                              // is between
	OpIsNotBetween                           // is not between
	OpContains                               // contains
	OpDoesNotContain                         // does not contain
	OpIsOneOf                                // is one of
	OpIsNotOneOf                             // is not one of
	OpStartsWith                             // starts with
	OpEndsWith                               // ends with
	OpIsBefore                               // is before
	OpIsOnOrBefore                           // is on or before
	OpIsAfter                                // is after
	OpIsOnOrAfter                            // is on or after
	OpIsTrue                                 // is true
	OpIsFalse                                // is false
	OpIsNull                                 // is null
	OpIsNotNull                              // is not null
)
