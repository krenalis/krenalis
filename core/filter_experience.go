// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterExperience selects an occurrence before verifying its properties.
// Verify is mandatory and accepts Record rules and occurredIn predicates.
type FilterExperience struct {
	Base   FilterExperienceBase     `json:"base"`
	Select FilterExperienceSelector `json:"select"`
	Verify *Filter                  `json:"verify"`
}

// UnmarshalJSON decodes an experience with a mandatory non-null verification.
func (experience *FilterExperience) UnmarshalJSON(data []byte) error {
	if experience == nil {
		return errors.New("core: cannot unmarshal an experience into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("experience must be an object")
	}
	var decoded FilterExperience
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() {
			return errors.New("experience fields cannot be null")
		}
		var err error
		switch name {
		case "base":
			err = json.Unmarshal(value, &decoded.Base)
		case "select":
			err = json.Unmarshal(value, &decoded.Select)
		case "verify":
			var rule FilterRule
			rule, _, err = unmarshalFilterRule(json.NewDecoder(bytes.NewReader(value)), 1, expectFilterGroup, selectedFilterRule)
			if err == nil {
				decoded.Verify = rule.(*Filter)
			}
		default:
			return errors.New("unknown experience property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if decoded.Verify == nil {
		return errors.New("experience requires verify")
	}
	*experience = decoded
	return nil
}

// FilterExperienceBase defines the population before chronological selection.
// Where has the same Record semantics and limits as Count qualification.
type FilterExperienceBase struct {
	Source FilterCountSource `json:"source"`
	Period FilterPeriod      `json:"period"`
	Where  *Filter           `json:"where,omitzero"`
}

// UnmarshalJSON decodes a selection base without accepting Count-only fields.
func (base *FilterExperienceBase) UnmarshalJSON(data []byte) error {
	if base == nil {
		return errors.New("core: cannot unmarshal an experience base into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("experience base must be an object")
	}
	var decoded FilterExperienceBase
	var hasSource, hasPeriod bool
	for name, value := range json.Value(data).Properties() {
		var err error
		switch name {
		case "source":
			hasSource = true
			err = json.Unmarshal(value, &decoded.Source)
		case "period":
			hasPeriod = true
			err = json.Unmarshal(value, &decoded.Period)
		case "where":
			if !value.IsNull() {
				var rule FilterRule
				rule, _, err = unmarshalFilterRule(json.NewDecoder(bytes.NewReader(value)),
					1, expectFilterGroup, occurrenceFilterRule)
				if err == nil {
					decoded.Where = rule.(*Filter)
				}
			}
		default:
			return errors.New("unknown experience base property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if !hasSource || !hasPeriod {
		return errors.New("experience base requires source and period")
	}
	*base = decoded
	return nil
}

// FilterExperienceCondition is a complete selected-occurrence predicate.
// It has no outer comparison operator or operands.
type FilterExperienceCondition struct {
	Experience FilterExperience `json:"experience"`
}

// MarshalJSON returns the condition without semantic preparation.
func (condition *FilterExperienceCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter experience condition cannot be nil")
	}
	type encoded FilterExperienceCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes an experience using the common rule discriminator.
func (condition *FilterExperienceCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal an experience condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, profileFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterExperienceCondition)
	if !ok {
		return errors.New("expected an experience condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterExperienceCondition) filterRule() {}

// FilterExperienceSelector names a selector; First and Last execute.
type FilterExperienceSelector struct {
	Kind     string     `json:"kind"`
	Position json.Value `json:"position,omitzero"`
}

// UnmarshalJSON decodes a selector without silently ignoring parameters.
func (selector *FilterExperienceSelector) UnmarshalJSON(data []byte) error {
	if selector == nil {
		return errors.New("core: cannot unmarshal an experience selector into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("experience selector must be an object")
	}
	for name, value := range json.Value(data).Properties() {
		if name != "kind" && name != "position" || value.IsNull() {
			return errors.New("invalid experience selector property")
		}
	}
	type encoded FilterExperienceSelector
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	*selector = FilterExperienceSelector(decoded)
	return nil
}

func prepareFilterExperience(condition *FilterExperienceCondition, calendar *unified.Calendar) (unified.Rule, error) {

	if condition == nil || condition.Experience.Verify == nil {
		return unified.Rule{}, errors.New("experience requires verify")
	}
	experience := condition.Experience
	scope, err := experience.Base.Period.scope(experience.Base.Source, calendar)
	var unavailable error
	if err != nil {
		if _, ok := errors.AsType[*unified.UnsupportedError](err); !ok {
			return unified.Rule{}, err
		}
		unavailable = err
	}
	selector := experience.Select
	switch selector.Kind {
	case "last", "first":
		if selector.Position != nil {
			return unified.Rule{}, errors.New("position requires nth selector")
		}
	case "nth":
		position, err := parseFilterCountInteger(selector.Position)
		if err != nil {
			return unified.Rule{}, errors.New("nth position must be an integer from 1 to 10")
		}
		if position == "0" || position.compare("10") > 0 {
			return unified.Rule{}, errors.New("nth position must be an integer from 1 to 10")
		}
	default:
		return unified.Rule{}, errors.New("invalid experience selector")
	}

	// Prepare base qualification and selected-record verification separately.
	var where *state.Where
	if experience.Base.Where != nil {
		validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties()}
		where, err = prepareFilterGroup(experience.Base.Where, &validation, 1)
		if err != nil {
			return unified.Rule{}, err
		}
	}
	validation := filterValidation{maxDepth: 2, properties: schemas.Event.Properties()}
	verify, err := prepareExperienceVerification(experience.Verify, &validation, 1, experience.Base.Source, calendar)
	if err != nil {
		if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
			return unified.Rule{}, err
		}
		unavailable = err
	}
	if selector.Kind == "nth" {
		return unified.Rule{}, &unified.UnsupportedError{Capability: "nth experience selector"}
	}
	if unavailable != nil {
		return unified.Rule{}, unavailable
	}

	return unified.Rule{Experience: &unified.Experience{First: selector.Kind == "first",
		Scope: scope, Where: where, Verify: verify}}, nil
}
