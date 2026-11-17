// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestBehavioralRecordStatesPostgreSQL checks native state predicates in SQL.
func TestBehavioralRecordStatesPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testBehavioralRecordStates(t, "PostgreSQL")
}

// TestBehavioralRecordStatesSnowflake uses an isolated Snowflake warehouse.
func TestBehavioralRecordStatesSnowflake(t *testing.T) {
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
	testBehavioralRecordStates(t, "Snowflake")
}

func testBehavioralRecordStates(t *testing.T, platform string) {

	t.Helper()
	columns := []warehouses.Column{{Name: "text", Type: types.String(), Nullable: true},
		{Name: "elements", Type: types.Array(types.Int(32)), Nullable: true}}
	fixture := newBehavioralRecordFixture(t, platform, columns)
	table := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: append(
		[]warehouses.Column{{Name: "_kpid", Type: types.UUID()}, {Name: "_anonymous_count", Type: types.Int(32)},
			{Name: "_recognized_count", Type: types.Int(32)}, {Name: "_updated_at", Type: types.DateTime()}},
		columns...)}
	err := fixture.warehouse.Merge(t.Context(), table, [][]any{
		{behavioralTestID(1), 0, 1, fixture.at, nil, nil},
		{behavioralTestID(2), 0, 1, fixture.at, "", []any{}},
		{behavioralTestID(3), 0, 1, fixture.at, "x", []any{1}},
	}, nil)
	if err != nil {
		t.Fatalf("expected native state fixture writes, got %v", err)
	}
	// This exercises prepared physical predicates; native is-null admission in
	// the public API still requires a nullable property and is unchanged.
	m, f := behavioral.Match, behavioral.NoMatch
	for _, test := range []struct {
		name string
		op   state.WhereOperator
		want []behavioral.Truth
	}{
		{"empty", state.OpIsEmpty, []behavioral.Truth{m, m, f}},
		{"not empty", state.OpIsNotEmpty, []behavioral.Truth{f, f, m}},
		{"null", state.OpIsNull, []behavioral.Truth{m, f, f}},
		{"not null", state.OpIsNotNull, []behavioral.Truth{f, m, m}},
		{"exists", state.OpExists, []behavioral.Truth{f, m, m}},
		{"does not exist", state.OpDoesNotExist, []behavioral.Truth{m, f, f}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &behavioral.Rule{Operator: state.OpAnd, Children: []behavioral.Rule{
				{Record: &state.WhereCondition{Property: []string{"text"}, Operator: test.op}},
				{Record: &state.WhereCondition{Property: []string{"elements"}, Operator: test.op}},
			}}
			assertBehavioralTruths(t, fixture.query(t, root, behavioral.Page{Limit: 10}), test.want)
			for i, want := range test.want {
				explained := fixture.explain(t, root, i+1)
				if len(explained.Nodes) != 3 {
					t.Fatalf("expected group and two native state leaves, got %+v", explained.Nodes)
				}
				for node, outcome := range explained.Nodes {
					if outcome.Truth != want {
						t.Fatalf("expected native state node %d truth %v for profile %d, got %+v",
							node, want, i+1, outcome)
					}
				}
			}
		})
	}
	t.Logf("Native state budget: query=%d explanation=%d", fixture.queries, fixture.explains)

}
