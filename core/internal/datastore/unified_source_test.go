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
	"github.com/krenalis/krenalis/tools/json"
)

// TestUnifiedSourcePostgreSQL checks the determined-identity source contract on
// a fixture without a PK. This models conflicts that a physical PK can hide.
func TestUnifiedSourcePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles VALUES ('00000000-0000-0000-0000-000000000001'),('00000000-0000-0000-0000-000000000002');
CREATE TABLE krenalis_events (kpid uuid,message_id text,event text,timestamp timestamp(3) NOT NULL,
 properties jsonb,type text NOT NULL DEFAULT 'track')`)
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
			explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, bindings, at, unifiedTestID(1))
			if err != nil {
				if !test.conflict || explained != nil {
					t.Fatalf("expected explanation conflict=%v without partial data, got %+v, %v",
						test.conflict, explained, err)
				}
			}
			if err == nil && (test.conflict || explained == nil || len(explained.Nodes) != 1 ||
				explained.Nodes[0].Truth != test.want) {
				t.Fatalf("expected explanation conflict=%v and truth %v, got %+v", test.conflict, test.want, explained)
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

	t.Run("resolved currency and independent where", func(t *testing.T) {
		currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
		if err != nil {
			t.Fatalf("expected resolved currency, got %v", err)
		}
		count := *rule.Count
		count.Scope.Event, count.Currency = "Order Completed", currency
		count.Where = recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"properties", "coupon"}, Operator: state.OpIs,
				Values: []any{json.Value(`"PROMO"`)}},
		}})
		bound := unified.Rule{Count: &count}
		for _, test := range []struct {
			name                    string
			rows                    string
			conflict, otherConflict bool
			truth                   unified.Truth
		}{
			{"same copies", `('1','same','{"currency":"EUR","coupon":"PROMO"}'),('1','same','{"currency":"EUR","coupon":"PROMO"}')`, false, false, unified.Match},
			{"distinct identities", `('1','a','{"currency":"EUR","coupon":"PROMO"}'),('1','b','{"currency":"EUR","coupon":"PROMO"}')`, false, false, unified.NoMatch},
			{"different where", `('1','same','{"currency":"EUR","coupon":"PROMO"}'),('1','same','{"currency":"EUR"}')`, true, false, 0},
			{"different currency", `('1','same','{"currency":"EUR","coupon":"PROMO"}'),('1','same','{"currency":null,"coupon":"PROMO"}')`, true, false, 0},
			{"false where unknown currency", `('1','a','{"coupon":"OTHER"}')`, false, false, unified.NoMatch},
			{"unknown where EUR", `('1','a','{"currency":"EUR"}')`, false, false, unified.Unknown},
			{"PROMO unknown currency", `('1','a','{"coupon":"PROMO"}')`, false, false, unified.Unknown},
			{"both unknown", `('1','a','{}')`, false, false, unified.Unknown},
			{"fault in other profile", `('1','target','{"currency":"EUR","coupon":"PROMO"}'),('2','other','{"currency":"EUR","coupon":"PROMO"}'),('2','other','{"coupon":"PROMO"}')`, false, true, unified.Match},
		} {
			t.Run(test.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events (kpid,message_id,event,timestamp,properties)
SELECT ('00000000-0000-0000-0000-'||lpad(profile,12,'0'))::uuid,id,'Order Completed','2026-01-15',properties::jsonb
FROM (VALUES `+test.rows+`) input(profile,id,properties)`)
				explained, err := store.ExplainUnifiedProfile(t.Context(), &bound, unified.Bindings{}, at, unifiedTestID(1))
				if err != nil {
					if !test.conflict || explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected target source fault=%v, got %+v, %v", test.conflict, explained, err)
					}
				}
				if err == nil && (test.conflict || explained == nil || len(explained.Nodes) != 1 ||
					explained.Nodes[0].Truth != test.truth) {
					t.Fatalf("expected target fault=%v/truth=%v, got %+v", test.conflict, test.truth, explained)
				}
				result, err := store.UnifiedProfiles(t.Context(), &bound, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					if !(test.conflict || test.otherConflict) || result != nil ||
						!strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected population source fault without partial result, got %+v, %v", result, err)
					}
					return
				}
				if test.conflict || test.otherConflict {
					t.Fatal("expected population source fault, got success")
				}
				assertUnifiedTruths(t, result, []unified.Truth{test.truth, unified.NoMatch})
			})
		}
	})

	t.Run("unbounded history identity contract", func(t *testing.T) {

		history, err := unified.ResolveHistory("registered-queryable-events", "1")
		if err != nil {
			t.Fatalf("expected history reference, got %v", err)
		}
		count := *rule.Count
		count.Scope = unified.Scope{Event: "Purchase", Grain: "registered-message", History: history}
		historyRule := unified.Rule{Count: &count}

		for _, test := range []struct {
			name     string
			rows     []event
			conflict bool
			want     unified.Truth
		}{
			{"copies across centuries", []event{row("1", "same", "0001-01-01", `{"currency":"EUR"}`), row("1", "same", "2026-01-15", `{"currency":"EUR"}`)}, false, unified.Match},
			{"distinct old identities", []event{row("1", "a", "0001-01-01", `{"currency":"EUR"}`), row("1", "b", "1900-01-01", `{"currency":"EUR"}`)}, false, unified.NoMatch},
			{"old qualification conflict", []event{row("1", "same", "1800-01-01", `{"currency":"EUR"}`), row("1", "same", "2026-01-15", `{}`)}, true, 0},
			{"old attribution conflict", []event{row("1", "same", "1800-01-01", `{"currency":"EUR"}`), row("2", "same", "1900-01-01", `{"currency":"EUR"}`)}, true, 0},
			{"crosses exclusive T", []event{row("1", "same", "1800-01-01", `{"currency":"EUR"}`), row("1", "same", "2026-02-01", `{"currency":"EUR"}`)}, true, 0},
			{"future conflicts irrelevant", []event{row("1", "same", "2026-02-01", `{"currency":"EUR"}`), row("2", "same", "2026-02-02", `{}`)}, false, unified.NoMatch},
			{"old missing identity", []event{row("1", "", "0001-01-01", `{"currency":"EUR"}`)}, true, 0},
		} {
			t.Run(test.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				for _, r := range test.rows {
					id := "NULL"
					if r.id != "" {
						id = "'" + r.id + "'"
					}
					execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO krenalis_events VALUES ('%s',%s,'Purchase','%s','%s')",
						unifiedTestID(int(r.profile[0]-'0')), id, r.date, r.properties))
				}
				result, err := store.UnifiedProfiles(t.Context(), &historyRule, bindings, at, unified.Page{Limit: 10})
				if err != nil {
					if !test.conflict || result != nil {
						t.Fatalf("expected history conflict=%v without partial result, got %+v, %v", test.conflict, result, err)
					}
					return
				}
				if test.conflict {
					t.Fatal("expected source contract error, got success")
				}
				assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
			})
		}

	})

	// A fault confined to another profile must not poison the target explanation.
	for _, test := range []struct{ name, rows string }{
		{"unrelated missing identity", `('00000000-0000-0000-0000-000000000002',NULL,'Purchase','2026-01-15','{"currency":"EUR"}')`},
		{"unrelated attribution conflict", `('00000000-0000-0000-0000-000000000002','other','Purchase','2026-01-15','{"currency":"EUR"}'),(NULL,'other','Purchase','2026-01-15','{"currency":"EUR"}')`},
	} {
		t.Run(test.name, func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			execUnifiedFixture(t, dw, `INSERT INTO krenalis_events VALUES
('00000000-0000-0000-0000-000000000001','target','Purchase','2026-01-15','{"currency":"EUR"}'),`+test.rows)
			explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, bindings, at, unifiedTestID(1))
			if err != nil {
				t.Fatalf("expected unrelated fault to leave target explanation complete, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != unified.Match ||
				explained.Nodes[0].Count == nil || explained.Nodes[0].Count.Certain != 1 ||
				explained.Nodes[0].Count.Possible != 1 {
				t.Fatalf("expected target MATCH with exact Count [1,1], got %+v", explained)
			}
			explained, err = store.ExplainUnifiedProfile(t.Context(), &rule, bindings, at, unifiedTestID(2))
			if err != nil {
				if explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected affected profile to fail without explanation, got %+v, %v", explained, err)
				}
			}
			if err == nil {
				t.Fatal("expected affected profile source-contract failure, got success")
			}
			result, err := store.UnifiedProfiles(t.Context(), &rule, bindings, at, unified.Page{Limit: 10})
			if err != nil {
				if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected population source-contract failure without partial result, got %+v, %v", result, err)
				}
				return
			}
			t.Fatal("expected population source-contract failure, got success")
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
