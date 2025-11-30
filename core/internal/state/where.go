// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/types"
)

// Where represents a where expression.
type Where struct {
	Logical    WhereLogical     `json:"logical"`    // can be OpAnd or OpAny.
	Conditions []WhereCondition `json:"conditions"` // cannot be empty.
}

// Equal reports whether the receiver is equal to w.
func (where *Where) Equal(w *Where) bool {
	if where == nil && w == nil {
		return true
	}
	if where == nil || w == nil {
		return false
	}
	if where.Logical != w.Logical {
		return false
	}
	if len(where.Conditions) != len(w.Conditions) {
		return false
	}
	for i, c := range where.Conditions {
		c2 := w.Conditions[i]
		if !slices.Equal(c.Property, c2.Property) {
			return false
		}
		if c.Operator != c2.Operator {
			return false
		}
		if len(c.Values) != len(c2.Values) {
			return false
		}
		for j, v := range c.Values {
			v2 := c2.Values[j]
			switch v := v.(type) {
			case decimal.Decimal:
				if v2, ok := v2.(decimal.Decimal); !ok || !v.Equal(v2) {
					return false
				}
			default:
				if v != v2 {
					return false
				}
			}
		}
	}
	return true
}

func (where *Where) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	b.WriteString(`{"logical":"`)
	b.WriteString(where.Logical.String())
	b.WriteString(`","conditions":`)
	err := enc.Encode(where.Conditions)
	if err != nil {
		return nil, err
	}
	b.Truncate(b.Len() - 1)
	b.WriteString(`}`)
	return b.Bytes(), nil
}

// WhereLogical represents the logical operator of a where.
// It can be OpAnd or OpOr.
type WhereLogical int

const (
	OpAnd WhereLogical = iota // and
	OpOr                      // or
)

var jsonLogicals = []byte(`"And"Or"`)
var jsonLogicalsIndexes = [...]uint{0, 4}

// MarshalJSON returns the JSON representation of op.
func (op WhereLogical) MarshalJSON() ([]byte, error) {
	i := jsonLogicalsIndexes[op]
	j := jsonLogicalsIndexes[op+1] + 1
	return jsonLogicals[i:j], nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *WhereLogical) UnmarshalJSON(data []byte) error {
	k := bytes.Index(jsonLogicals, data)
	if k < 0 {
		return errors.New("invalid logical operator")
	}
	h := slices.Index(jsonLogicalsIndexes[:], uint(k))
	if h < 0 {
		return errors.New("invalid logical operator")
	}
	*op = WhereLogical(h)
	return nil
}

// String returns the string representation of op.
func (op WhereLogical) String() string {
	if op == OpAnd {
		return "And"
	}
	return "Or"
}

// WhereCondition represents the condition of a where.
type WhereCondition struct {
	Property []string      `json:"property"`         // property's path.
	Operator WhereOperator `json:"operator"`         // operator.
	Values   []any         `json:"values,omitempty"` // values.
}

// WhereOperator is a where operator.
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

	// OpExists and OpDoesNotExist must be placed at the end, as the convertWhere function in
	// the "core/datastore" package relies on the other operators being in their expected positions.
	OpExists       // exists
	OpDoesNotExist // does not exist
)

var jsonOperators = []byte(`"Is"IsNot"IsLessThan"IsLessThanOrEqualTo"IsGreaterThan"IsGreaterThanOrEqualTo"` +
	`IsBetween"OpIsNotBetween"Contains"DoesNotContain"IsOneOf"IsNotOneOf"StartsWith"EndsWith"IsBefore"` +
	`IsOnOrBefore"IsAfter"IsOnOrAfter"IsTrue"IsFalse"IsEmpty"IsNotEmpty"IsNull"IsNotNull"Exists"DoesNotExist"`)
var jsonOperatorsIndexes = [...]uint16{0, 3, 9, 20, 40, 54, 77, 87, 102, 111, 126, 134, 145, 156, 165, 174, 187, 195, 207, 214, 222, 230, 241, 248, 258, 265, 278}

// MarshalJSON returns the JSON representation of op.
func (op WhereOperator) MarshalJSON() ([]byte, error) {
	i := jsonOperatorsIndexes[op]
	j := jsonOperatorsIndexes[op+1] + 1
	return jsonOperators[i:j], nil
}

// String returns the string representation of op.
func (op WhereOperator) String() string {
	s, _ := op.MarshalJSON()
	return string(s[1 : len(s)-1])
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (op *WhereOperator) UnmarshalJSON(data []byte) error {
	k := bytes.Index(jsonOperators, data)
	if k < 0 {
		return errors.New("invalid operator")
	}
	h := slices.Index(jsonOperatorsIndexes[:], uint16(k))
	if h < 0 {
		return errors.New("invalid operator")
	}
	*op = WhereOperator(h)
	return nil
}

// JSONConditionValue represents a value in a Where condition that refers to a
// json property.
//
// - String holds the raw value as a string.
// - Number is non-nil if the String represents a numeric value.
type JSONConditionValue struct {
	String string
	Number *decimal.Decimal
}

// MarshalJSON returns the JSON representation of v.
func (v JSONConditionValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String)
}

// unmarshalConditionValue unmarshals a Where condition value and returns it.
// v is the value to unmarshal, and t is the type of the property.
func unmarshalConditionValue(v any, t types.Type) (any, error) {
	switch t.Kind() {
	case types.StringKind, types.UUIDKind, types.InetKind:
		return v, nil
	case types.IntKind:
		n, err := strconv.ParseInt(string(v.(json.Number)), 10, 64)
		if err != nil {
			return nil, err
		}
		return int(n), nil
	case types.UintKind:
		n, err := strconv.ParseUint(string(v.(json.Number)), 10, 64)
		if err != nil {
			return nil, err
		}
		return uint(n), nil
	case types.FloatKind:
		return strconv.ParseFloat(string(v.(json.Number)), t.BitSize())
	case types.DecimalKind:
		return decimal.Parse(v.(json.Number), 0, 0)
	case types.DateTimeKind:
		t, err := time.Parse(time.RFC3339Nano, v.(string))
		if err != nil {
			return nil, err
		}
		return t.UTC(), nil
	case types.DateKind:
		t, err := time.Parse(time.DateOnly, v.(string))
		if err != nil {
			return nil, err
		}
		t = t.UTC()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	case types.TimeKind:
		t, err := time.Parse("15:04:05.999999999", v.(string))
		if err != nil {
			return nil, err
		}
		return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
	case types.JSONKind:
		v := JSONConditionValue{String: v.(string)}
		if d, err := decimal.Parse(v.String, 0, 0); err == nil {
			v.Number = &d
		}
		return v, nil
	case types.ArrayKind:
		return unmarshalConditionValue(v, t.Elem())
	}
	panic(fmt.Sprintf("unexpected kind %s", t.Kind()))
}

// unmarshalWhere unmarshals a where and returns it.
func unmarshalWhere(b []byte, schema types.Type) (*Where, error) {
	var where *Where
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	err := dec.Decode(&where)
	if err != nil {
		return nil, err
	}
	// Normalize values.
	properties := schema.Properties()
	for _, c := range where.Conditions {
		p, err := properties.ByPathSlice(c.Property)
		if err != nil && p.Type.Kind() != types.JSONKind {
			return nil, err
		}
		for i, value := range c.Values {
			c.Values[i], err = unmarshalConditionValue(value, p.Type)
			if err != nil {
				return nil, err
			}
		}
	}
	return where, nil
}
