// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
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
