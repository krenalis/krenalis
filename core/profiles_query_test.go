// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestFilterRequiresUnified(t *testing.T) {
	count := &FilterCountCondition{}
	record := &FilterCondition{Property: "email", Operator: OpExists}
	deep := &Filter{Operator: OpAnd, Rules: []FilterRule{count}}
	for range 3 {
		deep = &Filter{Operator: OpAnd, Rules: []FilterRule{deep}}
	}
	cycle := &Filter{Operator: OpAnd}
	cycle.Rules = []FilterRule{cycle}
	for _, test := range []struct {
		name string
		root *Filter
		want bool
	}{
		{"nil", nil, false},
		{"Record", &Filter{Operator: OpAnd, Rules: []FilterRule{record}}, false},
		{"mixed", &Filter{Operator: OpOr, Rules: []FilterRule{record, count}}, true},
		{"Sum", &Filter{Rules: []FilterRule{&FilterSumCondition{}}}, true},
		{"Average", &Filter{Rules: []FilterRule{&FilterAverageCondition{}}}, true},
		{"experience", &Filter{Rules: []FilterRule{&FilterExperienceCondition{}}}, true},
		{"latestN", &Filter{Rules: []FilterRule{&FilterLatestNCondition{}}}, true},
		{"lastVersusPrevious", &Filter{Rules: []FilterRule{&FilterLastVersusPreviousCondition{}}}, true},
		{"occurredIn", &Filter{Rules: []FilterRule{&FilterOccurredInCondition{}}}, true},
		{"money", &Filter{Rules: []FilterRule{&FilterMoneyCondition{}}}, true},
		{"four levels", deep, true},
		{"typed nil", &Filter{Rules: []FilterRule{(*Filter)(nil)}}, false},
		{"nil Count", &Filter{Rules: []FilterRule{(*FilterCountCondition)(nil)}}, true},
		{"cycle", cycle, false},
		{"too deep", &Filter{Rules: []FilterRule{deep}}, false},
		{"too wide", &Filter{Rules: make([]FilterRule, 101)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.root.RequiresUnified(); got != test.want {
				t.Fatalf("expected RequiresUnified %t, got %t", test.want, got)
			}
		})
	}
}

func TestQueryProfilesCalendarValidation(t *testing.T) {
	workspace := &Workspace{core: &Core{}, workspace: &state.Workspace{ProfileSchema: types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
	})}}
	for _, test := range []struct{ name, definition, revision string }{
		{"unknown definition", "unknown", "1"},
		{"unknown revision", "gregorian-monday/Europe/Rome", "2"},
		{"nested calendar", "gregorian-monday/Europe/Rome", "1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			filter := &Filter{Operator: OpAnd,
				DefaultCalendar: &FilterCalendar{Definition: test.definition, MeaningRevision: test.revision},
				Rules:           []FilterRule{&FilterCondition{Property: "email", Operator: OpExists}}}
			if test.name == "nested calendar" {
				filter = &Filter{Operator: OpAnd, Rules: []FilterRule{filter}}
			}
			result, err := workspace.QueryProfiles(t.Context(), ProfileQuery{Filter: filter, Limit: 10})
			if err != nil {
				if _, ok := errors.AsType[*errors.BadRequestError](err); !ok || result != nil {
					t.Fatalf("expected BadRequest without results, got %v, %v", result, err)
				}
				return
			}
			t.Fatal("expected invalid Record calendar, got success")
		})
	}
}

func TestQueryProfilesValidation(t *testing.T) {
	workspace := &Workspace{core: &Core{}, workspace: &state.Workspace{ProfileSchema: types.Object([]types.Property{
		{Name: "email", Type: types.String(), ReadOptional: true},
		{Name: "document", Type: types.JSON(), ReadOptional: true},
	})}}
	for _, test := range []struct {
		name   string
		change func(*ProfileQuery)
		code   errors.Code
	}{
		{"empty properties", func(q *ProfileQuery) { q.Properties = []string{} }, ""},
		{"duplicate property", func(q *ProfileQuery) { q.Properties = []string{"email", "email"} }, ""},
		{"invalid property", func(q *ProfileQuery) { q.Properties = []string{"bad-name"} }, ""},
		{"missing property", func(q *ProfileQuery) { q.Properties = []string{"missing"} }, PropertyNotExist},
		{"missing order", func(q *ProfileQuery) { q.Order = "missing" }, OrderNotExist},
		{"JSON order", func(q *ProfileQuery) { q.Order = "document" }, OrderTypeNotSortable},
		{"invalid order", func(q *ProfileQuery) { q.Order = "bad-name" }, ""},
		{"negative offset", func(q *ProfileQuery) { q.First = -1 }, ""},
		{"zero limit", func(q *ProfileQuery) { q.Limit = 0 }, ""},
		{"large limit", func(q *ProfileQuery) { q.Limit = 1001 }, ""},
		{"future", func(q *ProfileQuery) { q.At = time.Now().Add(time.Hour).Format(time.RFC3339Nano) }, ""},
		{"zero", func(q *ProfileQuery) { q.At = "0001-01-01T00:00:00Z" }, ""},
		{"nanoseconds", func(q *ProfileQuery) { q.At = "2026-02-01T00:00:00.000000001Z" }, ""},
		{"truncated fraction", func(q *ProfileQuery) { q.At = "2026-02-01T00:00:00.0000000001Z" }, ""},
		{"invalid instant", func(q *ProfileQuery) { q.At = "not-an-instant" }, ""},
		{"oversized instant", func(q *ProfileQuery) { q.At = strings.Repeat("0", 1000) }, ""},
		{"Record at", func(q *ProfileQuery) { q.Filter = nil }, ""},
		{"invalid Count", func(q *ProfileQuery) { q.Filter.Rules[0].(*FilterCountCondition).Values = filterValues(-1) }, ""},
		{"missing qualification property", func(q *ProfileQuery) {
			q.Filter.Rules[0].(*FilterCountCondition).Count.Where = &Filter{Operator: OpAnd, Rules: []FilterRule{
				&FilterCondition{Property: "missing", Operator: OpExists}}}
		}, PropertyNotExist},
		{"unavailable Count", func(q *ProfileQuery) {
			q.Filter.Rules[0].(*FilterCountCondition).Values = []json.Value{json.Value(`18446744073709551616`)}
		}, FilterCapabilityNotImplemented},
	} {
		t.Run(test.name, func(t *testing.T) {
			var root Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+countConditionJSON+`]}`), &root)
			if err != nil {
				t.Fatalf("expected Count fixture, got %v", err)
			}
			query := ProfileQuery{Filter: &root, Limit: 10, At: "2026-02-01T00:00:00Z"}
			test.change(&query)
			result, err := workspace.QueryProfiles(t.Context(), query)
			if err != nil {
				if test.code == "" {
					if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
						t.Fatalf("expected BadRequest, got %T: %v", err, err)
					}
				} else {
					failure, ok := errors.AsType[*errors.UnprocessableError](err)
					if !ok || failure.Code != test.code {
						t.Fatalf("expected Unprocessable %s, got %T: %v", test.code, err, err)
					}
				}
			}
			if err == nil || result != nil {
				t.Fatalf("expected validation error without result, got %+v, %v", result, err)
			}
		})
	}
}
