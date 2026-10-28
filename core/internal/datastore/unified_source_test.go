// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
)

// TestUnifiedSourcePostgreSQL checks the determined-identity source contract on
// a fixture without a PK. This models conflicts that a physical PK can hide.
func TestUnifiedSourcePostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles VALUES ('00000000-0000-0000-0000-000000000001'),('00000000-0000-0000-0000-000000000002');
CREATE TABLE krenalis_events (kpid uuid,message_id text,event text,timestamp timestamp(3) NOT NULL,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	rule := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase",
		Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
		Test:     unified.CountTest{Operator: unified.Equal, Value: 1},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"}}}
	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{Definition: "NativeCurrency",
		Revision: "1", Path: []string{"properties", "currency"}, Evidence: unified.IndependentOccurrenceValues}}}
	type event struct{ profile, id, date, properties string }
	row := func(profile, id, date, properties string) event { return event{profile, id, date, properties} }
	for _, test := range []struct {
		name     string
		rows     []event
		conflict bool
		want     unified.Truth
	}{
		{"same copies", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("1", "same",
			"2026-01-15", `{"currency":"EUR"}`)}, false, unified.Match},
		{"distinct identities same payload", []event{row("1", "a", "2026-01-15", `{"currency":"EUR"}`), row("1",
			"b", "2026-01-15", `{"currency":"EUR"}`)}, false, unified.NoMatch},
		{"true unknown", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("1", "same",
			"2026-01-15", `{"currency":null}`)}, true, 0},
		{"true false", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("1", "same", "2026-01-15",
			`{"currency":"USD"}`)}, true, 0},
		{"unknown false", []event{row("1", "same", "2026-01-15", `{}`), row("1", "same", "2026-01-15",
			`{"currency":"USD"}`)}, true, 0},
		{"unknown copies", []event{row("1", "same", "2026-01-15", `{}`), row("1", "same", "2026-01-15",
			`{"currency":null}`)}, false, unified.Unknown},
		{"different profiles", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("2", "same",
			"2026-01-15", `{"currency":"EUR"}`)}, true, 0},
		{"unassigned and assigned", []event{row("", "same", "2026-01-15", `{"currency":"EUR"}`), row("1", "same",
			"2026-01-15", `{"currency":"EUR"}`)}, true, 0},
		{"unassigned only", []event{row("", "same", "2026-01-15", `{"currency":"EUR"}`), row("", "same",
			"2026-01-15", `{}`)}, false, unified.NoMatch},
		{"before and inside", []event{row("1", "same", "2025-12-31", `{"currency":"EUR"}`), row("1", "same",
			"2026-01-15", `{"currency":"EUR"}`)}, true, 0},
		{"inside and end", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("1", "same",
			"2026-02-01", `{"currency":"EUR"}`)}, true, 0},
		{"outside unassigned and inside assigned", []event{row("", "same", "2025-12-31", `{}`), row("1", "same",
			"2026-01-15", `{"currency":"EUR"}`)}, true, 0},
		{"inside unassigned and outside assigned", []event{row("", "same", "2026-01-15", `{}`), row("1", "same",
			"2026-02-01", `{"currency":"EUR"}`)}, true, 0},
		{"different times same membership", []event{row("1", "same", "2026-01-15", `{"currency":"EUR"}`), row("1",
			"same", "2026-01-16", `{"currency":"EUR","irrelevant":100}`)}, false, unified.Match},
		{"all outside different profiles", []event{row("1", "same", "2025-12-31", `{"currency":"EUR"}`), row("2",
			"same", "2026-02-01", `{}`)}, false, unified.NoMatch},
		{"verbatim lowercase", []event{row("1", "same", "2026-01-15", `{"currency":"eur"}`)}, false, unified.NoMatch},
		{"verbatim arbitrary string", []event{row("1", "same", "2026-01-15", `{"currency":"not-a-code"}`)}, false,
			unified.NoMatch},
		{"business Boolean", []event{row("1", "same", "2026-01-15", `{"currency":true}`)}, false, unified.Unknown},
		{"business Number", []event{row("1", "same", "2026-01-15", `{"currency":100}`)}, false, unified.Unknown},
		{"business Object", []event{row("1", "same", "2026-01-15", `{"currency":{}}`)}, false, unified.Unknown},
		{"business Array", []event{row("1", "same", "2026-01-15", `{"currency":[]}`)}, false, unified.Unknown},
		{"missing identity", []event{row("1", "", "2026-01-15", `{"currency":"EUR"}`)}, true, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			for _, r := range test.rows {
				profile := "NULL"
				if r.profile != "" {
					profile = "'" + unifiedTestID(int(r.profile[0]-'0')) + "'"
				}
				id := "NULL"
				if r.id != "" {
					id = "'" + r.id + "'"
				}
				execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO krenalis_events VALUES (%s,%s,'Purchase','%s','%s')",
					profile, id, r.date, strings.ReplaceAll(r.properties, "'", "''")))
			}
			result, err := store.UnifiedProfiles(t.Context(), &rule, bindings, at, unified.Page{Limit: 10})
			if err != nil {
				if !test.conflict {
					t.Fatalf("expected valid determined source, got %v", err)
				}
				if result != nil {
					t.Fatalf("expected no partial result, got %+v", result)
				}
				return
			}
			if test.conflict {
				t.Fatal("expected source contract failure, got success")
			}
			assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
		})
	}
	// A failed operation must leave the connection usable for a valid request.
	execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
	result, err := store.UnifiedProfiles(t.Context(), &rule, bindings, at, unified.Page{Limit: 10})
	if err != nil {
		t.Fatalf("expected reusable connection after conflicts, got %v", err)
	}
	assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch})
}
