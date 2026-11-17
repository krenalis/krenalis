// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralConformancePostgreSQL checks the bounded cross-warehouse
// fixtures.
func TestBehavioralConformancePostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires PostgreSQL")
	}
	testBehavioralConformance(t, "PostgreSQL")
}

// TestBehavioralConformanceSnowflake uses an isolated Snowflake warehouse.
func TestBehavioralConformanceSnowflake(t *testing.T) {
	if testing.Short() {
		t.Skip("requires Snowflake")
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
	testBehavioralConformance(t, "Snowflake")
}

func testBehavioralConformance(t *testing.T, name string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, name)
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	switch name {
	case "PostgreSQL":
		execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,name text,
 timestamp timestamp(6),received_at timestamp(6),sent_at timestamp(6),properties jsonb)`)
	case "Snowflake":
		execBehavioralFixture(t, dw, `CREATE TABLE "PROFILES" ("_KPID" VARCHAR(36));
CREATE TABLE "KRENALIS_EVENTS" ("KPID" VARCHAR(36),"MESSAGE_ID" VARCHAR,"TYPE" VARCHAR,
 "EVENT" VARCHAR,"NAME" VARCHAR,"TIMESTAMP" TIMESTAMP_NTZ(9),
 "RECEIVED_AT" TIMESTAMP_NTZ(9),"SENT_AT" TIMESTAMP_NTZ(9),"PROPERTIES" VARIANT)`)
	default:
		t.Fatalf("expected supported warehouse, got %s", name)
	}
	table := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String()},
		{Name: "name", Type: types.String()}, {Name: "timestamp", Type: types.DateTime()},
		{Name: "received_at", Type: types.DateTime()}, {Name: "sent_at", Type: types.DateTime()},
		{Name: "properties", Type: types.JSON()},
	}}

	t.Run("selections", func(t *testing.T) { testBehavioralConformanceSelections(t, store, dw, table, name) })
	t.Run("periods", func(t *testing.T) { testBehavioralConformancePeriods(t, store, dw, table) })
	t.Run("pages", func(t *testing.T) { testBehavioralConformancePages(t, store, dw, table) })
	t.Run("count-money", func(t *testing.T) { testBehavioralConformanceCountMoney(t, store, dw, table) })
	t.Run("guards", func(t *testing.T) { testBehavioralConformanceGuards(t, store, dw, table) })
	t.Run("sources", func(t *testing.T) { testBehavioralConformanceSources(t, store, dw, table) })
	// Keep protocol last: it changes the profile schema and drops the event table.
	t.Run("protocol", func(t *testing.T) { testBehavioralConformanceProtocol(t, store, dw, table, name) })

}

func testBehavioralConformancePages(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table) {

	t.Helper()
	execBehavioralFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES ('"+behavioralTestID(1)+"'),('"+behavioralTestID(2)+"')")
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	var rows [][]any
	for _, typ := range []string{"page", "screen"} {
		for i, name := range []any{nil, "", "Pricing", "pricing"} {
			rows = append(rows, []any{behavioralTestID(1), fmt.Sprintf("%s/%d", typ, i), typ, nil, name,
				at.Add(-time.Hour), at, at, json.Value(`{"path":"/pricing","path.extra":"literal"}`)})
		}
	}
	err := dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected real page/screen writes, got %v", err)
	}

	for _, typ := range []string{"page", "screen"} {
		for _, selector := range []string{"Pricing", ""} {
			t.Run(typ+"/"+selector, func(t *testing.T) {
				count := &behavioral.Count{Scope: behavioral.Scope{Type: typ, Event: selector, Grain: "registered-message",
					Start: at.Add(-24 * time.Hour), End: at}, Test: behavioral.CountTest{Operator: behavioral.Equal, Value: 1}}
				want := behavioral.Match
				certain := int64(1)
				if selector == "" {
					want, certain = behavioral.NoMatch, 4
				}
				root := &behavioral.Rule{Count: count}
				result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 10})
				if err != nil {
					t.Fatalf("expected exact source selection, got %v", err)
				}
				assertBehavioralTruths(t, result, []behavioral.Truth{want, behavioral.NoMatch})
				explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at,
					behavioralTestID(1))
				if err != nil {
					t.Fatalf("expected source explanation, got %v", err)
				}
				if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Count == nil ||
					*explained.Nodes[0].Count != (warehouses.OccurrenceCountBounds{Certain: certain, Possible: certain}) {
					t.Fatalf("expected certain source count %d, got %+v", certain, explained)
				}
				// A literal dotted key is distinct from two successive JSON keys.
				count.Where = &behavioral.OccurrencePredicate{Record: &state.WhereCondition{
					Property: []string{"properties", "path.extra"}, Operator: state.OpIs,
					Values: []any{json.Value(`"literal"`)}}}
				qualified, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at,
					behavioral.Page{Limit: 10})
				if err != nil {
					t.Fatalf("expected literal JSON path, got %v", err)
				}
				if !reflect.DeepEqual(qualified, result) {
					t.Fatalf("expected literal path to preserve the selected population, got %+v", qualified)
				}
			})
		}
	}

}

func testBehavioralConformancePeriods(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table) {

	t.Helper()
	execBehavioralFixture(t, dw, "TRUNCATE TABLE profiles; TRUNCATE TABLE krenalis_events")
	calendar, err := behavioral.ResolveCalendar("gregorian-monday/Europe/Rome", "1")
	if err != nil {
		t.Fatalf("expected Rome calendar, got %v", err)
	}
	history, err := behavioral.ResolveHistory("registered-queryable-events", "1")
	if err != nil {
		t.Fatalf("expected registered history, got %v", err)
	}
	parse := func(text string) time.Time {
		instant, err := time.Parse(time.RFC3339Nano, text)
		if err != nil {
			t.Fatalf("expected fixture instant, got %v", err)
		}
		return instant
	}
	tests := []struct {
		name, at, start, end string
		scope                behavioral.Scope
	}{
		{"absolute", "2026-02-02T00:00:00Z", "2026-01-01T00:00:00.123456Z", "2026-02-01T00:00:00.123456Z", behavioral.Scope{}},
		{"elapsed", "2026-03-30T10:00:00.123456Z", "2026-03-29T10:00:00.123456Z", "2026-03-30T10:00:00.123456Z", behavioral.Scope{Elapsed: &behavioral.ElapsedPeriod{Older: 24 * time.Hour}}},
		{"band", "2026-04-01T00:00:00Z", "2026-01-01T00:00:00Z", "2026-01-31T00:00:00Z", behavioral.Scope{Elapsed: &behavioral.ElapsedPeriod{Older: 90 * 24 * time.Hour, Newer: 60 * 24 * time.Hour}}},
		{"spring day", "2026-03-30T12:00:00Z", "2026-03-28T23:00:00Z", "2026-03-29T22:00:00Z", behavioral.Scope{Calendar: &behavioral.CalendarPeriod{Calendar: calendar, Unit: "day", Count: 1}}},
		{"autumn day", "2026-10-26T12:00:00Z", "2026-10-24T22:00:00Z", "2026-10-25T23:00:00Z", behavioral.Scope{Calendar: &behavioral.CalendarPeriod{Calendar: calendar, Unit: "day", Count: 1}}},
		{"week", "2026-04-01T12:00:00Z", "2026-03-22T23:00:00Z", "2026-03-29T22:00:00Z", behavioral.Scope{Calendar: &behavioral.CalendarPeriod{Calendar: calendar, Unit: "week", Count: 1}}},
		{"month", "2026-04-01T12:00:00Z", "2026-02-28T23:00:00Z", "2026-03-31T22:00:00Z", behavioral.Scope{Calendar: &behavioral.CalendarPeriod{Calendar: calendar, Unit: "month", Count: 1}}},
	}
	var rows [][]any
	for i := 1; i <= 7; i++ {
		execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES ('"+behavioralTestID(i)+"')")
	}
	for i := range tests {
		test := &tests[i]
		at, start, end := parse(test.at), parse(test.start), parse(test.end)
		test.scope.Event, test.scope.Grain = fmt.Sprintf("Period-%d", i), "registered-message"
		if test.scope.Elapsed == nil && test.scope.Calendar == nil {
			test.scope.Start, test.scope.End = start, end
		}
		for p, stamp := range []time.Time{start.Add(-time.Microsecond), start, start.Add(time.Microsecond),
			end.Add(-time.Microsecond), end, end.Add(time.Microsecond), parse("0001-01-01T00:00:00Z")} {
			rows = append(rows, []any{behavioralTestID(p + 1), fmt.Sprintf("period/%d/%d", i, p), "track",
				test.scope.Event, nil, stamp, at.Add(7 * 24 * time.Hour), at.Add(7 * 24 * time.Hour), json.Value(`{}`)})
		}
	}
	err = dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected exact timestamp writes, got %v", err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			at, start, end := parse(test.at), parse(test.start), parse(test.end)
			count := &behavioral.Rule{Count: &behavioral.Count{Scope: test.scope,
				Test: behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1}}}
			base := behavioral.Scope{Event: test.scope.Event, Grain: "registered-message", History: history}
			experience := &behavioral.Rule{Experience: &behavioral.Experience{First: true, Scope: base,
				Verify: &behavioral.OccurrencePredicate{OccurredIn: &test.scope}}}
			for _, rule := range []*behavioral.Rule{count, experience} {
				result, err := store.QueryProfiles(t.Context(), rule, behavioral.Bindings{}, at, behavioral.Page{
					Limit: 10})
				if err != nil {
					t.Fatalf("expected microsecond half-open period, got %v", err)
				}
				assertBehavioralTruths(t, result,
					[]behavioral.Truth{behavioral.NoMatch, behavioral.Match, behavioral.Match,
						behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
				if !result.At.Equal(at) || result.At.Location() != time.UTC {
					t.Fatalf("expected single UTC evaluation time %s, got %s", at, result.At)
				}
			}
			explained, err := store.ExplainProfileFilter(t.Context(), count, behavioral.Bindings{}, at,
				behavioralTestID(2))
			if err != nil {
				t.Fatalf("expected period explanation, got %v", err)
			}
			if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.Match {
				t.Fatalf("expected start-inclusive explanation, got %+v", explained)
			}
			prepared := explained.Filter.Root.(*warehouses.IndependentOccurrenceCount)
			if prepared.Start != start || prepared.End != end {
				t.Fatalf("expected resolved bounds [%s,%s), got [%s,%s)", start, end, prepared.Start, prepared.End)
			}

			count.Count.Scope = base
			result, err := store.QueryProfiles(t.Context(), count, behavioral.Bindings{}, at, behavioral.Page{
				Limit: 10})
			if err != nil {
				t.Fatalf("expected registered history before T, got %v", err)
			}
			want := []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match,
				behavioral.Match, behavioral.Match, behavioral.Match}
			if end.Equal(at) {
				want[4], want[5] = behavioral.NoMatch, behavioral.NoMatch
			}
			assertBehavioralTruths(t, result, want)

		})
	}

}

func testBehavioralConformanceSelections(t *testing.T, store *Store, dw warehouses.Warehouse, table warehouses.Table, name string) {

	t.Helper()
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	histories := [][]pairOracleOccurrence{
		{}, {{1, 1, "150"}}, {{1, 2, "150"}},
		{{1, 1, "150"}, {2, 2, "150"}}, {{1, 1, "150"}, {2, 2, "80"}},
		{{1, 1, "150"}, {2, 0, "150"}},
		{{3, 2, "60"}, {2, 1, "80"}, {1, 1, "100"}},
		{{1, 1, "100"}, {1, 1, "50"}},
		{{1, 1, "150"}, {1, 1, "80"}, {1, 1, "150"}},
		{{1, 1, "150"}, {2, 2, "null"}},
		{{3, 2, "150"}, {2, 1, "80"}, {1, 1, "150"}},
		{{1, 1, "99999999999999.9999"}, {2, 1, "-99999999999999.9999"}},
	}
	// A fixed seed bounds remote cost while keeping the sample reproducible.
	random := rand.New(rand.NewPCG(26, 11011))
	for range 16 {
		var history []pairOracleOccurrence
		for range 1 + random.IntN(6) {
			history = append(history, pairOracleOccurrence{1 + random.IntN(3), random.IntN(3),
				[]string{"80", "150", "null"}[random.IntN(3)]})
		}
		histories = append(histories, history)
	}
	// Eleven identities exercise position/size 10, optional predecessors,
	// optional successors and a tie across the block boundary.
	for variant := range 4 {
		var history []pairOracleOccurrence
		for i := 1; i <= 11; i++ {
			row := pairOracleOccurrence{i, 1, "150"}
			if i == 11 {
				row.amount = "80"
			}
			if variant == 1 && i >= 10 || variant == 2 && i == 1 || variant == 3 && i == 11 {
				row.membership = 2
			}
			if variant == 3 && i == 11 {
				row.timestamp = 10
			}
			history = append(history, row)
		}
		histories = append(histories, history)
	}
	var profiles []string
	var rows [][]any
	for p, history := range histories {
		profiles = append(profiles, "('"+behavioralTestID(p+1)+"')")
		for i, row := range history {
			currency := `"currency":"EUR",`
			if row.membership == 0 {
				currency = `"currency":"USD",`
			} else if row.membership == 2 {
				currency = []string{"", `"currency":null,`, `"currency":false,`, `"currency":123,`}[i%4]
			}
			rows = append(rows, []any{behavioralTestID(p + 1), fmt.Sprintf("selection/%d/%d", p, i), "track",
				"Order Completed", nil, at.AddDate(0, -1, row.timestamp-1), at, at,
				json.Value(`{` + currency + `"total":` + row.amount + `}`)})
		}
	}
	execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
	err := dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected real selection writes, got %v", err)
	}
	// Duplicate every physical row after Merge: copies are not new identities.
	execBehavioralFixture(t, dw, "INSERT INTO krenalis_events SELECT * FROM krenalis_events")
	field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected amount binding, got %v", err)
	}
	currency, err := behavioral.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR",
		"Order Completed")
	if err != nil {
		t.Fatalf("expected currency binding, got %v", err)
	}
	scope := behavioral.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0),
		End: at}
	money := &behavioral.OccurrencePredicate{Money: &behavioral.Money{Field: field, NativeCurrency: "EUR",
		Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("100")}}}
	type check struct {
		name string
		rule behavioral.Rule
	}
	var checks []check
	for _, selector := range []struct {
		name  string
		first bool
		nth   int
	}{{"last", false, 0}, {"first", true, 0}, {"nth1", false, 1}, {"nth2", false, 2}, {"nth3", false, 3}, {"nth10", false, 10}} {
		checks = append(checks, check{selector.name, behavioral.Rule{Experience: &behavioral.Experience{
			First: selector.first, Nth: selector.nth, Scope: scope, Currency: currency, Verify: money}}})
	}
	for _, size := range []int{1, 2, 10} {
		for _, quantifier := range []struct {
			name  string
			value behavioral.LatestNQuantifier
		}{
			{"all", behavioral.LatestNQuantifier{}},
			{"some", behavioral.LatestNQuantifier{Kind: behavioral.LatestSome}},
			{"exactly1", behavioral.LatestNQuantifier{Kind: behavioral.LatestCardinality, Operator: behavioral.Equal, Value: "1"}},
			{"atMost1", behavioral.LatestNQuantifier{Kind: behavioral.LatestCardinality, Operator: behavioral.LessOrEqual, Value: "1"}},
		} {
			checks = append(checks, check{fmt.Sprintf("latest%d/%s", size, quantifier.name),
				behavioral.Rule{LatestN: &behavioral.LatestN{Size: size, Quantifier: quantifier.value, Scope: scope,
					Currency: currency, Predicate: money}}})
		}
	}
	for _, test := range []struct {
		name     string
		operator behavioral.Comparison
	}{
		{"lt", behavioral.Less}, {"le", behavioral.LessOrEqual},
		{"gt", behavioral.Greater}, {"ge", behavioral.GreaterOrEqual},
	} {
		checks = append(checks, check{"difference/" + test.name, behavioral.Rule{
			LastVersusPrevious: &behavioral.LastVersusPrevious{Scope: scope, Field: field, Currency: currency,
				Check: behavioral.LastPreviousCheck{Kind: behavioral.LastPreviousDifference,
					Test: behavioral.SumTest{Operator: test.operator, Value: decimal.MustParse("-30")}}}}})
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {

			want := make([]behavioral.Truth, len(histories))
			selections := make([]warehouses.OccurrenceSelection, len(histories))
			pairs := make([]warehouses.OccurrencePairSelection, len(histories))
			for p, history := range histories {
				var occurrences []nthOracleOccurrence
				for _, row := range history {
					verification := 2
					if row.amount != "null" {
						verification = 0
						if !decimal.MustParse(row.amount).Less(decimal.MustParse("100")) {
							verification = 1
						}
					}
					stamp := row.timestamp
					if check.rule.Experience != nil && check.name == "last" {
						stamp = -stamp
					}
					occurrences = append(occurrences, nthOracleOccurrence{stamp, row.membership, verification})
				}
				switch {
				case check.rule.Experience != nil:
					position := check.rule.Experience.Nth
					if position == 0 {
						position = 1
					}
					want[p], selections[p] = enumerateNth(occurrences, position)
				case check.rule.LatestN != nil:
					latest := check.rule.LatestN
					counts, selection := enumerateLatestCounts(occurrences, latest.Size)
					selections[p] = selection
					lower, upper := latest.Size, latest.Size
					if latest.Quantifier.Kind == behavioral.LatestSome {
						lower = 1
					} else if latest.Quantifier.Kind == behavioral.LatestCardinality {
						lower, upper = 1, 1
						if latest.Quantifier.Operator == behavioral.LessOrEqual {
							lower = 0
						}
					}
					want[p] = latestCountTruth(counts, selection, lower, upper)
				case check.rule.LastVersusPrevious != nil:
					want[p], pairs[p] = enumeratePairTest(history, check.rule.LastVersusPrevious.Check.Test.Operator, "-30")
				default:
					t.Fatal("expected selection fixture, got an unknown rule")
				}
			}

			result, err := store.QueryProfiles(t.Context(), &check.rule, behavioral.Bindings{}, at,
				behavioral.Page{Limit: 100})
			if err != nil {
				t.Fatalf("expected compatible selections, got %v", err)
			}
			assertBehavioralTruths(t, result, want)
			for i, profile := range result.Profiles {
				if profile.ID != behavioralTestID(i+1) {
					t.Fatalf("expected stable profile %s, got %s", behavioralTestID(i+1), profile.ID)
				}
			}
			// Check every profile in two metadata shapes. Other rules explain
			// the first six profiles, plus every position/size 10 border.
			limit := 6
			if check.rule.LatestN != nil && check.rule.LatestN.Size == 2 &&
				check.rule.LatestN.Quantifier.Kind == behavioral.LatestSome ||
				check.rule.LastVersusPrevious != nil &&
					check.rule.LastVersusPrevious.Check.Test.Operator == behavioral.LessOrEqual {
				limit = len(histories)
			}
			checkBorders := check.rule.Experience != nil && check.rule.Experience.Nth == 10 ||
				check.rule.LatestN != nil && check.rule.LatestN.Size == 10
			for p := range histories {
				if p >= limit && (!checkBorders || p < len(histories)-4) {
					continue
				}
				explained, err := store.ExplainProfileFilter(t.Context(), &check.rule, behavioral.Bindings{}, at,
					behavioralTestID(p+1))
				if err != nil {
					t.Fatalf("expected complete selection explanation, got %v", err)
				}
				if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want[p] {
					t.Fatalf("expected profile %d truth %v, got %+v", p+1, want[p], explained)
				}
				node := explained.Nodes[0]
				if check.rule.LastVersusPrevious != nil {
					if node.Pair == nil || *node.Pair != pairs[p] {
						t.Fatalf("expected profile %d pair %+v, got %+v", p+1, pairs[p], node.Pair)
					}
				} else if node.Experience == nil || *node.Experience != selections[p] {
					t.Fatalf("expected profile %d selection %+v, got %+v", p+1, selections[p], node.Experience)
				}
			}
			empty, err := store.QueryProfiles(t.Context(), &check.rule, behavioral.Bindings{}, at,
				behavioral.Page{First: 100, Limit: 1, MatchingOnly: true})
			if err != nil {
				t.Fatalf("expected empty page with totals, got %v", err)
			}
			if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
				empty.Unknown != result.Unknown || empty.At != result.At {
				t.Fatalf("expected unchanged totals and T on an empty page, got %+v", empty)
			}

		})
	}

	// Certainly excluded copies may disagree on timestamps; relevant copies
	// may not. Both checks exercise the same generated SQL and real scanner.
	copyRow := func(id string) {
		stamp := "timestamp + interval '1 microsecond'"
		if name == "Snowflake" {
			stamp = `DATEADD(microsecond,1,"TIMESTAMP")`
		}
		execBehavioralFixture(t, dw, `INSERT INTO krenalis_events SELECT kpid,message_id,type,event,name,`+stamp+
			`,received_at,sent_at,properties FROM krenalis_events WHERE message_id='`+id+`'`)
	}
	root := &checks[0].rule
	before, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 100})
	if err != nil {
		t.Fatalf("expected selection before excluded copies, got %v", err)
	}
	copyRow("selection/5/1")
	after, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 100})
	if err != nil {
		t.Fatalf("expected excluded timestamp copies to be ignored, got %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("expected excluded copies to preserve the selection %+v, got %+v", before, after)
	}
	copyRow("selection/1/0")
	result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 100})
	if err != nil {
		if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
			t.Fatalf("expected source conflict without partial data, got %+v, %v", result, err)
		}
	}
	if err == nil {
		t.Fatalf("expected timestamp source conflict, got %+v", result)
	}
	explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(1))
	if err != nil {
		t.Fatalf("expected explanation isolated from another profile's fault, got %v", err)
	}
	if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.NoMatch {
		t.Fatalf("expected unaffected empty profile, got %+v", explained)
	}

}
