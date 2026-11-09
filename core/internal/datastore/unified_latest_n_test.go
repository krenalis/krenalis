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

// TestUnifiedLatestNOraclePostgreSQL enumerates subsets, tie permutations and
// local body completions, never the compiler's cumulative-count formula.
func TestUnifiedLatestNOraclePostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	execUnifiedFixture(t, dw, `CREATE TABLE profiles (_kpid uuid PRIMARY KEY);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp,properties jsonb)`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	predicate := func(key string) *state.WhereCondition {
		return &state.WhereCondition{Property: []string{"properties", key}, Operator: state.OpIs,
			Values: []any{json.Value(`"yes"`)}}
	}

	for ti, times := range [][6]int{{1, 2, 3, 4, 5, 6}, {4, 1, 3, 3, 2, 4}, {1, 1, 1, 1, 1, 1}} {
		for bi, bodies := range [][6]int{{1, 1, 1, 1, 1, 1}, {0, 0, 0, 0, 0, 0}, {0, 1, 2, 1, 0, 2}} {
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
						var payload []string
						for j, value := range []int{qualification, bodies[i]} {
							if value < 2 {
								payload = append(payload, fmt.Sprintf(`"%s":"%s"`, []string{"q", "v"}[j],
									[]string{"no", "yes"}[value]))
							}
						}
						event := fmt.Sprintf("('%s','%d/%d','track','Order Completed','2026-01-0%d','{%s}')",
							id, code, i, timestamp, strings.Join(payload, ","))
						events = append(events, event, event)
					}
				}
				execUnifiedFixture(t, dw, "INSERT INTO profiles VALUES "+strings.Join(profiles, ","))
				execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

				for size := 1; size <= 4; size++ {

					latest := &unified.LatestN{Size: size,
						Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message",
							Start: at.AddDate(0, -1, 0), End: at},
						Where:     &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{predicate("q")}},
						Predicate: &unified.OccurrencePredicate{Record: predicate("v")}}
					root := &unified.Rule{LatestN: latest}
					want := make([]unified.Truth, len(histories))
					selections := make([]warehouses.OccurrenceSelection, len(histories))
					for code, history := range histories {
						want[code], selections[code] = enumerateLatestAll(history, size)
					}
					result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 1000})
					if err != nil {
						t.Fatalf("expected exact Latest(%d)/All enumeration, got %v", size, err)
					}
					assertUnifiedTruths(t, result, want)
					if size == 1 || bi == 0 {
						var equivalent unified.Rule
						if size == 1 {
							equivalent.Experience = &unified.Experience{Scope: latest.Scope, Where: latest.Where,
								Verify: latest.Predicate}
						} else {
							where, err := unified.RecordPredicate(latest.Where)
							if err != nil {
								t.Fatalf("expected Record predicate, got %v", err)
							}
							equivalent.Count = &unified.Count{Scope: latest.Scope, Where: where,
								Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: uint64(size)}}
						}
						other, err := store.UnifiedProfiles(t.Context(), &equivalent, unified.Bindings{}, at,
							unified.Page{Limit: 1000})
						if err != nil {
							t.Fatalf("expected differential Last/Count result, got %v", err)
						}
						if !reflect.DeepEqual(other, result) {
							t.Fatal("expected equivalent Last/Count outcomes, got different results")
						}
					}

					codes := []int{0, 121, 364, 728, (size*223 + 17) % 729}
					if ti == 1 && bi == 2 && size == 2 {
						codes = make([]int, len(histories))
						for code := range codes {
							codes[code] = code
						}
					}
					for _, code := range codes {
						explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(code+1))
						if err != nil {
							t.Fatalf("expected Latest(%d) explanation, got %v", size, err)
						}
						if len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want[code] ||
							explained.Nodes[0].Experience == nil || *explained.Nodes[0].Experience != selections[code] {
							t.Fatalf("expected size %d selection %+v and truth %v for %d, got %+v",
								size, selections[code], want[code], code, explained)
						}
						if size == 1 {
							last, err := store.ExplainUnifiedProfile(t.Context(), &unified.Rule{
								Experience: &unified.Experience{Scope: latest.Scope, Where: latest.Where, Verify: latest.Predicate}},
								unified.Bindings{}, at, unifiedTestID(code+1))
							if err != nil {
								t.Fatalf("expected differential Last explanation, got %v", err)
							}
							if last.At != explained.At || !reflect.DeepEqual(last.Nodes, explained.Nodes) {
								t.Fatal("expected Latest(1)/Last identical selection metadata, got different results")
							}
						}
					}

				}

			})
		}
	}

}

// TestUnifiedLatestNPostgreSQL discriminates compatible blocks and copy faults.
func TestUnifiedLatestNPostgreSQL(t *testing.T) {

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
	tests := []struct {
		name string
		size int
		rows []nthOracleOccurrence
	}{
		{"incompatible passing positions", 2, []nthOracleOccurrence{{3, 2, 1}, {2, 1, 0}, {1, 1, 1}}},
		{"ambiguous tied blocks", 2, []nthOracleOccurrence{{1, 1, 1}, {1, 1, 0}, {1, 1, 1}}},
		{"unique tied set", 3, []nthOracleOccurrence{{1, 1, 1}, {1, 1, 1}, {1, 1, 1}}},
		{"short base", 3, []nthOracleOccurrence{{1, 1, 1}, {2, 1, 1}}},
		{"possible absence", 3, []nthOracleOccurrence{{1, 1, 1}, {2, 1, 1}, {3, 2, 1}}},
		{"same currency conditioning", 2, []nthOracleOccurrence{{1, 1, 1}, {2, 1, 1}, {3, 2, 1}}},
		{"unknown amount", 2, []nthOracleOccurrence{{1, 1, 1}, {2, 1, 1}, {3, 2, 2}}},
		{"possible unknown boundary", 2, []nthOracleOccurrence{{1, 1, 1}, {2, 2, 2}}},
	}
	for mode := range 6 {
		rows := make([]nthOracleOccurrence, 11)
		for i := range rows {
			rows[i] = nthOracleOccurrence{i + 1, 1, 1}
		}
		switch mode {
		case 0:
			rows[0].verification = 0 // The eleventh, oldest event is outside the block.
		case 1:
			rows[10].qualification, rows[10].verification = 2, 0
		case 2:
			rows[10].qualification, rows[9].qualification = 2, 2
		case 3:
			rows[0].timestamp, rows[1].timestamp, rows[1].verification = 2, 2, 0
		case 4:
			rows[0].qualification, rows[0].timestamp = 2, 11
		case 5:
			rows[10].verification = 2
		default:
			t.Fatalf("expected six boundary fixtures, got %d", mode)
		}
		tests = append(tests, struct {
			name string
			size int
			rows []nthOracleOccurrence
		}{fmt.Sprintf("N10 boundary %d", mode), 10, rows})
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			var events []string
			for i, row := range test.rows {
				payload := `"total":` + []string{"80", "150", "null"}[row.verification]
				if row.qualification != 2 {
					payload += `,"currency":"` + []string{"USD", "EUR"}[row.qualification] + `"`
				}
				event := fmt.Sprintf("('%s','r%d','track','Order Completed','2026-01-%02d','{%s}')",
					unifiedTestID(1), i, row.timestamp, payload)
				events = append(events, event, event)
			}
			// Enough messages of another profile must not fill this profile's base.
			events = append(events, fmt.Sprintf("('%s','other','track','Order Completed','2026-01-20',"+
				`'{"currency":"EUR","total":150}')`, unifiedTestID(2)))
			execUnifiedFixture(t, dw, "INSERT INTO krenalis_events VALUES "+strings.Join(events, ","))

			latest := &unified.LatestN{Size: test.size, Currency: currency,
				Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
				Predicate: &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
					Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}}}
			root := &unified.Rule{LatestN: latest}
			want, selection := enumerateLatestAll(test.rows, test.size)
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected exact compatible blocks, got %v", err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{want, unified.NoMatch})
			explained, err := store.ExplainUnifiedProfile(t.Context(), root, unified.Bindings{}, at, unifiedTestID(1))
			if err != nil {
				t.Fatalf("expected selection explanation, got %v", err)
			}
			if *explained.Nodes[0].Experience != selection {
				t.Fatalf("expected selection %+v, got %+v", selection, explained.Nodes[0].Experience)
			}

			latest.Predicate.Money.NativeCurrency = "USD"
			result, err = store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected conditioned different-currency FALSE, got %v", err)
			}
			assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch})

			latest.Predicate = &unified.OccurrencePredicate{Operator: state.OpOr,
				Children: []unified.OccurrencePredicate{*latest.Predicate, {Record: &state.WhereCondition{
					Property: []string{"timestamp"}, Operator: state.OpExists}}}}
			result, err = store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
			if err != nil {
				t.Fatalf("expected conditional FALSE under OR, got %v", err)
			}
			passing := append([]nthOracleOccurrence(nil), test.rows...)
			for i := range passing {
				passing[i].verification = 1
			}
			always, _ := enumerateLatestAll(passing, test.size)
			assertUnifiedTruths(t, result, []unified.Truth{always, unified.NoMatch})

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
				root := &unified.Rule{LatestN: &unified.LatestN{Size: 10, Currency: currency,
					Scope: unified.Scope{Event: "Order Completed", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
					Predicate: &unified.OccurrencePredicate{Money: &unified.Money{Field: field, NativeCurrency: "EUR",
						Test: unified.SumTest{Operator: unified.Greater, Value: decimal.MustParse("100")}}}}}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					if !test.fault || result != nil || !strings.Contains(err.Error(), "occurrence source violates") {
						t.Fatalf("expected source fault=%v without partial result, got %+v, %v", test.fault, result, err)
					}
				}
				if err == nil {
					if test.fault {
						t.Fatal("expected source fault before insufficient base, got success")
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

	t.Run("page and screen sources", func(t *testing.T) {
		execUnifiedFixture(t, dw, "ALTER TABLE krenalis_events ADD COLUMN name text")
		for _, typ := range []string{"page", "screen"} {
			execUnifiedFixture(t, dw, "TRUNCATE krenalis_events")
			execUnifiedFixture(t, dw, `INSERT INTO krenalis_events (kpid,message_id,type,name,timestamp,properties)
SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid,id,'`+typ+`',name,stamp::timestamp,payload::jsonb
FROM (VALUES
(1,'named-a','Pricing','2026-01-01','{"v":"yes"}'),
(1,'named-b','Pricing','2026-01-02','{"v":"yes"}'),
(1,'anonymous',NULL,'2026-01-03','{"v":"no"}'),
(2,'empty','','2026-01-01','{"v":"yes"}'),
(2,'lowercase','pricing','2026-01-02','{"v":"yes"}')
) v(n,id,name,stamp,payload)`)
			for _, name := range []string{"Pricing", ""} {
				root := &unified.Rule{LatestN: &unified.LatestN{Size: 2,
					Scope: unified.Scope{Type: typ, Event: name, Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at},
					Predicate: &unified.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"properties", "v"},
						Operator: state.OpIs, Values: []any{json.Value(`"yes"`)}}}}}
				result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, at, unified.Page{Limit: 10})
				if err != nil {
					t.Fatalf("expected %s source %q, got %v", typ, name, err)
				}
				want := []unified.Truth{unified.Match, unified.NoMatch}
				if name == "" {
					want = []unified.Truth{unified.NoMatch, unified.Match}
				}
				assertUnifiedTruths(t, result, want)
			}
		}
	})

}

// enumerateLatestAll independently enumerates actual chronological selections.
func enumerateLatestAll(history []nthOracleOccurrence, size int) (unified.Truth, warehouses.OccurrenceSelection) {

	blocks := map[int]bool{}
	selection := warehouses.OccurrenceSelection{}
	union := 0
	for mask := range 1 << len(history) {
		var present []int
		valid := true
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
		if len(present) < size {
			selection.AbsencePossible = true
			continue
		}
		var permute func(int)
		permute = func(index int) {
			if index == len(present) {
				block := 0
				for _, i := range present[len(present)-size:] {
					block |= 1 << i
				}
				blocks[block] = true
				union |= block
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

	seen := [2]bool{selection.AbsencePossible, false}
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
		for block := range blocks {
			value := 0
			if mask&block == block {
				value = 1
			}
			seen[value] = true
		}
	}
	for i := range history {
		if union&(1<<i) != 0 {
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
