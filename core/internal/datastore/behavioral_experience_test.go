// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestBehavioralExperiencePostgreSQL checks exact selection on a coherent view.
func TestBehavioralExperiencePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,3) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,name text,
 timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	qualify := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"properties", "q"},
			Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
	}}
	verify := &behavioral.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"properties", "v"},
		Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
	}
	experience := behavioral.Experience{Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Where: qualify, Verify: verify}
	root := &behavioral.Rule{Experience: &experience}

	for _, test := range []struct {
		name, rows string
		truth      behavioral.Truth
		candidates int64
		absent     bool
		fault      bool
	}{
		{"empty", ``, behavioral.NoMatch, 0, true, false},
		{"latest true", `('a','2026-01-01','{"q":"yes","v":"no"}'),('b','2026-01-02','{"q":"yes","v":"yes"}')`, behavioral.Match, 1, false, false},
		{"no fallback", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"q":"yes","v":"no"}')`, behavioral.NoMatch, 1, false, false},
		{"latest unknown", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"q":"yes"}')`, behavioral.Unknown, 1, false, false},
		{"JSON null verification", `('a','2026-01-01','{"q":"yes","v":null}')`, behavioral.NoMatch, 1, false, false},
		{"different JSON kind", `('a','2026-01-01','{"q":"yes","v":100}')`, behavioral.NoMatch, 1, false, false},
		{"optional latest same body", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"v":"yes"}')`, behavioral.Match, 2, false, false},
		{"optional latest different body", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"v":"no"}')`, behavioral.Unknown, 2, false, false},
		{"older optional irrelevant", `('a','2026-01-01','{"v":"no"}'),('b','2026-01-02','{"q":"yes","v":"yes"}')`, behavioral.Match, 1, false, false},
		{"only possible true", `('a','2026-01-01','{"v":"yes"}')`, behavioral.Unknown, 1, true, false},
		{"only possible false", `('a','2026-01-01','{"v":"no"}')`, behavioral.NoMatch, 1, true, false},
		{"excluded unknown body", `('a','2026-01-01','{"q":"no"}')`, behavioral.NoMatch, 0, true, false},
		{"tie true", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes","v":"yes"}')`, behavioral.Match, 2, false, false},
		{"tie false", `('a','2026-01-01','{"q":"yes","v":"no"}'),('b','2026-01-01','{"q":"yes","v":"no"}')`, behavioral.NoMatch, 2, false, false},
		{"tie mixed", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes","v":"no"}')`, behavioral.Unknown, 2, false, false},
		{"tie unknown", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes"}')`, behavioral.Unknown, 2, false, false},
		{"optional tie", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"v":"no"}')`, behavioral.Unknown, 2, false, false},
		{"same copies", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"q":"yes","v":"yes"}')`, behavioral.Match, 1, false, false},
		{"exact timestamp conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01 00:00:00.000001','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"possible timestamp conflict", `('a','2026-01-01','{"v":"yes"}'),('a','2026-01-02','{"v":"yes"}')`, 0, 0, false, true},
		{"excluded timestamp conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-01-02','{"q":"no"}')`, behavioral.NoMatch, 0, true, false},
		{"mixed qualification and timestamp conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-01-02','{"q":"yes"}')`, 0, 0, false, true},
		{"period conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-02-01','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"qualification conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"v":"yes"}')`, 0, 0, false, true},
		{"verify conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"q":"yes"}')`, 0, 0, false, true},
		{"irrelevant verify copies", `('a','2026-01-01','{"q":"no","v":"yes"}'),('a','2026-01-01','{"q":"no"}')`, behavioral.NoMatch, 0, true, false},
		{"excluded period conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-02-01','{"q":"no"}')`, 0, 0, false, true},
		{"start included end excluded", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-02-01','{"q":"yes","v":"no"}')`, behavioral.Match, 1, false, false},
		{"microseconds order", `('a','2026-01-01 00:00:00.000002','{"q":"yes","v":"yes"}'),('b','2026-01-01 00:00:00.000001','{"q":"yes","v":"no"}')`, behavioral.Match, 1, false, false},
		{"missing timestamp", `('a',NULL,'{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"missing identity", `(NULL,'2026-01-01','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			if test.rows != "" {
				execBehavioralFixture(t, dw, `INSERT INTO krenalis_events
SELECT '`+behavioralTestID(1)+`',id,'track','Order Completed',NULL,stamp::timestamp,payload::jsonb
FROM (VALUES `+test.rows+`) v(id,stamp,payload)`)
			}
			// A later event of another profile must not suppress this profile's candidates.
			execBehavioralFixture(t, dw, `INSERT INTO krenalis_events VALUES
('`+behavioralTestID(2)+`','other-profile','track','Order Completed',NULL,'2026-01-31','{"q":"yes","v":"no"}')`)
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected fault=%v without partial result, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil {
				if test.fault {
					t.Fatal("expected source fault, got success")
				}
				assertBehavioralTruths(t, result, []behavioral.Truth{test.truth, behavioral.NoMatch, behavioral.NoMatch})
				empty, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at,
					behavioral.Page{First: 99, Limit: 1, MatchingOnly: true})
				if err != nil {
					t.Fatalf("expected complete empty page, got %v", err)
				}
				if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
					empty.Unknown != result.Unknown {
					t.Fatalf("expected identical totals on empty page, got %+v", empty)
				}
			}
			explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(1))
			if err != nil {
				if !test.fault || explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected fault=%v without partial explanation, got %+v, %v", test.fault, explained, err)
				}
				return
			}
			if test.fault || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != test.truth {
				t.Fatalf("expected truth %v with fault=%v, got %+v", test.truth, test.fault, explained)
			}
			selection := explained.Nodes[0].Experience
			if selection == nil || selection.Candidates != test.candidates || selection.AbsencePossible != test.absent {
				t.Fatalf("expected %d candidates and absence=%v, got %+v", test.candidates, test.absent, selection)
			}
		})
	}

	t.Run("fault scoped to another profile", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','broken','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}'),
('`+behavioralTestID(1)+`','broken','track','Order Completed',NULL,'2026-01-02','{"q":"yes","v":"yes"}')`)
		explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(2))
		if err != nil {
			t.Fatalf("expected complete unrelated-profile explanation, got %v", err)
		}
		if explained.Nodes[0].Truth != behavioral.NoMatch {
			t.Fatalf("expected NO_MATCH, got %+v", explained)
		}
	})

	t.Run("unrelated source timestamp conflict", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(2)+`','other','track','Product Viewed',NULL,'2026-01-01','{}'),
('`+behavioralTestID(2)+`','other','track','Product Viewed',NULL,'2026-01-02','{}')`)
		result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected irrelevant timestamp conflict ignored, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
	})

	t.Run("attribution conflicts", func(t *testing.T) {
		for _, other := range []string{"NULL", "'" + behavioralTestID(2) + "'"} {
			execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','same','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}'),
(`+other+`,'same','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}')`)
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				if result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected attribution fault without partial result, got %+v, %v", result, err)
				}
				continue
			}
			t.Fatal("expected attribution conflict, got success")
		}
	})

	t.Run("period restricts base", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','early','track','Order Completed',NULL,'2026-01-10','{"q":"yes","v":"yes"}'),
('`+behavioralTestID(1)+`','later','track','Order Completed',NULL,'2026-01-20','{"q":"yes","v":"no"}')`)
		bounded := experience
		bounded.Scope.End = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Experience: &bounded}, behavioral.Bindings{}, at,
			behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected Last within the selected period, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
		history, err := behavioral.ResolveHistory("registered-queryable-events", "1")
		if err != nil {
			t.Fatalf("expected recorded-history reference, got %v", err)
		}
		bounded.Scope.Start, bounded.Scope.End, bounded.Scope.History = time.Time{}, time.Time{}, history
		result, err = store.QueryProfiles(t.Context(), &behavioral.Rule{Experience: &bounded}, behavioral.Bindings{}, at,
			behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected Last over full recorded history, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
	})

	t.Run("page and screen", func(t *testing.T) {
		for _, typ := range []string{"page", "screen"} {
			execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','named','`+typ+`',NULL,'Pricing','2026-01-10','{"q":"yes","v":"yes"}'),
('`+behavioralTestID(1)+`','unnamed','`+typ+`',NULL,NULL,'2026-01-20','{"q":"yes","v":"no"}'),
('`+behavioralTestID(1)+`','track','track','Pricing',NULL,'2026-01-30','{"q":"yes","v":"no"}')`)
			selected := experience
			selected.Scope.Type, selected.Scope.Event = typ, "Pricing"
			result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Experience: &selected}, behavioral.Bindings{}, at,
				behavioral.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected named %s selection, got %v", typ, err)
			}
			assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
			selected.Scope.Event = ""
			result, err = store.QueryProfiles(t.Context(), &behavioral.Rule{Experience: &selected}, behavioral.Bindings{}, at,
				behavioral.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected all %s messages, got %v", typ, err)
			}
			assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
		}
	})

	t.Run("mixed monetary explanations", func(t *testing.T) {
		execBehavioralFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+behavioralTestID(1)+`','one','track','Order Completed',NULL,'2026-01-10',
'{"q":"yes","v":"yes","total":120,"currency":"EUR"}'),
('`+behavioralTestID(2)+`','two','track','Order Completed',NULL,'2026-01-20',
'{"q":"yes","v":"no","total":100,"currency":"EUR"}')`)
		field, err := behavioral.ResolveMonetaryField("commerce.order_completed.total", "1")
		if err != nil {
			t.Fatalf("expected resolved monetary field, got %v", err)
		}
		sum := behavioral.Sum{Scope: experience.Scope, Field: field, NativeCurrency: "EUR",
			Test: behavioral.SumTest{Operator: behavioral.GreaterOrEqual, Value: decimal.MustParse("100")}}
		average := sum
		average.Minimum = 1
		mixed := &behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{
			{Experience: &experience}, {Sum: &sum}, {Average: &behavioral.Average{Sum: average}},
		}}
		result, err := store.QueryProfiles(t.Context(), mixed, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected mixed listing, got %v", err)
		}
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.NoMatch})
		for n := 1; n <= 3; n++ {
			explained, err := store.ExplainProfileFilter(t.Context(), mixed, behavioral.Bindings{}, at, behavioralTestID(n))
			if err != nil {
				t.Fatalf("expected mixed explanation, got %v", err)
			}
			if len(explained.Nodes) != 4 || explained.Nodes[1].Experience == nil || explained.Nodes[2].Sum == nil ||
				explained.Nodes[3].Average == nil || explained.Nodes[1].Count != nil {
				t.Fatalf("expected distinct Experience/Sum/Average metadata, got %+v", explained)
			}
		}
	})

}

// TestBehavioralExperienceOraclePostgreSQL enumerates independent completions,
// including every tied winner, instead of using the compiler's cutoff rule.
func TestBehavioralExperienceOraclePostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		first bool
		times [3]int
	}{
		{false, [3]int{1, 2, 3}}, {false, [3]int{1, 2, 2}},
		{false, [3]int{1, 1, 1}}, {false, [3]int{3, 2, 1}},
		{true, [3]int{1, 2, 3}}, {true, [3]int{1, 2, 2}},
		{true, [3]int{1, 1, 1}}, {true, [3]int{3, 2, 1}},
	} {
		first, times := test.first, test.times
		t.Run(fmt.Sprintf("first=%v/times=%v", first, times), func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE profiles,krenalis_events")
			var profiles, events []string
			want := make([]behavioral.Truth, 729)
			for code := range want {
				id := behavioralTestID(code + 1)
				profiles = append(profiles, "('"+id+"')")
				n := code
				var qualifications, bodies [3]int
				for i := range 3 {
					qualifications[i], bodies[i] = n%3, n/3%3
					n /= 9
					payload := []string{}
					for j, state := range []int{qualifications[i], bodies[i]} {
						if state < 2 {
							key := "q"
							if j == 1 {
								key = "v"
							}
							payload = append(payload, fmt.Sprintf(`"%s":"%s"`, key, []string{"no", "yes"}[state]))
						}
					}
					events = append(events, fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-0%d','{%s}')",
						id, code, i, times[i], strings.Join(payload, ",")))
				}
				seen := [2]bool{}
				// Every membership subset and every admissible first/last witness.
				for mask := range 8 {
					valid, selected := true, 0
					for i, q := range qualifications {
						present := mask&(1<<i) != 0
						if q < 2 && present != (q == 1) {
							valid = false
						}
						if present && (selected == 0 || first && times[i] < selected || !first && times[i] > selected) {
							selected = times[i]
						}
					}
					if !valid {
						continue
					}
					if selected == 0 {
						seen[0] = true
					}
					for i, body := range bodies {
						if mask&(1<<i) == 0 || times[i] != selected {
							continue
						}
						seen[0] = seen[0] || body != 1
						seen[1] = seen[1] || body != 0
					}
				}
				want[code] = behavioral.Unknown
				if !seen[0] {
					want[code] = behavioral.Match
				} else if !seen[1] {
					want[code] = behavioral.NoMatch
				}
			}
			execBehavioralFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
			execBehavioralFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
			predicate := func(key string) *state.Where {
				return &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", key},
						Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
				}}
			}
			root := &behavioral.Rule{Experience: &behavioral.Experience{First: first, Scope: behavioral.Scope{Event: "Order Completed",
				Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}, Where: predicate("q"),
				Verify: &behavioral.OccurrencePredicate{Record: predicate("v").Rules[0].(*state.WhereCondition)}}}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected exact enumeration result, got %v", err)
			}
			assertBehavioralTruths(t, result, want)
		})
	}
}

// TestBehavioralExperienceTimePostgreSQL checks timestamp verification through
// the real Record binder and renderer, independently of the selection period.
func TestBehavioralExperienceTimePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	execBehavioralFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,7) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 3, 30, 10, 0, 0, 123456000, time.UTC)
	history, err := behavioral.ResolveHistory("registered-queryable-events", "1")
	if err != nil {
		t.Fatalf("expected recorded history resolved, got %v", err)
	}
	calendar, err := behavioral.ResolveCalendar("gregorian-monday/Europe/Rome", "1")
	if err != nil {
		t.Fatalf("expected fixed calendar resolved, got %v", err)
	}
	for _, test := range []struct {
		name       string
		period     behavioral.Scope
		start, end time.Time
	}{
		{"absolute microseconds", behavioral.Scope{Start: at.Add(-2 * time.Hour), End: at.Add(-time.Hour)}, at.Add(-2 * time.Hour), at.Add(-time.Hour)},
		{"lookback", behavioral.Scope{Elapsed: &behavioral.ElapsedPeriod{Older: 24 * time.Hour}}, at.Add(-24 * time.Hour), at},
		{"band", behavioral.Scope{Elapsed: &behavioral.ElapsedPeriod{Older: 72 * time.Hour, Newer: 48 * time.Hour}}, at.Add(-72 * time.Hour), at.Add(-48 * time.Hour)},
		{"Rome DST day", behavioral.Scope{Calendar: &behavioral.CalendarPeriod{Calendar: calendar, Unit: "day", Count: 1}}, time.Date(2026, 3, 28, 23, 0, 0, 0, time.UTC), time.Date(2026, 3, 29, 22, 0, 0, 0, time.UTC)},
	} {
		t.Run(test.name, func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			stamps := []time.Time{test.start.Add(-time.Microsecond), test.start, test.start.Add(time.Microsecond),
				test.end.Add(-time.Microsecond), test.end, test.end.Add(time.Microsecond)}
			for i, stamp := range stamps {
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','edge-%d','track','Order Completed','%s','{}')`, behavioralTestID(i+1), i,
					stamp.Format("2006-01-02 15:04:05.999999")))
			}
			for _, first := range []bool{false, true} {
				experience := &behavioral.Experience{First: first,
					Scope:  behavioral.Scope{Event: "Order Completed", Grain: "registered-message", History: history},
					Verify: &behavioral.OccurrencePredicate{OccurredIn: &test.period}}
				result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Experience: experience},
					behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
				if err != nil {
					t.Fatalf("expected exact temporal verification, got %v", err)
				}
				assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.Match, behavioral.Match,
					behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
			}
		})
	}

	for _, first := range []bool{false, true} {
		t.Run(fmt.Sprintf("uncertain first=%v", first), func(t *testing.T) {
			execBehavioralFixture(t, dw, "TRUNCATE krenalis_events")
			certain := at.Add(-96 * time.Hour)
			if first {
				certain = at.Add(-12 * time.Hour)
			}
			for _, event := range []struct {
				profile int
				hours   int
				payload string
			}{
				{1, 0, `{"q":"yes","v":"yes"}`},
				{1, 48, `{}`},
				{2, 48, `{}`},
				{3, 48, `{"q":"yes"}`},
				{3, 47, `{"q":"yes"}`},
			} {
				stamp := at.Add(-time.Duration(event.hours) * time.Hour)
				if event.hours == 0 {
					stamp = certain
				}
				execBehavioralFixture(t, dw, fmt.Sprintf(`INSERT INTO krenalis_events VALUES
('%s','p%d-%d','track','Order Completed','%s','%s')`, behavioralTestID(event.profile), event.profile, event.hours,
					stamp.Format("2006-01-02 15:04:05.999999"), event.payload))
			}
			period := behavioral.Scope{Elapsed: &behavioral.ElapsedPeriod{Older: 72 * time.Hour, Newer: 24 * time.Hour}}
			experience := &behavioral.Experience{First: first,
				Scope: behavioral.Scope{Event: "Order Completed", Grain: "registered-message", History: history},
				Where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", "q"}, Operator: state.OpIs,
						Values: []any{json.Value(`"yes"`)}},
				}}, Verify: &behavioral.OccurrencePredicate{OccurredIn: &period}}
			root := &behavioral.Rule{Experience: experience}
			for _, or := range []bool{false, true} {
				if or {
					experience.Verify = &behavioral.OccurrencePredicate{Operator: state.OpOr,
						Children: []behavioral.OccurrencePredicate{
							{OccurredIn: &period}, {Record: &state.WhereCondition{Property: []string{"properties", "v"},
								Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}}},
						}}
				}
				result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
				if err != nil {
					t.Fatalf("expected candidate-local time with OR=%v, got %v", or, err)
				}
				want := behavioral.Unknown
				if or {
					want = behavioral.Match
				}
				assertBehavioralTruths(t, result, []behavioral.Truth{want, behavioral.Unknown, behavioral.Match,
					behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
				explained, err := store.ExplainProfileFilter(t.Context(), root, behavioral.Bindings{}, at, behavioralTestID(1))
				if err != nil {
					t.Fatalf("expected temporal explanation, got %v", err)
				}
				if explained.Nodes[0].Truth != want || explained.Nodes[0].Experience.Candidates != 2 ||
					explained.Nodes[0].Experience.AbsencePossible {
					t.Fatalf("expected two candidates with certain nonemptiness and %v, got %+v", want, explained)
				}
			}
			experience.Verify = &behavioral.OccurrencePredicate{OccurredIn: &period}
			experience.Scope.History = nil
			experience.Scope.Elapsed = &behavioral.ElapsedPeriod{Older: 24 * time.Hour}
			result, err := store.QueryProfiles(t.Context(), root, behavioral.Bindings{}, at, behavioral.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected disjoint base and verification, got %v", err)
			}
			assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch,
				behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch})
		})
	}

}
