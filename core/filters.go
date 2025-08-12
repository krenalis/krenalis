//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"

	"github.com/relvacode/iso8601"
)

// Filter represents a filter.
type Filter struct {
	Logical    FilterLogical     `json:"logical"`    // can be OpAnd or OpOr.
	Conditions []FilterCondition `json:"conditions"` // cannot be empty.
}

// FilterLogical represents the logical operator of a filter.
// It can be OpAnd or OpOr.
type FilterLogical string

const (
	OpAnd FilterLogical = "and"
	OpOr  FilterLogical = "or"
)

// convertLogicalToWhere converts a filter logical operator into a where logical
// operator.
func convertFilterLogicalToWhere(op FilterLogical) state.WhereLogical {
	if op == OpAnd {
		return state.OpAnd
	}
	return state.OpOr
}

// convertLogicalFromWhere converts a where logical operator into a filter
// logical operator.
func convertLogicalFromWhere(op state.WhereLogical) FilterLogical {
	if op == state.OpAnd {
		return OpAnd
	}
	return OpOr
}

// FilterCondition represents the condition of a filter.
type FilterCondition struct {
	Property string         `json:"property"`        // property's path.
	Operator FilterOperator `json:"operator"`        // operator.
	Values   []string       `json:"values,omitzero"` // values; a value cannot be longer than 60 runes and cannot contain the NUL byte.
}

// FilterOperator represents a filter condition operator.
type FilterOperator string

const (
	OpIs                     FilterOperator = "is"
	OpIsNot                  FilterOperator = "is not"
	OpIsLessThan             FilterOperator = "is less than"
	OpIsLessThanOrEqualTo    FilterOperator = "is less than or equal to"
	OpIsGreaterThan          FilterOperator = "is greater than"
	OpIsGreaterThanOrEqualTo FilterOperator = "is greater than or equal to"
	OpIsBetween              FilterOperator = "is between"
	OpIsNotBetween           FilterOperator = "is not between"
	OpContains               FilterOperator = "contains"
	OpDoesNotContain         FilterOperator = "does not contain"
	OpIsOneOf                FilterOperator = "is one of"
	OpIsNotOneOf             FilterOperator = "is not one of"
	OpStartsWith             FilterOperator = "starts with"
	OpEndsWith               FilterOperator = "ends with"
	OpIsBefore               FilterOperator = "is before"
	OpIsOnOrBefore           FilterOperator = "is on or before"
	OpIsAfter                FilterOperator = "is after"
	OpIsOnOrAfter            FilterOperator = "is on or after"
	OpIsTrue                 FilterOperator = "is true"
	OpIsFalse                FilterOperator = "is false"
	OpIsNull                 FilterOperator = "is null"
	OpIsNotNull              FilterOperator = "is not null"
	OpExists                 FilterOperator = "exists"
	OpDoesNotExist           FilterOperator = "does not exist"
)

// operators contains all the operators in their order.
var operators = [...]FilterOperator{
	OpIs, OpIsNot, OpIsLessThan, OpIsLessThanOrEqualTo, OpIsGreaterThan, OpIsGreaterThanOrEqualTo, OpIsBetween,
	OpIsNotBetween, OpContains, OpDoesNotContain, OpIsOneOf, OpIsNotOneOf, OpStartsWith, OpEndsWith, OpIsBefore,
	OpIsOnOrBefore, OpIsAfter, OpIsOnOrAfter, OpIsTrue, OpIsFalse, OpIsNull, OpIsNotNull, OpExists, OpDoesNotExist,
}

// convertOperatorFromWhere converts a where operator into a filter operator.
func convertOperatorFromWhere(op state.WhereOperator) FilterOperator {
	return operators[op]
}

// convertOperatorFromWhere converts a filter operator into a where operator.
func convertOperatorToWhere(op FilterOperator) state.WhereOperator {
	return state.WhereOperator(slices.Index(operators[:], op))
}

// convertFilterToWhere converts the provided filter into a where.
// The filter must have been validated using the validateFilter function with
// the same schema.
// Panics if the filter is nil or the schema is not valid.
func convertFilterToWhere(filter *Filter, schema types.Type) *state.Where {
	where := &state.Where{
		Logical:    convertFilterLogicalToWhere(filter.Logical),
		Conditions: make([]state.WhereCondition, len(filter.Conditions)),
	}
	for i, cond := range filter.Conditions {
		p, _, _ := retrieveFilterProperty(schema, cond.Property)
		var values []any
		if cond.Values != nil {
			values = make([]any, len(cond.Values))
		}
		kind := p.Type.Kind()
		if kind == types.ArrayKind {
			kind = p.Type.Elem().Kind()
		}
		for i, value := range cond.Values {
			var v any
			switch kind {
			case types.BooleanKind:
				v = value == "true"
			case types.IntKind:
				v, _ = parseInt(value)
			case types.UintKind:
				v, _ = parseUint(value)
			case types.FloatKind:
				v, _ = parseFloat(value, p.Type.BitSize())
			case types.DecimalKind:
				v, _ = parseDecimal(value)
			case types.DateTimeKind:
				v, _ = iso8601.ParseString(value)
			case types.DateKind:
				v, _ = time.Parse(time.DateOnly, value)
			case types.TimeKind:
				v, _ = util.ParseTime(value)
			case types.YearKind:
				v, _ = parseYear(value)
			case types.UUIDKind:
				v, _ = types.ParseUUID(value)
			case types.JSONKind:
				jv := state.JSONConditionValue{String: value}
				if d, err := decimal.Parse(jv.String, 0, 0); err == nil {
					jv.Number = &d
				}
				v = jv
			case types.InetKind:
				addr, _ := netip.ParseAddr(value)
				v = addr.String()
			case types.TextKind:
				v = value
			default:
				panic(fmt.Errorf("unexpected type for property %s", cond.Property))
			}
			values[i] = v
		}
		where.Conditions[i] = state.WhereCondition{
			Property: strings.Split(cond.Property, "."),
			Operator: convertOperatorToWhere(cond.Operator),
			Values:   values,
		}
	}
	return where
}

// convertWhereToFilter converts the provided where into a filter.
// Panics if where is nil or the schema is not valid.
func convertWhereToFilter(where *state.Where, schema types.Type) *Filter {
	filter := &Filter{
		Logical:    convertLogicalFromWhere(where.Logical),
		Conditions: make([]FilterCondition, len(where.Conditions)),
	}
	for i, cond := range where.Conditions {
		var values []string
		if cond.Values != nil {
			values = make([]string, len(cond.Values))
		}
		for i, value := range cond.Values {
			var v string
			switch value := value.(type) {
			case bool:
				v = strconv.FormatBool(value)
			case float64:
				p, _ := types.PropertyByPathSlice(schema, cond.Property)
				v = strconv.FormatFloat(value, 'g', -1, p.Type.BitSize())
			case int:
				v = strconv.FormatInt(int64(value), 10)
			case uint:
				v = strconv.FormatUint(uint64(value), 10)
			case decimal.Decimal:
				v = value.String()
			case time.Time:
				p, _ := types.PropertyByPathSlice(schema, cond.Property)
				switch p.Type.Kind() {
				case types.DateTimeKind:
					v = value.Format("2006-01-02T15:04:05.999999999")
				case types.DateKind:
					v = value.Format("2006-01-02")
				case types.TimeKind:
					v = value.Format("15:04:05.999999999")
				}
			case state.JSONConditionValue:
				v = value.String
			case string:
				v = value
			}
			values[i] = v
		}
		filter.Conditions[i] = FilterCondition{
			Property: strings.Join(cond.Property, "."),
			Operator: convertOperatorFromWhere(cond.Operator),
			Values:   values,
		}
	}
	return filter
}

// parseDecimal parses a decimal from s and returns the parsed decimal value and
// true. If s is not a valid decimal, it returns 0 and false.
func parseDecimal(s string) (decimal.Decimal, bool) {
	d, err := decimal.Parse(s, 0, 0)
	if err != nil {
		return decimal.Decimal{}, false
	}
	return d, true
}

// parseDecimalDigits parses the string s and returns the index of the first
// byte in s that is not a decimal digit (0-9).
func parseDecimalDigits(s string) int {
	i := 0
	for ; i < len(s); i++ {
		var c = s[i]
		if c < '0' || c > '9' {
			break
		}
	}
	return i
}

// parseFloat parses a float(n) from s with the provided bit size and returns
// the parsed float value and true. If s is not a valid float, it returns 0
// and false. bitSize can be 32 for float(32) or 64 for float(64).
func parseFloat(s string, bitSize int) (float64, bool) {
	if s == "0" {
		return 0, true
	}
	if !isFloatingPoint(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, bitSize)
	if err != nil {
		return 0, false
	}
	return f, true
}

// isFloatingPoint checks whether the string s represents a valid floating-point
// value.
func isFloatingPoint(s string) bool {
	if len(s) == 0 {
		return false
	}
	if s[0] == '-' || s[0] == '+' {
		s = s[1:]
	}
	i := parseDecimalDigits(s)
	if i == 0 {
		return false
	}
	if i == len(s) {
		return true
	}
	c := s[i]
	s = s[i+1:]
	if c == '.' {
		i = parseDecimalDigits(s)
		if i == 0 {
			return false
		}
		if i == len(s) {
			return true
		}
		c = s[i]
		s = s[i+1:]
	}
	if c != 'e' && c != 'E' || len(s) == 0 {
		return false
	}
	c = s[0]
	if c == '-' || c == '+' {
		s = s[1:]
	}
	i = parseDecimalDigits(s)
	return i == len(s)
}

// parseInt parses an int(64) from s and returns the int(64) value and true.
// If s is not a valid int(64), it returns 0 and false.
func parseInt(s string) (int, bool) {
	if len(s) == 0 {
		return 0, false
	}
	if s == "0" {
		return 0, true
	}
	sign := 1
	switch s[0] {
	case '-':
		sign = -1
		fallthrough
	case '+':
		s = s[1:]
	}
	un, valid := parseUint(s)
	if !valid {
		return 0, false
	}
	if sign < 0 && un > math.MaxInt64+1 {
		return 0, false
	}
	if sign > 0 && un > math.MaxInt64 {
		return 0, false
	}
	return sign * int(un), true
}

// parseYear parses a year from s and returns the year and true.
// If s is not a valid year, it returns 0 and false.
func parseYear(s string) (int, bool) {
	if l := len(s); l == 0 || l > 4 {
		return 0, false
	}
	var year int
	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		year = year*10 + int(c-'0')
		if year > types.MaxYear {
			return 0, false
		}
	}
	if year < types.MinYear {
		return 0, false
	}
	return year, true
}

// parseUint parses an uint(64) from s and returns the uint(64) value and true.
// If s is not a valid uint(64), it returns 0 and false.
func parseUint(s string) (uint, bool) {
	if len(s) == 0 {
		return 0, false
	}
	if s == "0" {
		return 0, true
	}
	var n uint
	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' || i == 0 && c == '0' {
			return 0, false
		}
		n2 := n*10 + uint(c-'0')
		if n2 < n {
			return 0, false
		}
		n = n2
	}
	return n, true
}

// retrieveFilterProperty retrieves a property located at a specific path within
// a schema and returns the property along with its path. If the path points to
// a json property, it returns the path to that json property.
// path must be a valid property path.
func retrieveFilterProperty(schema types.Type, path string) (types.Property, string, error) {
	p, err := types.PropertyByPath(schema, path)
	if err != nil {
		if p.Type.Kind() != types.JSONKind {
			return types.Property{}, "", err
		}
		path = err.(types.PathNotExistError).Path
		i := strings.LastIndexByte(path, '.')
		if i < 0 {
			return types.Property{}, "", err
		}
		path = path[:i]
	}
	return p, path, nil
}

// validateFilter checks the validity of a filter and returns its properties.
// Returns an error if the filter is not valid. Specifically, it returns a
// types.PathNotExistError if a path does not exist.
// Panics if the filter is nil or the schema is not valid.
func validateFilter(filter *Filter, schema types.Type) ([]string, error) {

	if op := filter.Logical; op != OpAnd && op != OpOr {
		return nil, fmt.Errorf("invalid logical operator %q", op)
	}
	if len(filter.Conditions) == 0 {
		return nil, errors.New("conditions are missing")
	}

	var properties []string

	for _, cond := range filter.Conditions {

		if !types.IsValidPropertyPath(cond.Property) {
			return nil, errors.New("property path is not valid")
		}

		p, path, err := retrieveFilterProperty(schema, cond.Property)
		if err != nil {
			return nil, err
		}

		if i, ok := slices.BinarySearch(properties, path); !ok {
			properties = slices.Insert(properties, i, path)
		}

		op := cond.Operator
		kind := p.Type.Kind()

		// Validate the operator and its kind.
		//
		// is                          : int, uint, float, decimal, datetime, date, time, year, uuid, json, inet, text
		// is not                      : int, uint, float, decimal, datetime, date, time, year, uuid, json, inet, text
		// is less than                : int, uint, float, decimal, json, text [^1]
		// is less than or equal to    : int, uint, float, decimal, json, text [^1]
		// is greater than             : int, uint, float, decimal, json, text [^1]
		// is greater than or equal to : int, uint, float, decimal, json, text [^1]
		// is between                  : int, uint, float, decimal, year, datetime, date, time, json, text [^1]
		// is not between              : int, uint, float, decimal, year, datetime, date, time, json, text [^1]
		// contains                    : json, text, array [^2]
		// does not contain            : json, text, array [^2]
		// is one of                   : int, uint, float, decimal, year, datetime, date, time, json, text
		// is not one of               : int, uint, float, decimal, year, datetime, date, time, json, text
		// starts with                 : json, text [^1]
		// ends with                   : json, text [^1]
		// is before                   : datetime, date, time, year
		// is on or before             : datetime, date, time, year
		// is after                    : datetime, date, time, year
		// is on or after              : datetime, date, time, year
		// is true                     : boolean, json
		// is false                    : boolean, json
		// is null                     : All types
		// is not null                 : All types
		// exists                      : All types [^3]
		// does not exist              : All types [^3]
		//
		// [1]: text with values is not supported.
		// [2]: array(T) is supported if T is a type that is supported by the 'is' operator.
		// [3]: only if the property is read-optional or 'json' with a non-empty path.
		//
		switch op {
		case OpIs, OpIsNot:
			switch kind {
			case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
				return nil, fmt.Errorf("operator %q cannot be used with boolean properties", op)
			}
		case OpIsBetween, OpIsNotBetween:
			switch kind {
			case types.TextKind:
				if p.Type.Values() != nil {
					return nil, fmt.Errorf("operator %q cannot be used with text type that has values", op)
				}
			case types.BooleanKind, types.UUIDKind, types.InetKind, types.ArrayKind, types.ObjectKind, types.MapKind:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpIsOneOf, OpIsNotOneOf:
			switch kind {
			case types.BooleanKind, types.UUIDKind, types.InetKind, types.ArrayKind, types.ObjectKind, types.MapKind:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpIsLessThan, OpIsLessThanOrEqualTo, OpIsGreaterThan, OpIsGreaterThanOrEqualTo:
			switch kind {
			case types.IntKind, types.UintKind, types.FloatKind, types.DecimalKind:
			case types.TextKind:
				if p.Type.Values() != nil {
					return nil, fmt.Errorf("operator %q cannot be used with text type that has values", op)
				}
			case types.JSONKind:
			default:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpContains, OpDoesNotContain:
			switch kind {
			case types.JSONKind, types.TextKind:
			case types.ArrayKind:
				switch k := p.Type.Elem().Kind(); k {
				case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
					return nil, fmt.Errorf("operator %q cannot be used with array(%s) properties", op, k)
				}
			default:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpStartsWith, OpEndsWith:
			switch kind {
			case types.JSONKind:
			case types.TextKind:
				if p.Type.Values() != nil {
					return nil, fmt.Errorf("operator %q cannot be used with text type that has values", op)
				}
			default:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpIsBefore, OpIsAfter, OpIsOnOrBefore, OpIsOnOrAfter:
			switch kind {
			case types.DateTimeKind, types.DateKind, types.TimeKind:
			case types.YearKind:
			case types.JSONKind:
			default:
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpIsTrue, OpIsFalse:
			if kind != types.BooleanKind && kind != types.JSONKind {
				return nil, fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
			}
		case OpIsNull, OpIsNotNull:
			if !p.Nullable && kind != types.JSONKind {
				return nil, fmt.Errorf("operator %q can only be used with nullable or json properties", op)
			}
		case OpExists, OpDoesNotExist:
			if !p.ReadOptional && path == cond.Property {
				return nil, fmt.Errorf("operator %q can only be used with read-optional properties or with json properties that include a JSON path", op)
			}
		default:
			return nil, fmt.Errorf("operator %q is not valid", op)
		}

		// Validate the values.
		switch op {
		case OpIsNull, OpIsNotNull, OpIsTrue, OpIsFalse, OpExists, OpDoesNotExist:
			if cond.Values != nil {
				return nil, fmt.Errorf("values cannot be used with the operator %q", op)
			}
		default:
			if len(cond.Values) != 1 {
				return nil, fmt.Errorf("only one value can be used with the operator %q", op)
			}
		case OpIsBetween, OpIsNotBetween:
			if len(cond.Values) != 2 {
				return nil, fmt.Errorf("two values must be used with the operator %q", op)
			}
		case OpIsOneOf, OpIsNotOneOf:
			if len(cond.Values) == 0 {
				return nil, fmt.Errorf("at least one value must be used with the operator %q", op)
			}
		}
		if cond.Values == nil {
			continue
		}
		var k = kind
		if kind == types.ArrayKind {
			k = p.Type.Elem().Kind()
		}
		for _, value := range cond.Values {
			if err := util.ValidateStringField("condition value", value, 60); err != nil {
				return nil, err
			}
			var valid bool
			switch k {
			case types.IntKind:
				_, valid = parseInt(value)
			case types.UintKind:
				_, valid = parseUint(value)
			case types.FloatKind:
				_, valid = parseFloat(value, p.Type.BitSize())
			case types.DecimalKind:
				_, valid = parseDecimal(value)
			case types.DateTimeKind:
				if t, err := iso8601.ParseString(value); err == nil {
					y := t.UTC().Year()
					valid = types.MinYear <= y && y <= types.MaxYear
				}
			case types.DateKind:
				if t, err := time.Parse(time.DateOnly, value); err == nil {
					y := t.UTC().Year()
					valid = types.MinYear <= y && y <= types.MaxYear
				}
			case types.TimeKind:
				_, valid = util.ParseTime(value)
			case types.YearKind:
				_, valid = parseYear(value)
			case types.UUIDKind:
				_, valid = types.ParseUUID(value)
			case types.JSONKind, types.TextKind:
				valid = utf8.ValidString(value)
			case types.InetKind:
				_, err := netip.ParseAddr(value)
				valid = err == nil
			default:
				return nil, fmt.Errorf("unexpected type for property %q", cond.Property)
			}
			if !valid {
				return nil, fmt.Errorf("value of the %q property is not a valid %s", cond.Property, k)
			}
		}
	}

	return properties, nil
}
