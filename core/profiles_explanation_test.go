// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

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
