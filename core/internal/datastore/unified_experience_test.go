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
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// TestUnifiedExperiencePostgreSQL checks exact selection on a coherent view.
func TestUnifiedExperiencePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,3) n;
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,name text,
 timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	qualify := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"properties", "q"},
			Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
	}}
	verify := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
		&state.WhereCondition{Property: []string{"properties", "v"},
			Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
	}}
	experience := unified.Experience{Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Where: qualify, Verify: verify}
	root := &unified.Rule{Experience: &experience}

	for _, test := range []struct {
		name, rows string
		truth      unified.Truth
		candidates int64
		absent     bool
		fault      bool
	}{
		{"empty", ``, unified.NoMatch, 0, true, false},
		{"latest true", `('a','2026-01-01','{"q":"yes","v":"no"}'),('b','2026-01-02','{"q":"yes","v":"yes"}')`, unified.Match, 1, false, false},
		{"no fallback", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"q":"yes","v":"no"}')`, unified.NoMatch, 1, false, false},
		{"latest unknown", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"q":"yes"}')`, unified.Unknown, 1, false, false},
		{"JSON null verification", `('a','2026-01-01','{"q":"yes","v":null}')`, unified.NoMatch, 1, false, false},
		{"different JSON kind", `('a','2026-01-01','{"q":"yes","v":100}')`, unified.NoMatch, 1, false, false},
		{"optional latest same body", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"v":"yes"}')`, unified.Match, 2, false, false},
		{"optional latest different body", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-02','{"v":"no"}')`, unified.Unknown, 2, false, false},
		{"older optional irrelevant", `('a','2026-01-01','{"v":"no"}'),('b','2026-01-02','{"q":"yes","v":"yes"}')`, unified.Match, 1, false, false},
		{"only possible true", `('a','2026-01-01','{"v":"yes"}')`, unified.Unknown, 1, true, false},
		{"only possible false", `('a','2026-01-01','{"v":"no"}')`, unified.NoMatch, 1, true, false},
		{"excluded unknown body", `('a','2026-01-01','{"q":"no"}')`, unified.NoMatch, 0, true, false},
		{"tie true", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes","v":"yes"}')`, unified.Match, 2, false, false},
		{"tie false", `('a','2026-01-01','{"q":"yes","v":"no"}'),('b','2026-01-01','{"q":"yes","v":"no"}')`, unified.NoMatch, 2, false, false},
		{"tie mixed", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes","v":"no"}')`, unified.Unknown, 2, false, false},
		{"tie unknown", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"q":"yes"}')`, unified.Unknown, 2, false, false},
		{"optional tie", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-01-01','{"v":"no"}')`, unified.Unknown, 2, false, false},
		{"same copies", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"q":"yes","v":"yes"}')`, unified.Match, 1, false, false},
		{"exact timestamp conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01 00:00:00.000001','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"possible timestamp conflict", `('a','2026-01-01','{"v":"yes"}'),('a','2026-01-02','{"v":"yes"}')`, 0, 0, false, true},
		{"excluded timestamp conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-01-02','{"q":"no"}')`, unified.NoMatch, 0, true, false},
		{"mixed qualification and timestamp conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-01-02','{"q":"yes"}')`, 0, 0, false, true},
		{"period conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-02-01','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"qualification conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"v":"yes"}')`, 0, 0, false, true},
		{"verify conflict", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('a','2026-01-01','{"q":"yes"}')`, 0, 0, false, true},
		{"irrelevant verify copies", `('a','2026-01-01','{"q":"no","v":"yes"}'),('a','2026-01-01','{"q":"no"}')`, unified.NoMatch, 0, true, false},
		{"excluded period conflict", `('a','2026-01-01','{"q":"no"}'),('a','2026-02-01','{"q":"no"}')`, 0, 0, false, true},
		{"start included end excluded", `('a','2026-01-01','{"q":"yes","v":"yes"}'),('b','2026-02-01','{"q":"yes","v":"no"}')`, unified.Match, 1, false, false},
		{"microseconds order", `('a','2026-01-01 00:00:00.000002','{"q":"yes","v":"yes"}'),('b','2026-01-01 00:00:00.000001','{"q":"yes","v":"no"}')`, unified.Match, 1, false, false},
		{"missing timestamp", `('a',NULL,'{"q":"yes","v":"yes"}')`, 0, 0, false, true},
		{"missing identity", `(NULL,'2026-01-01','{"q":"yes","v":"yes"}')`, 0, 0, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			if test.rows != "" {
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events
SELECT '`+unifiedTestID(1)+`',id,'track','Order Completed',NULL,stamp::timestamp,payload::jsonb
FROM (VALUES `+test.rows+`) v(id,stamp,payload)`)
			}
			// A later event of another profile must not suppress this profile's candidates.
			execUnifiedFixture(t, dw, `INSERT INTO krenalis_events VALUES
('`+unifiedTestID(2)+`','other-profile','track','Order Completed',NULL,'2026-01-31','{"q":"yes","v":"no"}')`)
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
					t.Fatalf("expected fault=%v without partial result, got %+v, %v", test.fault, result, err)
				}
			}
			if err == nil {
				if test.fault {
					t.Fatal("expected source fault, got success")
				}
				assertUnifiedTruths(t, result, []unified.Truth{test.truth, unified.NoMatch, unified.NoMatch})
				empty, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at,
					unified.Page{First: 99, Limit: 1, MatchingOnly: true})
				if err != nil {
					t.Fatalf("expected complete empty page, got %v", err)
				}
				if len(empty.Profiles) != 0 || empty.Match != result.Match || empty.NoMatch != result.NoMatch ||
					empty.Unknown != result.Unknown {
					t.Fatalf("expected identical totals on empty page, got %+v", empty)
				}
			}
			explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
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
		execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','broken','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}'),
('`+unifiedTestID(1)+`','broken','track','Order Completed',NULL,'2026-01-02','{"q":"yes","v":"yes"}')`)
		explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(2))
		if err != nil {
			t.Fatalf("expected complete unrelated-profile explanation, got %v", err)
		}
		if explained.Nodes[0].Truth != unified.NoMatch {
			t.Fatalf("expected NO_MATCH, got %+v", explained)
		}
	})

	t.Run("unrelated source timestamp conflict", func(t *testing.T) {
		execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(2)+`','other','track','Product Viewed',NULL,'2026-01-01','{}'),
('`+unifiedTestID(2)+`','other','track','Product Viewed',NULL,'2026-01-02','{}')`)
		result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected irrelevant timestamp conflict ignored, got %v", err)
		}
		assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch, unified.NoMatch})
	})

	t.Run("attribution conflicts", func(t *testing.T) {
		for _, other := range []string{"NULL", "'" + unifiedTestID(2) + "'"} {
			execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','same','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}'),
(`+other+`,'same','track','Order Completed',NULL,'2026-01-01','{"q":"yes","v":"yes"}')`)
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
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
		execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','early','track','Order Completed',NULL,'2026-01-10','{"q":"yes","v":"yes"}'),
('`+unifiedTestID(1)+`','later','track','Order Completed',NULL,'2026-01-20','{"q":"yes","v":"no"}')`)
		bounded := experience
		bounded.Scope.End = time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
		result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: &bounded}, unified.Bindings{}, at,
			unified.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected Last within the selected period, got %v", err)
		}
		assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch})
		history, err := unified.ResolveHistory("registered-queryable-events", "1")
		if err != nil {
			t.Fatalf("expected recorded-history reference, got %v", err)
		}
		bounded.Scope.Start, bounded.Scope.End, bounded.Scope.History = time.Time{}, time.Time{}, history
		result, err = store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: &bounded}, unified.Bindings{}, at,
			unified.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected Last over full recorded history, got %v", err)
		}
		assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch, unified.NoMatch})
	})

	t.Run("page and screen", func(t *testing.T) {
		for _, typ := range []string{"page", "screen"} {
			execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','named','`+typ+`',NULL,'Pricing','2026-01-10','{"q":"yes","v":"yes"}'),
('`+unifiedTestID(1)+`','unnamed','`+typ+`',NULL,NULL,'2026-01-20','{"q":"yes","v":"no"}'),
('`+unifiedTestID(1)+`','track','track','Pricing',NULL,'2026-01-30','{"q":"yes","v":"no"}')`)
			selected := experience
			selected.Scope.Type, selected.Scope.Event = typ, "Pricing"
			result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: &selected}, unified.Bindings{}, at,
				unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected named %s selection, got %v", typ, err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch})
			selected.Scope.Event = ""
			result, err = store.UnifiedProfiles(t.Context(), &unified.Rule{Experience: &selected}, unified.Bindings{}, at,
				unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected all %s messages, got %v", typ, err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch, unified.NoMatch})
		}
	})

	t.Run("mixed monetary explanations", func(t *testing.T) {
		execUnifiedFixture(t, dw, `TRUNCATE krenalis_events;
INSERT INTO krenalis_events VALUES
('`+unifiedTestID(1)+`','one','track','Order Completed',NULL,'2026-01-10',
'{"q":"yes","v":"yes","total":120,"currency":"EUR"}'),
('`+unifiedTestID(2)+`','two','track','Order Completed',NULL,'2026-01-20',
'{"q":"yes","v":"no","total":100,"currency":"EUR"}')`)
		field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
		if err != nil {
			t.Fatalf("expected resolved monetary field, got %v", err)
		}
		sum := unified.Sum{Scope: experience.Scope, Field: field, NativeCurrency: "EUR",
			Test: unified.SumTest{Operator: unified.GreaterOrEqual, Value: decimal.MustParse("100")}}
		average := sum
		average.Minimum = 1
		mixed := &unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{
			{Experience: &experience}, {Sum: &sum}, {Average: &unified.Average{Sum: average}},
		}}
		result, err := store.UnifiedProfiles(t.Context(), mixed, unified.Bindings{}, at, unified.Page{Limit: 10})
		if err != nil {
			t.Fatalf("expected mixed listing, got %v", err)
		}
		assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch})
		for n := 1; n <= 3; n++ {
			explained, err := store.ExplainUnifiedProfile(t.Context(), mixed, unified.Bindings{}, at, unifiedTestID(n))
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

// TestUnifiedExperienceOraclePostgreSQL enumerates independent completions,
// including every tied winner, instead of using the compiler's cutoff rule.
func TestUnifiedExperienceOraclePostgreSQL(t *testing.T) {
	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, times := range [][3]int{{1, 2, 3}, {1, 2, 2}, {1, 1, 1}, {3, 2, 1}} {
		t.Run(fmt.Sprint(times), func(t *testing.T) {
			execUnifiedFixture(t, dw, "TRUNCATE profiles,krenalis_events")
			var profiles, events []string
			want := make([]unified.Truth, 729)
			for code := range want {
				id := unifiedTestID(code + 1)
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
				// Every membership subset and every admissible last witness.
				for mask := range 8 {
					valid, last := true, 0
					for i, q := range qualifications {
						present := mask&(1<<i) != 0
						if q < 2 && present != (q == 1) {
							valid = false
						}
						if present && times[i] > last {
							last = times[i]
						}
					}
					if !valid {
						continue
					}
					if last == 0 {
						seen[0] = true
					}
					for i, body := range bodies {
						if mask&(1<<i) == 0 || times[i] != last {
							continue
						}
						seen[0] = seen[0] || body != 1
						seen[1] = seen[1] || body != 0
					}
				}
				want[code] = unified.Unknown
				if !seen[0] {
					want[code] = unified.Match
				} else if !seen[1] {
					want[code] = unified.NoMatch
				}
			}
			execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
			execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))
			predicate := func(key string) *state.Where {
				return &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"properties", key},
						Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}},
				}}
			}
			root := &unified.Rule{Experience: &unified.Experience{Scope: unified.Scope{Event: "Order Completed",
				Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}, Where: predicate("q"), Verify: predicate("v")}}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
			if err != nil {
				t.Fatalf("expected exact enumeration result, got %v", err)
			}
			assertUnifiedTruths(t, result, want)
		})
	}
}
