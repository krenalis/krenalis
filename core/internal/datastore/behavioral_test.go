// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralProfilesPostgreSQL executes the Count subset on a real
// warehouse.
func TestBehavioralProfilesPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	// Publish coherent fixture views before any filtering request.
	// Its identity, time and profile attribution are determined; no writer or
	// Identity Resolution operation runs concurrently with these tests.
	execBehavioralFixture(t, dw, `CREATE TABLE profile_fixture (_kpid uuid PRIMARY KEY);
CREATE VIEW profiles AS SELECT * FROM profile_fixture;
CREATE TABLE event_fixture (kpid uuid NOT NULL, message_id text PRIMARY KEY,
 event text, timestamp timestamp(3) NOT NULL, properties jsonb,type text NOT NULL DEFAULT 'track');
CREATE VIEW krenalis_events AS SELECT * FROM event_fixture;
INSERT INTO profile_fixture SELECT ('00000000-0000-0000-0000-'||lpad(n::text,12,'0'))::uuid FROM generate_series(1,8) n;
INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000002','p2','Purchase','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000003','p3','Purchase','2026-01-15','{"currency":null}'),
('00000000-0000-0000-0000-000000000004','p4','Purchase','2026-01-15','{}'),
('00000000-0000-0000-0000-000000000005','p5','Purchase','2026-01-15','{"currency":"USD"}'),
('00000000-0000-0000-0000-000000000006','p6a','Purchase','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000006','p6b','Purchase','2026-01-16','{"currency":null}'),
('00000000-0000-0000-0000-000000000007','p7a','Purchase','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000007','p7b','Purchase','2026-01-15','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000008','p8a','Purchase','2025-12-31 23:59:59.999','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000008','p8b','Purchase','2026-02-01','{"currency":"EUR"}'),
('00000000-0000-0000-0000-000000000008','p8c','Refund','2026-01-15','{"currency":"EUR"}');`)
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := behavioral.Scope{Event: "Purchase", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}
	bindings := behavioral.Bindings{Currencies: []behavioral.CurrencyBinding{{
		Definition: "NativeCurrency", Revision: "registered-v1", Path: []string{"properties", "currency"},
		Evidence: behavioral.IndependentOccurrenceValues,
	}}}
	governed := behavioral.Rule{Count: &behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal},
		Currency: &behavioral.NativeCurrency{Definition: "NativeCurrency", Revision: "registered-v1", Value: "EUR"}}}
	raw := governed
	raw.Count = &behavioral.Count{Scope: scope, Test: governed.Count.Test,
		Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpIs,
				Values: []any{json.Value(`"EUR"`)}},
		}})}
	page := behavioral.Page{Limit: 1000}
	query := func(t *testing.T, rule *behavioral.Rule, p behavioral.Page) *behavioral.Result {
		t.Helper()
		result, err := store.QueryProfiles(t.Context(), rule, bindings, at, p)
		if err != nil {
			t.Fatalf("expected complete Unified query, got %v", err)
		}
		return result
	}
	t.Run("all profiles", func(t *testing.T) {
		assertBehavioralTruths(t, query(t, nil, page), []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Match,
			behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match})
	})

	t.Run("governed absence", func(t *testing.T) {
		result := query(t, &governed, page)
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Unknown,
			behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.Match})
	})

	t.Run("raw null and missing", func(t *testing.T) {
		result := query(t, &raw, page)
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Match,
			behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.Match})
	})

	t.Run("non-track events", func(t *testing.T) {

		execBehavioralFixture(t, dw, `INSERT INTO event_fixture (kpid,message_id,event,timestamp,properties,type)
SELECT _kpid,'non-track/'||event_type||'/'||_kpid,NULL,'2026-01-15',NULL,event_type
FROM profile_fixture CROSS JOIN (VALUES ('identify'),('page'),('screen'),('group'),('alias')) types(event_type);
INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000002','review/2','Review','2026-01-15','{}'),
('00000000-0000-0000-0000-000000000008','review/8','Review','2026-01-15','{}')`)
		absence := behavioral.Rule{Count: &behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal}}}
		purchase := behavioral.Rule{Count: &behavioral.Count{Scope: scope,
			Test: behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1}}}
		reviewScope := scope
		reviewScope.Event = "Review"
		review := behavioral.Rule{Count: &behavioral.Count{Scope: reviewScope,
			Test: behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1}}}

		for _, test := range []struct {
			name string
			rule behavioral.Rule
			want []behavioral.Truth
		}{
			{"absence", absence, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.Match}},
			{"existence", purchase, []behavioral.Truth{behavioral.NoMatch, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.NoMatch}},
			{"raw qualification", raw, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Match, behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.Match}},
			{"governed qualification", governed, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Unknown, behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.Match}},
			{"order and review", behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{purchase, review}}, []behavioral.Truth{behavioral.NoMatch, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}},
			{"order or review", behavioral.Rule{Operator: state.OpOr, Children: []behavioral.Rule{purchase, review}}, []behavioral.Truth{behavioral.NoMatch, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match, behavioral.Match}},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertBehavioralTruths(t, query(t, &test.rule, page), test.want)
			})
		}
		execBehavioralFixture(t, dw, `DELETE FROM event_fixture
WHERE message_id LIKE 'non-track/%' OR message_id LIKE 'review/%'`)

	})

	t.Run("unknown event name retains possible evidence", func(t *testing.T) {
		execBehavioralFixture(t, dw,
			`INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000001','unknown-name',NULL,'2026-01-15','{"currency":"EUR"}')`)
		result := query(t, &governed, page)
		if result.Profiles[0].Truth != behavioral.Unknown {
			t.Fatalf("expected unknown named-event membership, got %+v", result.Profiles[0])
		}
		execBehavioralFixture(t, dw, `DELETE FROM event_fixture WHERE message_id='unknown-name'`)
	})

	t.Run("membership page and empty page totals", func(t *testing.T) {
		all := query(t, &governed, page)
		result := query(t, &governed, behavioral.Page{Limit: 1, First: 1, MatchingOnly: true})
		if len(result.Profiles) != 1 || result.Profiles[0].ID != behavioralTestID(5) ||
			result.Profiles[0].Truth != behavioral.Match {
			t.Fatalf("expected second matching profile 5, got %+v", result.Profiles)
		}
		empty := query(t, &governed, behavioral.Page{First: 100, Limit: 2, MatchingOnly: true})
		if len(empty.Profiles) != 0 || empty.Match != all.Match || empty.NoMatch != all.NoMatch ||
			empty.Unknown != all.Unknown {
			t.Fatalf("expected empty page with complete totals %+v, got %+v", all, empty)
		}
	})

	t.Run("outer Kleene", func(t *testing.T) {
		exists := *governed.Count
		exists.Test = behavioral.CountTest{Operator: behavioral.GreaterOrEqual, Value: 1}
		for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
			root := behavioral.Rule{Operator: op, Children: []behavioral.Rule{governed, {Count: &exists}}}
			want := []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.Unknown, behavioral.Unknown,
				behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}
			if op == state.OpOr {
				want = []behavioral.Truth{behavioral.Match, behavioral.Match, behavioral.Unknown, behavioral.Unknown, behavioral.Match,
					behavioral.Match, behavioral.Match, behavioral.Match}
			}
			assertBehavioralTruths(t, query(t, &root, page), want)
		}
	})

	t.Run("Kleene decisive operands", func(t *testing.T) {
		base := query(t, &governed, page)
		for _, constant := range []bool{false, true} {
			terminal := behavioral.CountTest{Operator: behavioral.Less}
			if constant {
				terminal.Operator = behavioral.GreaterOrEqual
			}
			for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
				root := behavioral.Rule{Operator: op, Children: []behavioral.Rule{governed,
					{Count: &behavioral.Count{Scope: scope, Test: terminal}}}}
				want := make([]behavioral.Truth, len(base.Profiles))
				for i, profile := range base.Profiles {
					want[i] = profile.Truth
					if !constant && op == state.OpAnd {
						want[i] = behavioral.NoMatch
					}
					if constant && op == state.OpOr {
						want[i] = behavioral.Match
					}
				}
				assertBehavioralTruths(t, query(t, &root, page), want)
			}
		}
	})

	t.Run("identity grain and duplicate joins", func(t *testing.T) {
		count := behavioral.Rule{Count: &behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal, Value: 2}}}
		before := query(t, &count, page)
		execBehavioralFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS
SELECT e.* FROM event_fixture e CROSS JOIN (VALUES (1),(2),(3)) copies(n)`)
		after := query(t, &count, page)
		if !reflect.DeepEqual(before, after) || after.Profiles[6].Truth != behavioral.Match {
			t.Fatalf("expected same two distinct real occurrences despite join copies, got %+v and %+v", before, after)
		}
		execBehavioralFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
	})

	t.Run("time boundaries and transmitted precision", func(t *testing.T) {
		execBehavioralFixture(t, dw, `INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000001','t0','Time','2026-01-01','{}'),
('00000000-0000-0000-0000-000000000001','t1','Time','2026-01-01 00:00:00.001','{}'),
('00000000-0000-0000-0000-000000000001','t2','Time','2026-02-01','{}')`)
		for _, test := range []struct {
			start, end time.Time
			want       uint64
		}{
			{scope.Start, scope.Start.Add(time.Microsecond), 1},
			{scope.Start.Add(time.Microsecond), scope.End, 1},
		} {
			rule := behavioral.Rule{Count: &behavioral.Count{Scope: behavioral.Scope{Event: "Time",
				Grain: "registered-message", Start: test.start, End: test.end},
				Test: behavioral.CountTest{Operator: behavioral.Equal, Value: test.want}}}
			result := query(t, &rule, page)
			if result.Profiles[0].Truth != behavioral.Match {
				t.Fatalf("expected exact half-open and T boundary, got %+v", result)
			}
		}
		future := behavioral.Rule{Count: &behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal}}}
		future.Count.Scope.End = at.Add(time.Hour)
		result, err := store.QueryProfiles(t.Context(), &future, bindings, at, page)
		assertBehavioralUnsupported(t, err)
		if result != nil {
			t.Fatalf("expected no semantic result, got %+v", result)
		}
	})

	t.Run("recorded history raw and governed", func(t *testing.T) {
		history, err := behavioral.ResolveHistory("registered-queryable-events", "1")
		if err != nil {
			t.Fatalf("expected history binding, got %v", err)
		}
		for _, test := range []struct {
			name string
			rule behavioral.Rule
			want []behavioral.Truth
		}{
			{"raw", raw, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Match, behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}},
			{"governed", governed, []behavioral.Truth{behavioral.Match, behavioral.NoMatch, behavioral.Unknown, behavioral.Unknown, behavioral.Match, behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch}},
		} {
			t.Run(test.name, func(t *testing.T) {
				count := *test.rule.Count
				count.Scope = behavioral.Scope{Event: "Purchase", Grain: "registered-message", History: history}
				rule := behavioral.Rule{Count: &count}
				assertBehavioralTruths(t, query(t, &rule, page), test.want)
				for i, want := range test.want {
					explained, err := store.ExplainProfileFilter(t.Context(), &rule, bindings, at, behavioralTestID(i+1))
					if err != nil {
						t.Fatalf("expected history explanation, got %v", err)
					}
					if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want {
						t.Fatalf("expected history explanation %v for profile %d, got %+v", want, i+1, explained)
					}
				}
			})
		}
	})

	t.Run("sample minimum uses Kleene", func(t *testing.T) {
		execBehavioralFixture(t, dw, `INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000006','p6c','Purchase','2026-01-15','{}')`)
		defer execBehavioralFixture(t, dw, `DELETE FROM event_fixture WHERE message_id='p6c'`)
		count := *governed.Count
		count.Minimum, count.Test = 3, behavioral.CountTest{Operator: behavioral.Different, Value: 3}
		rule := behavioral.Rule{Count: &count}
		result := query(t, &rule, page)
		assertBehavioralTruths(t, result, []behavioral.Truth{behavioral.NoMatch, behavioral.NoMatch, behavioral.NoMatch,
			behavioral.NoMatch, behavioral.NoMatch, behavioral.Unknown, behavioral.NoMatch, behavioral.NoMatch})
		explained, err := store.ExplainProfileFilter(t.Context(), &rule, bindings, at, behavioralTestID(6))
		if err != nil {
			t.Fatalf("expected sample guard explanation, got %v", err)
		}
		if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != behavioral.Unknown ||
			explained.Nodes[0].Count == nil || explained.Nodes[0].Count.Certain != 1 ||
			explained.Nodes[0].Count.Possible != 3 {
			t.Fatalf("expected UNKNOWN with count [1,3], got %+v", explained)
		}
	})

	t.Run("independent memberships exhaustive oracle", func(t *testing.T) {
		var histories [][]int
		for i := range 27 {
			history := []int{i % 3, i / 3 % 3, i / 9 % 3}
			histories = append(histories, history)
			execBehavioralFixture(t, dw, fmt.Sprintf("INSERT INTO profile_fixture VALUES ('%s')", behavioralTestID(100+i)))
			for j, flag := range history {
				data := []string{`{"flag":false}`, `{"flag":true}`, `{}`}[flag]
				execBehavioralFixture(t, dw,
					fmt.Sprintf("INSERT INTO event_fixture VALUES ('%s','o%d_%d','Sample','2026-01-15','%s')",
						behavioralTestID(100+i), i, j, data))
			}
		}
		for _, op := range []behavioral.Comparison{behavioral.Equal, behavioral.Different, behavioral.Less,
			behavioral.LessOrEqual, behavioral.Greater, behavioral.GreaterOrEqual, behavioral.Between} {
			for threshold := uint64(0); threshold <= 4; threshold++ {
				for _, minimum := range []uint64{0, 2, 3} {
					if minimum > 0 && ((op == behavioral.Equal && threshold < minimum) || (op == behavioral.Less && threshold <= minimum) ||
						(op == behavioral.LessOrEqual && threshold < minimum) || (op == behavioral.Between && threshold+1 < minimum)) {
						continue // The same-condition parameters are statically incompatible.
					}
					count := behavioral.Count{Scope: scope, Minimum: minimum, Test: behavioral.CountTest{Operator: op,
						Value: threshold, Upper: threshold + 1},
						Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
							&state.WhereCondition{Property: []string{"properties", "flag"}, Operator: state.OpIsTrue},
						}})}
					count.Scope.Event = "Sample"
					var want []behavioral.Truth
					for range 8 {
						want = append(want, independentCountOracle(nil, minimum, count.Test))
					}
					for _, history := range histories {
						want = append(want, independentCountOracle(history, minimum, count.Test))
					}
					assertBehavioralTruths(t, query(t, &behavioral.Rule{Count: &count}, page), want)
				}
			}
		}
	})

	t.Run("existing typed JSON fixtures through compiled execution", func(t *testing.T) {
		fixtures, cases := warehouseexpr.JSONCases()
		for i, value := range fixtures {
			properties := json.Value(`{}`)
			if value != nil {
				var err error
				properties, err = json.Marshal(map[string]any{"value": value, "a.b": value})
				if err != nil {
					t.Fatalf("expected JSON fixture encoding, got %v", err)
				}
			}
			execBehavioralFixture(t, dw,
				fmt.Sprintf("INSERT INTO profile_fixture VALUES ('%s'); "+
					"INSERT INTO event_fixture VALUES ('%s','json%d','JSON','2026-01-15','%s')",
					behavioralTestID(200+i), behavioralTestID(200+i), i, strings.ReplaceAll(string(properties), "'", "''")))
		}
		for _, path := range [][]string{{"properties", "value"}, {"properties", "a.b"}} {
			for _, test := range cases {
				count := behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal, Value: 1},
					Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
						&state.WhereCondition{Property: path, Operator: state.WhereOperator(test.Operator), Values: test.Values},
					}})}
				count.Scope.Event = "JSON"
				result := query(t, &behavioral.Rule{Count: &count}, page)
				want := map[string]behavioral.Truth{}
				for _, i := range test.Matches {
					want[behavioralTestID(200+i)] = behavioral.Match
				}
				for _, i := range test.Nulls(len(fixtures)) {
					want[behavioralTestID(200+i)] = behavioral.Unknown
				}
				for _, profile := range result.Profiles {
					if profile.Truth != want[profile.ID] {
						t.Fatalf("expected %s/%v profile %s truth %d, got %d", test.Name, path, profile.ID, want[profile.ID],
							profile.Truth)
					}
				}
			}
		}
	})

	t.Run("shared governed facts rejected", func(t *testing.T) {
		for _, flags := range [][]bool{{true, true}, {true, false}} {
			eventName := "SharedSame"
			if !flags[1] {
				eventName = "SharedOpposite"
			}
			for i, flag := range flags {
				execBehavioralFixture(t, dw,
					fmt.Sprintf("INSERT INTO event_fixture VALUES ('%s','%s%d','%s','2026-01-15','{\"flag\":%t}')",
						behavioralTestID(1), eventName, i, eventName, flag))
			}
			var counts []int
			for _, g := range []bool{false, true} {
				n := 0
				for _, flag := range flags {
					if flag == g {
						n++
					}
				}
				counts = append(counts, n)
			}
			want := []int{0, 2}
			if !flags[1] {
				want = []int{1, 1}
			}
			if !reflect.DeepEqual(counts, want) {
				t.Fatalf("expected shared counts %v, got %v", want, counts)
			}
			count := behavioral.Count{Scope: scope, Test: behavioral.CountTest{Operator: behavioral.Equal, Value: 1},
				Shared: &behavioral.SharedBoolean{Fact: "profile.g@1", Field: []string{"properties", "flag"}, EqualTo: true}}
			count.Scope.Event = eventName
			result, err := store.QueryProfiles(t.Context(), &behavioral.Rule{Count: &count}, bindings, at, page)
			assertBehavioralUnsupported(t, err)
			if result != nil {
				t.Fatalf("expected no semantic result, got %+v", result)
			}
		}
	})

	t.Run("failed and canceled execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := store.QueryProfiles(ctx, &raw, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected canceled operation without result, got %+v, %v", result, err)
		}
		execBehavioralFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS SELECT kpid,message_id,event,timestamp,
jsonb_build_object('currency',(1/(length(message_id)-length(message_id)))::text) AS properties,type FROM event_fixture`)
		result, err = store.QueryProfiles(t.Context(), &raw, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected execution failure without result, got %+v, %v", result, err)
		}
		execBehavioralFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
		_ = query(t, &raw, page) // The connection remains usable after failure.
		execBehavioralFixture(t, dw, `DROP VIEW profiles; CREATE VIEW profiles AS SELECT 'bad-profile'::text AS _kpid`)
		result, err = store.QueryProfiles(t.Context(), nil, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected scan failure without result, got %+v, %v", result, err)
		}
		execBehavioralFixture(t, dw, `DROP VIEW profiles; CREATE VIEW profiles AS SELECT * FROM profile_fixture`)
		execBehavioralFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS SELECT e.* FROM event_fixture e CROSS JOIN (SELECT pg_sleep(2)) pause`)
		deadline, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer stop()
		result, err = store.QueryProfiles(deadline, &raw, bindings, at, page)
		if err == nil || result != nil || deadline.Err() == nil {
			t.Fatalf("expected timed out in-flight operation without result, got %+v, %v, %v", result, err, deadline.Err())
		}
		execBehavioralFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
		_ = query(t, &raw, page)
	})

	t.Run("empty population", func(t *testing.T) {
		execBehavioralFixture(t, dw, `DELETE FROM profile_fixture`)
		result := query(t, &raw, page)
		if len(result.Profiles) != 0 || result.Match != 0 || result.NoMatch != 0 || result.Unknown != 0 {
			t.Fatalf("expected empty population and zero totals, got %+v", result)
		}
	})

}

// TestBehavioralProfilesUnavailableBackend refuses an unverified provider.
func TestBehavioralProfilesUnavailableBackend(t *testing.T) {
	dw := &capabilityWarehouse{}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	result, err := store.QueryProfiles(t.Context(), nil, behavioral.Bindings{}, time.Date(2026, 1, 1, 0, 0, 0, 0,
		time.UTC), behavioral.Page{Limit: 1})
	if err != nil {
		if !errors.Is(err, ErrBehavioralFilteringUnavailable) {
			t.Fatalf("expected unavailable backend, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected unavailable backend error, got success")
	}
	if result != nil {
		t.Fatalf("expected no semantic result, got %+v", result)
	}
}

func assertBehavioralTruths(t *testing.T, result *behavioral.Result, want []behavioral.Truth) {
	t.Helper()
	got := make([]behavioral.Truth, len(result.Profiles))
	var m, f, u int64
	for i, p := range result.Profiles {
		got[i] = p.Truth
	}
	for _, truth := range want {
		switch truth {
		case behavioral.Match:
			m++
		case behavioral.NoMatch:
			f++
		case behavioral.Unknown:
			u++
		default:
			t.Fatalf("expected valid truth, got %d", truth)
		}
	}
	if !reflect.DeepEqual(got, want) || result.Match != m || result.NoMatch != f || result.Unknown != u {
		t.Fatalf("expected truths %v and totals %d/%d/%d, got %v and %d/%d/%d", want, m, f, u, got, result.Match,
			result.NoMatch, result.Unknown)
	}
}

func assertBehavioralUnsupported(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		if _, ok := errors.AsType[*behavioral.UnsupportedError](err); !ok {
			t.Fatalf("expected unavailable valid-language capability, got %v", err)
		}
		return
	}
	t.Fatal("expected unsupported capability error, got success")
}

func execBehavioralFixture(t *testing.T, dw warehouses.Warehouse, statement string) {
	t.Helper()
	// Fixture scripts contain no semicolons inside literals. Execute their
	// statements separately, as required by the real prepared-query transport.
	for _, part := range strings.Split(statement, ";") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		execBehavioralFixtureStatement(t, dw, part)
	}
}

func execBehavioralFixtureStatement(t *testing.T, dw warehouses.Warehouse, statement string) {
	t.Helper()
	raw := dw.(interface {
		RawQuery(context.Context, string) (warehouses.Rows, int, error)
	})
	rows, _, err := raw.RawQuery(t.Context(), statement)
	if err != nil {
		t.Fatalf("expected fixture statement, got %v", err)
	}
	for rows.Next() {
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		t.Fatalf("expected complete fixture statement, got %v", err)
	}
	if closeErr != nil {
		t.Fatalf("expected fixture close, got %v", closeErr)
	}
}

// independentCountOracle enumerates boolean completions; it uses no compiler
// bounds, terminal rendering or production evaluation helpers.
func independentCountOracle(flags []int, minimum uint64, test behavioral.CountTest) behavioral.Truth {
	testYes, testNo, guardYes, guardNo := false, false, false, false
	var visit func(int, uint64)
	visit = func(index int, count uint64) {
		if index < len(flags) {
			if flags[index] != 1 {
				visit(index+1, count)
			}
			if flags[index] != 0 {
				visit(index+1, count+1)
			}
			return
		}
		ok := false
		switch test.Operator {
		case behavioral.Equal:
			ok = count == test.Value
		case behavioral.Different:
			ok = count != test.Value
		case behavioral.Less:
			ok = count < test.Value
		case behavioral.LessOrEqual:
			ok = count <= test.Value
		case behavioral.Greater:
			ok = count > test.Value
		case behavioral.GreaterOrEqual:
			ok = count >= test.Value
		case behavioral.Between:
			ok = count >= test.Value && count <= test.Upper
		default:
			panic(fmt.Sprintf("core/internal/datastore: unexpected oracle comparison %d", test.Operator))
		}
		testYes, testNo = testYes || ok, testNo || !ok
		guardYes, guardNo = guardYes || count >= minimum, guardNo || count < minimum
	}
	visit(0, 0)
	if !testYes || !guardYes {
		return behavioral.NoMatch
	}
	if testNo || guardNo {
		return behavioral.Unknown
	}
	return behavioral.Match
}

func behavioralTestID(n int) string { return fmt.Sprintf("00000000-0000-0000-0000-%012d", n) }
