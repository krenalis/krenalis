// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/snowflakedb/gosnowflake/v2"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type behavioralRecordFixture struct {
	store     *Store
	warehouse warehouses.Warehouse
	at        time.Time
	queries   int
	explains  int
	legacy    int
}

func newBehavioralRecordFixture(t *testing.T, platform string, columns []warehouses.Column) *behavioralRecordFixture {
	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	err := dw.Initialize(t.Context(), columns)
	if err != nil {
		t.Fatalf("expected warehouse initialization, got %v", err)
	}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(wrapWarehouse(dw))
	store.columnByProperty.user = map[string]warehouses.Column{
		"_kpid": {Name: "_kpid", Type: types.UUID()},
	}
	for _, column := range columns {
		store.columnByProperty.user[column.Name] = column
	}
	return &behavioralRecordFixture{store: store, warehouse: dw,
		at: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
}

func (fixture *behavioralRecordFixture) explain(t *testing.T, root *behavioral.Rule, profile int) *behavioral.Explanation {
	t.Helper()
	fixture.explains++
	ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordExplanationTag)
	result, err := fixture.store.ExplainProfileFilter(ctx, root, behavioral.Bindings{}, fixture.at,
		behavioralTestID(profile))
	if err != nil {
		t.Fatalf("expected Record explanation, got %v", err)
	}
	if result == nil || !result.At.Equal(fixture.at) {
		t.Fatalf("expected explanation at %s, got %+v", fixture.at, result)
	}
	return result
}

func (fixture *behavioralRecordFixture) query(t *testing.T, root *behavioral.Rule, page behavioral.Page) *behavioral.Result {
	t.Helper()
	fixture.queries++
	ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag)
	result, err := fixture.store.QueryProfiles(ctx, root, behavioral.Bindings{}, fixture.at, page)
	if err != nil {
		t.Fatalf("expected Record query, got %v", err)
	}
	if !result.At.Equal(fixture.at) {
		t.Fatalf("expected one evaluation time %s, got %s", fixture.at, result.At)
	}
	return result
}

// TestBehavioralRecordConformancePostgreSQL checks all original Record
// operators.
func TestBehavioralRecordConformancePostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testBehavioralRecordConformance(t, "PostgreSQL")
}

// TestBehavioralRecordConformanceSnowflake uses an isolated Snowflake
// warehouse.
func TestBehavioralRecordConformanceSnowflake(t *testing.T) {
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
	testBehavioralRecordConformance(t, "Snowflake")
}

func testBehavioralRecordConformance(t *testing.T, platform string) {

	t.Helper()
	cases := append(warehouseexpr.ValueCases(), warehouseexpr.SQLFloatCases()...)
	columns := make([]warehouses.Column, len(cases))
	for i, c := range cases {
		columns[i] = warehouses.Column{Name: fmt.Sprintf("v%d", i), Type: c.Type, Nullable: true}
	}
	columns = append(columns, warehouses.Column{Name: "document", Type: types.JSON(), Nullable: true},
		warehouses.Column{Name: "scalar", Type: types.JSON(), Nullable: true})
	fixture := newBehavioralRecordFixture(t, platform, columns)
	table := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: append(
		[]warehouses.Column{{Name: "_kpid", Type: types.UUID()}, {Name: "_anonymous_count", Type: types.Int(32)},
			{Name: "_recognized_count", Type: types.Int(32)}, {Name: "_updated_at", Type: types.DateTime()}},
		columns...)}
	present := []any{behavioralTestID(1), 0, 1, fixture.at}
	absent := []any{behavioralTestID(2), 0, 1, fixture.at}
	for _, c := range cases {
		present = append(present, c.Value)
		absent = append(absent, nil)
	}
	present, absent = append(present, nil, nil), append(absent, nil, nil)
	err := fixture.warehouse.Merge(t.Context(), table, [][]any{present, absent}, nil)
	if err != nil {
		t.Fatalf("expected native fixture writes, got %v", err)
	}
	covered := map[warehouses.Operator]bool{}
	for operator := warehouses.OpIs; operator <= warehouses.OpDoesNotExist; operator++ {
		var children []behavioral.Rule
		var matches []bool
		for i, c := range cases {
			if c.Operator == operator {
				children = append(children, behavioral.Rule{Record: &state.WhereCondition{
					Property: []string{columns[i].Name}, Operator: state.WhereOperator(operator), Values: c.Values}})
				matches = append(matches, c.Matches)
			}
		}
		if len(children) == 0 {
			continue
		}
		covered[operator] = true
		t.Run(fmt.Sprintf("native/%d", operator), func(t *testing.T) {

			root := &behavioral.Rule{Operator: state.OpAnd, Children: children}
			truth := behavioral.Match
			for _, match := range matches {
				if !match {
					truth = behavioral.NoMatch
				}
			}
			result := fixture.query(t, root, behavioral.Page{Limit: 100})
			assertBehavioralTruths(t, result, []behavioral.Truth{truth, behavioral.Unknown})
			for profile := 1; profile <= 2; profile++ {
				explained := fixture.explain(t, root, profile)
				if len(explained.Nodes) != len(children)+1 {
					t.Fatalf("expected %d explanation nodes, got %d", len(children)+1, len(explained.Nodes))
				}
				for i, match := range matches {
					want := behavioral.Unknown
					if profile == 1 {
						want = behavioral.NoMatch
						if match {
							want = behavioral.Match
						}
					}
					if explained.Nodes[i+1].Truth != want {
						t.Fatalf("expected native leaf %d truth %v for profile %d, got %+v",
							i, want, profile, explained.Nodes[i+1])
					}
				}
			}
			root.Operator = state.OpOr
			truth = behavioral.NoMatch
			for _, match := range matches {
				if match {
					truth = behavioral.Match
				}
			}
			result = fixture.query(t, root, behavioral.Page{Limit: 100})
			assertBehavioralTruths(t, result, []behavioral.Truth{truth, behavioral.Unknown})
			where := &state.Where{Operator: state.OpOr}
			for _, child := range children {
				where.Rules = append(where.Rules, child.Record)
			}
			fixture.legacy++
			ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordLegacyTag)
			legacy, total, err := fixture.store.Profiles(ctx,
				Query{Properties: []string{"_kpid"}, Where: where, Limit: 100})
			if err != nil {
				t.Fatalf("expected legacy Record query, got %v", err)
			}
			if int64(total) != result.Match || len(legacy) != int(result.Match) {
				t.Fatalf("expected legacy membership count %d, got %d and %d rows", result.Match, total, len(legacy))
			}
			if len(legacy) == 1 && legacy[0]["_kpid"] != behavioralTestID(1) {
				t.Fatalf("expected present profile in legacy membership, got %v", legacy)
			}

		})
	}

	// Reuse every JSON truth table in both scalar and literal nested form.
	execBehavioralFixture(t, fixture.warehouse, "TRUNCATE TABLE krenalis_profiles_0")
	values, jsonCases := warehouseexpr.JSONCases()
	jsonTable := warehouses.Table{Name: table.Name, Keys: table.Keys, Columns: append(
		slices.Clone(table.Columns[:4]), warehouses.Column{Name: "document", Type: types.JSON(), Nullable: true},
		warehouses.Column{Name: "scalar", Type: types.JSON(), Nullable: true})}
	var rows [][]any
	for i, value := range values {
		var document any
		if value != nil {
			document = json.Value(`{"literal.key":{"quote'key":{"back\\slash":` + string(value) + `}}}`)
		}
		var scalar any
		if value != nil {
			scalar = value
		}
		rows = append(rows, []any{behavioralTestID(i + 1), 0, 1, fixture.at, document, scalar})
	}
	err = fixture.warehouse.Merge(t.Context(), jsonTable, rows, nil)
	if err != nil {
		t.Fatalf("expected JSON fixture writes, got %v", err)
	}
	explainedOperators := map[warehouses.Operator]bool{}
	for index, c := range jsonCases {
		t.Run("json/"+c.Name, func(t *testing.T) {
			condition := &state.WhereCondition{
				Property: []string{"document", "literal.key", "quote'key", "back\\slash"},
				Operator: state.WhereOperator(c.Operator), Values: c.Values}
			if index%2 != 0 {
				condition.Property = []string{"scalar"}
			}
			root := &behavioral.Rule{Record: condition}
			nulls := c.Nulls(len(values))
			want := make([]behavioral.Truth, len(values))
			for i := range want {
				want[i] = behavioral.NoMatch
				if slices.Contains(c.Matches, i) {
					want[i] = behavioral.Match
				} else if slices.Contains(nulls, i) {
					want[i] = behavioral.Unknown
				}
			}
			result := fixture.query(t, root, behavioral.Page{Limit: 100})
			assertBehavioralTruths(t, result, want)
			if !covered[c.Operator] && !explainedOperators[c.Operator] {
				explained := fixture.explain(t, root, 1)
				if len(explained.Nodes) != 1 || explained.Nodes[0].Truth != want[0] {
					t.Fatalf("expected JSON state explanation %v, got %+v", want[0], explained)
				}
				explainedOperators[c.Operator] = true
			}
		})
		covered[c.Operator] = true
	}
	if len(covered) != 26 {
		t.Fatalf("expected 26 Record operator families, got %d", len(covered))
	}
	root := &behavioral.Rule{Record: &state.WhereCondition{Property: []string{"document"}, Operator: state.OpExists}}
	full := fixture.query(t, root, behavioral.Page{Limit: 100, MatchingOnly: true})
	empty := fixture.query(t, root, behavioral.Page{First: 100, Limit: 1, MatchingOnly: true})
	if len(empty.Profiles) != 0 || full.Match != empty.Match || full.NoMatch != empty.NoMatch ||
		full.Unknown != empty.Unknown {
		t.Fatalf("expected empty page preserving totals %+v, got %+v", full, empty)
	}
	// Pure Record must still execute when no event table exists.
	execBehavioralFixture(t, fixture.warehouse, "DROP VIEW events; DROP TABLE krenalis_events")
	afterDrop := fixture.query(t, root, behavioral.Page{Limit: 100, MatchingOnly: true})
	if !reflect.DeepEqual(afterDrop, full) {
		t.Fatalf("expected pure Record independent of events, got %+v", afterDrop)
	}
	t.Logf("Record budget: native_cases=%d JSON_cases=%d query=%d explanation=%d legacy=%d", len(cases),
		len(jsonCases), fixture.queries, fixture.explains, fixture.legacy)

}
