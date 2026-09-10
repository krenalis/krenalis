// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/relvacode/iso8601"
)

// Filter complexity limits.
const (
	maxFilterDepth     = 4
	maxFilterRuleCount = 100
)

// Filter represents a logical expression whose rules are combined
// using AND or OR.
type Filter struct {
	Operator FilterLogical `json:"operator"`
	Rules    []FilterRule  `json:"rules"`
}

// FilterRule represents a condition or a nested filter group.
// It is implemented by *FilterCondition and *Filter.
type FilterRule interface {
	filterRule()
}

// MarshalJSON returns the JSON representation of filter.
func (filter *Filter) MarshalJSON() ([]byte, error) {
	if filter == nil {
		return nil, errors.New("filter cannot be nil")
	}
	var b json.Buffer
	err := marshalFilterJSON(&b, filter)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// UnmarshalJSON sets filter to the filter group represented by data.
func (filter *Filter) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	group, _, err := unmarshalFilterRule(dec, 1, expectFilterGroup)
	if err != nil {
		return err
	}
	*filter = *(group.(*Filter))
	return nil
}

// filterRule marks Filter as a filter rule.
func (*Filter) filterRule() {}

// FilterLogical represents the logical operator of a filter.
// It can be OpAnd or OpOr.
type FilterLogical string

const (
	OpAnd FilterLogical = "and"
	OpOr  FilterLogical = "or"
)

// convertLogicalToWhere converts a filter logical operator into a where logical
// operator.
func convertLogicalToWhere(op FilterLogical) state.WhereLogical {
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

// FilterCondition represents a single filter condition.
type FilterCondition struct {
	// Property path.
	Property string `json:"property"`

	// Operator to apply.
	Operator FilterOperator `json:"operator"`

	// Values.
	// If the property has a string type with allowed values, each value must be one of the allowed values.
	// In all other cases, each value must be at most 60 runes and must not contain the NUL byte.
	Values []string `json:"values,omitzero"`
}

// MarshalJSON returns the JSON representation of condition.
func (condition *FilterCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter condition cannot be nil")
	}
	var b json.Buffer
	err := marshalFilterConditionJSON(&b, condition)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// filterRule marks FilterCondition as a filter rule.
func (*FilterCondition) filterRule() {}

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
	OpIsEmpty                FilterOperator = "is empty"
	OpIsNotEmpty             FilterOperator = "is not empty"
	OpIsNull                 FilterOperator = "is null"
	OpIsNotNull              FilterOperator = "is not null"
	OpExists                 FilterOperator = "exists"
	OpDoesNotExist           FilterOperator = "does not exist"
)

// operators contains all the operators in their order.
var operators = [...]FilterOperator{
	OpIs, OpIsNot, OpIsLessThan, OpIsLessThanOrEqualTo, OpIsGreaterThan, OpIsGreaterThanOrEqualTo, OpIsBetween,
	OpIsNotBetween, OpContains, OpDoesNotContain, OpIsOneOf, OpIsNotOneOf, OpStartsWith, OpEndsWith, OpIsBefore,
	OpIsOnOrBefore, OpIsAfter, OpIsOnOrAfter, OpIsTrue, OpIsFalse, OpIsEmpty, OpIsNotEmpty, OpIsNull, OpIsNotNull,
	OpExists, OpDoesNotExist,
}

// filterValidation contains the state shared while validating a filter tree.
type filterValidation struct {
	paths      []string
	properties types.Properties
	role       state.Role
	ruleCount  int
	target     state.Target
}

// convertOperatorFromWhere converts a where operator into a filter operator.
func convertOperatorFromWhere(op state.WhereOperator) FilterOperator {
	return operators[op]
}

// convertOperatorToWhere converts a filter operator into a where operator.
func convertOperatorToWhere(op FilterOperator) state.WhereOperator {
	return state.WhereOperator(slices.Index(operators[:], op))
}

// convertFilterToWhere converts the provided filter into a where.
// The filter must have been validated using the validateFilter function with
// the same schema. Panics if the filter is nil or the schema is not valid.
func convertFilterToWhere(filter *Filter, schema types.Type) *state.Where {

	where := &state.Where{
		Operator: convertLogicalToWhere(filter.Operator),
		Rules:    make([]state.WhereRule, len(filter.Rules)),
	}
	properties := schema.Properties()

	for i, rule := range filter.Rules {
		if group, ok := rule.(*Filter); ok {
			where.Rules[i] = convertFilterToWhere(group, schema)
			continue
		}
		cond := rule.(*FilterCondition)
		p, _, _ := resolveFilterProperty(properties, cond.Property)
		var values []any
		if len(cond.Values) > 0 {
			values = make([]any, len(cond.Values))
		}
		t := p.Type
		if t.Kind() == types.ArrayKind {
			t = t.Elem()
		}
		kind := t.Kind()
		for i, value := range cond.Values {
			var v any
			switch kind {
			case types.StringKind:
				v = value
			case types.BooleanKind:
				v = value == "true"
			case types.IntKind:
				if t.IsUnsigned() {
					v, _ = parseUnsigned(value)
				} else {
					v, _ = parseInt(value)
				}
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
			case types.IPKind:
				addr, _ := netip.ParseAddr(value)
				v = addr.String()
			default:
				panic(fmt.Errorf("unexpected type for property %s", cond.Property))
			}
			values[i] = v
		}
		where.Rules[i] = &state.WhereCondition{
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
		Operator: convertLogicalFromWhere(where.Operator),
		Rules:    make([]FilterRule, len(where.Rules)),
	}
	properties := schema.Properties()

	for i, rule := range where.Rules {
		if group, ok := rule.(*state.Where); ok {
			filter.Rules[i] = convertWhereToFilter(group, schema)
			continue
		}
		cond := rule.(*state.WhereCondition)
		var values []string
		if len(cond.Values) > 0 {
			values = make([]string, len(cond.Values))
		}
		for i, value := range cond.Values {
			var v string
			switch value := value.(type) {
			case string:
				v = value
			case bool:
				v = strconv.FormatBool(value)
			case float64:
				p, _ := properties.ByPathSlice(cond.Property)
				v = strconv.FormatFloat(value, 'g', -1, p.Type.BitSize())
			case int:
				v = strconv.FormatInt(int64(value), 10)
			case uint:
				v = strconv.FormatUint(uint64(value), 10)
			case decimal.Decimal:
				v = value.String()
			case time.Time:
				p, _ := properties.ByPathSlice(cond.Property)
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
			}
			values[i] = v
		}
		filter.Rules[i] = &FilterCondition{
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

// marshalFilterConditionJSON appends the JSON representation of condition to
// b.
func marshalFilterConditionJSON(b *json.Buffer, condition *FilterCondition) error {
	b.WriteString(`{"property":`)
	err := b.Encode(condition.Property)
	if err != nil {
		return err
	}
	b.WriteString(`,"operator":`)
	err = b.Encode(condition.Operator)
	if err != nil {
		return err
	}
	if condition.Values != nil {
		b.WriteString(`,"values":[`)
		for i, value := range condition.Values {
			if i > 0 {
				b.WriteByte(',')
			}
			err = b.Encode(value)
			if err != nil {
				return err
			}
		}
		b.WriteByte(']')
	}
	b.WriteByte('}')
	return nil
}

// marshalFilterJSON appends the JSON representation of filter to b.
func marshalFilterJSON(b *json.Buffer, filter *Filter) error {
	b.WriteString(`{"operator":`)
	err := b.Encode(filter.Operator)
	if err != nil {
		return err
	}
	b.WriteString(`,"rules":[`)
	for i, rule := range filter.Rules {
		if i > 0 {
			b.WriteByte(',')
		}
		switch rule := rule.(type) {
		case *Filter:
			if rule == nil {
				return errors.New("filter group cannot be nil")
			}
			err = marshalFilterJSON(b, rule)
			if err != nil {
				return err
			}
		case *FilterCondition:
			if rule == nil {
				return errors.New("filter condition cannot be nil")
			}
			err = marshalFilterConditionJSON(b, rule)
			if err != nil {
				return err
			}
		case nil:
			return errors.New("filter rule cannot be nil")
		default:
			return fmt.Errorf("unsupported filter rule type %T", rule)
		}
	}
	b.WriteString(`]}`)
	return nil
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
	un, valid := parseUnsigned(s)
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

// parseUnsigned parses an unsigned int(64) from s and returns the unsigned
// int(64) value and true.
// If s is not a valid unsigned int(64), it returns 0 and false.
func parseUnsigned(s string) (uint, bool) {
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

// resolveFilterProperty resolves path against properties.
//
// It returns the property at path and its schema path or, if path extends into
// a JSON property, that JSON property and its schema path. path must be a valid
// property path.
func resolveFilterProperty(properties types.Properties, path string) (types.Property, string, error) {
	p, err := properties.ByPath(path)
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

// Filter rule decoding modes.
const (
	expectAnyFilterRule = false
	expectFilterGroup   = true
)

// unmarshalFilterRule unmarshals a filter rule and returns it and its number of
// descendant rules. If expectGroup is true, it accepts only a filter group.
func unmarshalFilterRule(dec *json.Decoder, depth int, expectGroup bool) (FilterRule, int, error) {

	tok, err := dec.ReadToken()
	if err != nil {
		return nil, 0, err
	}
	if tok.Kind() != json.Object {
		return nil, 0, errors.New("filter rule must be an object")
	}

	var operator string
	var property string
	var rules []FilterRule
	var values []string
	var descendantRuleCount int
	var hasOperator, hasProperty, hasRules, hasValues bool

	for dec.PeekKind() != '}' {
		tok, err = dec.ReadToken()
		if err != nil {
			return nil, 0, err
		}
		name := tok.String()
		switch name {
		case "operator":
			if hasOperator {
				return nil, 0, fmt.Errorf("duplicate filter property %q", name)
			}
			hasOperator = true
			err = dec.Decode(&operator)
		case "property":
			if hasProperty {
				return nil, 0, fmt.Errorf("duplicate filter property %q", name)
			}
			hasProperty = true
			if expectGroup {
				err = dec.SkipValue()
			} else {
				err = dec.Decode(&property)
			}
		case "rules":
			if hasRules {
				return nil, 0, fmt.Errorf("duplicate filter property %q", name)
			}
			if depth > maxFilterDepth {
				return nil, 0, fmt.Errorf("filter exceeds maximum depth of %d", maxFilterDepth)
			}
			hasRules = true
			rules, descendantRuleCount, err = unmarshalFilterRules(dec, depth)
		case "values":
			if hasValues {
				return nil, 0, fmt.Errorf("duplicate filter property %q", name)
			}
			hasValues = true
			if expectGroup {
				err = dec.SkipValue()
			} else {
				err = dec.Decode(&values)
			}
		default:
			return nil, 0, fmt.Errorf("unknown filter property %q", name)
		}
		if err != nil {
			return nil, 0, err
		}
	}

	_ = dec.SkipToken() // skip '}'

	if hasRules && hasProperty {
		return nil, 0, errors.New(`filter rule cannot contain both "rules" and "property"`)
	}
	if hasRules {
		if hasValues {
			return nil, 0, errors.New(`unknown filter property "values"`)
		}
		if !hasOperator {
			return nil, 0, errors.New("filter group must contain operator and rules")
		}
		return &Filter{Operator: FilterLogical(operator), Rules: rules}, descendantRuleCount, nil
	}
	if expectGroup {
		if hasProperty || hasValues {
			return nil, 0, errors.New("filter must be a group")
		}
		return nil, 0, errors.New("filter group must contain operator and rules")
	}
	if hasProperty {
		if !hasOperator {
			return nil, 0, errors.New("filter condition must contain property and operator")
		}
		return &FilterCondition{Property: property, Operator: FilterOperator(operator), Values: values}, 0, nil
	}

	return nil, 0, errors.New(`filter rule must contain either "rules" or "property"`)
}

// unmarshalFilterRules unmarshals the rules of a filter group and returns them
// and their total rule count.
func unmarshalFilterRules(dec *json.Decoder, depth int) ([]FilterRule, int, error) {

	tok, err := dec.ReadToken()
	if err != nil {
		return nil, 0, err
	}
	if tok.Kind() != json.Array {
		return nil, 0, errors.New("filter rules must be an array")
	}

	rules := []FilterRule{}
	ruleCount := 0
	for dec.PeekKind() != ']' {
		rule, descendantRuleCount, err := unmarshalFilterRule(dec, depth+1, expectAnyFilterRule)
		if err != nil {
			return nil, 0, err
		}
		ruleCount += descendantRuleCount + 1
		if ruleCount > maxFilterRuleCount {
			return nil, 0, fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)
		}
		rules = append(rules, rule)
	}

	// Consume ']'.
	_ = dec.SkipToken()

	return rules, ruleCount, nil
}

// validateFilter checks the validity of a filter and returns the referenced
// property paths.
// Returns an error if the filter is not valid. Specifically, it returns a
// types.PathNotExistError if a path does not exist.
// Panics if the filter is nil or the schema is not valid.
func validateFilter(filter *Filter, schema types.Type, role state.Role, target state.Target) ([]string, error) {
	validation := filterValidation{properties: schema.Properties(), role: role, target: target}
	if err := validateFilterGroup(filter, &validation, 1); err != nil {
		return nil, err
	}
	return validation.paths, nil
}

// validateFilterCondition checks the validity of a filter condition and
// returns its property path.
func validateFilterCondition(cond *FilterCondition, validation *filterValidation) (string, error) {

	// disallowJSON indicates whether JSON properties are disallowed.
	disallowJSON := validation.role == state.Destination && validation.target == state.TargetUser

	// disallowEmptyOnObject indicates whether the "is empty" and "is not empty"
	// operators are disallowed on object properties.
	disallowEmptyOnObject := validation.role == state.Destination && validation.target == state.TargetUser || validation.target == state.TargetEvent

	if !types.IsValidPropertyPath(cond.Property) {
		return "", errors.New("property path is not valid")
	}

	p, path, err := resolveFilterProperty(validation.properties, cond.Property)
	if err != nil {
		return "", err
	}

	op := cond.Operator
	kind := p.Type.Kind()

	if disallowJSON && kind == types.JSONKind {
		return "", fmt.Errorf("property %q has type json, which is not supported in data warehouse exports", path)
	}

	// Validate the operator for the property kind.
	//
	// is                          : int, float, decimal, datetime, date, time, year, uuid, json, ip, string
	// is not                      : int, float, decimal, datetime, date, time, year, uuid, json, ip, string
	// is less than                : int, float, decimal, json, string [^1]
	// is less than or equal to    : int, float, decimal, json, string [^1]
	// is greater than             : int, float, decimal, json, string [^1]
	// is greater than or equal to : int, float, decimal, json, string [^1]
	// is between                  : int, float, decimal, year, datetime, date, time, json, string [^1]
	// is not between              : int, float, decimal, year, datetime, date, time, json, string [^1]
	// contains                    : json, string, array [^2]
	// does not contain            : json, string, array [^2]
	// is one of                   : int, float, decimal, year, datetime, date, time, json, string
	// is not one of               : int, float, decimal, year, datetime, date, time, json, string
	// starts with                 : json, string [^1]
	// ends with                   : json, string [^1]
	// is before                   : datetime, date, time, year
	// is on or before             : datetime, date, time, year
	// is after                    : datetime, date, time, year
	// is on or after              : datetime, date, time, year
	// is true                     : boolean, json
	// is false                    : boolean, json
	// is empty                    : json, string, object, array, map [^3]
	// is not empty                : json, string, object, array, map [^3]
	// is null                     : All types [^4]
	// is not null                 : All types [^4]
	// exists                      : All types [^5]
	// does not exist              : All types [^5]
	//
	// [1]: Strings with allowed values are not supported.
	// [2]: array(T) is supported if T is supported by the "is" operator.
	// [3]: Object properties are not supported for destination pipelines on users or for pipelines on events.
	// [4]: Only if the property is nullable or has type JSON.
	// [5]: Only if the property is read-optional or the path continues inside a JSON property.
	//
	switch op {
	case OpIs, OpIsNot:
		switch kind {
		case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsBetween, OpIsNotBetween:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.BooleanKind, types.UUIDKind, types.IPKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsOneOf, OpIsNotOneOf:
		switch kind {
		case types.BooleanKind, types.UUIDKind, types.IPKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsLessThan, OpIsLessThanOrEqualTo, OpIsGreaterThan, OpIsGreaterThanOrEqualTo:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.IntKind, types.FloatKind, types.DecimalKind:
		case types.JSONKind:
		default:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpContains, OpDoesNotContain:
		switch kind {
		case types.StringKind, types.JSONKind:
		case types.ArrayKind:
			switch k := p.Type.Elem().Kind(); k {
			case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
				return "", fmt.Errorf("operator %q cannot be used with array(%s) properties", op, k)
			}
		default:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpStartsWith, OpEndsWith:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.JSONKind:
		default:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsBefore, OpIsAfter, OpIsOnOrBefore, OpIsOnOrAfter:
		switch kind {
		case types.DateTimeKind, types.DateKind, types.TimeKind:
		case types.YearKind:
		case types.JSONKind:
		default:
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsTrue, OpIsFalse:
		if kind != types.BooleanKind && kind != types.JSONKind {
			return "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsEmpty, OpIsNotEmpty:
		switch kind {
		case types.StringKind:
			if values := p.Type.Values(); len(values) > 0 {
				if !slices.Contains(values, "") {
					return "", fmt.Errorf("operator %q cannot be used on string properties that exclude the empty string from allowed values", op)
				}
			}
		case types.JSONKind, types.ArrayKind, types.MapKind:
		case types.ObjectKind:
			if disallowEmptyOnObject {
				if validation.target == state.TargetEvent {
					return "", fmt.Errorf("operator %q cannot be used on object properties for pipelines on events", op)
				}
				return "", fmt.Errorf("operator %q cannot be used on object properties for destination pipelines on users", op)
			}
		default:
			return "", fmt.Errorf("operator %q can only be used with json, string, object, array, and map properties", op)
		}
	case OpIsNull, OpIsNotNull:
		if !p.Nullable && kind != types.JSONKind {
			return "", fmt.Errorf("operator %q can only be used with nullable or json properties", op)
		}
	case OpExists, OpDoesNotExist:
		if !p.ReadOptional && path == cond.Property {
			return "", fmt.Errorf("operator %q can only be used with read-optional properties or with json properties that include a JSON path", op)
		}
	default:
		return "", fmt.Errorf("operator %q is not valid", op)
	}

	// Validate the values.
	switch op {
	case OpIsTrue, OpIsFalse, OpIsEmpty, OpIsNotEmpty, OpIsNull, OpIsNotNull, OpExists, OpDoesNotExist:
		if cond.Values != nil {
			return "", fmt.Errorf("values cannot be used with the operator %q", op)
		}
	default:
		if len(cond.Values) != 1 {
			return "", fmt.Errorf("only one value can be used with the operator %q", op)
		}
	case OpIsBetween, OpIsNotBetween:
		if len(cond.Values) != 2 {
			return "", fmt.Errorf("two values must be used with the operator %q", op)
		}
	case OpIsOneOf, OpIsNotOneOf:
		if len(cond.Values) == 0 {
			return "", fmt.Errorf("at least one value must be used with the operator %q", op)
		}
	}
	if cond.Values == nil {
		return path, nil
	}
	// Handle string properties with allowed values separately.
	if kind == types.StringKind {
		if values := p.Type.Values(); values != nil {
			for _, value := range cond.Values {
				if !slices.Contains(values, value) {
					return "", fmt.Errorf("value of the %q property is not among the allowed values", cond.Property)
				}
			}
			return path, nil
		}
	}
	t := p.Type
	if t.Kind() == types.ArrayKind {
		t = t.Elem()
	}
	k := t.Kind()
	for _, value := range cond.Values {
		if err := util.ValidateStringField("condition value", value, 60); err != nil {
			return "", err
		}
		var valid bool
		switch k {
		case types.StringKind, types.JSONKind:
			valid = utf8.ValidString(value)
		case types.IntKind:
			if t.IsUnsigned() {
				_, valid = parseUnsigned(value)
			} else {
				_, valid = parseInt(value)
			}
		case types.FloatKind:
			_, valid = parseFloat(value, t.BitSize())
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
		case types.IPKind:
			_, err := netip.ParseAddr(value)
			valid = err == nil
		default:
			return "", fmt.Errorf("unexpected type for property %q", cond.Property)
		}
		if !valid {
			return "", fmt.Errorf("value of the %q property is not a valid %s", cond.Property, k)
		}
	}

	return path, nil
}

// validateFilterGroup checks the validity of a filter group and its rules.
func validateFilterGroup(filter *Filter, validation *filterValidation, depth int) error {

	if depth > maxFilterDepth {
		return fmt.Errorf("filter exceeds maximum depth of %d", maxFilterDepth)
	}
	if op := filter.Operator; op != OpAnd && op != OpOr {
		return fmt.Errorf("invalid logical operator %q", op)
	}
	if len(filter.Rules) == 0 {
		return errors.New("rules are missing")
	}

	for _, rule := range filter.Rules {
		validation.ruleCount++
		if validation.ruleCount > maxFilterRuleCount {
			return fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)
		}
		switch rule := rule.(type) {
		case *Filter:
			if rule == nil {
				return errors.New("filter group cannot be nil")
			}
			if err := validateFilterGroup(rule, validation, depth+1); err != nil {
				return err
			}
		case *FilterCondition:
			if rule == nil {
				return errors.New("filter condition cannot be nil")
			}
			path, err := validateFilterCondition(rule, validation)
			if err != nil {
				return err
			}
			if i, ok := slices.BinarySearch(validation.paths, path); !ok {
				validation.paths = slices.Insert(validation.paths, i, path)
			}
		default:
			return errors.New("unsupported filter rule")
		}
	}

	return nil
}
