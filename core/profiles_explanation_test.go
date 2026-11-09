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
