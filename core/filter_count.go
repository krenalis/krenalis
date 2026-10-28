// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterCount describes the occurrence population counted by a condition.
// Where is a Record filter on events, limited to two group levels and 100
// descendants. Minimum, when supplied, is a positive JSON integer sample guard.
type FilterCount struct {
	Source  FilterCountSource `json:"source"`
	Period  FilterPeriod      `json:"period"`
	Where   *Filter           `json:"where,omitzero"`
	Minimum json.Value        `json:"minimum,omitzero"`
}

// UnmarshalJSON decodes the Count subject without preparing its operands.
func (count *FilterCount) UnmarshalJSON(data []byte) error {
	if count == nil {
		return errors.New("core: cannot unmarshal Count into nil")
	}
	if !json.Valid(data) {
		return errors.New("invalid count JSON")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.ReadToken()
	if err != nil {
		return errors.New(err.Error())
	}
	if tok.Kind() != json.Object {
		return errors.New("count must be an object")
	}
	var decoded FilterCount
	var hasSource, hasPeriod bool
	for dec.PeekKind() != '}' {
		tok, err = dec.ReadToken()
		if err != nil {
			return errors.New(err.Error())
		}
		switch tok.String() {
		case "source":
			hasSource = true
			err = dec.Decode(&decoded.Source)
		case "period":
			hasPeriod = true
			err = dec.Decode(&decoded.Period)
		case "minimum":
			err = dec.Decode(&decoded.Minimum)
		case "where":
			if dec.PeekKind() == json.Null {
				err = dec.SkipValue()
			} else {
				var rule FilterRule
				rule, _, err = unmarshalFilterRule(dec, 1, expectFilterGroup, true)
				if err == nil {
					decoded.Where = rule.(*Filter)
				}
			}
		default:
			return errors.New("unknown count property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	err = dec.SkipToken()
	if err != nil {
		return errors.New(err.Error())
	}
	if !hasSource || !hasPeriod {
		return errors.New("count must contain source and period")
	}
	*count = decoded
	return nil
}

// FilterCountCondition compares a Count to one or two nonnegative JSON integer
// literals using the seven Count comparisons. It requires Unified filtering.
// Literal values retain their exact JSON spelling through encode/decode.
type FilterCountCondition struct {
	Count    FilterCount    `json:"count"`
	Operator FilterOperator `json:"operator"`
	Values   []json.Value   `json:"values"`
}

// MarshalJSON returns the condition's JSON without semantic validation.
func (condition *FilterCountCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter count condition cannot be nil")
	}
	type encoded FilterCountCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes a Count condition with the common rule discriminator.
func (condition *FilterCountCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a Count condition into nil")
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	rule, _, err := unmarshalFilterRule(dec, 1, expectAnyFilterRule, false)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterCountCondition)
	if !ok {
		return errors.New("expected a count condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterCountCondition) filterRule() {}

// FilterCountSource selects recorded events by their raw name and occurrence
// grain. The current source kind is recordedEvents, with registered-message
// grain. It is not a reference to a governed Purchase definition.
type FilterCountSource struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Grain string `json:"grain"`
}

// UnmarshalJSON decodes a source and rejects unknown members.
func (source *FilterCountSource) UnmarshalJSON(data []byte) error {
	if source == nil || !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("Count source must be an object")
	}
	for name := range json.Value(data).Properties() {
		switch name {
		case "kind", "name", "grain":
			// These are the complete source fields.
		default:
			return errors.New("unknown Count source property")
		}
	}
	type encoded FilterCountSource
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*source = FilterCountSource(decoded)
	return nil
}

// FilterDuration is a nonnegative integer JSON value in a fixed elapsed unit:
// microsecond, millisecond, second, minute, hour, day or week. A day is 24 hours.
// Window widths must be positive; the newer offset of an ago band may be zero.
type FilterDuration struct {
	Value json.Value `json:"value"`
	Unit  string     `json:"unit"`
}

// UnmarshalJSON decodes an elapsed duration and rejects unknown members.
func (duration *FilterDuration) UnmarshalJSON(data []byte) error {
	if duration == nil || !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("filter duration must be an object")
	}
	for name := range json.Value(data).Properties() {
		switch name {
		case "value", "unit":
			// These are the complete elapsed duration fields.
		default:
			return errors.New("unknown filter duration property")
		}
	}
	type encoded FilterDuration
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*duration = FilterDuration(decoded)
	return nil
}

func (duration FilterDuration) elapsed() (time.Duration, error) {
	value, err := parseFilterCountInteger(duration.Value)
	if err != nil {
		return 0, err
	}
	var unit time.Duration
	switch duration.Unit {
	case "microsecond":
		unit = time.Microsecond
	case "millisecond":
		unit = time.Millisecond
	case "second":
		unit = time.Second
	case "minute":
		unit = time.Minute
	case "hour":
		unit = time.Hour
	case "day":
		unit = 24 * time.Hour
	case "week":
		unit = 7 * 24 * time.Hour
	default:
		return 0, errors.New("invalid fixed elapsed duration unit")
	}
	if value > uint64(math.MaxInt64/int64(unit)) {
		return 0, &unified.UnsupportedError{Capability: "elapsed duration beyond time.Duration range"}
	}
	return time.Duration(value) * unit, nil
}

// FilterPeriod describes an absoluteHalfOpen, elapsedLookback or elapsedAgoBand
// period. Exactly the fields belonging to Kind must be supplied. Absolute
// instants use RFC 3339 with an explicit offset. Relative periods remain
// relative when serialized; their UTC bounds are resolved only for execution.
type FilterPeriod struct {
	Kind     string          `json:"kind"`
	Start    *time.Time      `json:"start,omitzero"`
	End      *time.Time      `json:"end,omitzero"`
	Duration *FilterDuration `json:"duration,omitzero"`
	Older    *FilterDuration `json:"older,omitzero"`
	Newer    *FilterDuration `json:"newer,omitzero"`
}

// UnmarshalJSON decodes a period and rejects unknown or null fields.
func (period *FilterPeriod) UnmarshalJSON(data []byte) error {
	if period == nil || !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("filter period must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "start", "end":
			if !value.IsString() || len(value) > 80 {
				return errors.New("period instant must be an RFC 3339 string")
			}
		case "kind", "duration", "older", "newer":
			if value.IsNull() {
				return errors.New("filter period fields cannot be null")
			}
		default:
			return errors.New("unknown filter period property")
		}
	}
	type encoded FilterPeriod
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New("invalid filter period fields")
	}
	*period = FilterPeriod(decoded)
	return nil
}

func (period FilterPeriod) scope(source FilterCountSource) (unified.Scope, error) {
	if source.Kind != "recordedEvents" {
		return unified.Scope{}, errors.New("invalid Count source kind")
	}
	if source.Grain == "" {
		return unified.Scope{}, errors.New("Count source must declare its grain")
	}
	scope := unified.Scope{Event: source.Name, Grain: source.Grain}
	switch period.Kind {
	case "absoluteHalfOpen":
		if period.Start == nil || period.End == nil || period.Duration != nil ||
			period.Older != nil || period.Newer != nil {
			return unified.Scope{}, errors.New("absolute period requires only start and end")
		}
		scope.Start, scope.End = *period.Start, *period.End
	case "elapsedLookback":
		if period.Duration == nil || period.Start != nil || period.End != nil ||
			period.Older != nil || period.Newer != nil {
			return unified.Scope{}, errors.New("elapsed lookback requires only duration")
		}
		older, err := period.Duration.elapsed()
		if err != nil {
			return unified.Scope{}, err
		}
		scope.Elapsed = &unified.ElapsedPeriod{Older: older}
	case "elapsedAgoBand":
		if period.Older == nil || period.Newer == nil || period.Start != nil ||
			period.End != nil || period.Duration != nil {
			return unified.Scope{}, errors.New("elapsed ago band requires only older and newer")
		}
		older, err := period.Older.elapsed()
		if err != nil {
			return unified.Scope{}, err
		}
		newer, err := period.Newer.elapsed()
		if err != nil {
			return unified.Scope{}, err
		}
		scope.Elapsed = &unified.ElapsedPeriod{Older: older, Newer: newer}
	default:
		return unified.Scope{}, errors.New("invalid Count period kind")
	}
	return scope, nil
}

// parseFilterCountInteger reads the same integer JSON spelling as Record Int
// operands, retaining out-of-range natural literals as unsupported capability.
func parseFilterCountInteger(value json.Value) (uint64, error) {
	if !json.Valid(value) || !value.IsNumber() {
		return 0, errors.New("Count operand must be a nonnegative JSON integer")
	}
	if bytes.Equal(json.TrimSpace(value), []byte("-0")) {
		return 0, nil
	}
	n, err := strconv.ParseUint(string(json.TrimSpace(value)), 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, &unified.UnsupportedError{Capability: "Count integer beyond 64-bit representation"}
		}
		return 0, errors.New("Count operand must be a nonnegative JSON integer")
	}
	return n, nil
}

func prepareFilterCount(condition *FilterCountCondition) (unified.Rule, error) {
	if condition == nil {
		return unified.Rule{}, errors.New("filter count condition cannot be nil")
	}
	if len(condition.Operator) > 64 {
		return unified.Rule{}, errors.New("invalid Count operator")
	}
	var operator unified.Comparison
	switch condition.Operator {
	case OpIs:
		operator = unified.Equal
	case OpIsNot:
		operator = unified.Different
	case OpIsLessThan:
		operator = unified.Less
	case OpIsLessThanOrEqualTo:
		operator = unified.LessOrEqual
	case OpIsGreaterThan:
		operator = unified.Greater
	case OpIsGreaterThanOrEqualTo:
		operator = unified.GreaterOrEqual
	case OpIsBetween:
		operator = unified.Between
	default:
		return unified.Rule{}, fmt.Errorf("operator %q cannot be used with Count", condition.Operator)
	}
	arity := 1
	if operator == unified.Between {
		arity = 2
	}
	if len(condition.Values) != arity {
		return unified.Rule{}, fmt.Errorf("Count operator requires %d operands", arity)
	}
	value, err := parseFilterCountInteger(condition.Values[0])
	if err != nil {
		return unified.Rule{}, err
	}
	test := unified.CountTest{Operator: operator, Value: value}
	if arity == 2 {
		test.Upper, err = parseFilterCountInteger(condition.Values[1])
		if err != nil {
			return unified.Rule{}, err
		}
	}
	scope, err := condition.Count.Period.scope(condition.Count.Source)
	if err != nil {
		return unified.Rule{}, err
	}
	rule, err := prepareUnifiedRecordCount(scope, condition.Count.Where, test)
	if err != nil {
		return unified.Rule{}, err
	}
	if condition.Count.Minimum != nil {
		rule.Count.Minimum, err = parseFilterCountInteger(condition.Count.Minimum)
		if err != nil {
			return unified.Rule{}, err
		}
		if rule.Count.Minimum == 0 {
			return unified.Rule{}, errors.New("Count sample minimum must be positive")
		}
	}
	return rule, nil
}
