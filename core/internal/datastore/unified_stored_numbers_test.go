// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedStoredNumbersPostgreSQL checks the monetary path on real writes.
func TestUnifiedStoredNumbersPostgreSQL(t *testing.T) {
	testUnifiedStoredNumbers(t, "PostgreSQL")
}

// TestUnifiedStoredNumbersSnowflake needs a verified or test-overlay build;
// it does not provide a runtime bypass of the production verification gate.
func TestUnifiedStoredNumbersSnowflake(t *testing.T) {
	dw := warehouses.Registered("Snowflake").New(newTestSettingsLoader(json.Value(`{}`)), nil)
	if !dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified() {
		t.Fatal("expected verified Snowflake provider, got closed gate")
	}
	testUnifiedStoredNumbers(t, "Snowflake")
}

func testUnifiedStoredNumbers(t *testing.T, name string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, name)
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	// Fixtures are coherent and fully attributed before querying. This tests
	// their numeric transport, not publication or asynchronous attribution.
	switch name {
	case "PostgreSQL":
		execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,
 timestamp timestamp(6),properties jsonb)`)
	case "Snowflake":
		execUnifiedFixture(t, dw, `CREATE TABLE "PROFILES" ("_KPID" VARCHAR(36));
CREATE TABLE "KRENALIS_EVENTS" ("KPID" VARCHAR(36),"MESSAGE_ID" VARCHAR,"TYPE" VARCHAR,"EVENT" VARCHAR,
 "TIMESTAMP" TIMESTAMP_NTZ(9),"PROPERTIES" VARIANT)`)
	default:
		t.Fatalf("expected supported warehouse, got %s", name)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	var population []string
	for i := 1; i <= 6; i++ {
		population = append(population, "('"+unifiedTestID(i)+"')")
	}
	execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(population, ","))
	events := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String()},
		{Name: "timestamp", Type: types.DateTime()}, {Name: "properties", Type: types.JSON()},
	}}
	var data [][]any
	for i, fixture := range []struct {
		profile  int
		amount   string
		currency string
	}{
		{1, "100", "EUR"}, {1, "70", "EUR"},
		{2, "20", "EUR"}, {2, "200", ""},
		{4, "-30", "EUR"}, {4, "", "EUR"},
		{5, "1000", "USD"},
		{6, "99999999999999.9999", "EUR"}, {6, "-99999999999999.9999", "EUR"},
	} {
		properties := map[string]any{}
		if fixture.amount != "" {
			properties["total"] = decimal.MustParse(fixture.amount)
		}
		if fixture.currency != "" {
			properties["currency"] = fixture.currency
		}
		payload, err := json.Marshal(properties)
		if err != nil {
			t.Fatalf("expected exact decimal serialization, got %v", err)
		}
		data = append(data, []any{unifiedTestID(fixture.profile), fmt.Sprintf("m%d", i), "track", "Order Completed",
			at.AddDate(0, 0, -15).Add(time.Duration(i) * time.Hour), json.Value(payload)})
	}
	err := dw.Merge(t.Context(), events, data, nil)
	if err != nil {
		t.Fatalf("expected recorded event write, got %v", err)
	}
	// The duplicate is a technical copy; it must not add a third occurrence.
	switch name {
	case "PostgreSQL":
		execUnifiedFixture(t, dw, `INSERT INTO krenalis_events SELECT * FROM krenalis_events WHERE message_id='m0'`)
	case "Snowflake":
		execUnifiedFixture(t, dw, `INSERT INTO "KRENALIS_EVENTS" SELECT * FROM "KRENALIS_EVENTS" WHERE "MESSAGE_ID"='m0'`)
	default:
		t.Fatalf("expected supported warehouse, got %s", name)
	}

	scope := unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", scope.Event)
	if err != nil {
		t.Fatalf("expected native currency, got %v", err)
	}
	count := unified.Rule{Count: &unified.Count{Scope: scope, Currency: currency,
		Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 2}}}
	sum := unified.Rule{Sum: &unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR", Minimum: 2,
		Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("150")}}}
	average := unified.Rule{Average: &unified.Average{Sum: unified.Sum{Scope: scope, Field: field,
		NativeCurrency: "EUR", Minimum: 2,
		Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("85")}}}}
	money := unified.Rule{Experience: &unified.Experience{Scope: scope, Currency: currency,
		Verify: &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
			Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("50")}}}}}
	pair := unified.Rule{LastVersusPrevious: &unified.LastVersusPrevious{Scope: scope, Field: field, Currency: currency,
		Check: unified.LastPreviousCheck{Kind: unified.LastPreviousDifference,
			Test: unified.SumTest{Operator: unified.LessOrEqual, Value: decimal.MustParse("-30")}}}}
	mixed := unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{count, sum, average, money, pair}}
	for _, test := range []struct {
		name string
		rule unified.Rule
		want []unified.Truth
	}{
		{"count", count, []unified.Truth{unified.Match, unified.Unknown, unified.NoMatch, unified.Match, unified.NoMatch, unified.Match}},
		{"sum", sum, []unified.Truth{unified.Match, unified.Unknown, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
		{"average", average, []unified.Truth{unified.Match, unified.Unknown, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
		{"money", money, []unified.Truth{unified.Match, unified.Unknown, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
		{"difference", pair, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.Match}},
		{"mixed", mixed, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch}},
	} {
		t.Run(test.name, func(t *testing.T) {

			result, err := store.UnifiedProfiles(t.Context(), &test.rule, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected complete monetary evaluation, got %v", err)
			}
			assertUnifiedTruths(t, result, test.want)
			for i, want := range test.want {
				explained, err := store.ExplainUnifiedProfile(t.Context(), &test.rule, unified.Bindings{}, at, unifiedTestID(i+1))
				if err != nil {
					t.Fatalf("expected complete monetary explanation, got %v", err)
				}
				if explained == nil || len(explained.Nodes) == 0 || explained.Nodes[0].Truth != want {
					t.Fatalf("expected profile %d explanation outcome %v, got %+v", i+1, want, explained)
				}
			}

			empty, err := store.UnifiedProfiles(t.Context(), &test.rule, unified.Bindings{}, at,
				unified.Page{First: 100, Limit: 1, MatchingOnly: true})
			if err != nil {
				t.Fatalf("expected complete empty monetary page, got %v", err)
			}
			if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
				empty.Unknown != result.Unknown || !empty.At.Equal(at) {
				t.Fatalf("expected empty page with the same evaluation and totals %+v, got %+v", result, empty)
			}

		})
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result, err := store.UnifiedProfiles(ctx, &sum, unified.Bindings{}, at, unified.Page{Limit: 10})
	if err != nil {
		if result != nil {
			t.Fatalf("expected cancellation without partial result, got %+v", result)
		}
		return
	}
	t.Fatalf("expected cancellation error, got success %+v", result)

}
