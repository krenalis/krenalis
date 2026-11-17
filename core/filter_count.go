// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterCount describes the occurrence population counted by a condition.
// Where filters events with Record and money predicates, with at most two group
// levels and 100 descendants. Minimum is a positive JSON integer sample guard.
// CurrencyField and NativeCurrency must be supplied together when restricting
// the population by governed currency; they never convert currencies.
type FilterCount struct {
	Source         FilterCountSource    `json:"source"`
	Period         FilterPeriod         `json:"period"`
	Where          *Filter              `json:"where,omitzero"`
	Minimum        json.Value           `json:"minimum,omitzero"`
	CurrencyField  *FilterCurrencyField `json:"currencyField,omitzero"`
	NativeCurrency *string              `json:"nativeCurrency,omitzero"`
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
		case "currencyField":
			if dec.PeekKind() == json.Null {
				return errors.New("currencyField cannot be null")
			}
			err = dec.Decode(&decoded.CurrencyField)
		case "nativeCurrency":
			if dec.PeekKind() == json.Null {
				return errors.New("nativeCurrency cannot be null")
			}
			err = dec.Decode(&decoded.NativeCurrency)
		case "where":
			if dec.PeekKind() == json.Null {
				err = dec.SkipValue()
			} else {
				var rule FilterRule
				rule, _, err = unmarshalFilterRule(dec, 1, expectFilterGroup, occurrenceFilterRule)
				if err != nil {
					return errors.New(err.Error())
				}
				decoded.Where = rule.(*Filter)
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
	rule, _, err := unmarshalFilterRule(dec, 1, expectAnyFilterRule, profileFilterRule)
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

// FilterCountSource selects recorded events at registered-message grain.
// Type defaults to track. Name selects event for track and name for
// page/screen; omitting Name selects all messages of that page/screen type.
// It is not a reference to a governed business definition.
type FilterCountSource struct {
	Kind  string `json:"kind"`
	Name  string `json:"name,omitzero"`
	Grain string `json:"grain"`
	Type  string `json:"type,omitzero"`
}

// UnmarshalJSON decodes a source and rejects unknown members.
func (source *FilterCountSource) UnmarshalJSON(data []byte) error {
	if source == nil {
		return errors.New("core: cannot unmarshal a Count source into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("Count source must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "name", "type":
			if !value.IsString() || value.String() == "" {
				return errors.New("source name and type must be nonempty strings when supplied")
			}
		case "kind", "grain":
			// Kind and grain are checked during preparation.
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
// microsecond, millisecond, second, minute, hour, day or week. A day is 24h.
// Window widths must be positive; the newer offset of an ago band may be zero.
type FilterDuration struct {
	Value json.Value `json:"value"`
	Unit  string     `json:"unit"`
}

// UnmarshalJSON decodes an elapsed duration and rejects unknown members.
func (duration *FilterDuration) UnmarshalJSON(data []byte) error {
	if duration == nil {
		return errors.New("core: cannot unmarshal a filter duration into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
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

// elapsed returns exact nanoseconds, before applying execution range limits.
func (duration FilterDuration) elapsed() (filterCountInteger, error) {
	value, err := parseFilterCountInteger(duration.Value)
	if err != nil {
		return "", fmt.Errorf("invalid elapsed duration value: %w", err)
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
		return "", errors.New("invalid fixed elapsed duration unit")
	}
	return value.scaled(uint64(unit)), nil
}

// FilterPeriod describes an absoluteHalfOpen, elapsedLookback, elapsedAgoBand,
// lastCompleteCalendarPeriods or fullRelevantHistory period. Calendar periods
// require root defaultCalendar; fullRelevantHistory requires an explicit
// History reference.
// Exactly the fields belonging to Kind must be supplied. Absolute instants use
// RFC 3339 with an explicit offset. Relative periods remain relative when
// serialized; their UTC bounds are resolved only for execution.
type FilterPeriod struct {
	Kind     string          `json:"kind"`
	Start    *time.Time      `json:"start,omitzero"`
	End      *time.Time      `json:"end,omitzero"`
	Duration *FilterDuration `json:"duration,omitzero"`
	Older    *FilterDuration `json:"older,omitzero"`
	Newer    *FilterDuration `json:"newer,omitzero"`
	Unit     string          `json:"unit,omitzero"`
	Count    json.Value      `json:"count,omitzero"`
	History  *FilterHistory  `json:"history,omitzero"`
}

// UnmarshalJSON decodes a period and rejects unknown or null fields.
func (period *FilterPeriod) UnmarshalJSON(data []byte) error {
	if period == nil {
		return errors.New("core: cannot unmarshal a filter period into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("filter period must be an object")
	}
	var calendarFields bool
	for name, value := range json.Value(data).Properties() {
		switch name {
		case "start", "end":
			if !value.IsString() || len(value) > 80 {
				return errors.New("period instant must be an RFC 3339 string")
			}
			_, err := parseFilterInstant(value.String())
			if err != nil {
				return err
			}
		case "unit", "count":
			calendarFields = true
			if value.IsNull() {
				return errors.New("filter period fields cannot be null")
			}
		case "kind", "duration", "older", "newer", "history":
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
	if calendarFields && decoded.Kind != "lastCompleteCalendarPeriods" {
		return errors.New("calendar period fields require lastCompleteCalendarPeriods")
	}
	*period = FilterPeriod(decoded)
	return nil
}

func (period FilterPeriod) scope(source FilterCountSource, calendar *behavioral.Calendar) (behavioral.Scope, error) {
	if source.Kind != "recordedEvents" {
		return behavioral.Scope{}, errors.New("invalid recorded event source kind")
	}
	scope, err := behavioral.RecordedScope(source.Type, source.Name, source.Grain)
	if err != nil {
		return behavioral.Scope{}, err
	}
	var older, newer filterCountInteger
	if period.Kind != "lastCompleteCalendarPeriods" && (period.Unit != "" || period.Count != nil) {
		return behavioral.Scope{}, errors.New("calendar period fields require lastCompleteCalendarPeriods")
	}
	if period.Kind != "fullRelevantHistory" && period.History != nil {
		return behavioral.Scope{}, errors.New("history reference requires fullRelevantHistory")
	}
	switch period.Kind {
	case "fullRelevantHistory":
		if period.History == nil || period.Start != nil || period.End != nil || period.Duration != nil ||
			period.Older != nil || period.Newer != nil {
			return behavioral.Scope{}, errors.New("full relevant history requires only a history reference")
		}
		history, err := behavioral.ResolveHistory(period.History.Definition, period.History.MeaningRevision)
		if err != nil {
			return behavioral.Scope{}, err
		}
		scope.History = history
	case "lastCompleteCalendarPeriods":
		if period.Start != nil || period.End != nil || period.Duration != nil ||
			period.Older != nil || period.Newer != nil || calendar == nil {
			return behavioral.Scope{}, errors.New("complete calendar periods require defaultCalendar, unit and count only")
		}
		switch period.Unit {
		case "day", "week", "month":
		default:
			return behavioral.Scope{}, errors.New("invalid calendar period unit")
		}
		count, err := parseFilterCountInteger(period.Count)
		if err != nil {
			return behavioral.Scope{}, fmt.Errorf("invalid calendar period count: %w", err)
		}
		if count == "0" {
			return behavioral.Scope{}, errors.New("calendar period count must be positive")
		}
		n, ok := count.uint64()
		if !ok {
			return behavioral.Scope{}, &behavioral.UnsupportedError{Capability: "calendar count beyond 64-bit representation"}
		}
		scope.Calendar = &behavioral.CalendarPeriod{Calendar: calendar, Unit: period.Unit, Count: n}
	case "absoluteHalfOpen":
		if period.Start == nil || period.End == nil || period.Duration != nil ||
			period.Older != nil || period.Newer != nil {
			return behavioral.Scope{}, errors.New("absolute period requires only start and end")
		}
		scope.Start, scope.End = *period.Start, *period.End
		for _, bound := range []time.Time{scope.Start, scope.End} {
			if bound.IsZero() || bound.UTC().Year() < 1 || bound.UTC().Year() > 9999 {
				return behavioral.Scope{}, errors.New("invalid Unified timestamp")
			}
		}
		if !scope.Start.Before(scope.End) {
			return behavioral.Scope{}, errors.New("invalid absolute observation period")
		}
	case "elapsedLookback":
		if period.Duration == nil || period.Start != nil || period.End != nil ||
			period.Older != nil || period.Newer != nil {
			return behavioral.Scope{}, errors.New("elapsed lookback requires only duration")
		}
		var err error
		older, err = period.Duration.elapsed()
		if err != nil {
			return behavioral.Scope{}, err
		}
		newer = "0"
	case "elapsedAgoBand":
		if period.Older == nil || period.Newer == nil || period.Start != nil ||
			period.End != nil || period.Duration != nil {
			return behavioral.Scope{}, errors.New("elapsed ago band requires only older and newer")
		}
		var err error
		older, err = period.Older.elapsed()
		if err != nil {
			return behavioral.Scope{}, err
		}
		newer, err = period.Newer.elapsed()
		if err != nil {
			return behavioral.Scope{}, err
		}
	default:
		return behavioral.Scope{}, errors.New("invalid filter period kind")
	}
	if older != "" {
		if older.compare(newer) <= 0 {
			return behavioral.Scope{}, errors.New("invalid elapsed observation period")
		}
		old, ook := older.uint64()
		recent, rok := newer.uint64()
		if !ook || !rok || old > math.MaxInt64 || recent > math.MaxInt64 {
			return behavioral.Scope{}, &behavioral.UnsupportedError{Capability: "elapsed duration beyond time.Duration range"}
		}
		scope.Elapsed = &behavioral.ElapsedPeriod{Older: time.Duration(old), Newer: time.Duration(recent)}
	}
	return scope, nil
}

// filterCountInteger is a canonical natural decimal with linear comparisons.
type filterCountInteger string

func (n filterCountInteger) compare(other filterCountInteger) int {
	if len(n) < len(other) {
		return -1
	}
	if len(n) > len(other) {
		return 1
	}
	return strings.Compare(string(n), string(other))
}

// scaled multiplies a natural by a fixed elapsed unit, in linear time.
// The unit is at most one week in nanoseconds, so each carry fits uint64.
func (n filterCountInteger) scaled(unit uint64) filterCountInteger {
	result := make([]byte, len(n)+16)
	position := len(result)
	var carry uint64
	for i := len(n) - 1; i >= 0; i-- {
		value := uint64(n[i]-'0')*unit + carry
		position--
		result[position] = byte(value%10) + '0'
		carry = value / 10
	}
	for carry > 0 {
		position--
		result[position] = byte(carry%10) + '0'
		carry /= 10
	}
	return filterCountInteger(result[position:])
}

func (n filterCountInteger) uint64() (uint64, bool) {
	value, err := strconv.ParseUint(string(n), 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// parseFilterCountInteger validates the lexical natural before range limits.
func parseFilterCountInteger(value json.Value) (filterCountInteger, error) {
	if !json.Valid(value) || !value.IsNumber() {
		return "", errors.New("Count operand must be a nonnegative JSON integer")
	}
	value = json.TrimSpace(value)
	if bytes.Equal(value, []byte("-0")) {
		return "0", nil
	}
	if value[0] == '-' || bytes.ContainsAny(value, ".eE") {
		return "", errors.New("Count operand must be a nonnegative JSON integer")
	}
	return filterCountInteger(value), nil
}

// parseFilterInstant reads an exact RFC 3339 instant without truncating digits.
func parseFilterInstant(text string) (time.Time, error) {
	if len(text) > 80 {
		return time.Time{}, errors.New("invalid timestamp")
	}
	if separator := strings.IndexAny(text, ".,"); separator >= 0 {
		end := separator + 1
		for end < len(text) && text[end] >= '0' && text[end] <= '9' {
			end++
		}
		if end-separator-1 > 9 {
			return time.Time{}, errors.New("timestamp has more than nine fractional digits")
		}
	}
	quoted, err := json.Quote([]byte(text))
	if err != nil {
		return time.Time{}, errors.New("invalid timestamp")
	}
	var instant time.Time
	err = json.Unmarshal(quoted, &instant)
	if err != nil {
		return time.Time{}, errors.New("invalid RFC 3339 timestamp")
	}
	return instant, nil
}

func prepareFilterCount(condition *FilterCountCondition, calendar *behavioral.Calendar) (behavioral.Rule, error) {

	if condition == nil {
		return behavioral.Rule{}, errors.New("filter count condition cannot be nil")
	}
	if len(condition.Operator) > 64 {
		return behavioral.Rule{}, errors.New("invalid Count operator")
	}
	// Validate the operator and arity.
	var operator behavioral.Comparison
	switch condition.Operator {
	case OpIs:
		operator = behavioral.Equal
	case OpIsNot:
		operator = behavioral.Different
	case OpIsLessThan:
		operator = behavioral.Less
	case OpIsLessThanOrEqualTo:
		operator = behavioral.LessOrEqual
	case OpIsGreaterThan:
		operator = behavioral.Greater
	case OpIsGreaterThanOrEqualTo:
		operator = behavioral.GreaterOrEqual
	case OpIsBetween:
		operator = behavioral.Between
	default:
		return behavioral.Rule{}, fmt.Errorf("operator %q cannot be used with Count", condition.Operator)
	}
	arity := 1
	if operator == behavioral.Between {
		arity = 2
	}
	if len(condition.Values) != arity {
		return behavioral.Rule{}, fmt.Errorf("Count operator requires %d operands", arity)
	}

	// Validate exact operands and the sample minimum before execution limits.
	value, err := parseFilterCountInteger(condition.Values[0])
	if err != nil {
		return behavioral.Rule{}, err
	}
	upper := filterCountInteger("0")
	if arity == 2 {
		upper, err = parseFilterCountInteger(condition.Values[1])
		if err != nil {
			return behavioral.Rule{}, err
		}
		if upper.compare(value) < 0 {
			return behavioral.Rule{}, errors.New("invalid Count terminal")
		}
	}
	minimum := filterCountInteger("0")
	if condition.Count.Minimum != nil {
		minimum, err = parseFilterCountInteger(condition.Count.Minimum)
		if err != nil {
			return behavioral.Rule{}, errors.New("Count sample minimum must be a positive JSON integer")
		}
		if minimum == "0" {
			return behavioral.Rule{}, errors.New("Count sample minimum must be positive")
		}
	}
	if minimum != "0" && ((operator == behavioral.Equal && value.compare(minimum) < 0) ||
		(operator == behavioral.Less && value.compare(minimum) <= 0) ||
		(operator == behavioral.LessOrEqual && value.compare(minimum) < 0) ||
		(operator == behavioral.Between && upper.compare(minimum) < 0)) {
		return behavioral.Rule{}, errors.New("Count sample minimum and terminal are statically incompatible")
	}

	// Validate the period, source and occurrence qualification before availability.
	scope, periodErr := condition.Count.Period.scope(condition.Count.Source, calendar)
	if periodErr != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](periodErr); !unsupported {
			return behavioral.Rule{}, periodErr
		}
		// Preserve the validated source while deferring period availability.
		scope.Type, scope.Event, scope.Grain = condition.Count.Source.Type, condition.Count.Source.Name,
			condition.Count.Source.Grain
	}
	rule, err := prepareBehavioralCount(scope, condition.Count.Where, behavioral.CountTest{})
	var unavailable error
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	count := condition.Count
	if (count.CurrencyField == nil) != (count.NativeCurrency == nil) {
		return behavioral.Rule{}, errors.New("Count requires currencyField and nativeCurrency together")
	}
	if count.CurrencyField != nil {
		if count.Source.Type != "" && count.Source.Type != "track" {
			return behavioral.Rule{}, errors.New("currency field is incompatible with the occurrence source")
		}
		rule.Count.Currency, err = behavioral.ResolveNativeCurrency(count.CurrencyField.Definition,
			count.CurrencyField.MeaningRevision, *count.NativeCurrency, count.Source.Name)
		if err != nil {
			return behavioral.Rule{}, err
		}
	}
	err = rule.Count.ValidateWhere()
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	if periodErr != nil {
		return behavioral.Rule{}, periodErr
	}
	if unavailable != nil {
		return behavioral.Rule{}, unavailable
	}
	v, vok := value.uint64()
	u, uok := upper.uint64()
	m, mok := minimum.uint64()
	if !vok || !uok || !mok {
		return behavioral.Rule{}, &behavioral.UnsupportedError{Capability: "Count integer beyond 64-bit representation"}
	}
	rule.Count.Test = behavioral.CountTest{Operator: operator, Value: v, Upper: u}
	rule.Count.Minimum = m

	return rule, nil
}
