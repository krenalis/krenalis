// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

type nthOracleOccurrence struct {
	timestamp, qualification, verification int
}

// TestUnifiedExperienceNthOraclePostgreSQL enumerates membership subsets,
// chronological permutations and per-identity verification completions.
func TestUnifiedExperienceNthOraclePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	for _, times := range [][6]int{{1, 2, 3, 4, 5, 6}, {4, 1, 3, 3, 2, 4}, {1, 1, 1, 1, 1, 1}} {
		for _, bodies := range [][6]int{{1, 1, 1, 1, 1, 1}, {0, 0, 0, 0, 0, 0}, {0, 1, 2, 1, 0, 2}} {
			t.Run(fmt.Sprintf("times=%v/bodies=%v", times, bodies), func(t *testing.T) {

				execUnifiedFixture(t, dw, "TRUNCATE profiles,krenalis_events")
				var profiles, events []string
				histories := make([][]nthOracleOccurrence, 729)
				for code := range histories {
					id, n := unifiedTestID(code+1), code
					profiles = append(profiles, "('"+id+"')")
					for i, timestamp := range times {
						qualification := n % 3
						n /= 3
						histories[code] = append(histories[code], nthOracleOccurrence{timestamp, qualification, bodies[i]})
						payload := []string{}
						for j, value := range []int{qualification, bodies[i]} {
							if value < 2 {
								payload = append(payload, fmt.Sprintf(`"%s":"%s"`, []string{"q", "v"}[j],
									[]string{"no", "yes"}[value]))
							}
						}
						event := fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-0%d','{%s}')",
							id, code, i, timestamp, strings.Join(payload, ","))
						events = append(events, event, event) // Copy multiplicity cannot move the ordinal.
					}
				}
				execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
				execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

				predicate := func(key string) *state.WhereCondition {
					return &state.WhereCondition{Property: []string{"properties", key}, Operator: state.OpIs,
						Values: []any{json.Value(`"yes"`)}}
				}
				for position := 1; position <= 3; position++ {

					want := make([]unified.Truth, len(histories))
					selections := make([]warehouses.OccurrenceSelection, len(histories))
					for code, history := range histories {
						want[code], selections[code] = enumerateNth(history, position)
					}

					experience := &unified.Experience{Nth: position,
						Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
							Start: at.AddDate(0, -1, 0), End: at},
						Where:  &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{predicate("q")}},
						Verify: &unified.OccurrencePredicate{Record: predicate("v")}}
					root := &unified.Rule{Experience: experience}
					result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
					if err != nil {
						t.Fatalf("expected exact Nth(%d) enumeration, got %v", position, err)
					}
					assertUnifiedTruths(t, result, want)
					if position == 1 {
						experience.Nth, experience.First = 0, true
						first, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
						if err != nil {
							t.Fatalf("expected differential First result, got %v", err)
						}
						if !reflect.DeepEqual(first, result) {
							t.Fatal("expected Nth(1) and First identical outcomes, got different results")
						}
						experience.Nth, experience.First = 1, false
					}
					for _, code := range []int{0, 121, 364, 728, (position*223 + 17) % 729} {
						explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(code+1))
						if err != nil {
							t.Fatalf("expected Nth(%d) selection metadata, got %v", position, err)
						}
						if len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want[code] ||
							explained.Nodes[0].Experience == nil || *explained.Nodes[0].Experience != selections[code] {
							t.Fatalf("expected position %d selection %+v and truth %v for %d, got %+v",
								position, selections[code], want[code], code, explained)
						}
						if position == 1 {
							experience.Nth, experience.First = 0, true
							first, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(code+1))
							if err != nil {
								t.Fatalf("expected differential First explanation, got %v", err)
							}
							if first.At != explained.At || !reflect.DeepEqual(first.Nodes, explained.Nodes) {
								t.Fatal("expected Nth(1) and First identical selection metadata, got different results")
							}
							experience.Nth, experience.First = 1, false
						}
					}

				}

			})
		}
	}

}

// TestUnifiedExperienceNthPostgreSQL checks ranks and currency conditioning.
func TestUnifiedExperienceNthPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
INSERT INTO profiles VALUES ('`+unifiedTestID(1)+`'),('`+unifiedTestID(2)+`');
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	currency, err := unified.ResolveNativeCurrency("commerce.order_completed.currency", "1", "EUR", "Order Completed")
	if err != nil {
		t.Fatalf("expected resolved currency, got %v", err)
	}
	field, err := unified.ResolveMonetaryField("commerce.order_completed.total", "1")
	if err != nil {
		t.Fatalf("expected resolved amount, got %v", err)
	}
	for _, test := range []struct {
		name         string
		position     int
		certain      int
		optionalDays []int
		optionalBody string
		want         unified.Truth
		candidates   int64
		absent       bool
	}{
		{"eleventh does not cancel tenth", 10, 11, nil, "", unified.Match, 1, false},
		{"certain shortage", 10, 9, nil, "", unified.NoMatch, 0, true},
		{"possible shortage", 10, 9, []int{20}, "150", unified.Unknown, 1, true},
		{"two possible after nine", 10, 9, []int{20, 21}, "150", unified.Unknown, 2, true},
		{"possible false with shortage", 10, 9, []int{20}, "80", unified.NoMatch, 1, true},
		{"earlier optional", 10, 10, []int{1}, "80", unified.Match, 2, false},
		{"tied optional", 10, 10, []int{12}, "80", unified.Unknown, 2, false},
		{"tied optional passes", 10, 10, []int{12}, "150", unified.Match, 2, false},
		{"tied optional unreadable", 10, 10, []int{12}, "null", unified.Unknown, 2, false},
		{"optional before mixed tie", 2, 1, []int{1, 3}, "150", unified.Unknown, 2, true},
	} {
		t.Run(test.name, func(t *testing.T) {

			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			var events []string
			for i := range test.certain {
				amount := "150"
				if i == 10 {
					amount = "80"
				}
				events = append(events, fmt.Sprintf("('%s','c%d','track','Order Completed','2026-01-%02d',"+
					`'{"currency":"EUR","total":%s}')`, unifiedTestID(1), i, i+3, amount))
			}
			for _, day := range test.optionalDays {
				events = append(events, fmt.Sprintf("('%s','optional-%d','track','Order Completed','2026-01-%02d',"+
					`'{"total":%s}')`, unifiedTestID(1), day, day, test.optionalBody))
			}
			// Another profile has enough events; they must not fill this profile's shortage.
			for i := range 11 {
				events = append(events, fmt.Sprintf("('%s','other%d','track','Order Completed','2026-01-%02d',"+
					`'{"currency":"EUR","total":80}')`, unifiedTestID(2), i, i+1))
			}
			execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

			experience := &unified.Experience{Nth: test.position, Currency: currency,
				Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
				Verify: &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
					Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}}}
			root := &unified.Rule{Experience: experience}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected exact Nth(%d) occurrence, got %v", test.position, err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{test.want, unified.NoMatch})
			explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
			if err != nil {
				t.Fatalf("expected Nth(%d) selection explanation, got %v", test.position, err)
			}
			selection := explained.Nodes[0].Experience
			if selection.Candidates != test.candidates || selection.AbsencePossible != test.absent {
				t.Fatalf("expected %d candidates, absence=%v, got %+v", test.candidates, test.absent, selection)
			}
			experience.Verify.Money.NativeCurrency = "USD"
			result, err = store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected conditioned different-currency FALSE, got %v", err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch})

		})
	}

	t.Run("source faults before shortage", func(t *testing.T) {
		for _, test := range []struct {
			name, rows string
			fault      bool
		}{
			{"timestamps", `('same','2026-01-10','{"currency":"EUR","total":150}'),('same','2026-01-11','{"currency":"EUR","total":150}')`, true},
			{"possible timestamps", `('same','2026-01-10','{"total":150}'),('same','2026-01-11','{"total":150}')`, true},
			{"qualification", `('same','2026-01-10','{"currency":"EUR","total":150}'),('same','2026-01-10','{"total":150}')`, true},
			{"verification", `('same','2026-01-10','{"currency":"EUR","total":150}'),('same','2026-01-10','{"currency":"EUR","total":80}')`, true},
			{"missing timestamp", `('same',NULL,'{"currency":"EUR","total":150}')`, true},
			{"missing identity", `(NULL,'2026-01-10','{"currency":"EUR","total":150}')`, true},
			{"excluded timestamps", `('same','2026-01-10','{"currency":"USD"}'),('same','2026-01-11','{"currency":"GBP"}')`, false},
		} {
			t.Run(test.name, func(t *testing.T) {
				execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
				execUnifiedFixture(t, dw, `INSERT INTO krenalis_events
SELECT '`+unifiedTestID(1)+`',id,'track','Order Completed',stamp::timestamp,payload::jsonb
FROM (VALUES `+test.rows+`) v(id,stamp,payload)`)
				experience := &unified.Experience{Nth: 10, Currency: currency,
					Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
					Verify: &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
						Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}}}
				root := &unified.Rule{Experience: experience}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected source fault=%v without partial result, got %+v, %v", test.fault, result, err)
					}
				}
				if err == nil {
					if test.fault {
						t.Fatal("expected source fault before insufficient sample, got success")
					}
					assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch})
				}
				explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
				if err != nil {
					if !test.fault || explained != nil || !strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected explanation fault=%v without partial result, got %+v, %v", test.fault, explained, err)
					}
				}
				if err == nil && test.fault {
					t.Fatal("expected source fault before explanation, got success")
				}
				other, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(2))
				if err != nil {
					t.Fatalf("expected unrelated profile unaffected by faults, got %v", err)
				}
				if other.Nodes[0].Truth != unified.NoMatch {
					t.Fatalf("expected NO_MATCH for unrelated profile, got %+v", other.Nodes)
				}
			})
		}
	})

}

// enumerateNth deliberately uses no rank bounds or certain/possible cutoffs.
func enumerateNth(history []nthOracleOccurrence, position int) (unified.Truth, warehouses.OccurrenceSelection) {

	candidates := make([]bool, len(history))
	absent := false
	for mask := range 1 << len(history) {
		valid := true
		var present []int
		for i, row := range history {
			included := mask&(1<<i) != 0
			if row.qualification < 2 && included != (row.qualification == 1) {
				valid = false
			}
			if included {
				present = append(present, i)
			}
		}
		if !valid {
			continue
		}
		var permute func(int)
		permute = func(index int) {
			if index == len(present) {
				if len(present) < position {
					absent = true
					return
				}
				candidates[present[position-1]] = true
				return
			}
			for i := index; i < len(present); i++ {
				if index > 0 && history[present[i]].timestamp < history[present[index-1]].timestamp {
					continue
				}
				present[index], present[i] = present[i], present[index]
				permute(index + 1)
				present[index], present[i] = present[i], present[index]
			}
		}
		permute(0)
	}

	seen := [2]bool{absent, false}
	for mask := range 1 << len(history) {
		valid := true
		for i, row := range history {
			if row.verification < 2 && (mask&(1<<i) != 0) != (row.verification == 1) {
				valid = false
			}
		}
		if !valid {
			continue
		}
		for i, candidate := range candidates {
			if candidate {
				value := 0
				if mask&(1<<i) != 0 {
					value = 1
				}
				seen[value] = true
			}
		}
	}

	selection := warehouses.OccurrenceSelection{AbsencePossible: absent}
	for _, candidate := range candidates {
		if candidate {
			selection.Candidates++
		}
	}
	truth := unified.Unknown
	if !seen[1] {
		truth = unified.NoMatch
	} else if !seen[0] {
		truth = unified.Match
	}

	return truth, selection
}
