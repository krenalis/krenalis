// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"slices"
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

// TestUnifiedProfileRecordPostgreSQL verifies profile Record and Count on a
// coherent fixture, including the complete three-by-three Kleene table.
func TestUnifiedProfileRecordPostgreSQL(t *testing.T) {

	const (
		match   = unified.Match
		noMatch = unified.NoMatch
		unknown = unified.Unknown
	)
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	schema := types.Object([]types.Property{
		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "profile_id", Type: types.String(), ReadOptional: true},
		{Name: "event", Type: types.String(), ReadOptional: true},
		{Name: "document", Type: types.JSON(), ReadOptional: true},
		{Name: "amount", Type: types.Decimal(38, 10), ReadOptional: true},
		{Name: "settings", ReadOptional: true, Type: types.Object([]types.Property{
			{Name: "value", Type: types.String(), ReadOptional: true},
			{Name: "json", Type: types.JSON(), ReadOptional: true},
		})},
	})
	store.columnByProperty.user = profileColumnByProperty(schema)
	store.columnByProperty.user["_kpid"] = warehouses.Column{Name: "_kpid", Type: types.UUID()}

	// The source has no PK so the failure path can exercise conflicting copies.
	// All fixtures are published before their query; no asynchronous writer runs.
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY,country text,
 profile_id text,event text,document jsonb,amount numeric(38,10),settings_value text,settings_json jsonb);
CREATE TABLE krenalis_events (kpid uuid,message_id text,event text,timestamp timestamp(3) NOT NULL,
 properties jsonb,type text NOT NULL DEFAULT 'track');
INSERT INTO profiles (_kpid,country,profile_id,event,document,amount,settings_value,settings_json)
SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,
 CASE WHEN n<=3 THEN 'IT' WHEN n<=6 THEN 'FR' END,'customer-'||n,'Refund',
 CASE n WHEN 1 THEN '{}' WHEN 2 THEN '{"currency":null}' WHEN 3 THEN '{"currency":"EUR"}'
 WHEN 4 THEN '{"currency":100}' WHEN 5 THEN '{"currency":true}'
 WHEN 7 THEN '{"literal.key":{"quote''key":9007199254740993}}'
 WHEN 8 THEN '{"currency":"USD"}' WHEN 9 THEN '{"currency":"EUR"}' END::jsonb,
 CASE n WHEN 1 THEN 9007199254740993.0000000001 WHEN 2 THEN 9007199254740993.0000000002
 WHEN 3 THEN NULL ELSE 0 END,CASE WHEN n=1 THEN '' END,
 CASE WHEN n=2 THEN 'null' WHEN n=3 THEN '{}' END::jsonb
 FROM generate_series(1,9) n;
INSERT INTO krenalis_events
 SELECT _kpid,'event-'||right(_kpid::text,1),'Purchase','2026-01-15',
 CASE WHEN right(_kpid::text,1)::int%3=2 THEN '{"currency":"EUR"}' ELSE '{"currency":null}' END::jsonb
 FROM profiles WHERE right(_kpid::text,1)::int%3<>1;`)

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{Definition: "NativeCurrency",
		Revision: "1", Path: []string{"properties", "currency"}, Evidence: unified.IndependentOccurrenceValues}}}
	count := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Test: unified.CountTest{Operator: unified.Equal},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "1", Value: "EUR"}}}
	record := unified.Rule{Record: &state.WhereCondition{Property: []string{"country"}, Operator: state.OpIs,
		Values: []any{"IT"}}}
	page := unified.Page{Limit: 100}
	query := func(t *testing.T, rule *unified.Rule, page unified.Page) *unified.Result {
		t.Helper()
		result, err := store.UnifiedProfiles(t.Context(), rule, bindings, at, page)
		if err != nil {
			t.Fatalf("expected complete profile query, got %v", err)
		}
		return result
	}

	for _, test := range []struct {
		name     string
		operator state.WhereLogical
		want     []unified.Truth
	}{
		{"and", state.OpAnd, []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown,
			unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.Unknown}},
		{"or", state.OpOr, []unified.Truth{unified.Match, unified.Match, unified.Match,
			unified.Match, unified.NoMatch, unified.Unknown, unified.Match, unified.Unknown, unified.Unknown}},
	} {
		t.Run(test.name, func(t *testing.T) {

			root := unified.Rule{Operator: test.operator, Children: []unified.Rule{record, count}}
			all := query(t, &root, page)
			assertUnifiedTruths(t, all, test.want)
			for i, expected := range test.want {
				explained, err := store.ExplainUnifiedProfile(t.Context(), &root, bindings, at, unifiedTestID(i+1))
				if err != nil {
					t.Fatalf("expected explained Record/Count composition, got %v", err)
				}
				if explained == nil || len(explained.Nodes) != 3 || explained.Nodes[0].Truth != expected ||
					explained.Nodes[1].Count != nil || explained.Nodes[2].Count == nil || !explained.At.Equal(at) {
					t.Fatalf("expected three explained nodes with root %v, got %+v", expected, explained)
				}
				lo, hi := int64(0), int64(0)
				if i%3 == 1 {
					lo, hi = 1, 1
				}
				if i%3 == 2 {
					hi = 1
				}
				if explained.Nodes[2].Count.Certain != lo || explained.Nodes[2].Count.Possible != hi {
					t.Fatalf("expected NativeCurrency Count bounds [%d,%d], got %+v", lo, hi, explained.Nodes[2].Count)
				}
			}

			matched := query(t, &root, unified.Page{Limit: 100, MatchingOnly: true})
			var ids []string
			for i, truth := range test.want {
				if truth == unified.Match {
					ids = append(ids, unifiedTestID(i+1))
				}
			}
			if len(matched.Profiles) != len(ids) || matched.Match != all.Match ||
				matched.NoMatch != all.NoMatch || matched.Unknown != all.Unknown {
				t.Fatalf("expected membership and unchanged totals, got %+v", matched)
			}
			for i, outcome := range matched.Profiles {
				if outcome.ID != ids[i] || outcome.Truth != unified.Match {
					t.Fatalf("expected matching profile %s, got %+v", ids[i], outcome)
				}
			}

			empty := query(t, &root, unified.Page{First: 100, Limit: 1, MatchingOnly: true})
			if len(empty.Profiles) != 0 || empty.Match != all.Match ||
				empty.NoMatch != all.NoMatch || empty.Unknown != all.Unknown {
				t.Fatalf("expected empty page with complete totals, got %+v", empty)
			}

		})
	}

	t.Run("record binding reads profile not event", func(t *testing.T) {
		root := unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{count,
			{Record: &state.WhereCondition{Property: []string{"event"}, Operator: state.OpIs, Values: []any{"Refund"}}}}}
		assertUnifiedTruths(t, query(t, &root, page), []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown,
			unified.Match, unified.NoMatch, unified.Unknown, unified.Match, unified.NoMatch, unified.Unknown})
	})

	for _, test := range []struct {
		name      string
		condition state.WhereCondition
		want      []unified.Truth
	}{
		{"ordinary negative", state.WhereCondition{Property: []string{"country"}, Operator: state.OpIsNot, Values: []any{"IT"}}, []unified.Truth{noMatch, noMatch, noMatch, match, match, match, unknown, unknown, unknown}},
		{"ordinary not empty", state.WhereCondition{Property: []string{"country"}, Operator: state.OpIsNotEmpty}, []unified.Truth{match, match, match, match, match, match, noMatch, noMatch, noMatch}},
		{"alias collision", state.WhereCondition{Property: []string{"profile_id"}, Operator: state.OpIs, Values: []any{"customer-1"}}, []unified.Truth{match, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch}},
		{"JSON equality", state.WhereCondition{Property: []string{"document", "currency"}, Operator: state.OpIs, Values: []any{json.Value(`"EUR"`)}}, []unified.Truth{unknown, noMatch, match, noMatch, noMatch, unknown, unknown, noMatch, match}},
		{"JSON negative", state.WhereCondition{Property: []string{"document", "currency"}, Operator: state.OpIsNot, Values: []any{json.Value(`"EUR"`)}}, []unified.Truth{unknown, match, noMatch, match, match, unknown, unknown, match, noMatch}},
		{"JSON existence", state.WhereCondition{Property: []string{"document", "currency"}, Operator: state.OpExists}, []unified.Truth{noMatch, match, match, match, match, noMatch, noMatch, match, match}},
		{"JSON null", state.WhereCondition{Property: []string{"document", "currency"}, Operator: state.OpIsNull}, []unified.Truth{noMatch, match, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch}},
		{"JSON precise number and literal keys", state.WhereCondition{Property: []string{"document", "literal.key", "quote'key"}, Operator: state.OpIs, Values: []any{json.Value(`9007199254740993`)}}, []unified.Truth{unknown, unknown, unknown, unknown, unknown, unknown, match, unknown, unknown}},
		{"exact Decimal", state.WhereCondition{Property: []string{"amount"}, Operator: state.OpIs, Values: []any{decimal.MustParse("9007199254740993.0000000001")}}, []unified.Truth{match, noMatch, unknown, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch}},
		{"flattened object exists", state.WhereCondition{Property: []string{"settings"}, Operator: state.OpExists}, []unified.Truth{match, match, match, noMatch, noMatch, noMatch, noMatch, noMatch, noMatch}},
		{"flattened object absent", state.WhereCondition{Property: []string{"settings"}, Operator: state.OpDoesNotExist}, []unified.Truth{noMatch, noMatch, noMatch, match, match, match, match, match, match}},
	} {
		t.Run(test.name, func(t *testing.T) {

			root := unified.Rule{Record: &test.condition}
			result := query(t, &root, page)
			assertUnifiedTruths(t, result, test.want)

			where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{&test.condition}}
			existing, _, err := store.Profiles(t.Context(), Query{Properties: []string{"_kpid"}, Where: where,
				OrderBy: "_kpid", Limit: 100})
			if err != nil {
				t.Fatalf("expected existing Record listing, got %v", err)
			}
			var expected, actual []string
			for _, row := range existing {
				expected = append(expected, row["_kpid"].(string))
			}
			for _, outcome := range result.Profiles {
				if outcome.Truth == unified.Match {
					actual = append(actual, outcome.ID)
				}
			}
			if !slices.Equal(actual, expected) {
				t.Fatalf("expected existing Record membership %v, got %v", expected, actual)
			}

		})
	}

	t.Run("deepest object leaf", func(t *testing.T) {
		root := unified.Rule{Record: &state.WhereCondition{Property: []string{"settings"}, Operator: state.OpExists}}
		for range 4 {
			root = unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{root}}
		}
		assertUnifiedTruths(t, query(t, &root, page), []unified.Truth{match, match, match,
			noMatch, noMatch, noMatch, noMatch, noMatch, noMatch})
	})

	t.Run("unmapped property fails before execution", func(t *testing.T) {
		root := unified.Rule{Record: &state.WhereCondition{Property: []string{"unmapped"}, Operator: state.OpIsTrue}}
		result, err := store.UnifiedProfiles(t.Context(), &root, bindings, at, page)
		if err != nil {
			if result != nil {
				t.Fatalf("expected no result on binding failure, got %+v", result)
			}
			return
		}
		t.Fatal("expected unmapped property failure, got success")
	})

	t.Run("false Record does not hide technical failure", func(t *testing.T) {
		execUnifiedFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','conflict','Purchase','2026-01-15','{"currency":"EUR"}'),
('%s','conflict','Purchase','2026-01-15','{"currency":null}')`, unifiedTestID(1), unifiedTestID(1)))
		root := unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{count,
			{Record: &state.WhereCondition{Property: []string{"profile_id"}, Operator: state.OpDoesNotExist}}}}
		result, err := store.UnifiedProfiles(t.Context(), &root, bindings, at, page)
		if err != nil {
			if _, ok := errors.AsType[*UnavailableError](err); !ok {
				t.Fatalf("expected occurrence source UnavailableError, got %T (%v)", err, err)
			}
			if !strings.Contains(err.Error(), "occurrence source violates the prepared occurrence contract") {
				t.Fatalf("expected occurrence source contract failure, got %v", err)
			}
			if result != nil {
				t.Fatalf("expected no partial result on source conflict, got %+v", result)
			}
			return
		}
		t.Fatal("expected source conflict failure, got success")
	})

	// Pure Record must not depend on an event table or an occurrence binding.
	execUnifiedFixture(t, dw, "DROP TABLE krenalis_events")
	assertUnifiedTruths(t, query(t, &record, page), []unified.Truth{match, match, match,
		noMatch, noMatch, noMatch, unknown, unknown, unknown})
	execUnifiedFixture(t, dw, "TRUNCATE profiles")
	result := query(t, &record, page)
	if len(result.Profiles) != 0 || result.Match != 0 || result.NoMatch != 0 || result.Unknown != 0 {
		t.Fatalf("expected empty population with zero totals, got %+v", result)
	}

}
