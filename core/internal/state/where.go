// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"
	"strconv"
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
	Operator WhereLogical
	Rules    []WhereRule
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

// WhereCondition applies an operator to a property and zero or more values.
type WhereCondition struct {
	Property []string
	Operator WhereOperator
	Values   []any
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
		case json.Value:
			otherValue, ok := otherValue.(json.Value)
			if !ok || len(value) == 0 || len(otherValue) == 0 || !json.Valid(value) || !json.Valid(otherValue) {
				return false
			}
			if value.Kind() != otherValue.Kind() {
				return false
			}
			switch value.Kind() {
			case json.Number:
				number, err := value.Decimal(0, 0)
				if err != nil {
					return false
				}
				otherNumber, err := otherValue.Decimal(0, 0)
				if err != nil {
					return false
				}
				if !number.Equal(otherNumber) {
					return false
				}
			case json.String:
				if value.String() != otherValue.String() {
					return false
				}
			case json.True, json.False, json.Null:
				// Matching kinds identify the same boolean or null value.
			default:
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
	OpExists                                      // exists
	OpDoesNotExist                                // does not exist
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

// String returns the string representation of op.
// It panics if op is not a valid WhereOperator value.
func (op WhereOperator) String() string {
	if op < OpIs || op > OpDoesNotExist {
		panic(fmt.Errorf("invalid operator %d", op))
	}
	i := jsonOperatorOffsets[op] + 1
	j := jsonOperatorOffsets[op+1]
	return string(jsonOperators[i:j])
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

// whereConditionJSON is the persisted representation of a WhereCondition.
type whereConditionJSON struct {
	Property []string      `json:"property"`
	Operator WhereOperator `json:"operator"`
	Values   []json.Value  `json:"values,omitzero"`
}

// MarshalWhere marshals where into JSON using the provided schema.
// where must be valid and conform to schema.
func MarshalWhere(where *Where, schema types.Type) json.Value {
	var b bytes.Buffer
	marshalWhereGroup(&b, where, schema.Properties())
	return b.Bytes()
}

// marshalWhereCondition writes the JSON representation of condition to b.
func marshalWhereCondition(b *bytes.Buffer, condition *WhereCondition, properties types.Properties) {

	b.WriteString(`{"property":[`)
	for i, name := range condition.Property {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		b.WriteString(name)
		b.WriteByte('"')
	}
	b.WriteString(`],"operator":`)
	i := jsonOperatorOffsets[condition.Operator]
	j := jsonOperatorOffsets[condition.Operator+1] + 1
	b.Write(jsonOperators[i:j])

	if condition.Values != nil {
		p, _ := properties.ByPathSlice(condition.Property)
		t := p.Type
		if t.IsArray() {
			t = t.Elem()
		}
		b.WriteString(`,"values":[`)
		for i, value := range condition.Values {
			if i > 0 {
				b.WriteByte(',')
			}
			marshalWhereValue(b, value, t)
		}
		b.WriteByte(']')
	}

	b.WriteByte('}')

}

// marshalWhereGroup writes the JSON representation of where to b.
func marshalWhereGroup(b *bytes.Buffer, where *Where, properties types.Properties) {
	b.WriteString(`{"operator":`)
	b.Write(jsonLogicals[jsonLogicalOffsets[where.Operator] : jsonLogicalOffsets[where.Operator+1]+1])
	b.WriteString(`,"rules":[`)
	for i, rule := range where.Rules {
		if i > 0 {
			b.WriteByte(',')
		}
		switch rule := rule.(type) {
		case *Where:
			marshalWhereGroup(b, rule, properties)
		case *WhereCondition:
			marshalWhereCondition(b, rule, properties)
		}
	}
	b.WriteString("]}")
}

// marshalWhereValue writes a condition value according to its property type.
func marshalWhereValue(b *bytes.Buffer, value any, t types.Type) {
	switch v := value.(type) {
	case string:
		quoted, _ := json.Quote([]byte(v))
		b.Write(quoted)
	case json.Value:
		b.Write(v)
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'g', -1, t.BitSize()))
	case decimal.Decimal:
		encoded, err := v.MarshalJSON()
		if err != nil {
			panic(err)
		}
		b.Write(encoded)
	case time.Time:
		b.WriteByte('"')
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.UTC().Format(time.RFC3339Nano))
		case types.DateKind:
			b.WriteString(v.Format(time.DateOnly))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999999"))
		}
		b.WriteByte('"')
	default:
		panic(fmt.Errorf("unexpected where condition value %T", value))
	}
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
	if err != nil && !p.Type.IsJSON() {
		return nil, err
	}
	switch condition.Operator {
	case OpIsTrue, OpIsFalse, OpIsEmpty, OpIsNotEmpty, OpIsNull, OpIsNotNull, OpExists, OpDoesNotExist:
		if condition.Values != nil {
			return nil, errors.New("unary filter operator cannot have values")
		}
	case OpIsBetween, OpIsNotBetween:
		if len(condition.Values) != 2 {
			return nil, errors.New("filter interval requires two values")
		}
	case OpIsOneOf, OpIsNotOneOf:
		if len(condition.Values) == 0 {
			return nil, errors.New("filter membership requires at least one value")
		}
	default:
		if len(condition.Values) != 1 {
			return nil, errors.New("filter comparison requires one value")
		}
	}
	var values []any
	if condition.Values != nil {
		values = make([]any, len(condition.Values))
	}
	for i, value := range condition.Values {
		values[i], err = DecodeFilterOperand(value, p.Type, condition.Operator)
		if err != nil {
			return nil, err
		}
	}
	if (condition.Operator == OpIsBetween || condition.Operator == OpIsNotBetween) && p.Type.IsJSON() {
		if values[0].(json.Value).Kind() != values[1].(json.Value).Kind() {
			return nil, errors.New("JSON interval bounds must have the same kind")
		}
	}
	if p.Type.Kind() == types.DateTimeKind || p.Type.IsArray() && p.Type.Elem().Kind() == types.DateTimeKind {
		for i, value := range values {
			values[i] = value.(time.Time).UTC()
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
