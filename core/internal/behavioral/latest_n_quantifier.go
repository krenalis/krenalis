// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"fmt"
	"strconv"

	"github.com/krenalis/krenalis/warehouses"
)

// LatestNQuantifier preserves a requested quantifier on complete latest blocks.
// Value and Upper are canonical natural digits, not JSON or bounded integers.
// This keeps large redundant thresholds exact without large-integer parsing.
// All is the zero value; Some and All have no operator or operands.
type LatestNQuantifier struct {
	Kind     LatestNQuantifierKind
	Operator Comparison
	Value    string
	Upper    string
}

// Accepted validates the quantifier and derives its bounded inclusive range.
// Nil represents All. Cardinality operands stay unchanged for explanations.
func (quantifier LatestNQuantifier) Accepted(size int) (*warehouses.CountRange, error) {

	if size < 1 || size > 10 {
		return nil, fmt.Errorf("latestN size must be from 1 to 10")
	}
	if quantifier.Kind == LatestAll || quantifier.Kind == LatestSome {
		if quantifier.Operator != 0 || quantifier.Value != "" || quantifier.Upper != "" {
			return nil, fmt.Errorf("latestN all and some admit only kind")
		}
		if quantifier.Kind == LatestAll {
			return nil, nil
		}
		upper := int64(size)
		return &warehouses.CountRange{Lower: 1, Upper: &upper}, nil
	}
	if quantifier.Kind != LatestCardinality {
		return nil, fmt.Errorf("invalid latestN quantifier kind")
	}
	switch quantifier.Operator {
	case Equal, GreaterOrEqual, LessOrEqual, Between:
		// These are the four canonical cardinalities.
	default:
		return nil, fmt.Errorf("invalid latestN cardinality operator")
	}
	values := []string{quantifier.Value}
	if quantifier.Operator == Between {
		values = append(values, quantifier.Upper)
	} else if quantifier.Upper != "" {
		return nil, fmt.Errorf("latestN upper operand requires between")
	}
	for _, value := range values {
		if value == "" || len(value) > 1 && value[0] == '0' {
			return nil, fmt.Errorf("invalid canonical cardinality natural")
		}
		for i := range value {
			if value[i] < '0' || value[i] > '9' {
				return nil, fmt.Errorf("invalid canonical cardinality natural")
			}
		}
	}
	if quantifier.Operator == Between && naturalLess(quantifier.Upper, quantifier.Value) {
		return nil, fmt.Errorf("inverted latestN cardinality interval")
	}
	if quantifier.Operator != LessOrEqual && naturalLess(strconv.Itoa(size), quantifier.Value) {
		return nil, fmt.Errorf("latestN cardinality is statically incompatible with size")
	}

	// Every executable endpoint is at most N; retain larger requested digits.
	bounded := func(value string) int64 {
		if len(value) > 2 {
			return int64(size)
		}
		n, _ := strconv.Atoi(value) // At most two validated digits.
		return int64(min(n, size))
	}
	lower, upper := int64(0), int64(size)
	switch quantifier.Operator {
	case Equal:
		lower, upper = bounded(quantifier.Value), bounded(quantifier.Value)
	case GreaterOrEqual:
		lower = bounded(quantifier.Value)
	case LessOrEqual:
		upper = bounded(quantifier.Value)
	case Between:
		lower, upper = bounded(quantifier.Value), bounded(quantifier.Upper)
	default:
		return nil, fmt.Errorf("core/internal/behavioral: unexpected latestN cardinality operator %d", quantifier.Operator)
	}

	return &warehouses.CountRange{Lower: lower, Upper: &upper}, nil
}

// LatestNQuantifierKind identifies the three canonical block quantifiers.
type LatestNQuantifierKind uint8

const (
	LatestAll LatestNQuantifierKind = iota
	LatestSome
	LatestCardinality
)

// naturalLess compares validated canonical naturals without a numeric limit.
func naturalLess(left, right string) bool {
	if len(left) != len(right) {
		return len(left) < len(right)
	}
	return left < right
}
