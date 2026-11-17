// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralPrepare checks resolved source metadata without a
// backend/dialect.
func TestBehavioralPrepare(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	rule := behavioral.Rule{Count: &behavioral.Count{Scope: behavioral.Scope{Event: "Purchase'",
		Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Currency: &behavioral.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"},
		Test:     behavioral.CountTest{Operator: behavioral.Different, Value: 2}, Minimum: 1}}
	bindings := behavioral.Bindings{Currencies: []behavioral.CurrencyBinding{{Definition: "NativeCurrency",
		Revision: "1", Path: []string{"properties", "a.b"}, Evidence: behavioral.IndependentOccurrenceValues}}}
	calls := 0
	source := behavioral.SourceBinding{
		EventType: eventColumnByProperty["type"], Event: eventColumnByProperty["event"],
		Name: eventColumnByProperty["name"],
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		Record: func(where *state.Where) (warehouses.Expr, error) {
			calls++
			return convertWhere(where, eventColumnByProperty)
		},
	}
	filter, err := behavioral.Prepare(&rule, bindings, at, source)
	if err != nil {
		t.Fatalf("expected prepared filter, got %v", err)
	}
	count := filter.Root.(*warehouses.IndependentOccurrenceCount)
	if !reflect.DeepEqual(count.Source, source.Occurrences) || !count.Start.Equal(rule.Count.Scope.Start) ||
		!count.End.Equal(at) {
		t.Fatalf("expected resolved source and period, got %+v", count)
	}
	quals := count.Qualification.(*warehouses.QualificationGroup)
	selector := quals.Children[0].(*warehouses.QualificationGroup)
	wantSelector := &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: []warehouses.RowQualification{
		&warehouses.ExactColumnText{Column: eventColumnByProperty["event"], Value: "Purchase'"},
		&warehouses.ExactColumnText{Column: eventColumnByProperty["type"], Value: "track"},
	}}
	if !reflect.DeepEqual(selector, wantSelector) || calls != 0 {
		t.Fatalf("expected exact track name binding, got %+v and %d calls", selector, calls)
	}
	currency := quals.Children[1].(*warehouses.KnownTextEqual)
	if currency.Value != "EUR" || !reflect.DeepEqual(currency.Keys, []string{"a.b"}) {
		t.Fatalf("expected exact bound currency path, got %+v", currency)
	}
	if len(count.Accept) != 2 || count.Accept[0].Lower != 0 || *count.Accept[0].Upper != 1 ||
		count.Accept[1].Lower != 3 || count.Accept[1].Upper != nil {
		t.Fatalf("expected [0,1] and [3,infinity], got %+v", count.Accept)
	}
	if count.Minimum != 1 {
		t.Fatalf("expected separate sample minimum 1, got %d", count.Minimum)
	}
	history, err := behavioral.ResolveHistory("registered-queryable-events", "1")
	if err != nil {
		t.Fatalf("expected resolved history, got %v", err)
	}
	historyCount := *rule.Count
	historyCount.Scope = behavioral.Scope{Event: "Purchase'", Grain: "registered-message", History: history}
	historyFilter, err := behavioral.Prepare(&behavioral.Rule{Count: &historyCount}, bindings, at, source)
	if err != nil {
		t.Fatalf("expected prepared history filter, got %v", err)
	}
	preparedHistory := historyFilter.Root.(*warehouses.IndependentOccurrenceCount)
	if !preparedHistory.UnboundedStart || !preparedHistory.Start.IsZero() || !preparedHistory.End.Equal(at) ||
		!historyCount.Scope.End.IsZero() {
		t.Fatalf("expected explicit unbounded start and resolved T without changing input, got %+v", preparedHistory)
	}
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		dw := newWarehouse(platform, newTestSettingsLoader(nil), nil)
		backend, ok := dw.(warehouses.ProfileFilterWarehouse)
		if !ok || !backend.ProfileFilterVerified() {
			t.Fatalf("expected verified profile capability through warehouse factory, got %s", platform)
		}
		if _, ok := dw.(warehouses.ProfileFilterExplanationWarehouse); !ok {
			t.Fatalf("expected explanation capability through warehouse factory, got %s", platform)
		}
		if _, ok := dw.(warehouses.EventDiscoveryWarehouse); !ok {
			t.Fatalf("expected discovery capability through warehouse factory, got %s", platform)
		}
		err := dw.Close()
		if err != nil {
			t.Fatalf("expected warehouse cleanup, got %v", err)
		}
	}
	for _, change := range []func(*behavioral.Rule){
		func(r *behavioral.Rule) { r.Count.Shared = &behavioral.SharedBoolean{Fact: "g"} },
		func(r *behavioral.Rule) { r.Count.Scope.End = at.Add(time.Hour) },
		func(r *behavioral.Rule) {
			r.Count.Scope.Start = time.Time{}
			r.Count.Scope.End = time.Time{}
			r.Count.Scope.Elapsed = &behavioral.ElapsedPeriod{Older: time.Nanosecond}
		},
	} {
		c := *rule.Count
		r := behavioral.Rule{Count: &c}
		change(&r)
		calls = 0
		_, err := behavioral.Prepare(&r, bindings, at, source)
		if err != nil {
			if _, ok := errors.AsType[*behavioral.UnsupportedError](err); !ok {
				t.Fatalf("expected unsupported capability, got %v", err)
			}
		}
		if err == nil || calls != 0 {
			t.Fatalf("expected rejection before binding, got %v and %d calls", err, calls)
		}
	}
	for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
		group, err := behavioral.Prepare(&behavioral.Rule{Operator: op, Children: []behavioral.Rule{rule}},
			bindings, at, source)
		if err != nil {
			t.Fatalf("expected prepared logical group, got %v", err)
		}
		expected := warehouses.OpAnd
		if op == state.OpOr {
			expected = warehouses.OpOr
		}
		if group.Root.(*warehouses.FilterGroup).Operator != expected {
			t.Fatalf("expected logical operator %d, got %+v", expected, group.Root)
		}
	}
	_, err = behavioral.Prepare(&rule, behavioral.Bindings{}, at, source)
	if err == nil {
		t.Fatal("expected unresolved meaning revision error, got success")
	}
	bindings.Currencies[0].Evidence = 0
	_, err = behavioral.Prepare(&rule, bindings, at, source)
	if err == nil {
		t.Fatal("expected unsupported evidence model, got success")
	}
}

// TestBehavioralPrepareElapsedPeriods resolves offsets before the warehouse
// boundary without mutating the scope or relying on a dialect or Unix nanos.
func TestBehavioralPrepareElapsedPeriods(t *testing.T) {
	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatalf("expected Rome calendar, got %v", err)
	}
	for _, test := range []struct {
		name       string
		at         time.Time
		period     behavioral.ElapsedPeriod
		start, end time.Time
	}{
		{"spring lookback", time.Date(2026, 3, 29, 12, 0, 0, 0, rome), behavioral.ElapsedPeriod{Older: 24 * time.Hour}, time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC), time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)},
		{"spring ago band", time.Date(2026, 3, 29, 12, 0, 0, 0, rome), behavioral.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24 * time.Hour}, time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC), time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)},
		{"autumn lookback", time.Date(2026, 10, 25, 12, 0, 0, 0, rome), behavioral.ElapsedPeriod{Older: 24 * time.Hour}, time.Date(2026, 10, 24, 11, 0, 0, 0, time.UTC), time.Date(2026, 10, 25, 11, 0, 0, 0, time.UTC)},
		{"microsecond lookback", time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC), behavioral.ElapsedPeriod{Older: time.Microsecond}, time.Date(2026, 2, 1, 0, 0, 0, 123455000, time.UTC), time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)},
		{"microsecond ago band", time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC), behavioral.ElapsedPeriod{Older: 3 * time.Microsecond, Newer: time.Microsecond}, time.Date(2026, 2, 1, 0, 0, 0, 123453000, time.UTC), time.Date(2026, 2, 1, 0, 0, 0, 123455000, time.UTC)},
		{"maximum microsecond duration", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), behavioral.ElapsedPeriod{Older: time.Duration(math.MaxInt64/1000) * time.Microsecond}, time.Date(1677, 9, 21, 0, 12, 43, 145225000, time.UTC), time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"far future", time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), behavioral.ElapsedPeriod{Older: time.Microsecond}, time.Date(9999, 12, 31, 23, 59, 59, 999998000, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)},
		{"far past", time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC), behavioral.ElapsedPeriod{Older: time.Microsecond}, time.Date(1, 1, 1, 23, 59, 59, 999999000, time.UTC), time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {
			scope := behavioral.Scope{Event: "Purchase", Grain: "registered-message", Elapsed: &test.period}
			rule := behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{{Count: &behavioral.Count{Scope: scope}}}}
			source := behavioral.SourceBinding{
				EventType: eventColumnByProperty["type"], Event: eventColumnByProperty["event"],
				Name: eventColumnByProperty["name"],
				Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events",
					Profile:  eventColumnByProperty["kpid"],
					Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
				Record: func(where *state.Where) (warehouses.Expr, error) {
					return convertWhere(where, eventColumnByProperty)
				},
			}
			before := test.period
			filter, err := behavioral.Prepare(&rule, behavioral.Bindings{}, test.at, source)
			if err != nil {
				t.Fatalf("expected resolved elapsed period, got %v", err)
			}
			count := filter.Root.(*warehouses.FilterGroup).Children[0].(*warehouses.IndependentOccurrenceCount)
			if !count.Start.Equal(test.start) || !count.End.Equal(test.end) ||
				count.Start.Location() != time.UTC || count.End.Location() != time.UTC {
				t.Fatalf("expected UTC period [%s,%s), got [%s,%s)", test.start, test.end, count.Start, count.End)
			}
			if *rule.Children[0].Count.Scope.Elapsed != before ||
				!rule.Children[0].Count.Scope.Start.IsZero() || !rule.Children[0].Count.Scope.End.IsZero() {
				t.Fatalf("expected unchanged relative input %+v, got %+v", before, rule.Children[0].Count.Scope)
			}
		})
	}
}

// TestBehavioralPrepareProfileRecord binds without an occurrence source.
func TestBehavioralPrepareProfileRecord(t *testing.T) {

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	condition := &state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs, Values: []any{"Refund"}}
	rule := behavioral.Rule{Record: condition}
	source := behavioral.SourceBinding{
		Profiles: warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		ProfileRecord: func(where *state.Where) (warehouses.Expr, error) {
			return convertWhere(where, map[string]warehouses.Column{"event": {Name: "profile_event",
				Type: types.String(), Nullable: true}})
		},
	}
	filter, err := behavioral.Prepare(&rule, behavioral.Bindings{}, at, source)
	if err != nil {
		t.Fatalf("expected prepared profile Record, got %v", err)
	}
	expr := filter.Root.(*warehouses.RecordFilter).Expr.(*warehouses.MultiExpr).Operands[0].(*warehouses.BaseExpr)
	if expr.Column.Name != "profile_event" || expr.Operator != warehouses.OpIs || expr.Values[0] != "Refund" {
		t.Fatalf("expected population binding and exact operands, got %+v", expr)
	}

	for _, change := range []func(*behavioral.SourceBinding){
		func(s *behavioral.SourceBinding) { s.ProfileRecord = nil },
		func(s *behavioral.SourceBinding) { s.Profiles.Table = "" },
		func(s *behavioral.SourceBinding) {
			s.ProfileRecord = func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, nil) }
		},
	} {
		binding := source
		change(&binding)
		_, err := behavioral.Prepare(&rule, behavioral.Bindings{}, at, binding)
		if err == nil {
			t.Fatal("expected missing profile binding error, got success")
		}
	}

}

// TestBehavioralPrepareReferencePrecedence rejects unresolved revisions before
// unavailable evidence, regardless of their position in nested groups.
func TestBehavioralPrepareReferencePrecedence(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := behavioral.SourceBinding{
		EventType: eventColumnByProperty["type"], Event: eventColumnByProperty["event"],
		Name: eventColumnByProperty["name"],
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		Record: func(where *state.Where) (warehouses.Expr, error) {
			return convertWhere(where, eventColumnByProperty)
		},
	}
	for _, unresolved := range []bool{false, true} {
		for _, operator := range []state.WhereLogical{state.OpAnd, state.OpOr} {
			for _, reversed := range []bool{false, true} {
				t.Run(fmt.Sprintf("unresolved=%v/%d/reversed=%v", unresolved, operator, reversed), func(t *testing.T) {
					bindings := behavioral.Bindings{Currencies: []behavioral.CurrencyBinding{
						{Definition: "NativeCurrency", Revision: "1", Path: []string{"properties", "currency"}},
						{Definition: "NativeCurrency", Revision: "2", Path: []string{"properties", "currency"}},
					}}
					if unresolved {
						bindings.Currencies = bindings.Currencies[:1]
					}
					children := make([]behavioral.Rule, 2)
					for i := range children {
						children[i].Count = &behavioral.Count{Scope: behavioral.Scope{Event: "Purchase",
							Grain: "registered-message", Start: at.Add(-time.Hour), End: at},
							Currency: &behavioral.NativeCurrency{Definition: "NativeCurrency",
								Revision: fmt.Sprint(i + 1), Value: "EUR"}}
					}
					if reversed {
						children[0], children[1] = children[1], children[0]
					}
					root := behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{
						{Operator: operator, Children: children}}}
					filter, err := behavioral.Prepare(&root, bindings, at, source)
					if err != nil {
						if filter.Root != nil {
							t.Fatalf("expected no partial prepared filter, got %+v", filter)
						}
						if unresolved {
							if err.Error() != "unresolved NativeCurrency meaning revision" {
								t.Fatalf("expected unresolved NativeCurrency meaning revision, got %v", err)
							}
							return
						}
						if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
							t.Fatalf("expected unavailable evidence capability, got %v", err)
						}
						return
					}
					t.Fatal("expected binding rejection, got success")
				})
			}
		}
	}
}

// TestBehavioralPreparedBudget rejects physical limits before backend
// execution.
func TestBehavioralPreparedBudget(t *testing.T) {

	dw := warehouses.Registered("PostgreSQL").New(newTestSettingsLoader(nil), nil)
	t.Cleanup(func() {
		err := dw.Close()
		if err != nil {
			t.Errorf("expected warehouse cleanup, got %v", err)
		}
	})
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	properties := make([]types.Property, 170)
	for i := range properties {
		properties[i] = types.Property{Name: fmt.Sprintf("field%d", i), Type: types.String(), ReadOptional: true}
	}
	schema := types.Object([]types.Property{{Name: "wide", Type: types.Object(properties), ReadOptional: true}})
	store.columnByProperty.user = profileColumnByProperty(schema)
	root := behavioral.Rule{Operator: state.OpAnd, Children: make([]behavioral.Rule, 99)}
	for i := range root.Children {
		root.Children[i] = behavioral.Rule{Record: &state.WhereCondition{
			Property: []string{"wide"}, Operator: state.OpExists}}
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	page := behavioral.Page{Limit: 1}
	err := behavioral.Validate(&root, at, page)
	if err != nil {
		t.Fatalf("expected valid semantic tree, got %v", err)
	}

	// The unconfigured backend requires no connection for this rejection.
	result, err := store.QueryProfiles(t.Context(), &root, behavioral.Bindings{}, at, page)
	if err != nil {
		if _, unavailable := errors.AsType[*WarehouseError](err); unavailable {
			t.Fatalf("expected filter admission error, got warehouse failure %v", err)
		}
		if !strings.Contains(err.Error(), "exceeds structural budget") {
			t.Fatalf("expected physical budget error, got %v", err)
		}
		if result != nil {
			t.Fatalf("expected no result on admission failure, got %+v", result)
		}
		return
	}
	t.Fatal("expected physical budget rejection, got success")

}

// TestBehavioralRecordBodyBinding keeps each Count or E base Record body
// atomic.
func TestBehavioralRecordBodyBinding(t *testing.T) {
	where := &state.Where{Operator: state.OpOr, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"properties", "a.b"}, Operator: state.OpIsNot,
			Values: []any{json.Value(`100`)}},
		&state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"context", "page", "path"}, Operator: state.OpDoesNotContain,
				Values: []any{"/private"}},
		}},
	}}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, selected := range []bool{false, true} {
		source := profileFilterSourceBinding(nil)
		bind := source.Record
		calls := 0
		source.Record = func(actual *state.Where) (warehouses.Expr, error) {
			if calls == 0 && !reflect.DeepEqual(actual, where) {
				t.Fatalf("expected original whole Record group, got %+v", actual)
			}
			calls++
			return bind(actual)
		}
		scope := behavioral.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-time.Hour),
			End: at}
		root := &behavioral.Rule{Count: &behavioral.Count{Scope: scope, Where: recordCountWhere(t, where)}}
		expectedCalls := 1 // The whole where; source identity has its own binding.
		if selected {
			root = &behavioral.Rule{Experience: &behavioral.Experience{Scope: scope, Where: where,
				Verify: &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
					Operator: state.OpStartsWith, Values: []any{"WEB-"}}}}}
			expectedCalls++
		}
		_, err := behavioral.Prepare(root, behavioral.Bindings{}, at, source)
		if err != nil {
			t.Fatalf("expected prepared selection=%v, got %v", selected, err)
		}
		if calls != expectedCalls {
			t.Fatalf("expected %d atomic Record binds, got %d", expectedCalls, calls)
		}
	}
}

func recordCountWhere(t *testing.T, where *state.Where) *behavioral.OccurrencePredicate {
	t.Helper()
	predicate, err := behavioral.RecordPredicate(where)
	if err != nil {
		t.Fatalf("expected prepared Record predicate, got %v", err)
	}
	return predicate
}
