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

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedProfilesPostgreSQL executes the Count subset on a real warehouse.
func TestUnifiedProfilesPostgreSQL(t *testing.T) {

	dw := newJSONFilterTestWarehouse(t, "PostgreSQL")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	// Publish coherent fixture views before any filtering request.
	// Its identity, time and profile attribution are determined; no writer or
	// Identity Resolution operation runs concurrently with these tests.
	execUnifiedFixture(t, dw, `CREATE TABLE profile_fixture (_kpid uuid PRIMARY KEY);
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
	scope := unified.Scope{Event: "Purchase", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}
	bindings := unified.Bindings{Currencies: []unified.CurrencyBinding{{
		Definition: "NativeCurrency", Revision: "registered-v1", Path: []string{"properties", "currency"},
		Evidence: unified.IndependentOccurrenceValues,
	}}}
	governed := unified.Rule{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal},
		Currency: &unified.NativeCurrency{Definition: "NativeCurrency", Revision: "registered-v1", Value: "EUR"}}}
	raw := governed
	raw.Count = &unified.Count{Scope: scope, Test: governed.Count.Test,
		Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"properties", "currency"}, Operator: state.OpIs,
				Values: []any{json.Value(`"EUR"`)}},
		}})}
	page := unified.Page{Limit: 1000}
	query := func(t *testing.T, rule *unified.Rule, p unified.Page) *unified.Result {
		t.Helper()
		result, err := store.UnifiedProfiles(t.Context(), rule, bindings, at, p)
		if err != nil {
			t.Fatalf("expected complete Unified query, got %v", err)
		}
		return result
	}
	t.Run("all profiles", func(t *testing.T) {
		assertUnifiedTruths(t, query(t, nil, page), []unified.Truth{unified.Match, unified.Match, unified.Match,
			unified.Match, unified.Match, unified.Match, unified.Match, unified.Match})
	})

	t.Run("governed absence", func(t *testing.T) {
		result := query(t, &governed, page)
		assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown,
			unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.Match})
	})

	t.Run("raw null and missing", func(t *testing.T) {
		result := query(t, &raw, page)
		assertUnifiedTruths(t, result, []unified.Truth{unified.Match, unified.NoMatch, unified.Match,
			unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.Match})
	})

	t.Run("non-track events", func(t *testing.T) {

		execUnifiedFixture(t, dw, `INSERT INTO event_fixture (kpid,message_id,event,timestamp,properties,type)
SELECT _kpid,'non-track/'||event_type||'/'||_kpid,NULL,'2026-01-15',NULL,event_type
FROM profile_fixture CROSS JOIN (VALUES ('identify'),('page'),('screen'),('group'),('alias')) types(event_type);
INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000002','review/2','Review','2026-01-15','{}'),
('00000000-0000-0000-0000-000000000008','review/8','Review','2026-01-15','{}')`)
		absence := unified.Rule{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal}}}
		purchase := unified.Rule{Count: &unified.Count{Scope: scope,
			Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}}}
		reviewScope := scope
		reviewScope.Event = "Review"
		review := unified.Rule{Count: &unified.Count{Scope: reviewScope,
			Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}}}

		for _, test := range []struct {
			name string
			rule unified.Rule
			want []unified.Truth
		}{
			{"absence", absence, []unified.Truth{unified.Match, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.Match}},
			{"existence", purchase, []unified.Truth{unified.NoMatch, unified.Match, unified.Match, unified.Match, unified.Match, unified.Match, unified.Match, unified.NoMatch}},
			{"raw qualification", raw, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.Match}},
			{"governed qualification", governed, []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown, unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.Match}},
			{"order and review", unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{purchase, review}}, []unified.Truth{unified.NoMatch, unified.Match, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch}},
			{"order or review", unified.Rule{Operator: state.OpOr, Children: []unified.Rule{purchase, review}}, []unified.Truth{unified.NoMatch, unified.Match, unified.Match, unified.Match, unified.Match, unified.Match, unified.Match, unified.Match}},
		} {
			t.Run(test.name, func(t *testing.T) {
				assertUnifiedTruths(t, query(t, &test.rule, page), test.want)
			})
		}
		execUnifiedFixture(t, dw, `DELETE FROM event_fixture
WHERE message_id LIKE 'non-track/%' OR message_id LIKE 'review/%'`)

	})

	t.Run("unknown event name retains possible evidence", func(t *testing.T) {
		execUnifiedFixture(t, dw,
			`INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000001','unknown-name',NULL,'2026-01-15','{"currency":"EUR"}')`)
		result := query(t, &governed, page)
		if result.Profiles[0].Truth != unified.Unknown {
			t.Fatalf("expected unknown named-event membership, got %+v", result.Profiles[0])
		}
		execUnifiedFixture(t, dw, `DELETE FROM event_fixture WHERE message_id='unknown-name'`)
	})

	t.Run("membership page and empty page totals", func(t *testing.T) {
		all := query(t, &governed, page)
		result := query(t, &governed, unified.Page{Limit: 1, First: 1, MatchingOnly: true})
		if len(result.Profiles) != 1 || result.Profiles[0].ID != unifiedTestID(5) ||
			result.Profiles[0].Truth != unified.Match {
			t.Fatalf("expected second matching profile 5, got %+v", result.Profiles)
		}
		empty := query(t, &governed, unified.Page{First: 100, Limit: 2, MatchingOnly: true})
		if len(empty.Profiles) != 0 || empty.Match != all.Match || empty.NoMatch != all.NoMatch ||
			empty.Unknown != all.Unknown {
			t.Fatalf("expected empty page with complete totals %+v, got %+v", all, empty)
		}
	})

	t.Run("outer Kleene", func(t *testing.T) {
		exists := *governed.Count
		exists.Test = unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}
		for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
			root := unified.Rule{Operator: op, Children: []unified.Rule{governed, {Count: &exists}}}
			want := []unified.Truth{unified.NoMatch, unified.NoMatch, unified.Unknown, unified.Unknown,
				unified.NoMatch, unified.NoMatch, unified.NoMatch, unified.NoMatch}
			if op == state.OpOr {
				want = []unified.Truth{unified.Match, unified.Match, unified.Unknown, unified.Unknown, unified.Match,
					unified.Match, unified.Match, unified.Match}
			}
			assertUnifiedTruths(t, query(t, &root, page), want)
		}
	})

	t.Run("Kleene decisive operands", func(t *testing.T) {
		base := query(t, &governed, page)
		for _, constant := range []bool{false, true} {
			terminal := unified.CountTest{Operator: unified.Less}
			if constant {
				terminal.Operator = unified.GreaterOrEqual
			}
			for _, op := range []state.WhereLogical{state.OpAnd, state.OpOr} {
				root := unified.Rule{Operator: op, Children: []unified.Rule{governed,
					{Count: &unified.Count{Scope: scope, Test: terminal}}}}
				want := make([]unified.Truth, len(base.Profiles))
				for i, profile := range base.Profiles {
					want[i] = profile.Truth
					if !constant && op == state.OpAnd {
						want[i] = unified.NoMatch
					}
					if constant && op == state.OpOr {
						want[i] = unified.Match
					}
				}
				assertUnifiedTruths(t, query(t, &root, page), want)
			}
		}
	})

	t.Run("identity grain and duplicate joins", func(t *testing.T) {
		count := unified.Rule{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal, Value: 2}}}
		before := query(t, &count, page)
		execUnifiedFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS
SELECT e.* FROM event_fixture e CROSS JOIN (VALUES (1),(2),(3)) copies(n)`)
		after := query(t, &count, page)
		if !reflect.DeepEqual(before, after) || after.Profiles[6].Truth != unified.Match {
			t.Fatalf("expected same two distinct real occurrences despite join copies, got %+v and %+v", before, after)
		}
		execUnifiedFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
	})

	t.Run("time boundaries and transmitted precision", func(t *testing.T) {
		execUnifiedFixture(t, dw, `INSERT INTO event_fixture VALUES
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
			rule := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Time",
				Grain: "registered-message", Start: test.start, End: test.end},
				Test: unified.CountTest{Operator: unified.Equal, Value: test.want}}}
			result := query(t, &rule, page)
			if result.Profiles[0].Truth != unified.Match {
				t.Fatalf("expected exact half-open and T boundary, got %+v", result)
			}
		}
		future := unified.Rule{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal}}}
		future.Count.Scope.End = at.Add(time.Hour)
		result, err := store.UnifiedProfiles(t.Context(), &future, bindings, at, page)
		assertUnifiedUnsupported(t, err)
		if result != nil {
			t.Fatalf("expected no semantic result, got %+v", result)
		}
	})

	t.Run("recorded history raw and governed", func(t *testing.T) {
		history, err := unified.ResolveHistory("registered-queryable-events", "1")
		if err != nil {
			t.Fatalf("expected history binding, got %v", err)
		}
		for _, test := range []struct {
			name string
			rule unified.Rule
			want []unified.Truth
		}{
			{"raw", raw, []unified.Truth{unified.Match, unified.NoMatch, unified.Match, unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.NoMatch}},
			{"governed", governed, []unified.Truth{unified.Match, unified.NoMatch, unified.Unknown, unified.Unknown, unified.Match, unified.NoMatch, unified.NoMatch, unified.NoMatch}},
		} {
			t.Run(test.name, func(t *testing.T) {
				count := *test.rule.Count
				count.Scope = unified.Scope{Event: "Purchase", Grain: "registered-message", History: history}
				rule := unified.Rule{Count: &count}
				assertUnifiedTruths(t, query(t, &rule, page), test.want)
				for i, want := range test.want {
					explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, bindings, at, unifiedTestID(i+1))
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
		execUnifiedFixture(t, dw, `INSERT INTO event_fixture VALUES
('00000000-0000-0000-0000-000000000006','p6c','Purchase','2026-01-15','{}')`)
		defer execUnifiedFixture(t, dw, `DELETE FROM event_fixture WHERE message_id='p6c'`)
		count := *governed.Count
		count.Minimum, count.Test = 3, unified.CountTest{Operator: unified.Different, Value: 3}
		rule := unified.Rule{Count: &count}
		result := query(t, &rule, page)
		assertUnifiedTruths(t, result, []unified.Truth{unified.NoMatch, unified.NoMatch, unified.NoMatch,
			unified.NoMatch, unified.NoMatch, unified.Unknown, unified.NoMatch, unified.NoMatch})
		explained, err := store.ExplainUnifiedProfile(t.Context(), &rule, bindings, at, unifiedTestID(6))
		if err != nil {
			t.Fatalf("expected sample guard explanation, got %v", err)
		}
		if explained == nil || len(explained.Nodes) != 1 || explained.Nodes[0].Truth != unified.Unknown ||
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
			execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO profile_fixture VALUES ('%s')", unifiedTestID(100+i)))
			for j, flag := range history {
				data := []string{`{"flag":false}`, `{"flag":true}`, `{}`}[flag]
				execUnifiedFixture(t, dw,
					fmt.Sprintf("INSERT INTO event_fixture VALUES ('%s','o%d_%d','Sample','2026-01-15','%s')",
						unifiedTestID(100+i), i, j, data))
			}
		}
		for _, op := range []unified.Comparison{unified.Equal, unified.Different, unified.Less,
			unified.LessOrEqual, unified.Greater, unified.GreaterOrEqual, unified.Between} {
			for threshold := uint64(0); threshold <= 4; threshold++ {
				for _, minimum := range []uint64{0, 2, 3} {
					if minimum > 0 && ((op == unified.Equal && threshold < minimum) || (op == unified.Less && threshold <= minimum) ||
						(op == unified.LessOrEqual && threshold < minimum) || (op == unified.Between && threshold+1 < minimum)) {
						continue // The same-condition parameters are statically incompatible.
					}
					count := unified.Count{Scope: scope, Minimum: minimum, Test: unified.CountTest{Operator: op,
						Value: threshold, Upper: threshold + 1},
						Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
							&state.WhereCondition{Property: []string{"properties", "flag"}, Operator: state.OpIsTrue},
						}})}
					count.Scope.Event = "Sample"
					var want []unified.Truth
					for range 8 {
						want = append(want, independentCountOracle(nil, minimum, count.Test))
					}
					for _, history := range histories {
						want = append(want, independentCountOracle(history, minimum, count.Test))
					}
					assertUnifiedTruths(t, query(t, &unified.Rule{Count: &count}, page), want)
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
			execUnifiedFixture(t, dw,
				fmt.Sprintf("INSERT INTO profile_fixture VALUES ('%s'); "+
					"INSERT INTO event_fixture VALUES ('%s','json%d','JSON','2026-01-15','%s')",
					unifiedTestID(200+i), unifiedTestID(200+i), i, strings.ReplaceAll(string(properties), "'", "''")))
		}
		for _, path := range [][]string{{"properties", "value"}, {"properties", "a.b"}} {
			for _, test := range cases {
				count := unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal, Value: 1},
					Where: recordCountWhere(t, &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
						&state.WhereCondition{Property: path, Operator: state.WhereOperator(test.Operator), Values: test.Values},
					}})}
				count.Scope.Event = "JSON"
				result := query(t, &unified.Rule{Count: &count}, page)
				want := map[string]unified.Truth{}
				for _, i := range test.Matches {
					want[unifiedTestID(200+i)] = unified.Match
				}
				for _, i := range test.Nulls(len(fixtures)) {
					want[unifiedTestID(200+i)] = unified.Unknown
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
				execUnifiedFixture(t, dw,
					fmt.Sprintf("INSERT INTO event_fixture VALUES ('%s','%s%d','%s','2026-01-15','{\"flag\":%t}')",
						unifiedTestID(1), eventName, i, eventName, flag))
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
			count := unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal, Value: 1},
				Shared: &unified.SharedBoolean{Fact: "profile.g@1", Field: []string{"properties", "flag"}, EqualTo: true}}
			count.Scope.Event = eventName
			result, err := store.UnifiedProfiles(t.Context(), &unified.Rule{Count: &count}, bindings, at, page)
			assertUnifiedUnsupported(t, err)
			if result != nil {
				t.Fatalf("expected no semantic result, got %+v", result)
			}
		}
	})

	t.Run("failed and canceled execution", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		result, err := store.UnifiedProfiles(ctx, &raw, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected canceled operation without result, got %+v, %v", result, err)
		}
		execUnifiedFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS SELECT kpid,message_id,event,timestamp,
jsonb_build_object('currency',(1/(length(message_id)-length(message_id)))::text) AS properties,type FROM event_fixture`)
		result, err = store.UnifiedProfiles(t.Context(), &raw, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected execution failure without result, got %+v, %v", result, err)
		}
		execUnifiedFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
		_ = query(t, &raw, page) // The connection remains usable after failure.
		execUnifiedFixture(t, dw, `DROP VIEW profiles; CREATE VIEW profiles AS SELECT 'bad-profile'::text AS _kpid`)
		result, err = store.UnifiedProfiles(t.Context(), nil, bindings, at, page)
		if err == nil || result != nil {
			t.Fatalf("expected scan failure without result, got %+v, %v", result, err)
		}
		execUnifiedFixture(t, dw, `DROP VIEW profiles; CREATE VIEW profiles AS SELECT * FROM profile_fixture`)
		execUnifiedFixture(t, dw,
			`CREATE OR REPLACE VIEW krenalis_events AS SELECT e.* FROM event_fixture e CROSS JOIN (SELECT pg_sleep(2)) pause`)
		deadline, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer stop()
		result, err = store.UnifiedProfiles(deadline, &raw, bindings, at, page)
		if err == nil || result != nil || deadline.Err() == nil {
			t.Fatalf("expected timed out in-flight operation without result, got %+v, %v, %v", result, err, deadline.Err())
		}
		execUnifiedFixture(t, dw, `CREATE OR REPLACE VIEW krenalis_events AS SELECT * FROM event_fixture`)
		_ = query(t, &raw, page)
	})

	t.Run("empty population", func(t *testing.T) {
		execUnifiedFixture(t, dw, `DELETE FROM profile_fixture`)
		result := query(t, &raw, page)
		if len(result.Profiles) != 0 || result.Match != 0 || result.NoMatch != 0 || result.Unknown != 0 {
			t.Fatalf("expected empty population and zero totals, got %+v", result)
		}
	})

}

// TestUnifiedProfilesSnowflakeGate refuses the unverified remote path.
func TestUnifiedProfilesSnowflakeGate(t *testing.T) {
	dw := warehouses.Registered("Snowflake").New(newTestSettingsLoader(json.Value(`{}`)), nil)
	t.Cleanup(func() {
		err := dw.Close()
		if err != nil {
			t.Errorf("expected warehouse close, got %v", err)
		}
	})
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	result, err := store.UnifiedProfiles(t.Context(), nil, unified.Bindings{}, time.Date(2026, 1, 1, 0, 0, 0, 0,
		time.UTC), unified.Page{Limit: 1})
	if err != nil {
		if !errors.Is(err, ErrUnifiedUnavailable) {
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

func assertUnifiedTruths(t *testing.T, result *unified.Result, want []unified.Truth) {
	t.Helper()
	got := make([]unified.Truth, len(result.Profiles))
	var m, f, u int64
	for i, p := range result.Profiles {
		got[i] = p.Truth
	}
	for _, truth := range want {
		switch truth {
		case unified.Match:
			m++
		case unified.NoMatch:
			f++
		case unified.Unknown:
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

func assertUnifiedUnsupported(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		if _, ok := errors.AsType[*unified.UnsupportedError](err); !ok {
			t.Fatalf("expected unavailable valid-language capability, got %v", err)
		}
		return
	}
	t.Fatal("expected unsupported capability error, got success")
}

func execUnifiedFixture(t *testing.T, dw warehouses.Warehouse, statement string) {
	t.Helper()
	// Fixture scripts contain no semicolons inside literals. Execute their
	// statements separately, as required by the real prepared-query transport.
	for _, part := range strings.Split(statement, ";") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		execUnifiedFixtureStatement(t, dw, part)
	}
}

func execUnifiedFixtureStatement(t *testing.T, dw warehouses.Warehouse, statement string) {
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
func independentCountOracle(flags []int, minimum uint64, test unified.CountTest) unified.Truth {
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
		case unified.Equal:
			ok = count == test.Value
		case unified.Different:
			ok = count != test.Value
		case unified.Less:
			ok = count < test.Value
		case unified.LessOrEqual:
			ok = count <= test.Value
		case unified.Greater:
			ok = count > test.Value
		case unified.GreaterOrEqual:
			ok = count >= test.Value
		case unified.Between:
			ok = count >= test.Value && count <= test.Upper
		default:
			panic(fmt.Sprintf("core/internal/datastore: unexpected oracle comparison %d", test.Operator))
		}
		testYes, testNo = testYes || ok, testNo || !ok
		guardYes, guardNo = guardYes || count >= minimum, guardNo || count < minimum
	}
	visit(0, 0)
	if !testYes || !guardYes {
		return unified.NoMatch
	}
	if testNo || guardNo {
		return unified.Unknown
	}
	return unified.Match
}

func unifiedTestID(n int) string { return fmt.Sprintf("00000000-0000-0000-0000-%012d", n) }
