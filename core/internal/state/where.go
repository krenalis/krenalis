// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// WhereRule represents a condition or a nested where expression.
// It is implemented by *WhereCondition and *Where.
type WhereRule interface {
	whereRule()
}

// Where combines rules using a logical operator.
type Where struct {
	Operator WhereLogical `json:"operator"`
	Rules    []WhereRule  `json:"rules"`
}

// Equal reports whether the receiver is equal to other.
func (where *Where) Equal(other *Where) bool {
	if where == nil && other == nil {
		return true
	}
	if where == nil || other == nil {
		return false
	}
	if where.Operator != other.Operator {
		return false
	}
	if len(where.Rules) != len(other.Rules) {
		return false
	}
	for i, rule := range where.Rules {
		switch rule := rule.(type) {
		case *Where:
			otherGroup, ok := other.Rules[i].(*Where)
			if !ok || !rule.Equal(otherGroup) {
				return false
			}
		case *WhereCondition:
			otherCondition, ok := other.Rules[i].(*WhereCondition)
			if !ok || !rule.Equal(otherCondition) {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// MarshalJSON returns the JSON representation of where.
func (where *Where) MarshalJSON() ([]byte, error) {
	type plainWhere Where
	return json.Marshal((*plainWhere)(where))
}

// whereRule marks Where as a where rule.
func (*Where) whereRule() {}

// WhereLogical identifies the logical operator used by a Where.
type WhereLogical int

const (
	OpAnd WhereLogical = iota // and
	OpOr                      // or
)

// jsonLogicals contains the JSON representations of the logical operators.
var jsonLogicals = []byte(`"And"Or"`)

// jsonLogicalOffsets contains the start offset of each value in jsonLogicals
// followed by a terminal offset.
var jsonLogicalOffsets = [...]uint{0, 4, 7}

// MarshalJSON returns the JSON representation of op.
func (op WhereLogical) MarshalJSON() ([]byte, error) {
	if op < OpAnd || op > OpOr {
		return nil, fmt.Errorf("invalid logical operator %d", op)
	}
	i := jsonLogicalOffsets[op]
	j := jsonLogicalOffsets[op+1] + 1
	return jsonLogicals[i:j], nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *WhereLogical) UnmarshalJSON(data []byte) error {
	k := bytes.Index(jsonLogicals, data)
	if k < 0 {
		return errors.New("invalid logical operator")
	}
	h := slices.Index(jsonLogicalOffsets[:], uint(k))
	if h < 0 {
		return errors.New("invalid logical operator")
	}
	*op = WhereLogical(h)
	return nil
}

// String returns the string representation of op.
// It panics if op is not a valid WhereLogical value.
func (op WhereLogical) String() string {
	s, err := op.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return string(s[1 : len(s)-1])
}

// WhereCondition applies an operator to a property and zero or more values.
type WhereCondition struct {
	Property []string      `json:"property"`
	Operator WhereOperator `json:"operator"`
	Values   []any         `json:"values,omitzero"`
}

// Equal reports whether the receiver is equal to other.
func (condition *WhereCondition) Equal(other *WhereCondition) bool {
	if condition == nil && other == nil {
		return true
	}
	if condition == nil || other == nil {
		return false
	}
	if !slices.Equal(condition.Property, other.Property) {
		return false
	}
	if condition.Operator != other.Operator {
		return false
	}
	if len(condition.Values) != len(other.Values) {
		return false
	}
	for i, value := range condition.Values {
		otherValue := other.Values[i]
		switch value := value.(type) {
		case decimal.Decimal:
			if otherValue, ok := otherValue.(decimal.Decimal); !ok || !value.Equal(otherValue) {
				return false
			}
		case JSONConditionValue:
			otherValue, ok := otherValue.(JSONConditionValue)
			if !ok || value.String != otherValue.String || value.Number == nil != (otherValue.Number == nil) {
				return false
			}
			if value.Number != nil && !value.Number.Equal(*otherValue.Number) {
				return false
			}
		default:
			if !reflect.DeepEqual(value, otherValue) {
				return false
			}
		}
	}
	return true
}

// whereRule marks WhereCondition as a where rule.
func (*WhereCondition) whereRule() {}

// WhereOperator identifies an operator supported by WhereCondition.
type WhereOperator int

const (
	OpIs                     WhereOperator = iota // is
	OpIsNot                                       // is not
	OpIsLessThan                                  // is less than
	OpIsLessThanOrEqualTo                         // is less than or equal to
	OpIsGreaterThan                               // is greater than
	OpIsGreaterThanOrEqualTo                      // is greater than or equal to
	OpIsBetween                                   // is between
	OpIsNotBetween                                // is not between
	OpContains                                    // contains
	OpDoesNotContain                              // does not contain
	OpIsOneOf                                     // is one of
	OpIsNotOneOf                                  // is not one of
	OpStartsWith                                  // starts with
	OpEndsWith                                    // ends with
	OpIsBefore                                    // is before
	OpIsOnOrBefore                                // is on or before
	OpIsAfter                                     // is after
	OpIsOnOrAfter                                 // is on or after
	OpIsTrue                                      // is true
	OpIsFalse                                     // is false
	OpIsEmpty                                     // is empty
	OpIsNotEmpty                                  // is not empty
	OpIsNull                                      // is null
	OpIsNotNull                                   // is not null

	// OpExists and OpDoesNotExist must remain at the end because convertWhere in
	// core/internal/datastore relies on the positions of the preceding operators.
	OpExists       // exists
	OpDoesNotExist // does not exist
)

// jsonOperators contains the JSON representations of the condition operators.
var jsonOperators = []byte(`"Is"IsNot"IsLessThan"IsLessThanOrEqualTo"IsGreaterThan"IsGreaterThanOrEqualTo"` +
	`IsBetween"IsNotBetween"Contains"DoesNotContain"IsOneOf"IsNotOneOf"StartsWith"EndsWith"IsBefore"` +
	`IsOnOrBefore"IsAfter"IsOnOrAfter"IsTrue"IsFalse"IsEmpty"IsNotEmpty"IsNull"IsNotNull"Exists"DoesNotExist"`)

// jsonOperatorOffsets contains the start offset of each value in jsonOperators
// followed by a terminal offset.
var jsonOperatorOffsets = [...]uint16{
	0, 3, 9, 20, 40, 54, 77, 87, 100, 109, 124, 132, 143, 154,
	163, 172, 185, 193, 205, 212, 220, 228, 239, 246, 256, 263, 276,
}

// MarshalJSON returns the JSON representation of op.
func (op WhereOperator) MarshalJSON() ([]byte, error) {
	if op < OpIs || op > OpDoesNotExist {
		return nil, fmt.Errorf("invalid operator %d", op)
	}
	i := jsonOperatorOffsets[op]
	j := jsonOperatorOffsets[op+1] + 1
	return jsonOperators[i:j], nil
}

// String returns the string representation of op.
// It panics if op is not a valid WhereOperator value.
func (op WhereOperator) String() string {
	s, err := op.MarshalJSON()
	if err != nil {
		panic(err)
	}
	return string(s[1 : len(s)-1])
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *WhereOperator) UnmarshalJSON(data []byte) error {
	k := bytes.Index(jsonOperators, data)
	if k < 0 {
		return errors.New("invalid operator")
	}
	h := slices.Index(jsonOperatorOffsets[:], uint16(k))
	if h < 0 {
		return errors.New("invalid operator")
	}
	*op = WhereOperator(h)
	return nil
}

// JSONConditionValue represents a value in a Where condition that refers to a
// JSON property.
//
// - String holds the raw value.
// - Number is non-nil if String represents a numeric value.
type JSONConditionValue struct {
	String string
	Number *decimal.Decimal
}

// MarshalJSON returns the JSON representation of v.
func (v JSONConditionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String)
}

// whereConditionJSON is the persisted representation of a WhereCondition.
type whereConditionJSON struct {
	Property []string      `json:"property"`
	Operator WhereOperator `json:"operator"`
	Values   []json.Value  `json:"values,omitzero"`
}

// unmarshalConditionValue unmarshals a Where condition value and returns it.
// v is the value to unmarshal, and t is the type of the property.
func unmarshalConditionValue(v json.Value, t types.Type) (any, error) {
	switch t.Kind() {
	case types.StringKind, types.UUIDKind, types.IPKind:
		if !v.IsString() {
			return nil, errors.New("where condition value must be a string")
		}
		return v.String(), nil
	case types.IntKind:
		if t.IsUnsigned() {
			return v.Uint()
		}
		return v.Int()
	case types.FloatKind:
		return v.Float(t.BitSize())
	case types.DecimalKind:
		return v.Decimal(0, 0)
	case types.DateTimeKind:
		t, err := time.Parse(time.RFC3339Nano, v.String())
		if err != nil {
			return nil, err
		}
		return t.UTC(), nil
	case types.DateKind:
		t, err := time.Parse(time.DateOnly, v.String())
		if err != nil {
			return nil, err
		}
		t = t.UTC()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	case types.TimeKind:
		t, err := time.Parse("15:04:05.999999999", v.String())
		if err != nil {
			return nil, err
		}
		return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
	case types.YearKind:
		return v.Int()
	case types.JSONKind:
		if !v.IsString() {
			return nil, errors.New("where condition value must be a string")
		}
		value := JSONConditionValue{String: v.String()}
		if d, err := decimal.Parse(value.String, 0, 0); err == nil {
			value.Number = &d
		}
		return value, nil
	case types.ArrayKind:
		return unmarshalConditionValue(v, t.Elem())
	}
	return nil, fmt.Errorf("unexpected kind %s in where condition", t.Kind())
}

// unmarshalWhere unmarshals a Where value and returns it.
func unmarshalWhere(b []byte, schema types.Type) (*Where, error) {
	var value json.Value
	if err := json.Unmarshal(b, &value); err != nil {
		return nil, err
	}
	return unmarshalWhereGroup(value, schema.Properties())
}

// unmarshalWhereCondition unmarshals a WhereCondition value and returns it.
func unmarshalWhereCondition(value json.Value, properties types.Properties) (*WhereCondition, error) {
	var condition whereConditionJSON
	if err := value.Unmarshal(&condition); err != nil {
		return nil, err
	}
	p, err := properties.ByPathSlice(condition.Property)
	if err != nil && p.Type.Kind() != types.JSONKind {
		return nil, err
	}
	var values []any
	if condition.Values != nil {
		values = make([]any, len(condition.Values))
	}
	for i, value := range condition.Values {
		values[i], err = unmarshalConditionValue(value, p.Type)
		if err != nil {
			return nil, err
		}
	}
	return &WhereCondition{Property: condition.Property, Operator: condition.Operator, Values: values}, nil
}

// unmarshalWhereGroup unmarshals a Where group and returns it.
func unmarshalWhereGroup(value json.Value, properties types.Properties) (*Where, error) {

	if !value.IsObject() {
		return nil, errors.New("where group must be an object")
	}
	rawOperator, hasOperator := value.Get([]string{"operator"})
	rawRules, hasRules := value.Get([]string{"rules"})
	if !hasOperator || !hasRules {
		return nil, errors.New("where group must contain operator and rules")
	}
	if !rawRules.IsArray() {
		return nil, errors.New("where rules must be an array")
	}

	var operator WhereLogical
	if err := rawOperator.Unmarshal(&operator); err != nil {
		return nil, err
	}
	rules := make([]WhereRule, 0, rawRules.NumElement())

	for _, rawRule := range rawRules.Elements() {
		_, isGroup := rawRule.Get([]string{"rules"})
		_, isCondition := rawRule.Get([]string{"property"})
		switch {
		case isGroup && !isCondition:
			group, err := unmarshalWhereGroup(rawRule, properties)
			if err != nil {
				return nil, err
			}
			rules = append(rules, group)
		case isCondition && !isGroup:
			condition, err := unmarshalWhereCondition(rawRule, properties)
			if err != nil {
				return nil, err
			}
			rules = append(rules, condition)
		default:
			return nil, errors.New("where rule must be either a group or a condition")
		}
	}

	return &Where{Operator: operator, Rules: rules}, nil
}
