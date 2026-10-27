// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedCompileDialects checks binding and both physical adapters.
func TestUnifiedCompileDialects(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	rule := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase'",
		Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"},
		Test:     unified.CountTest{Operator: unified.Equal}}}
	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{Definition: "NativeCurrency",
		Revision: "1", Path: []string{"properties", "currency"}, Evidence: unified.IndependentOccurrenceValues}}}
	bind := func(where *state.Where) (warehouses.Expr, error) { return convertWhere(where, eventColumnByProperty) }
	for _, platform := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(platform, func(t *testing.T) {
			dw := warehouses.Registered(platform).New(newTestSettingsLoader(json.Value(`{}`)), nil)
			t.Cleanup(func() {
				err := dw.Close()
				if err != nil {
					t.Errorf("expected warehouse close, got %v", err)
				}
			})
			dialect := dw.(warehouses.CompiledQueryWarehouse).CompiledQueryDialect()
			query, err := unified.Compile(&rule, bindings, at, unified.Page{Limit: 10}, dialect, bind)
			if err != nil {
				t.Fatalf("expected shared compiler result, got %v", err)
			}
			values, err := query.Values()
			if err != nil {
				t.Fatalf("expected typed arguments, got %v", err)
			}
			if len(values) != 5 || values[0] != "Purchase'" || values[1] != "EUR" ||
				values[4] != "2026-02-01 00:00:00.123456" || strings.Contains(query.SQL, "Purchase'") {
				t.Fatalf("expected bound name/currency and one exact T, got %v", values)
			}
			if !strings.Contains(query.SQL, dialect.QuoteIdentifier("properties")) || len(query.Columns) != 6 {
				t.Fatalf("expected dialect quoting and normalized schema, got %s", query.SQL)
			}
			if platform == "Snowflake" && (strings.Count(query.SQL, "?") != len(values) || dialect.CountVerified) {
				t.Fatalf("expected ordered Snowflake binds and inactive gate, got %s", query.SQL)
			}
			_, err = unified.Compile(&rule, unified.Bindings{}, at, unified.Page{Limit: 10}, dialect, bind)
			if err == nil {
				t.Fatal("expected unresolved meaning revision error, got success")
			}
			uncertified := bindings
			uncertified.Currencies = append([]unified.CurrencyBinding(nil), bindings.Currencies...)
			uncertified.Currencies[0].Evidence = 0
			_, err = unified.Compile(&rule, uncertified, at, unified.Page{Limit: 10}, dialect, bind)
			if err == nil {
				t.Fatal("expected unsupported independence contract, got success")
			}
		})
	}
}
