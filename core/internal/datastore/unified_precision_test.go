// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedTimestampPrecisionPostgreSQL checks native timestamp borders.
func TestUnifiedTimestampPrecisionPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testUnifiedTimestampPrecision(t, "PostgreSQL")
}

// TestUnifiedTimestampPrecisionSnowflake also checks native nanosecond borders.
func TestUnifiedTimestampPrecisionSnowflake(t *testing.T) {
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
	testUnifiedTimestampPrecision(t, "Snowflake")
}

func testUnifiedTimestampPrecision(t *testing.T, platform string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	timestampType := "timestamp"
	precisions := []int{3, 6}
	if platform == "Snowflake" {
		timestampType = "TIMESTAMP_NTZ"
		precisions = append(precisions, 9) // PostgreSQL cannot store nanosecond timestamps.
	}
	q := func(name string) string { return quoteUnifiedRecordIdentifier(platform, name) }
	execUnifiedFixture(t, dw, "CREATE TABLE "+q("profiles")+" ("+q("_kpid")+" VARCHAR(36));"+
		"INSERT INTO "+q("profiles")+" VALUES ('"+unifiedTestID(1)+"')")
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	fixture := &unifiedRecordFixture{store: store, warehouse: dw}

	for _, precision := range precisions {
		t.Run(fmt.Sprintf("precision-%d", precision), func(t *testing.T) {

			step, fraction := time.Microsecond, 123456000
			if precision == 3 {
				step, fraction = time.Millisecond, 123000000
			} else if precision == 9 {
				step = time.Nanosecond
			}
			fixture.at = time.Date(2026, 2, 1, 0, 0, 0, fraction, time.UTC)
			scope := unified.Scope{Event: "Order Completed", Grain: "registered-message",
				Start: fixture.at.Add(-24 * time.Hour), End: fixture.at}
			execUnifiedFixture(t, dw, "CREATE TABLE "+q("krenalis_events")+" ("+q("kpid")+
				" VARCHAR(36),"+q("message_id")+" VARCHAR,"+q("type")+" VARCHAR,"+q("event")+
				" VARCHAR,"+q("timestamp")+fmt.Sprintf(" %s(%d))", timestampType, precision))
			defer execUnifiedFixture(t, dw, "DROP TABLE "+q("krenalis_events"))
			for _, event := range []struct {
				id string
				at time.Time
			}{
				{"before-start", scope.Start.Add(-step)}, {"at-start", scope.Start},
				{"after-start", scope.Start.Add(step)}, {"before-end", scope.End.Add(-step)},
				{"at-end", scope.End}, {"after-end", scope.End.Add(step)},
			} {
				execUnifiedFixture(t, dw, fmt.Sprintf("INSERT INTO %s SELECT '%s','%s','track','Order Completed',"+
					"CAST('%s' AS %s(%d))", q("krenalis_events"), unifiedTestID(1), event.id,
					event.at.Format("2006-01-02 15:04:05.999999999"), timestampType, precision))
			}

			root := &unified.Rule{Operator: state.OpAnd, Children: []unified.Rule{
				{Count: &unified.Count{Scope: scope, Test: unified.CountTest{Operator: unified.Equal, Value: 3}}},
				{Experience: &unified.Experience{Scope: scope, First: true,
					Verify: &unified.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
						Operator: state.OpIs, Values: []any{"at-start"}}}}},
				{Experience: &unified.Experience{Scope: scope,
					Verify: &unified.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"messageId"},
						Operator: state.OpIs, Values: []any{"before-end"}}}}},
			}}
			assertUnifiedTruths(t, fixture.query(t, root, unified.Page{Limit: 10}), []unified.Truth{unified.Match})
			explained := fixture.explain(t, root, 1)
			if len(explained.Nodes) != 4 || explained.Nodes[1].Count == nil ||
				*explained.Nodes[1].Count != (warehouses.OccurrenceCountBounds{Certain: 3, Possible: 3}) {
				t.Fatal("expected four time nodes with exact count 3, got incomplete metadata")
			}
			for i, node := range explained.Nodes {
				if node.Truth != unified.Match {
					t.Fatalf("expected timestamp node %d MATCH, got %v", i, node.Truth)
				}
				if i > 1 && (node.Experience == nil ||
					*node.Experience != (warehouses.OccurrenceSelection{Candidates: 1})) {
					t.Fatalf("expected timestamp node %d with one determined candidate, got %+v", i, node.Experience)
				}
			}

		})
	}

}
