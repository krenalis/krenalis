//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package warehouses

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
type LogicalOperator string

const (
	LogicalOperatorAnd LogicalOperator = "And"
	LogicalOperatorOr  LogicalOperator = "Or"
)

// BaseExpr represents an SQL expression that refers to a property, on which an
// operator is applied, an eventually an operand, if the operator is binary.
type BaseExpr struct {
	Column   Column
	Operator Operator
	Value    any // may be nil for unary expressions.
}

func (*BaseExpr) expr() {}

// NewBaseExpr returns a new BaseExpr expression that applies to the given
// column with the given operator and value.
// If the operator is unary, value should be nil.
func NewBaseExpr(column Column, operator Operator, value any) *BaseExpr {
	return &BaseExpr{Column: column, Operator: operator, Value: value}
}

// Operator presents a unary or binary operator of a BaseExpr.
type Operator string

const (
	OperatorEqual        Operator = "Equal"
	OperatorNotEqual     Operator = "NotEqual"
	OperatorGreater      Operator = "Greater"
	OperatorGreaterEqual Operator = "GreaterEqual"
	OperatorLess         Operator = "Less"
	OperatorLessEqual    Operator = "LessEqual"
	OperatorIsNull       Operator = "IsNull"
	OperatorIsNotNull    Operator = "IsNotNull"
)
