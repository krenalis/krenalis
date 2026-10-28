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

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedPrepare checks resolved source metadata without a backend/dialect.
func TestUnifiedPrepare(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	rule := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase'",
		Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"},
		Test:     unified.CountTest{Operator: unified.Different, Value: 2}, Minimum: 1}}
	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{Definition: "NativeCurrency",
		Revision: "1", Path: []string{"properties", "a.b"}, Evidence: unified.IndependentOccurrenceValues}}}
	calls := 0
	source := unified.SourceBinding{
		Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events", Profile: eventColumnByProperty["kpid"],
			Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
		Properties: eventColumnByProperty["properties"],
		Record: func(where *state.Where) (warehouses.Expr, error) {
			calls++
			return convertWhere(where, eventColumnByProperty)
		},
	}
	filter, err := unified.Prepare(&rule, bindings, at, source)
	if err != nil {
		t.Fatalf("expected prepared filter, got %v", err)
	}
	count := filter.Root.(*warehouses.IndependentOccurrenceCount)
	if !reflect.DeepEqual(count.Source, source.Occurrences) || !count.Start.Equal(rule.Count.Scope.Start) ||
		!count.End.Equal(at) {
		t.Fatalf("expected resolved source and period, got %+v", count)
	}
	quals := count.Qualification.(*warehouses.QualificationGroup)
	selector := quals.Children[0].(*warehouses.RecordQualification).Expr.(*warehouses.MultiExpr).Operands[0].(*warehouses.BaseExpr)
	if selector.Values[0] != "Purchase'" || selector.Operator != warehouses.OpIs ||
		selector.Column.Name != "event" || calls != 1 {
		t.Fatalf("expected existing raw name binding, got %+v and %d calls", selector, calls)
	}
	currency := quals.Children[1].(*warehouses.KnownTextEqual)
	if currency.Value != "EUR" || !reflect.DeepEqual(currency.Keys, []string{"a.b"}) {
		t.Fatalf("expected exact bound currency path, got %+v", currency)
	}
	if len(count.Accept) != 2 || count.Accept[0].Lower != 1 || *count.Accept[0].Upper != 1 ||
		count.Accept[1].Lower != 3 || count.Accept[1].Upper != nil {
		t.Fatalf("expected [1,1] and [3,infinity], got %+v", count.Accept)
	}
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		dw := warehouses.Registered(platform).New(newTestSettingsLoader(nil), nil)
		backend, ok := dw.(warehouses.ProfileFilterWarehouse)
		if !ok || backend.ProfileFilterVerified() != (platform == "PostgreSQL") {
			t.Fatalf("expected mirrored interfaces and verification gates, got %s", platform)
		}
		err := dw.Close()
		if err != nil {
			t.Fatalf("expected warehouse cleanup, got %v", err)
		}
	}
	for _, change := range []func(*unified.Rule){
		func(r *unified.Rule) { r.Count.Shared = &unified.SharedBoolean{Fact: "g"} },
		func(r *unified.Rule) { r.Count.Scope.End = at.Add(time.Hour) },
		func(r *unified.Rule) {
			r.Count.Scope.FullHistory = true
			r.Count.Scope.Start = time.Time{}
			r.Count.Scope.End = time.Time{}
		},
	} {
		c := *rule.Count
		r := unified.Rule{Count: &c}
		change(&r)
		calls = 0
		_, err := unified.Prepare(&r, bindings, at, source)
		if err != nil {
			if _, ok := errors.AsType[*unified.UnsupportedError](err); !ok {
				t.Fatalf("expected unsupported capability, got %v", err)
			}
		}
		if err == nil || calls != 0 {
			t.Fatalf("expected rejection before binding, got %v and %d calls", err, calls)
		}
	}
	for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
		group, err := unified.Prepare(&unified.Rule{Operator: op, Children: []unified.Rule{rule}}, bindings, at, source)
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
	_, err = unified.Prepare(&rule, unified.Bindings{}, at, source)
	if err == nil {
		t.Fatal("expected unresolved meaning revision error, got success")
	}
	bindings.Currencies[0].Evidence = 0
	_, err = unified.Prepare(&rule, bindings, at, source)
	if err == nil {
		t.Fatal("expected unsupported evidence model, got success")
	}
}

// TestUnifiedPrepareElapsedPeriods resolves offsets before the warehouse
// boundary without mutating the scope or relying on a dialect or Unix nanos.
func TestUnifiedPrepareElapsedPeriods(t *testing.T) {

	rome, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		t.Fatalf("expected Rome calendar, got %v", err)
	}
	for _, test := range []struct {
		name       string
		at         time.Time
		period     unified.ElapsedPeriod
		start, end time.Time
	}{
		{"spring lookback", time.Date(2026, 3, 29, 12, 0, 0, 0, rome), unified.ElapsedPeriod{Older: 24 * time.Hour}, time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC), time.Date(2026, 3, 29, 10, 0, 0, 0, time.UTC)},
		{"spring ago band", time.Date(2026, 3, 29, 12, 0, 0, 0, rome), unified.ElapsedPeriod{Older: 48 * time.Hour, Newer: 24 * time.Hour}, time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC), time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)},
		{"autumn lookback", time.Date(2026, 10, 25, 12, 0, 0, 0, rome), unified.ElapsedPeriod{Older: 24 * time.Hour}, time.Date(2026, 10, 24, 11, 0, 0, 0, time.UTC), time.Date(2026, 10, 25, 11, 0, 0, 0, time.UTC)},
		{"microsecond lookback", time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC), unified.ElapsedPeriod{Older: time.Microsecond}, time.Date(2026, 2, 1, 0, 0, 0, 123455000, time.UTC), time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)},
		{"microsecond ago band", time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC), unified.ElapsedPeriod{Older: 3 * time.Microsecond, Newer: time.Microsecond}, time.Date(2026, 2, 1, 0, 0, 0, 123453000, time.UTC), time.Date(2026, 2, 1, 0, 0, 0, 123455000, time.UTC)},
		{"maximum microsecond duration", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), unified.ElapsedPeriod{Older: time.Duration(math.MaxInt64/1000) * time.Microsecond}, time.Date(1677, 9, 21, 0, 12, 43, 145225000, time.UTC), time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"far future", time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC), unified.ElapsedPeriod{Older: time.Microsecond}, time.Date(9999, 12, 31, 23, 59, 59, 999998000, time.UTC), time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC)},
		{"far past", time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC), unified.ElapsedPeriod{Older: time.Microsecond}, time.Date(1, 1, 1, 23, 59, 59, 999999000, time.UTC), time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {

			scope := unified.Scope{Event: "Purchase", Grain: "registered-message", Elapsed: &test.period}
			rule := unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{{Count: &unified.Count{Scope: scope}}}}
			source := unified.SourceBinding{
				Occurrences: warehouses.OccurrenceSource{Table: "krenalis_events",
					Profile:  eventColumnByProperty["kpid"],
					Identity: eventColumnByProperty["messageId"], Timestamp: eventColumnByProperty["timestamp"]},
				Record: func(where *state.Where) (warehouses.Expr, error) {
					return convertWhere(where, eventColumnByProperty)
				},
			}
			before := test.period
			filter, err := unified.Prepare(&rule, unified.Bindings{}, test.at, source)
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

// TestUnifiedPrepareProfileRecord binds without an occurrence source.
func TestUnifiedPrepareProfileRecord(t *testing.T) {

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	condition := &state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs, Values: []any{"Refund"}}
	rule := unified.Rule{Record: condition}
	source := unified.SourceBinding{
		Profiles: warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		ProfileRecord: func(where *state.Where) (warehouses.Expr, error) {
			return convertWhere(where, map[string]warehouses.Column{"event": {Name: "profile_event",
				Type: types.String(), Nullable: true}})
		},
	}
	filter, err := unified.Prepare(&rule, unified.Bindings{}, at, source)
	if err != nil {
		t.Fatalf("expected prepared profile Record, got %v", err)
	}
	expr := filter.Root.(*warehouses.RecordFilter).Expr.(*warehouses.MultiExpr).Operands[0].(*warehouses.BaseExpr)
	if expr.Column.Name != "profile_event" || expr.Operator != warehouses.OpIs || expr.Values[0] != "Refund" {
		t.Fatalf("expected population binding and exact operands, got %+v", expr)
	}

	for _, change := range []func(*unified.SourceBinding){
		func(s *unified.SourceBinding) { s.ProfileRecord = nil },
		func(s *unified.SourceBinding) { s.Profiles.Table = "" },
		func(s *unified.SourceBinding) {
			s.ProfileRecord = func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, nil) }
		},
	} {
		binding := source
		change(&binding)
		_, err := unified.Prepare(&rule, unified.Bindings{}, at, binding)
		if err == nil {
			t.Fatal("expected missing profile binding error, got success")
		}
	}

}

// TestUnifiedPreparedBudget rejects physical limits before backend execution.
func TestUnifiedPreparedBudget(t *testing.T) {

	dw := warehouses.Registered("PostgreSQL").New(newTestSettingsLoader(nil), nil)
	t.Cleanup(func() {
		err := dw.Close()
		if err != nil {
			t.Errorf("expected warehouse cleanup, got %v", err)
		}
	})
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	properties := make([]types.Property, 170)
	for i := range properties {
		properties[i] = types.Property{Name: fmt.Sprintf("field%d", i), Type: types.String(), ReadOptional: true}
	}
	schema := types.Object([]types.Property{{Name: "wide", Type: types.Object(properties), ReadOptional: true}})
	store.columnByProperty.user = profileColumnByProperty(schema)
	root := unified.Rule{Operator: state.OpAnd, Children: make([]unified.Rule, 99)}
	for i := range root.Children {
		root.Children[i] = unified.Rule{Record: &state.WhereCondition{
			Property: []string{"wide"}, Operator: state.OpExists}}
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	page := unified.Page{Limit: 1}
	err := unified.Validate(&root, at, page)
	if err != nil {
		t.Fatalf("expected valid semantic tree, got %v", err)
	}

	// The unconfigured backend requires no connection for this rejection.
	result, err := store.UnifiedProfiles(t.Context(), &root, unified.Bindings{}, at, page)
	if err != nil {
		if _, unavailable := errors.AsType[*UnavailableError](err); unavailable {
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
