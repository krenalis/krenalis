// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// FilterLatestN selects a complete block before applying one uniform predicate.
// Predicate admits Record and money, not occurredIn. Size is from 1 to 10.
// Only All is executable; Some and Cardinality remain unavailable.
type FilterLatestN struct {
	Base       FilterExperienceBase    `json:"base"`
	Size       json.Value              `json:"size"`
	Predicate  *Filter                 `json:"predicate"`
	Quantifier FilterLatestNQuantifier `json:"quantifier"`
}

// UnmarshalJSON decodes the closed, non-null latestN fields.
func (latest *FilterLatestN) UnmarshalJSON(data []byte) error {
	if latest == nil {
		return errors.New("core: cannot unmarshal latestN into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("latestN must be an object")
	}
	var decoded FilterLatestN
	seen := map[string]bool{}
	for name, value := range json.Value(data).Properties() {
		if value.IsNull() || seen[name] {
			return errors.New("latestN fields must be non-null and unique")
		}
		seen[name] = true
		var err error
		switch name {
		case "base":
			err = json.Unmarshal(value, &decoded.Base)
		case "size":
			err = json.Unmarshal(value, &decoded.Size)
		case "predicate":
			var rule FilterRule
			rule, _, err = unmarshalFilterRule(json.NewDecoder(bytes.NewReader(value)),
				1, expectFilterGroup, occurrenceFilterRule)
			if err == nil {
				decoded.Predicate = rule.(*Filter)
			}
		case "quantifier":
			err = json.Unmarshal(value, &decoded.Quantifier)
		default:
			return errors.New("unknown latestN property")
		}
		if err != nil {
			return errors.New(err.Error())
		}
	}
	if len(seen) != 4 {
		return errors.New("latestN requires base, size, predicate and quantifier")
	}
	*latest = decoded
	return nil
}

// FilterLatestNCondition is a block predicate without an outer comparison.
type FilterLatestNCondition struct {
	LatestN FilterLatestN `json:"latestN"`
}

// MarshalJSON returns the condition without semantic preparation.
func (condition *FilterLatestNCondition) MarshalJSON() ([]byte, error) {
	if condition == nil {
		return nil, errors.New("filter latestN condition cannot be nil")
	}
	type encoded FilterLatestNCondition
	data, err := json.Marshal((*encoded)(condition))
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return data, nil
}

// UnmarshalJSON decodes latestN using the common rule discriminator.
func (condition *FilterLatestNCondition) UnmarshalJSON(data []byte) error {
	if condition == nil {
		return errors.New("core: cannot unmarshal a latestN condition into nil")
	}
	rule, _, err := unmarshalFilterRule(json.NewDecoder(bytes.NewReader(data)), 1, expectAnyFilterRule, profileFilterRule)
	if err != nil {
		return errors.New(err.Error())
	}
	decoded, ok := rule.(*FilterLatestNCondition)
	if !ok {
		return errors.New("expected a latestN condition")
	}
	*condition = *decoded
	return nil
}

func (*FilterLatestNCondition) filterRule() {}

// FilterLatestNQuantifier is All, Some or a canonical Cardinality test.
// All and Some admit only Kind. Cardinality requires Operator and Values:
// is (exactly), >= (at least), <= (at most), or inclusive between.
// Explanations retain the requested quantifier without normalization.
type FilterLatestNQuantifier struct {
	Kind     string         `json:"kind"`
	Operator FilterOperator `json:"operator,omitzero"`
	Values   []json.Value   `json:"values,omitzero"`
}

// UnmarshalJSON rejects unknown, null and variant-incompatible fields.
func (quantifier *FilterLatestNQuantifier) UnmarshalJSON(data []byte) error {
	if quantifier == nil {
		return errors.New("core: cannot unmarshal a latestN quantifier into nil")
	}
	if !json.Valid(data) || !json.Value(data).IsObject() {
		return errors.New("latestN quantifier must be an object")
	}
	seen := map[string]bool{}
	for name, value := range json.Value(data).Properties() {
		if name != "kind" && name != "operator" && name != "values" || value.IsNull() || seen[name] {
			return errors.New("invalid latestN quantifier property")
		}
		seen[name] = true
	}
	type encoded FilterLatestNQuantifier
	var decoded encoded
	err := json.Unmarshal(data, &decoded)
	if err != nil {
		return errors.New(err.Error())
	}
	if decoded.Kind != "all" && decoded.Kind != "some" && decoded.Kind != "cardinality" {
		return errors.New("invalid latestN quantifier kind")
	}
	if decoded.Kind == "cardinality" {
		if !seen["operator"] || !seen["values"] {
			return errors.New("latestN cardinality requires operator and values")
		}
	} else if seen["operator"] || seen["values"] {
		return errors.New("latestN all and some admit only kind")
	}
	*quantifier = FilterLatestNQuantifier(decoded)
	return nil
}

func (quantifier FilterLatestNQuantifier) validate(size filterCountInteger) error {
	if quantifier.Kind == "all" || quantifier.Kind == "some" {
		if quantifier.Operator != "" || quantifier.Values != nil {
			return errors.New("latestN all and some admit only kind")
		}
		return nil
	}
	if quantifier.Kind != "cardinality" {
		return errors.New("invalid latestN quantifier kind")
	}
	arity := 1
	switch quantifier.Operator {
	case OpIs, OpIsGreaterThanOrEqualTo, OpIsLessThanOrEqualTo:
		// These canonical cardinalities have one natural operand.
	case OpIsBetween:
		arity = 2
	default:
		return errors.New("latestN cardinality accepts only is, is greater than or equal to, " +
			"is less than or equal to, and is between")
	}
	if len(quantifier.Values) != arity {
		return errors.New("invalid latestN cardinality arity")
	}
	lower, err := parseFilterCountInteger(quantifier.Values[0])
	if err != nil {
		return err
	}
	if arity == 2 {
		upper, err := parseFilterCountInteger(quantifier.Values[1])
		if err != nil {
			return err
		}
		if upper.compare(lower) < 0 {
			return errors.New("inverted latestN cardinality interval")
		}
	}
	if quantifier.Operator != OpIsLessThanOrEqualTo && lower.compare(size) > 0 {
		return errors.New("latestN cardinality is statically incompatible with size")
	}
	return nil
}

func prepareFilterLatestN(condition *FilterLatestNCondition, calendar *unified.Calendar) (unified.Rule, error) {
	if condition == nil || condition.LatestN.Predicate == nil {
		return unified.Rule{}, errors.New("latestN requires predicate")
	}
	latest := condition.LatestN
	size, err := parseFilterCountInteger(latest.Size)
	if err != nil {
		return unified.Rule{}, errors.New("latestN size must be an integer from 1 to 10")
	}
	if size == "0" || size.compare("10") > 0 {
		return unified.Rule{}, errors.New("latestN size must be an integer from 1 to 10")
	}
	err = latest.Quantifier.validate(size)
	if err != nil {
		return unified.Rule{}, err
	}
	base, err := prepareFilterSelection(latest.Base, latest.Predicate, calendar, occurrenceFilterRule)
	if err != nil {
		return unified.Rule{}, err
	}
	if latest.Quantifier.Kind != "all" {
		return unified.Rule{}, &unified.UnsupportedError{Capability: "latestN Some and Cardinality"}
	}
	n, _ := size.uint64() // The validated size is at most 10.
	return unified.Rule{LatestN: &unified.LatestN{Size: int(n), Scope: base.Scope, Where: base.Where,
		Currency: base.Currency, Predicate: base.Verify}}, nil
}
