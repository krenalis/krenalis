// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

const lastPreviousConditionJSON = `{"lastVersusPrevious":{"base":{"source":{"kind":"recordedEvents",` +
	`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory",` +
	`"history":{"definition":"registered-queryable-events","meaningRevision":"1"}},` +
	`"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"},` +
	`"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},` +
	`"check":{"kind":"difference","operator":"is less than or equal to","values":[-30]}}}`

// TestExplainLastPreviousNumbers preserves exact requested comparisons.
func TestExplainLastPreviousNumbers(t *testing.T) {
	for _, value := range []string{"100", "-30", "1e2", "100.00001", "1e75", "-1e-37"} {
		t.Run(value, func(t *testing.T) {
			var condition FilterLastVersusPreviousCondition
			err := json.Unmarshal([]byte(strings.Replace(lastPreviousConditionJSON, "[-30]", "["+value+"]", 1)), &condition)
			if err != nil {
				t.Fatalf("expected condition decoding, got %v", err)
			}
			rule, err := prepareFilterLastVersusPrevious(&condition, nil)
			if err != nil {
				t.Fatalf("expected prepared pair, got %v", err)
			}
			at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			for _, selection := range []warehouses.OccurrencePairSelection{
				{Candidates: 2, Unique: true}, {Candidates: 2},
				{Candidates: 2, AbsencePossible: true}, {AbsencePossible: true},
			} {
				index := 0
				result, err := explainProfileRule(&warehouses.LastPreviousDifference{}, rule,
					[]warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown, Pair: &selection}}, &index, at)
				if err != nil {
					t.Fatalf("expected pair explanation, got %v", err)
				}
				data, err := json.Marshal(result.LastVersusPrevious.Check.Requested)
				if err != nil {
					t.Fatalf("expected exact requested number, got %v", err)
				}
				var requested struct {
					Operator FilterOperator `json:"operator"`
					Values   []json.Value   `json:"values"`
				}
				err = json.Unmarshal(data, &requested)
				if err != nil {
					t.Fatalf("expected echoed number decoding, got %v", err)
				}
				echoed, err := (FilterLastPreviousCheck{Kind: "difference", Operator: requested.Operator,
					Values: requested.Values}).prepare()
				if err != nil {
					t.Fatalf("expected echoed test accepted as input, got %v", err)
				}
				original := rule.LastVersusPrevious.Check
				if echoed.Kind != original.Kind || echoed.Test.Operator != original.Test.Operator ||
					!echoed.Test.Value.Equal(original.Test.Value) {
					t.Fatalf("expected same echoed check %+v, got %+v", original, echoed)
				}
				kind := "ambiguous"
				if selection.Unique {
					kind = "unique"
				}
				if selection.Candidates == 0 {
					kind = "absent"
				}
				if result.LastVersusPrevious.Selection.Kind != kind {
					t.Fatalf("expected selection %s, got %+v", kind, result.LastVersusPrevious.Selection)
				}
				_, err = json.Marshal(result)
				if err != nil {
					t.Fatalf("expected normalized explanation encoding, got %v", err)
				}
			}
		})
	}
}

// TestFilterLastPreviousChecks validates checks independently of availability.
func TestFilterLastPreviousChecks(t *testing.T) {
	for _, kind := range []string{"difference", "ratio"} {
		for _, operator := range []FilterOperator{OpIs, OpIsNot, OpIsLessThan, OpIsLessThanOrEqualTo,
			OpIsGreaterThan, OpIsGreaterThanOrEqualTo, OpIsBetween} {
			values := []json.Value{json.Value("-3e1")}
			if operator == OpIsBetween {
				values = append(values, json.Value("3e1"))
			}
			check := FilterLastPreviousCheck{Kind: kind, Operator: operator, Values: values}
			_, err := check.prepare()
			if err != nil {
				t.Fatalf("expected valid %s/%s, got %v", kind, operator, err)
			}
		}
	}
	for _, change := range []struct{ old, replacement string }{
		{`,"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"`, ``},
		{`"operator":"is less than or equal to","values":[-30]`, `"operator":"is between","values":[-30,30]`},
	} {
		input := strings.Replace(lastPreviousConditionJSON, change.old, change.replacement, 1)
		for _, invalid := range []struct{ old, replacement string }{
			{`commerce.order_completed.total`, `unknown.total`},
			{`"name":"Order Completed"`, `"name":"Refund"`},
			{`"is less than or equal to"`, `"contains"`},
			{`"source":`, `"where":{"operator":"and","rules":[{"property":"unknown","operator":"is","values":[0]}]},"source":`},
		} {
			changed := strings.Replace(input, invalid.old, invalid.replacement, 1)
			if changed == input {
				continue
			}
			var condition FilterLastVersusPreviousCondition
			err := json.Unmarshal([]byte(changed), &condition)
			if err != nil {
				t.Fatalf("expected structural decoding before validation, got %v", err)
			}
			_, err = prepareFilterLastVersusPrevious(&condition, nil)
			if err != nil {
				if _, unavailable := errors.AsType[*behavioral.UnsupportedError](err); unavailable {
					t.Fatalf("expected invalid parameter before unsupported check/currency, got %v", err)
				}
				continue
			}
			t.Fatal("expected invalid parameter, got success")
		}
	}
}

// TestFilterLastPreviousCodec checks the closed objects and Record boundary.
func TestFilterLastPreviousCodec(t *testing.T) {
	var filter Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+lastPreviousConditionJSON+`]}`), &filter)
	if err != nil {
		t.Fatalf("expected public pair decoding, got %v", err)
	}
	if !filter.HasBehavioralConditions() {
		t.Fatal("expected behavioral filter requirement, got false")
	}
	encoded, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected pair encoding, got %v", err)
	}
	var roundTrip Filter
	err = json.Unmarshal(encoded, &roundTrip)
	if err != nil {
		t.Fatalf("expected round trip decoding, got %v", err)
	}
	if !reflect.DeepEqual(filter, roundTrip) {
		t.Fatal("expected identical round trip, got different filters")
	}
	_, _, err = prepareFilter(&filter, types.Object([]types.Property{{Name: "x", Type: types.String()}}))
	if err == nil {
		t.Fatal("expected Record-only rejection, got success")
	}
	for _, test := range []struct{ name, old, replacement string }{
		{"missing check", `,"check":{"kind":"difference","operator":"is less than or equal to","values":[-30]}`, ``},
		{"null subject", `"lastVersusPrevious":{`, `"lastVersusPrevious":null,"extra":{`},
		{"null field", `"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"}`, `"field":null`},
		{"null check", `"check":{"kind":"difference","operator":"is less than or equal to","values":[-30]}`, `"check":null`},
		{"unknown property", `"field":`, `"other":0,"field":`},
		{"other subject", `"lastVersusPrevious":`, `"latestN":{},"lastVersusPrevious":`},
		{"outer values", `"lastVersusPrevious":`, `"values":[1],"lastVersusPrevious":`},
		{"null currency", `"nativeCurrency":"EUR"`, `"nativeCurrency":null`},
		{"null currency reference", `"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"}`, `"currencyField":null`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterLastVersusPreviousCondition
			err := json.Unmarshal([]byte(strings.Replace(lastPreviousConditionJSON, test.old, test.replacement, 1)), &condition)
			if err == nil {
				t.Fatal("expected closed codec rejection, got success")
			}
		})
	}
	for _, check := range []string{`{}`, `{"kind":null}`, `{"kind":"other"}`, `{"kind":"sameValue","values":[]}`,
		`{"kind":"differentValue","operator":"is"}`, `{"kind":"difference","operator":"is"}`,
		`{"kind":"ratio","values":[1]}`, `{"kind":"difference","operator":"is","values":null}`,
		`{"kind":"difference","kind":"ratio","operator":"is","values":[1]}`,
		`{"kind":"difference","operator":"is","values":[1],"extra":1}`} {
		var decoded FilterLastPreviousCheck
		err := json.Unmarshal([]byte(check), &decoded)
		if err == nil {
			t.Fatalf("expected closed check rejection for %s, got success", check)
		}
	}
}

// TestFilterLastPreviousValidation preserves invalid-before-unavailable order.
func TestFilterLastPreviousValidation(t *testing.T) {
	for _, test := range []struct {
		name, old, replacement string
		unavailable            bool
	}{
		{"same", `{"kind":"difference","operator":"is less than or equal to","values":[-30]}`, `{"kind":"sameValue"}`, true},
		{"different", `{"kind":"difference","operator":"is less than or equal to","values":[-30]}`, `{"kind":"differentValue"}`, true},
		{"ratio", `"kind":"difference"`, `"kind":"ratio"`, true},
		{"equality", `"is less than or equal to"`, `"is"`, true},
		{"inequality", `"is less than or equal to"`, `"is not"`, true},
		{"between", `"operator":"is less than or equal to","values":[-30]`, `"operator":"is between","values":[-30,30]`, true},
		{"missing currency pair", `,"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR"`, ``, true},
		{"raw parent", `"source":`, `"where":{"operator":"and","rules":[{"property":"properties","operator":"exists"}]},"source":`, true},
		{"raw child", `"source":`, `"where":{"operator":"or","rules":[{"property":"properties.total.x","operator":"exists"}]},"source":`, true},
		{"raw currency", `"source":`, `"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]},"source":`, true},
		{"currency alone", `,"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"}`, ``, false},
		{"reference alone", `,"nativeCurrency":"EUR"`, ``, false},
		{"bad reference", `commerce.order_completed.total`, `unknown.total`, false},
		{"wrong source", `"name":"Order Completed"`, `"name":"Refund"`, false},
		{"non-track source", `"name":"Order Completed"`, `"type":"page","name":"Order Completed"`, false},
		{"wrong operator", `"is less than or equal to"`, `"contains"`, false},
		{"inverted", `"operator":"is less than or equal to","values":[-30]`, `"operator":"is between","values":[30,-30]`, false},
		{"string threshold", `[-30]`, `["30"]`, false},
		{"huge literal", `[-30]`, `[1` + strings.Repeat("0", 4097) + `]`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterLastVersusPreviousCondition
			err := json.Unmarshal([]byte(strings.Replace(lastPreviousConditionJSON, test.old, test.replacement, 1)), &condition)
			if err != nil {
				if test.unavailable {
					t.Fatalf("expected structurally valid condition, got %v", err)
				}
				return
			}
			_, err = prepareFilterLastVersusPrevious(&condition, nil)
			if err != nil {
				_, unsupported := errors.AsType[*behavioral.UnsupportedError](err)
				if unsupported != test.unavailable {
					t.Fatalf("expected unavailable=%v, got %v", test.unavailable, err)
				}
			}
			if err == nil {
				t.Fatal("expected invalid or unavailable condition, got success")
			}
			if !test.unavailable {
				return
			}
			invalid := &FilterCondition{Property: "x", Operator: OpContains, Values: []json.Value{json.Value("1")}}
			schema := types.Object([]types.Property{{Name: "x", Type: types.Int(64)}})
			for _, rules := range [][]FilterRule{{&condition, invalid}, {invalid, &condition}} {
				nodes := 0
				_, err := prepareBehavioralRule(&Filter{Operator: OpAnd, Rules: rules}, schema, 0, &nodes)
				if err != nil {
					if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalid sibling first, got %v", err)
					}
					continue
				}
				t.Fatal("expected invalid sibling, got success")
			}
		})
	}
}

// TestNormalizedDifferenceForms rejects ambiguous explanation representations.
func TestNormalizedDifferenceForms(t *testing.T) {
	for _, test := range []struct {
		comparison ProfileNormalizedDifferenceComparison
		invalid    bool
	}{
		{ProfileNormalizedDifferenceComparison{}, true},
		{ProfileNormalizedDifferenceComparison{ReadableDifferences: "unknown"}, true},
		{ProfileNormalizedDifferenceComparison{ReadableDifferences: "all", Operator: OpIsGreaterThanOrEqualTo, Values: []decimal.Decimal{decimal.Decimal{}}}, true},
		{ProfileNormalizedDifferenceComparison{Operator: OpIsGreaterThanOrEqualTo, Values: []decimal.Decimal{decimal.MustParse("1.00001")}}, true},
		{ProfileNormalizedDifferenceComparison{Operator: OpIsGreaterThanOrEqualTo, Values: []decimal.Decimal{decimal.MustParse("199999999999999.9999")}}, true},
		{ProfileNormalizedDifferenceComparison{Operator: OpIsGreaterThanOrEqualTo, Values: []decimal.Decimal{decimal.MustParse("-199999999999999.9999")}}, true},
		{ProfileNormalizedDifferenceComparison{ReadableDifferences: "none"}, false},
		{ProfileNormalizedDifferenceComparison{ReadableDifferences: "all"}, false},
		{ProfileNormalizedDifferenceComparison{Operator: OpIsGreaterThanOrEqualTo, Values: []decimal.Decimal{decimal.Decimal{}}}, false},
	} {
		_, err := json.Marshal(test.comparison)
		if err != nil {
			if !test.invalid {
				t.Fatalf("expected normalized form, got %v", err)
			}
			continue
		}
		if test.invalid {
			t.Fatal("expected invalid normalized form, got success")
		}
	}
}
