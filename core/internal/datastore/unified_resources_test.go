// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedResourcesPostgreSQL checks budgets, errors and bounded load.
func TestUnifiedResourcesPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testUnifiedResources(t, "PostgreSQL")
}

// TestUnifiedResourcesSnowflake requires an isolated gate-overlay build.
func TestUnifiedResourcesSnowflake(t *testing.T) {
	if testing.Short() {
		t.Skip("requires explicitly configured Snowflake")
	}
	dw := warehouses.Registered("Snowflake").New(newTestSettingsLoader(json.Value(`{}`)), nil)
	verified := dw.(warehouses.ProfileFilterWarehouse).ProfileFilterVerified()
	err := dw.Close()
	if err != nil {
		t.Fatalf("expected unopened warehouse cleanup, got %v", err)
	}
	if !verified {
		t.Skip("Snowflake verification requires an isolated test-overlay build")
	}
	testUnifiedResources(t, "Snowflake")
}

func testUnifiedResources(t *testing.T, platform string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	textType := "text"
	switch platform {
	case "PostgreSQL":
		// A text key also models malformed rows from a warehouse-owned view.
		execUnifiedFixture(t, dw, `CREATE TABLE profile_data (_kpid text,payload text);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,
 name text,timestamp timestamp(6),properties jsonb)`)
	case "Snowflake":
		textType = "VARCHAR"
		execUnifiedFixture(t, dw, `CREATE TABLE "PROFILE_DATA" ("_KPID" VARCHAR(36),"PAYLOAD" VARCHAR);
CREATE TABLE "KRENALIS_EVENTS" ("KPID" VARCHAR(36),"MESSAGE_ID" VARCHAR,"TYPE" VARCHAR,"EVENT" VARCHAR,
 "NAME" VARCHAR,"TIMESTAMP" TIMESTAMP_NTZ(6),"PROPERTIES" VARIANT)`)
	default:
		t.Fatalf("expected supported warehouse, got %s", platform)
	}
	q := func(name string) string { return quoteUnifiedRecordIdentifier(platform, name) }
	view := "CREATE OR REPLACE VIEW " + q("profiles") + " AS SELECT " + q("_kpid") + "," + q("payload") +
		" FROM " + q("profile_data")
	execUnifiedFixture(t, dw, view)
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	store.columnByProperty.user = map[string]warehouses.Column{
		"_kpid": {Name: "_kpid", Type: types.UUID()}, "payload": {Name: "payload", Type: types.String()},
	}
	fixture := &unifiedRecordFixture{store: store, warehouse: dw,
		at: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
	scope := unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: fixture.at.Add(-24 * time.Hour), End: fixture.at}
	root := &unified.Rule{Count: &unified.Count{Scope: scope,
		Test: unified.CountTest{Operator: unified.GreaterOrEqual}}}

	for _, test := range []struct {
		name, unit string
		repeats    int
		profiles   int
		oversized  bool
	}{
		{"single oversized", "x", warehouses.MaxProfilePageBytes + 1, 1, true},
		{"cumulative oversized", "x", warehouses.MaxProfilePageBytes/2 + 1, 2, true},
		{"exact budget", "x", warehouses.MaxProfilePageBytes, 1, false},
		{"UTF8 oversized", "é", warehouses.MaxProfilePageBytes/2 + 1, 1, true},
		{"UTF8 exact budget", "é", warehouses.MaxProfilePageBytes / 2, 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE TABLE "+q("profile_data"))
			for profile := 1; profile <= test.profiles; profile++ {
				execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO %s SELECT '%s',REPEAT('%s',%d)",
					q("profile_data"), unifiedTestID(profile), test.unit, test.repeats))
			}
			result, err := store.UnifiedProfiles(gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag),
				root, unified.Bindings{}, fixture.at, unified.Page{Limit: 10, Properties: []string{"payload"}})
			if err != nil {
				if !test.oversized || !errors.Is(err, ErrProfileResultTooLarge) || result != nil {
					t.Fatalf("expected size rejection=%t with no result, got %T, result present=%t",
						test.oversized, err, result != nil)
				}
				// The same pool must remain usable; omit the oversized projection.
				healthy := fixture.query(t, root, unified.Page{Limit: 10})
				want := make([]unified.Truth, test.profiles)
				for i := range want {
					want[i] = unified.Match
				}
				assertUnifiedTruths(t, healthy, want)
				return
			}
			if test.oversized || result == nil {
				t.Fatalf("expected size rejection=%t, got successful result present=%t", test.oversized, result != nil)
			}
			assertUnifiedTruths(t, result, []unified.Truth{unified.Match})
			payload, ok := result.Profiles[0].Attributes["payload"].(string)
			if !ok || payload != strings.Repeat(test.unit, test.repeats) {
				t.Fatalf("expected %d exact payload bytes, got type %T and %d bytes",
					len(test.unit)*test.repeats, result.Profiles[0].Attributes["payload"], len(payload))
			}
		})
	}

	t.Run("server error and recovery", func(t *testing.T) {

		execUnifiedFixture(t, dw, "TRUNCATE TABLE "+q("profile_data")+";INSERT INTO "+q("profile_data")+
			" VALUES ('"+unifiedTestID(1)+"','x')")
		record := &unified.Rule{Record: &state.WhereCondition{Property: []string{"payload"},
			Operator: state.OpIs, Values: []any{"x"}}}

		for _, operation := range []struct {
			name string
			call func(*testing.T) (bool, error)
		}{
			{"query", func(t *testing.T) (bool, error) {
				ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag)
				result, err := store.UnifiedProfiles(ctx, record, unified.Bindings{}, fixture.at, unified.Page{Limit: 10})
				if err != nil {
					return result != nil, err
				}
				if result == nil {
					t.Fatal("expected complete query result, got nil")
				}
				assertUnifiedTruths(t, result, []unified.Truth{unified.Match})
				return true, nil
			}},
			{"explanation", func(t *testing.T) (bool, error) {
				ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordExplanationTag)
				result, err := store.ExplainUnifiedProfile(ctx, record, unified.Bindings{}, fixture.at, unifiedTestID(1))
				if err != nil {
					return result != nil, err
				}
				if result == nil || len(result.Nodes) != 1 || result.Nodes[0].Truth != unified.Match {
					t.Fatal("expected one matching explanation node, got incomplete explanation")
				}
				return true, nil
			}},
		} {
			t.Run(operation.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "CREATE OR REPLACE VIEW "+q("profiles")+" AS SELECT "+q("_kpid")+
					",CAST(1/(LENGTH("+q("payload")+")-LENGTH("+q("payload")+")) AS "+textType+") AS "+q("payload")+
					" FROM "+q("profile_data"))
				present, err := operation.call(t)
				if err != nil {
					unavailable, ok := errors.AsType[*UnavailableError](err)
					if !ok || present {
						t.Fatalf("expected opaque server error without partial result, got %T and present=%t", err, present)
					}
					switch platform {
					case "PostgreSQL":
						failure, ok := errors.AsType[*pgconn.PgError](unavailable.Err)
						if !ok || failure.Code != "22012" {
							t.Fatalf("expected server division-by-zero code 22012, got %T", unavailable.Err)
						}
					case "Snowflake":
						failure, ok := errors.AsType[*gosnowflake.SnowflakeError](unavailable.Err)
						if !ok || failure.SQLState != "22012" {
							t.Fatalf("expected server division-by-zero SQLSTATE 22012, got %T", unavailable.Err)
						}
					default:
						t.Fatalf("expected supported warehouse, got %s", platform)
					}
				}
				if err == nil {
					t.Fatal("expected server evaluation failure, got success")
				}
				execUnifiedFixture(t, dw, view)
				present, err = operation.call(t)
				if err != nil {
					t.Fatalf("expected complete recovery on the same pool, got %v", err)
				}
				if !present {
					t.Fatal("expected recovery result, got none")
				}
			})
		}

	})

	t.Run("invalid profile identity", func(t *testing.T) {
		execUnifiedFixture(t, dw, "TRUNCATE TABLE "+q("profile_data")+";INSERT INTO "+q("profile_data")+
			" VALUES ('bad','x')")
		result, err := store.UnifiedProfiles(gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag),
			root, unified.Bindings{}, fixture.at, unified.Page{Limit: 10})
		if err != nil {
			if _, ok := errors.AsType[*UnavailableError](err); !ok || result != nil {
				t.Fatalf("expected malformed identity failure without result, got %T and present=%t", err, result != nil)
			}
			return
		}
		t.Fatal("expected malformed identity rejection, got success")
	})

	t.Run("bounded scale", func(t *testing.T) {

		execUnifiedFixture(t, dw, "TRUNCATE TABLE "+q("profile_data"))
		switch platform {
		case "PostgreSQL":
			execUnifiedFixture(t, dw, `INSERT INTO profile_data
 SELECT '00000000-0000-0000-0000-'||lpad(i::text,12,'0'),'x' FROM generate_series(1,128) i;
INSERT INTO krenalis_events SELECT _kpid::uuid,_kpid||'/'||i::text,'track','Order Completed',NULL,
 '2026-01-31 12:00:00'::timestamp-i*INTERVAL '1 second','{"currency":"EUR","total":100}'::jsonb
 FROM profile_data CROSS JOIN generate_series(1,128) i`)
		case "Snowflake":
			execUnifiedFixture(t, dw, `INSERT INTO "PROFILE_DATA" SELECT
 '00000000-0000-0000-0000-'||LPAD(TO_VARCHAR(ROW_NUMBER() OVER(ORDER BY SEQ4())),12,'0'),'x'
 FROM TABLE(GENERATOR(ROWCOUNT=>128));
INSERT INTO "KRENALIS_EVENTS" SELECT "_KPID","_KPID"||'/'||TO_VARCHAR(i),'track','Order Completed',NULL,
 DATEADD('second',-i,'2026-01-31 12:00:00'::TIMESTAMP_NTZ),PARSE_JSON('{"currency":"EUR","total":100}')
 FROM "PROFILE_DATA" CROSS JOIN
 (SELECT ROW_NUMBER() OVER(ORDER BY SEQ4()) AS i FROM TABLE(GENERATOR(ROWCOUNT=>128)))`)
		default:
			t.Fatalf("expected supported warehouse, got %s", platform)
		}

		field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
		if err != nil {
			t.Fatalf("expected scale monetary field, got %v", err)
		}
		currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", scope.Event)
		if err != nil {
			t.Fatalf("expected scale native currency, got %v", err)
		}
		verify := &unified.OccurrencePredicate{Record: &state.WhereCondition{
			Property: []string{"messageId"}, Operator: state.OpExists}}
		root := &unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{
			{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal, Value: 128}}},
			{Sum: &unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
				Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("12800")}}},
			{Average: &unified.Average{Sum: unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR", Minimum: 128,
				Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}}}},
			{Experience: &unified.Experience{Scope: scope, First: true, Verify: verify}},
			{Experience: &unified.Experience{Scope: scope, Verify: verify}},
			{Experience: &unified.Experience{Scope: scope, Nth: 2, Verify: verify}},
			{LatestN: &unified.LatestN{Scope: scope, Size: 2, Predicate: verify}},
			{LastVersusPrevious: &unified.LastVersusPrevious{Scope: scope, Field: field, Currency: currency,
				Check: unified.LastPreviousCheck{Kind: unified.LastPreviousDifference,
					Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}},
		}}

		for _, first := range []int{120, 128} {
			result := fixture.query(t, root, unified.Page{First: first, Limit: 10, MatchingOnly: true})
			if result.Match != 128 || result.NoMatch != 0 || result.Unknown != 0 || len(result.Profiles) != 128-first {
				t.Fatalf("expected 128 matches and %d page rows, got totals %d/%d/%d and %d rows",
					128-first, result.Match, result.NoMatch, result.Unknown, len(result.Profiles))
			}
			for i, profile := range result.Profiles {
				if profile.ID != unifiedTestID(first+i+1) || profile.Truth != unified.Match {
					t.Fatalf("expected matching profile %s, got %+v", unifiedTestID(first+i+1), profile)
				}
			}
		}

		explained := fixture.explain(t, root, 1)
		if len(explained.Nodes) != 9 {
			t.Fatalf("expected nine scale explanation nodes, got %d", len(explained.Nodes))
		}
		for i, node := range explained.Nodes {
			if node.Truth != unified.Match {
				t.Fatalf("expected scale node %d MATCH, got %v", i, node.Truth)
			}
		}
		count, sum, average := explained.Nodes[1].Count, explained.Nodes[2].Sum, explained.Nodes[3].Average
		total := decimal.MustParse("12800")
		if count == nil || *count != (warehouses.OccurrenceCountBounds{Certain: 128, Possible: 128}) ||
			sum == nil || sum.Certain != 128 || sum.Possible != 128 || sum.Lower == nil || sum.Upper == nil ||
			!sum.Lower.Equal(total) || !sum.Upper.Equal(total) || average == nil || average.Certain != 128 ||
			average.Possible != 128 || average.Count == nil || *average.Count != 128 || average.Sum == nil ||
			!average.Sum.Equal(total) {
			t.Fatal("expected exact scale counts 128 and monetary inputs 12800, got inconsistent metadata")
		}

	})

}
