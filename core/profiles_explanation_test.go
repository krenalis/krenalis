// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestExplainCountMoney retains monetary metadata in the complete explanation.
func TestExplainCountMoney(t *testing.T) {

	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved monetary field, got %v", err)
	}
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "USD", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved Count currency, got %v", err)
	}
	record := unified.OccurrencePredicate{Record: &state.WhereCondition{
		Property: []string{"properties", "flag"}, Operator: state.OpIsTrue}}
	money := unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
		Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("9007199254740.00001")}}}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	physical := &warehouses.IndependentOccurrenceCount{Start: at.Add(-time.Hour), End: at}
	nodes := []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown,
		Count: &warehouses.OccurrenceCountBounds{Certain: 2, Possible: 3}}}

	for _, test := range []struct {
		name  string
		where *unified.OccurrencePredicate
		path  []int
	}{
		{"Record only", &unified.OccurrencePredicate{Children: []unified.OccurrencePredicate{record}}, nil},
		{"money", &unified.OccurrencePredicate{Operator: state.OpOr, Children: []unified.OccurrencePredicate{record, money}}, []int{1}},
		{"nested money", &unified.OccurrencePredicate{Operator: state.OpOr, Children: []unified.OccurrencePredicate{record,
			{Children: []unified.OccurrencePredicate{money}}}}, []int{1, 0}},
	} {
		t.Run(test.name, func(t *testing.T) {

			semantic := unified.Rule{Count: &unified.Count{Currency: currency, Where: test.where,
				Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 3}}}
			index := 0
			result, err := explainProfileRule(physical, semantic, nodes, &index, at)
			if err != nil {
				t.Fatalf("expected complete Count explanation, got %v", err)
			}
			count := result.Count
			if result.Kind != "count" || result.Result != "UNKNOWN" || count == nil || index != 1 {
				t.Fatalf("expected one UNKNOWN Count explanation, got %+v at index %d", result, index)
			}
			if !count.Start.Equal(physical.Start) || !count.End.Equal(at) || count.Certain != 2 || count.Uncertain != 1 ||
				count.CurrencyField == nil || count.CurrencyField.Definition != "commerce.order_completed.currency" ||
				count.CurrencyField.MeaningRevision != "1" || count.NativeCurrency != "USD" {
				t.Fatalf("expected measured Count bounds and separate USD currency, got %+v", count)
			}
			if test.path == nil {
				encoded, err := json.Marshal(result)
				if err != nil {
					t.Fatalf("expected encoded Record-only Count explanation, got %v", err)
				}
				if count.Money != nil || strings.Contains(string(encoded), `"money":`) {
					t.Fatalf("expected money omitted without a monetary predicate, got %s", encoded)
				}
				return
			}
			if len(count.Money) != 1 {
				t.Fatalf("expected one monetary explanation, got %+v", count.Money)
			}
			leaf := count.Money[0]
			if !reflect.DeepEqual(leaf.Path, test.path) || leaf.Field.Definition != "commerce.order_completed.total" ||
				leaf.Field.MeaningRevision != "1" || leaf.NativeCurrency != "EUR" {
				t.Fatalf("expected monetary reference and where path %v, got %+v", test.path, leaf)
			}
			if leaf.Requested.Operator != OpIsGreaterThan || len(leaf.Requested.Values) != 1 ||
				!leaf.Requested.Values[0].Equal(decimal.MustParse("9007199254740.00001")) ||
				leaf.Normalized.Operator != OpIsGreaterThanOrEqualTo || len(leaf.Normalized.Values) != 1 ||
				!leaf.Normalized.Values[0].Equal(decimal.MustParse("9007199254740.0001")) {
				t.Fatalf("expected exact requested >9007199254740.00001 and normalized >=9007199254740.0001, got %+v", leaf)
			}

		})
	}

}

// TestExplainExperienceNth retains the request even when Nth(1) equals First.
func TestExplainExperienceNth(t *testing.T) {
	for _, position := range []int{1, 3, 10} {
		physical := &warehouses.SelectedOccurrence{Nth: position}
		semantic := unified.Rule{Experience: &unified.Experience{Nth: position}}
		nodes := []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterMatch,
			Experience: &warehouses.OccurrenceSelection{Candidates: 2}}}
		index := 0
		result, err := explainProfileRule(physical, semantic, nodes, &index, time.Time{})
		if err != nil {
			t.Fatalf("expected ordinal explanation, got %v", err)
		}
		selector := result.Experience.Select
		if selector.Kind != "nth" || string(selector.Position) != strconv.Itoa(position) ||
			result.Experience.Selection.Kind != "ambiguous" || result.Experience.Selection.Candidates != 2 {
			t.Fatalf("expected Nth(%d) with two candidates, got %+v", position, result.Experience)
		}
	}
}

// TestExplainExperienceSelection distinguishes candidate and absence states.
func TestExplainExperienceSelection(t *testing.T) {
	for _, test := range []struct {
		first      bool
		candidates int64
		absent     bool
		kind       string
		absence    string
	}{
		{false, 0, true, "absent", "certain"},
		{false, 1, true, "ambiguous", "possible"},
		{false, 1, false, "unique", "impossible"},
		{false, 2, false, "ambiguous", "impossible"},
		{true, 0, true, "absent", "certain"},
		{true, 1, true, "ambiguous", "possible"},
		{true, 1, false, "unique", "impossible"},
		{true, 2, false, "ambiguous", "impossible"},
	} {
		nodes := []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown,
			Experience: &warehouses.OccurrenceSelection{Candidates: test.candidates, AbsencePossible: test.absent}}}
		if test.candidates == 0 {
			nodes[0].Truth = warehouses.FilterNoMatch
		}
		index := 0
		rule := &warehouses.SelectedOccurrence{First: test.first}
		semantic := unified.Rule{Experience: &unified.Experience{First: test.first}}
		result, err := explainProfileRule(rule, semantic, nodes, &index, time.Time{})
		if err != nil {
			t.Fatalf("expected experience explanation, got %v", err)
		}
		selector := "last"
		if test.first {
			selector = "first"
		}
		if result.Experience == nil || result.Experience.Select.Kind != selector ||
			result.Experience.Selection.Kind != test.kind ||
			result.Experience.Selection.Absence != test.absence || result.Experience.Selection.Candidates != test.candidates {
			t.Fatalf("expected %s/%s with %d candidates, got %+v", test.kind, test.absence, test.candidates, result)
		}
	}
}

// TestExplainMoneyLiterals maps each new test into Count and E explanations.
func TestExplainMoneyLiterals(t *testing.T) {
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved monetary field, got %v", err)
	}
	for _, test := range []struct {
		name                    string
		operator                FilterOperator
		comparison              unified.Comparison
		value, upper, low, high string
		domain                  string
	}{
		{"equality", OpIs, unified.Equal, "100.00000", "", "100", "", ""},
		{"different", OpIsNot, unified.Different, "100", "", "100", "", ""},
		{"interval", OpIsBetween, unified.Between, "50.00001", "100.00009", "50.0001", "100", ""},
		{"collapsed interval", OpIsBetween, unified.Between, "99.99995", "100.00004", "100", "100", ""},
		{"no readable equality", OpIs, unified.Equal, "100.00001", "", "", "", "none"},
		{"all readable different", OpIsNot, unified.Different, "100.00001", "", "", "", "all"},
		{"no readable interval", OpIsBetween, unified.Between, "100.00001", "100.00009", "", "", "none"},
		{"all readable interval", OpIsBetween, unified.Between, "-1e20", "1e20", "", "", "all"},
	} {
		for _, kind := range []string{"count", "experience", "latestN"} {
			t.Run(test.name+"/"+kind, func(t *testing.T) {
				requested := []decimal.Decimal{decimal.MustParse(test.value)}
				money := &unified.Money{Field: field, NativeCurrency: "EUR",
					Test: unified.SumTest{Operator: test.comparison, Value: requested[0]}}
				if test.upper != "" {
					money.Test.Upper = decimal.MustParse(test.upper)
					requested = append(requested, money.Test.Upper)
				}
				body := &unified.OccurrencePredicate{Children: []unified.OccurrencePredicate{
					{Operator: state.OpOr, Children: []unified.OccurrencePredicate{{Money: money}}},
				}}
				var physical warehouses.FilterCondition = &warehouses.IndependentOccurrenceCount{}
				semantic := unified.Rule{Count: &unified.Count{Where: body}}
				nodes := []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown,
					Count: &warehouses.OccurrenceCountBounds{Possible: 1}}}
				if kind == "experience" {
					physical = &warehouses.SelectedOccurrence{}
					semantic = unified.Rule{Experience: &unified.Experience{Verify: body}}
					nodes[0].Experience = &warehouses.OccurrenceSelection{Candidates: 1, AbsencePossible: true}
				}
				if kind == "latestN" {
					physical = &warehouses.LatestOccurrences{Size: 1}
					semantic = unified.Rule{LatestN: &unified.LatestN{Size: 1, Predicate: body}}
					nodes[0].Experience = &warehouses.OccurrenceSelection{Candidates: 1, AbsencePossible: true}
				}
				index := 0
				result, err := explainProfileRule(physical, semantic, nodes, &index, time.Time{})
				if err != nil {
					t.Fatalf("expected complete monetary explanation, got %v", err)
				}
				if result.Kind != kind || result.Result != "UNKNOWN" || index != 1 ||
					kind == "count" && result.Count == nil || kind == "experience" && result.Experience == nil ||
					kind == "latestN" && result.LatestN == nil {
					t.Fatalf("expected one UNKNOWN %s explanation, got %+v", kind, result)
				}
				var leaves []ProfileMoneyExplanation
				if kind == "count" {
					leaves = result.Count.Money
				} else if kind == "experience" {
					leaves = result.Experience.Money
				} else {
					leaves = result.LatestN.Money
				}
				if len(leaves) != 1 {
					t.Fatalf("expected one explained monetary leaf, got %+v", leaves)
				}
				leaf := leaves[0]
				if !reflect.DeepEqual(leaf.Path, []int{0, 0}) || leaf.NativeCurrency != "EUR" ||
					leaf.Field.Definition != "commerce.order_completed.total" || leaf.Field.MeaningRevision != "1" ||
					leaf.Requested.Operator != test.operator || len(leaf.Requested.Values) != len(requested) {
					t.Fatalf("expected resolved reference, owner path and requested test, got %+v", leaf)
				}
				for i, value := range requested {
					if !leaf.Requested.Values[i].Equal(value) {
						t.Fatalf("expected requested operand %s, got %s", value, leaf.Requested.Values[i])
					}
				}
				if leaf.Normalized.ReadableAmounts != test.domain {
					t.Fatalf("expected readable-amount domain %q, got %+v", test.domain, leaf.Normalized)
				}
				if test.domain != "" {
					if leaf.Normalized.Operator != "" || leaf.Normalized.Values != nil {
						t.Fatalf("expected only the readable-amount domain, got %+v", leaf.Normalized)
					}
					return
				}
				if leaf.Normalized.Operator != test.operator || len(leaf.Normalized.Values) != len(requested) ||
					!leaf.Normalized.Values[0].Equal(decimal.MustParse(test.low)) ||
					test.high != "" && !leaf.Normalized.Values[1].Equal(decimal.MustParse(test.high)) {
					t.Fatalf("expected normalized %s [%s,%s], got %+v", test.operator, test.low, test.high, leaf.Normalized)
				}
			})
		}
	}
}

func TestExplainProfileFilterValidation(t *testing.T) {
	workspace := &Workspace{core: &Core{}, workspace: &state.Workspace{ProfileSchema: types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
	})}}
	for _, test := range []struct {
		name, kpid, at string
		filter         string
		code           errors.Code
	}{
		{"bad KPID", "bad", "", `{"operator":"and","rules":[]}`, ""},
		{"missing filter", "00000000-0000-0000-0000-000000000001", "", `null`, ""},
		{"bad at", "00000000-0000-0000-0000-000000000001", "bad", `{"operator":"and","rules":[{"property":"email","operator":"exists"}]}`, ""},
		{"empty group", "00000000-0000-0000-0000-000000000001", "", `{"operator":"and","rules":[]}`, ""},
		{"missing property", "00000000-0000-0000-0000-000000000001", "", `{"operator":"and","rules":[{"property":"missing","operator":"exists"}]}`, PropertyNotExist},
		{"unsupported", "00000000-0000-0000-0000-000000000001", "", strings.Replace(`{"operator":"and","rules":[`+countConditionJSON+`]}`, `"values":[0]`, `"values":[18446744073709551616]`, 1), FilterCapabilityNotImplemented},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter *Filter
			err := json.Unmarshal([]byte(test.filter), &filter)
			if err != nil {
				t.Fatalf("expected decodable filter fixture, got %v", err)
			}
			result, err := workspace.ExplainProfileFilter(t.Context(), test.kpid, filter, test.at)
			if err != nil {
				if result != nil {
					t.Fatalf("expected no partial explanation, got %+v", result)
				}
				if test.code != "" {
					failure, ok := errors.AsType[*errors.UnprocessableError](err)
					if !ok || failure.Code != test.code {
						t.Fatalf("expected Unprocessable %s, got %v", test.code, err)
					}
					return
				}
				if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
					t.Fatalf("expected BadRequest, got %v", err)
				}
				return
			}
			t.Fatal("expected request validation failure, got success")
		})
	}
	for _, at := range []string{time.Now().Add(time.Hour).Format(time.RFC3339),
		"2026-01-01T00:00:00.000000001Z", "2026-01-01T00:00:00.0000000001Z"} {
		root := &Filter{Operator: OpAnd, Rules: []FilterRule{&FilterCondition{Property: "email", Operator: OpExists}}}
		_, err := workspace.ExplainProfileFilter(t.Context(), "00000000-0000-0000-0000-000000000001", root, at)
		if err != nil {
			if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
				t.Fatalf("expected invalid evaluation instant, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid evaluation instant, got success")
	}
}

// TestProfileNormalizedMoneyComparison checks the two closed JSON forms.
func TestProfileNormalizedMoneyComparison(t *testing.T) {
	point := []decimal.Decimal{decimal.MustParse("100")}
	for _, test := range []struct {
		name  string
		value ProfileNormalizedMoneyComparison
		valid bool
	}{
		{"comparison", ProfileNormalizedMoneyComparison{Operator: OpIs, Values: point}, true},
		{"none", ProfileNormalizedMoneyComparison{ReadableAmounts: "none"}, true},
		{"all", ProfileNormalizedMoneyComparison{ReadableAmounts: "all"}, true},
		{"neither", ProfileNormalizedMoneyComparison{}, false},
		{"both", ProfileNormalizedMoneyComparison{ReadableAmounts: "all", Operator: OpIs, Values: point}, false},
		{"empty values with domain", ProfileNormalizedMoneyComparison{ReadableAmounts: "none", Values: []decimal.Decimal{}}, false},
		{"unknown domain", ProfileNormalizedMoneyComparison{ReadableAmounts: "some"}, false},
		{"operator only", ProfileNormalizedMoneyComparison{Operator: OpIs}, false},
		{"values only", ProfileNormalizedMoneyComparison{Values: point}, false},
		{"wrong arity", ProfileNormalizedMoneyComparison{Operator: OpIsBetween, Values: point}, false},
		{"inverted", ProfileNormalizedMoneyComparison{Operator: OpIsBetween, Values: []decimal.Decimal{point[0], decimal.MustParse("50")}}, false},
		{"fine", ProfileNormalizedMoneyComparison{Operator: OpIs, Values: []decimal.Decimal{decimal.MustParse("100.00001")}}, false},
		{"large", ProfileNormalizedMoneyComparison{Operator: OpIs, Values: []decimal.Decimal{decimal.MustParse("1e15")}}, false},
		{"unknown operator", ProfileNormalizedMoneyComparison{Operator: OpExists, Values: point}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.value)
			if err != nil {
				if test.valid {
					t.Fatalf("expected normalized comparison encoded, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid normalized comparison rejected, got %s", encoded)
			}
			if test.value.ReadableAmounts != "" && string(encoded) != `{"readableAmounts":"`+test.value.ReadableAmounts+`"}` {
				t.Fatalf("expected only the readable-amount domain, got %s", encoded)
			}
		})
	}
}
