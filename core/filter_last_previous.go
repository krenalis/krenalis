// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterLastPreviousCheck describes one of the four canonical pair checks.
// Difference means Last minus Previous. Difference and Ratio use the seven
// monetary comparisons; value checks admit only Kind.
type FilterLastPreviousCheck struct {
	Kind     string         `json:"kind"`
	Operator FilterOperator `json:"operator,omitzero"`
	Values   []json.Value   `json:"values,omitzero"`
}

// UnmarshalJSON decodes a closed, explicitly discriminated pair check.
func (check *FilterLastPreviousCheck) UnmarshalJSON(data []byte) error {
	if check == nil {
		return errors.New("core: cannot unmarshal a pair check into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("lastVersusPrevious check must be an object")
	}
	seen := map[string]bool{}
	for name, value := range json.Value(data).Properties() {
		if name != "kind" && name != "operator" && name != "values" || value.IsNull() || seen[name] {
			return errors.New("invalid lastVersusPrevious check property")
		}
		seen[name] = true
	}
	type encoded FilterLastPreviousCheck
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	switch decoded.Kind {
	case "sameValue", "differentValue":
		if seen["operator"] || seen["values"] {
			return errors.New("lastVersusPrevious value checks admit only kind")
		}
	case "difference", "ratio":
		if !seen["operator"] || !seen["values"] {
			return errors.New("lastVersusPrevious numeric checks require operator and values")
		}
	default:
		return errors.New("invalid lastVersusPrevious check kind")
	}
	*check = FilterLastPreviousCheck(decoded)
	return nil
}

func (check FilterLastPreviousCheck) prepare() (behavioral.LastPreviousCheck, error) {
	result := behavioral.LastPreviousCheck{}
	switch check.Kind {
	case "sameValue", "differentValue":
		if check.Operator != "" || check.Values != nil {
			return result, errors.New("lastVersusPrevious value checks admit only kind")
		}
		result.Kind = behavioral.LastPreviousSameValue
		if check.Kind == "differentValue" {
			result.Kind = behavioral.LastPreviousDifferentValue
		}
	case "difference", "ratio":
		result.Kind = behavioral.LastPreviousDifference
		if check.Kind == "ratio" {
			result.Kind = behavioral.LastPreviousRatio
		}
		test, err := prepareMonetaryTest(check.Operator, check.Values)
		if err != nil {
			return result, err
		}
		result.Test = test
	default:
		return result, errors.New("invalid lastVersusPrevious check kind")
	}
	err := result.Validate()
	return result, err
}

// FilterLastVersusPrevious compares one field on the last two messages.
// The initial execution subset requires governed currency in Base and an
// ordered Difference check. It never converts currencies or selects a fallback.
type FilterLastVersusPrevious struct {
	Base  FilterExperienceBase    `json:"base"`
	Field FilterSumField          `json:"field"`
	Check FilterLastPreviousCheck `json:"check"`
}

// UnmarshalJSON decodes the three mandatory, non-null properties.
func (last *FilterLastVersusPrevious) UnmarshalJSON(data []byte) error {
	if last == nil {
		return errors.New("core: cannot unmarshal lastVersusPrevious into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("lastVersusPrevious must be an object")
	}
	var decoded FilterLastVersusPrevious
	seen := map[string]bool{}
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() || seen[name] {
			return errors.New("lastVersusPrevious fields must be non-null and unique")
		}
		seen[name] = true
		var err error
		switch name {
		case "base":
			err = json.Unmarshal(value, &decoded.Base)
		case "field":
			err = json.Unmarshal(value, &decoded.Field)
		case "check":
			err = json.Unmarshal(value, &decoded.Check)
		default:
			return errors.New("unknown lastVersusPrevious property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if len(seen) != 3 {
		return errors.New("lastVersusPrevious requires base, field and check")
	}
	*last = decoded
	return nil
}

// FilterLastVersusPreviousCondition is a pair test with no outer comparison.
type FilterLastVersusPreviousCondition struct {
	LastVersusPrevious FilterLastVersusPrevious `json:"lastVersusPrevious"`
}

// MarshalJSON returns the public condition without semantic preparation.
func (condition *FilterLastVersusPreviousCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter lastVersusPrevious condition cannot be nil")
	}
	type encoded FilterLastVersusPreviousCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes the condition with the common rule discriminator.
func (condition *FilterLastVersusPreviousCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a lastVersusPrevious condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, profileFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterLastVersusPreviousCondition)
	if !ok {
		return errors.New("expected a lastVersusPrevious condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterLastVersusPreviousCondition) filterRule() {}

// prepareFilterLastVersusPrevious validates the full request before admission.
func prepareFilterLastVersusPrevious(condition *FilterLastVersusPreviousCondition, calendar *behavioral.Calendar) (behavioral.Rule, error) {
	if condition == nil {
		return behavioral.Rule{}, errors.New("filter lastVersusPrevious condition cannot be nil")
	}
	last := condition.LastVersusPrevious
	base := last.Base
	// Validate the source, period, reading and complete check.
	scope, err := base.Period.scope(base.Source, calendar)
	var unavailable error
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	field, err := behavioral.ResolveMonetaryField(last.Field.Definition, last.Field.MeaningRevision)
	if err != nil {
		return behavioral.Rule{}, err
	}
	if base.Source.Type != "" && base.Source.Type != "track" || base.Source.Name != "Order Completed" {
		return behavioral.Rule{}, errors.New("lastVersusPrevious field is incompatible with the occurrence source")
	}
	check, err := last.Check.prepare()
	if err != nil {
		return behavioral.Rule{}, err
	}
	// Prepare the independent Record qualification.
	var where *state.Where
	if base.Where != nil {
		source := behavioral.Scope{Type: base.Source.Type, Event: base.Source.Name, Grain: base.Source.Grain}
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties(), source: &source}
		where, err = prepareFilterGroup(base.Where, &validation, 1)
		if err != nil {
			if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
				return behavioral.Rule{}, err
			}
			unavailable = err
		}
	}
	// Resolve the pair as a whole before checking available capabilities.
	var currency *behavioral.NativeCurrency
	if (base.CurrencyField == nil) != (base.NativeCurrency == nil) {
		return behavioral.Rule{},
			errors.New("lastVersusPrevious base requires currencyField and nativeCurrency together")
	}
	if base.CurrencyField != nil {
		currency, err = behavioral.ResolveNativeCurrency(base.CurrencyField.Definition,
			base.CurrencyField.MeaningRevision, *base.NativeCurrency, base.Source.Name)
		if err != nil {
			return behavioral.Rule{}, err
		}
	}
	err = field.ValidateWhere(where)
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	if currency == nil {
		unavailable = &behavioral.UnsupportedError{Capability: "lastVersusPrevious without base currency"}
	}
	_, err = check.NormalizedTest()
	if err != nil {
		if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
			return behavioral.Rule{}, err
		}
		unavailable = err
	}
	if unavailable != nil {
		return behavioral.Rule{}, unavailable
	}
	return behavioral.Rule{LastVersusPrevious: &behavioral.LastVersusPrevious{
		Scope: scope, Where: where, Currency: currency, Field: field, Check: check}}, nil
}
