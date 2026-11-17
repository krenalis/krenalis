// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedIdentityPostgreSQL checks literal message identities under ICU.
func TestUnifiedIdentityPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testUnifiedIdentity(t, "PostgreSQL")
}

// TestUnifiedIdentitySnowflake uses an isolated Snowflake warehouse.
func TestUnifiedIdentitySnowflake(t *testing.T) {
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
		t.Fatal("expected verified Snowflake provider, got closed gate")
	}
	testUnifiedIdentity(t, "Snowflake")
}

func testUnifiedIdentity(t *testing.T, platform string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	profileType, timestampType := "uuid", "timestamp(6)"
	switch platform {
	case "PostgreSQL":
		execUnifiedFixture(t, dw, `CREATE COLLATION krenalis_ci
 (provider=icu,locale='und-u-ks-level2',deterministic=false);
CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text COLLATE krenalis_ci,
 type text,event text,name text,timestamp timestamp(6),properties jsonb)`)
	case "Snowflake":
		profileType, timestampType = "VARCHAR(36)", "TIMESTAMP_NTZ(6)"
		execUnifiedFixture(t, dw, `CREATE TABLE "PROFILES" ("_KPID" VARCHAR(36));
CREATE TABLE "KRENALIS_EVENTS" ("KPID" VARCHAR(36),"MESSAGE_ID" VARCHAR COLLATE 'en-ci-rtrim',
 "TYPE" VARCHAR,"EVENT" VARCHAR,"NAME" VARCHAR,"TIMESTAMP" TIMESTAMP_NTZ(6),"PROPERTIES" VARIANT)`)
	default:
		t.Fatalf("expected supported warehouse, got %s", platform)
	}
	execUnifiedFixture(t, dw, "INSERT INTO "+quoteUnifiedRecordIdentifier(platform, "profiles")+
		" VALUES ('"+unifiedTestID(1)+"'),('"+unifiedTestID(2)+"')")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.Add(-24 * time.Hour), End: at}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected monetary field, got %v", err)
	}
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", scope.Event)
	if err != nil {
		t.Fatalf("expected native currency, got %v", err)
	}
	verify := &unified.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"properties", "ok"},
		Operator: state.OpIsOneOf, Values: []any{json.Value(`true`)}}}
	root := &unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{
		{Count: &unified.Count{Scope: scope, Currency: currency,
			Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}}},
		{Sum: &unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
			Test: unified.SumTest{Operator: unified.GreaterOrEqual}}},
		{Average: &unified.Average{Sum: unified.Sum{Scope: scope, Field: field, NativeCurrency: "EUR",
			Minimum: 1, Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}},
		{Experience: &unified.Experience{Scope: scope, Currency: currency, First: true, Verify: verify}},
		{Experience: &unified.Experience{Scope: scope, Currency: currency, Verify: verify}},
		{Experience: &unified.Experience{Scope: scope, Currency: currency, Nth: 2, Verify: verify}},
		{LatestN: &unified.LatestN{Scope: scope, Currency: currency, Size: 2, Predicate: verify}},
		{LastVersusPrevious: &unified.LastVersusPrevious{Scope: scope, Field: field, Currency: currency,
			Check: unified.LastPreviousCheck{Kind: unified.LastPreviousDifference,
				Test: unified.SumTest{Operator: unified.GreaterOrEqual}}}},
	}}

	type event struct {
		profile int
		id      string
		at      time.Time
		payload string
	}
	type expectation struct {
		count      int64
		total      string
		selections [4]int64
		pair       warehouses.OccurrencePairSelection
		truths     [8]unified.Truth
	}
	const m, f = unified.Match, unified.NoMatch
	base := at.Add(-time.Hour)
	known := `{"currency":"EUR","total":10,"ok":true}`
	foreign := `{"currency":"USD","total":10,"ok":true}`
	one := expectation{1, "10", [4]int64{1, 1, 0, 0},
		warehouses.OccurrencePairSelection{AbsencePossible: true}, [8]unified.Truth{m, m, m, m, m, f, f, f}}
	empty := expectation{total: "0", pair: warehouses.OccurrencePairSelection{AbsencePossible: true},
		truths: [8]unified.Truth{f, m, f, f, f, f, f, f}}
	var ordered, tied []event
	for i, id := range []string{"abc", "ABC", "abc ", "é", "e\u0301"} {
		ordered = append(ordered, event{1, id, base.Add(time.Duration(i) * time.Second),
			fmt.Sprintf(`{"currency":"EUR","total":%d,"ok":%t}`, 10*(i+1), i%2 == 0)})
		tied = append(tied, event{1, id, base, known})
	}
	for _, test := range []struct {
		name                    string
		rows                    []event
		want                    expectation
		fault, otherFault, both bool
	}{
		{"ordered identities", ordered, expectation{5, "150", [4]int64{1, 1, 1, 2}, warehouses.OccurrencePairSelection{Candidates: 2, Unique: true}, [8]unified.Truth{m, m, m, m, m, f, f, m}}, false, false, false},
		{"tied identities", tied, expectation{5, "50", [4]int64{5, 5, 5, 5}, warehouses.OccurrencePairSelection{Candidates: 5}, [8]unified.Truth{m, m, m, m, m, m, m, m}}, false, false, false},
		{"same copies", []event{{1, "abc", base, known}, {1, "abc", base, known}}, one, false, false, false},
		{"distinct profiles", []event{{1, "abc", base, known}, {2, "ABC", base, known}}, one, false, false, true},
		{"distinct periods", []event{{1, "abc", base, known}, {1, "ABC", at.Add(-48 * time.Hour), known}}, one, false, false, false},
		{"distinct qualifications", []event{{1, "abc", base, known}, {1, "ABC", base, foreign}}, one, false, false, false},
		{"copy profile conflict", []event{{1, "abc", base, known}, {2, "abc", base, known}}, one, true, false, false},
		{"copy period conflict", []event{{1, "abc", base, known}, {1, "abc", at.Add(-48 * time.Hour), known}}, one, true, false, false},
		{"copy qualification conflict", []event{{1, "abc", base, known}, {1, "abc", base, foreign}}, one, true, false, false},
		{"other profile fault", []event{{1, "abc", base, known}, {2, "ABC", base, known}, {2, "bad", base, known}, {2, "bad", base, foreign}}, one, false, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {

			execUnifiedFixture(t, dw, "TRUNCATE TABLE "+quoteUnifiedRecordIdentifier(platform, "krenalis_events"))
			// Controlled DML preserves all physical copies; Merge has its own
			// collation behavior and is deliberately outside this reader fixture.
			for _, event := range test.rows {
				payload := "'" + strings.ReplaceAll(event.payload, "'", "''") + "'"
				if platform == "Snowflake" {
					payload = "PARSE_JSON(" + payload + ")"
				} else {
					payload = "CAST(" + payload + " AS jsonb)"
				}
				execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO %s SELECT CAST('%s' AS %s),'%s',"+
					"'track','Order Completed',NULL,CAST('%s' AS %s),%s",
					quoteUnifiedRecordIdentifier(platform, "krenalis_events"), unifiedTestID(event.profile), profileType,
					strings.ReplaceAll(event.id, "'", "''"), event.at.Format("2006-01-02 15:04:05.999999"),
					timestampType, payload))
			}
			counts, err := dw.Counts(t.Context(), "krenalis_events", nil)
			if err != nil {
				t.Fatalf("expected physical row count, got %v", err)
			}
			if len(counts) != 1 || counts[0] != len(test.rows) {
				t.Fatalf("expected %d physical rows, got %v", len(test.rows), counts)
			}

			result, err := store.UnifiedProfiles(gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag),
				root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				if _, ok := errors.AsType[*WarehouseError](err); !ok || !(test.fault || test.otherFault) ||
					result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected source failure=%t without partial results, got %+v, %v",
						test.fault || test.otherFault, result, err)
				}
			}
			if err == nil {
				if test.fault || test.otherFault {
					t.Fatal("expected source failure, got success")
				}
				truth := m
				for _, leaf := range test.want.truths {
					if leaf == f {
						truth = f
					}
				}
				assertUnifiedTruths(t, result, []unified.Truth{truth, f})
				for i, profile := range result.Profiles {
					if profile.ID != unifiedTestID(i+1) {
						t.Fatalf("expected profile %s, got %s", unifiedTestID(i+1), profile.ID)
					}
				}
			}

			profiles := []int{1, 2}
			if test.fault {
				profiles = profiles[:1]
			}
			for _, profile := range profiles {
				want := test.want
				if profile == 2 && !test.both {
					want = empty
				}
				fault := test.fault || profile == 2 && test.otherFault
				explained, err := store.ExplainUnifiedProfile(
					gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordExplanationTag),
					root, unified.Bindings{}, at, unifiedTestID(profile))
				if err != nil {
					if _, ok := errors.AsType[*WarehouseError](err); !ok || !fault || explained != nil ||
						!strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected profile %d source failure=%t without explanation, got %+v, %v",
							profile, fault, explained, err)
					}
					continue
				}
				if fault || explained == nil || len(explained.Nodes) != 9 {
					t.Fatalf("expected nine complete explanation nodes with failure=%t, got %+v", fault, explained)
				}
				truth := m
				for i, expected := range want.truths {
					if explained.Nodes[i+1].Truth != expected {
						t.Fatalf("expected profile %d node %d truth %d, got %+v", profile, i+1, expected,
							explained.Nodes[i+1])
					}
					if expected == f {
						truth = f
					}
				}
				if explained.Nodes[0].Truth != truth {
					t.Fatalf("expected root truth %d, got %d", truth, explained.Nodes[0].Truth)
				}
				count := explained.Nodes[1].Count
				if count == nil || count.Certain != want.count || count.Possible != want.count {
					t.Fatalf("expected exact count %d, got %+v", want.count, count)
				}
				total := decimal.MustParse(want.total)
				sum := explained.Nodes[2].Sum
				if sum == nil || sum.Certain != want.count || sum.Possible != want.count ||
					sum.Lower == nil || sum.Upper == nil || !sum.Lower.Equal(total) || !sum.Upper.Equal(total) {
					t.Fatalf("expected exact sum %s and count %d, got %+v", total, want.count, sum)
				}
				average := explained.Nodes[3].Average
				if average == nil || average.Certain != want.count || average.Possible != want.count ||
					average.Count == nil || *average.Count != want.count || average.Sum == nil ||
					!average.Sum.Equal(total) {
					t.Fatalf("expected average inputs %s and %d, got %+v", total, want.count, average)
				}
				for i, candidates := range want.selections {
					selection := explained.Nodes[i+4].Experience
					if selection == nil || selection.Candidates != candidates ||
						selection.AbsencePossible != (candidates == 0) {
						t.Fatalf("expected node %d candidates %d with absence=%t, got %+v",
							i+4, candidates, candidates == 0, selection)
					}
				}
				pair := explained.Nodes[8].Pair
				if pair == nil || *pair != want.pair {
					t.Fatalf("expected pair selection %+v, got %+v", want.pair, pair)
				}
			}

		})
	}

}
