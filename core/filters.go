// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/util"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
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

	// Values are JSON operands whose kinds must match the property's type.
	// String operands are limited to 80 runes without NUL, except allowed values.
	// Decimal and JSON Number operands must fit precision 76 and scale 37.
	Values []json.Value `json:"values,omitzero"`
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

// UnmarshalJSON replaces condition with its JSON representation.
func (condition *FilterCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a filter condition into nil")
	}
	type conditionJSON FilterCondition
	var decoded conditionJSON
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*condition = FilterCondition(decoded)
	return nil
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
	ruleCount  int
}

// convertOperatorFromWhere converts a where operator into a filter operator.
func convertOperatorFromWhere(op state.WhereOperator) FilterOperator {
	return operators[op]
}

// convertOperatorToWhere converts a filter operator into a where operator.
func convertOperatorToWhere(op FilterOperator) state.WhereOperator {
	return state.WhereOperator(slices.Index(operators[:], op))
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
		var values []json.Value
		if len(cond.Values) > 0 {
			values = make([]json.Value, len(cond.Values))
		}
		for i, value := range cond.Values {
			var v json.Value
			switch value := value.(type) {
			case string:
				v, _ = json.Quote([]byte(value))
			case bool:
				v = json.Value(strconv.FormatBool(value))
			case float64:
				p, _ := properties.ByPathSlice(cond.Property)
				t := p.Type
				if t.IsArray() {
					t = t.Elem()
				}
				v = json.Value(strconv.FormatFloat(value, 'g', -1, t.BitSize()))
			case int:
				v = json.Value(strconv.FormatInt(int64(value), 10))
			case uint:
				v = json.Value(strconv.FormatUint(uint64(value), 10))
			case decimal.Decimal:
				v, _ = value.MarshalJSON()
			case time.Time:
				p, _ := properties.ByPathSlice(cond.Property)
				t := p.Type
				if t.IsArray() {
					t = t.Elem()
				}
				var text string
				switch t.Kind() {
				case types.DateTimeKind:
					text = value.Format(time.RFC3339Nano)
				case types.DateKind:
					text = value.Format(time.DateOnly)
				case types.TimeKind:
					text = value.Format("15:04:05.999999999")
				default:
					panic(fmt.Errorf("core: unexpected filter time type %s", t))
				}
				v, _ = json.Quote([]byte(text))
			case json.Value:
				v = bytes.Clone(value)
			default:
				panic(fmt.Errorf("core: unexpected filter operand %T", value))
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

// resolveFilterProperty resolves path against properties.
//
// It returns the property at path and its schema path or, if path extends into
// a JSON property, that JSON property and its schema path. path must be a valid
// property path.
func resolveFilterProperty(properties types.Properties, path string) (types.Property, string, error) {
	p, err := properties.ByPath(path)
	if err != nil {
		if !p.Type.IsJSON() {
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
	var values []json.Value
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

// prepareFilter validates filter and returns its typed expression and used
// paths.
func prepareFilter(filter *Filter, schema types.Type) (*state.Where, []string, error) {
	if filter == nil {
		return nil, nil, nil
	}
	if !schema.Valid() || !schema.IsObject() {
		return nil, nil, errors.New("filter schema must be an object")
	}
	validation := filterValidation{properties: schema.Properties()}
	where, err := prepareFilterGroup(filter, &validation, 1)
	if err != nil {
		return nil, nil, err
	}
	return where, validation.paths, nil
}

// prepareFilterCondition validates a condition and returns its typed form and
// base path.
func prepareFilterCondition(cond *FilterCondition, validation *filterValidation) (*state.WhereCondition, string, error) {

	if !types.IsValidPropertyPath(cond.Property) {
		return nil, "", errors.New("property path is not valid")
	}

	p, path, err := resolveFilterProperty(validation.properties, cond.Property)
	if err != nil {
		return nil, "", err
	}

	op := cond.Operator
	kind := p.Type.Kind()

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
	// is empty                    : json, string, array, map
	// is not empty                : json, string, array, map
	// is null                     : All types [^3]
	// is not null                 : All types [^3]
	// exists                      : All types [^4]
	// does not exist              : All types [^4]
	//
	// [1]: Strings with allowed values are not supported.
	// [2]: array(T) is supported if T is supported by the "is" operator.
	// [3]: Only if the property is nullable or is of type json.
	// [4]: Only if the property is read-optional, or if it is of type json and has a JSON path.
	//
	switch op {
	case OpIs, OpIsNot:
		switch kind {
		case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsBetween, OpIsNotBetween:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return nil, "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.BooleanKind, types.UUIDKind, types.IPKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		default:
			// Other scalar kinds support intervals.
		}
	case OpIsOneOf, OpIsNotOneOf:
		switch kind {
		case types.BooleanKind, types.UUIDKind, types.IPKind, types.ArrayKind, types.ObjectKind, types.MapKind:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsLessThan, OpIsLessThanOrEqualTo, OpIsGreaterThan, OpIsGreaterThanOrEqualTo:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return nil, "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.IntKind, types.FloatKind, types.DecimalKind:
		case types.JSONKind:
		default:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpContains, OpDoesNotContain:
		switch kind {
		case types.StringKind, types.JSONKind:
		case types.ArrayKind:
			switch k := p.Type.Elem().Kind(); k {
			case types.BooleanKind, types.ArrayKind, types.ObjectKind, types.MapKind:
				return nil, "", fmt.Errorf("operator %q cannot be used with array(%s) properties", op, k)
			}
		default:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpStartsWith, OpEndsWith:
		switch kind {
		case types.StringKind:
			if p.Type.Values() != nil {
				return nil, "", fmt.Errorf("operator %q cannot be used with string type that has values", op)
			}
		case types.JSONKind:
		default:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsBefore, OpIsAfter, OpIsOnOrBefore, OpIsOnOrAfter:
		switch kind {
		case types.DateTimeKind, types.DateKind, types.TimeKind:
		case types.YearKind:
		default:
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsTrue, OpIsFalse:
		if kind != types.BooleanKind && kind != types.JSONKind {
			return nil, "", fmt.Errorf("operator %q cannot be used with %s properties", op, kind)
		}
	case OpIsEmpty, OpIsNotEmpty:
		switch kind {
		case types.StringKind:
			if values := p.Type.Values(); len(values) > 0 {
				if !slices.Contains(values, "") {
					return nil, "", fmt.Errorf("operator %q cannot be used on string properties "+
						"that exclude the empty string from allowed values", op)
				}
			}
		case types.JSONKind, types.ArrayKind, types.MapKind:
		default:
			return nil, "", fmt.Errorf("operator %q can only be used with json, string, array, and map properties", op)
		}
	case OpIsNull, OpIsNotNull:
		if !p.Nullable && kind != types.JSONKind {
			return nil, "", fmt.Errorf("operator %q can only be used with nullable or json properties", op)
		}
	case OpExists, OpDoesNotExist:
		if !p.ReadOptional && path == cond.Property {
			return nil, "", fmt.Errorf("operator %q can only be used with read-optional properties "+
				"or with json properties that include a JSON path", op)
		}
	default:
		return nil, "", fmt.Errorf("operator %q is not valid", op)
	}

	// Validate the values.
	switch op {
	case OpIsTrue, OpIsFalse, OpIsEmpty, OpIsNotEmpty, OpIsNull, OpIsNotNull, OpExists, OpDoesNotExist:
		if cond.Values != nil {
			return nil, "", fmt.Errorf("values cannot be used with the operator %q", op)
		}
	default:
		if len(cond.Values) != 1 {
			return nil, "", fmt.Errorf("only one value can be used with the operator %q", op)
		}
	case OpIsBetween, OpIsNotBetween:
		if len(cond.Values) != 2 {
			return nil, "", fmt.Errorf("two values must be used with the operator %q", op)
		}
	case OpIsOneOf, OpIsNotOneOf:
		if len(cond.Values) == 0 {
			return nil, "", fmt.Errorf("at least one value must be used with the operator %q", op)
		}
	}
	condition := &state.WhereCondition{
		Property: strings.Split(cond.Property, "."),
		Operator: convertOperatorToWhere(op),
	}
	if cond.Values == nil {
		return condition, path, nil
	}
	t := p.Type
	if t.IsArray() {
		t = t.Elem()
	}
	condition.Values = make([]any, len(cond.Values))
	for i, value := range cond.Values {
		if len(value) == 0 || !json.Valid(value) {
			return nil, "", errors.New("condition value must contain valid JSON")
		}
		if value.IsString() {
			text := value.String()
			if kind == types.StringKind && p.Type.Values() != nil {
				if !slices.Contains(p.Type.Values(), text) {
					return nil, "", fmt.Errorf("value of the %q property is not among the allowed values", cond.Property)
				}
			} else if !t.IsJSON() || text != "" {
				err := util.ValidateStringField("condition value", text, 80)
				if err != nil {
					return nil, "", err
				}
			}
		}
		value, err := state.DecodeFilterOperand(value, t, condition.Operator)
		if err != nil {
			return nil, "", fmt.Errorf("value of the %q property is not a valid %s: %w", cond.Property, t.Kind(), err)
		}
		condition.Values[i] = value
	}
	if t.IsJSON() && (op == OpIsBetween || op == OpIsNotBetween) {
		if condition.Values[0].(json.Value).Kind() != condition.Values[1].(json.Value).Kind() {
			return nil, "", errors.New("JSON interval bounds must have the same kind")
		}
	}

	return condition, path, nil
}

// prepareFilterGroup validates and prepares a group and its descendant rules.
func prepareFilterGroup(filter *Filter, validation *filterValidation, depth int) (*state.Where, error) {

	if depth > maxFilterDepth {
		return nil, fmt.Errorf("filter exceeds maximum depth of %d", maxFilterDepth)
	}
	if op := filter.Operator; op != OpAnd && op != OpOr {
		return nil, fmt.Errorf("invalid logical operator %q", op)
	}
	if len(filter.Rules) == 0 {
		return nil, errors.New("rules are missing")
	}
	if len(filter.Rules) > maxFilterRuleCount-validation.ruleCount {
		return nil, fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)
	}

	where := &state.Where{
		Operator: convertLogicalToWhere(filter.Operator),
		Rules:    make([]state.WhereRule, len(filter.Rules)),
	}
	for i, rule := range filter.Rules {
		validation.ruleCount++
		if validation.ruleCount > maxFilterRuleCount {
			return nil, fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)
		}
		switch rule := rule.(type) {
		case *Filter:
			if rule == nil {
				return nil, errors.New("filter group cannot be nil")
			}
			group, err := prepareFilterGroup(rule, validation, depth+1)
			if err != nil {
				return nil, err
			}
			where.Rules[i] = group
		case *FilterCondition:
			if rule == nil {
				return nil, errors.New("filter condition cannot be nil")
			}
			condition, path, err := prepareFilterCondition(rule, validation)
			if err != nil {
				return nil, err
			}
			where.Rules[i] = condition
			if i, ok := slices.BinarySearch(validation.paths, path); !ok {
				validation.paths = slices.Insert(validation.paths, i, path)
			}
		default:
			return nil, errors.New("unsupported filter rule")
		}
	}

	return where, nil
}
