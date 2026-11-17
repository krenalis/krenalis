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

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

type unifiedRecordFixture struct {
	store     *Store
	warehouse warehouses.Warehouse
	at        time.Time
	queries   int
	explains  int
	legacy    int
}

func newUnifiedRecordFixture(t *testing.T, platform string, columns []warehouses.Column) *unifiedRecordFixture {
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
	return &unifiedRecordFixture{store: store, warehouse: dw,
		at: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
}

func (fixture *unifiedRecordFixture) explain(t *testing.T, root *unified.Rule, profile int) *unified.Explanation {
	t.Helper()
	fixture.explains++
	ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordExplanationTag)
	result, err := fixture.store.ExplainUnifiedProfile(ctx, root, unified.Bindings{}, fixture.at,
		unifiedTestID(profile))
	if err != nil {
		t.Fatalf("expected Record explanation, got %v", err)
	}
	if result == nil || !result.At.Equal(fixture.at) {
		t.Fatalf("expected explanation at %s, got %+v", fixture.at, result)
	}
	return result
}

func (fixture *unifiedRecordFixture) query(t *testing.T, root *unified.Rule, page unified.Page) *unified.Result {
	t.Helper()
	fixture.queries++
	ctx := gosnowflake.WithQueryTag(t.Context(), snowflaketester.RecordQueryTag)
	result, err := fixture.store.UnifiedProfiles(ctx, root, unified.Bindings{}, fixture.at, page)
	if err != nil {
		t.Fatalf("expected Record query, got %v", err)
	}
	if !result.At.Equal(fixture.at) {
		t.Fatalf("expected one evaluation time %s, got %s", fixture.at, result.At)
	}
	return result
}

// TestUnifiedRecordConformancePostgreSQL checks all original Record operators.
func TestUnifiedRecordConformancePostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testUnifiedRecordConformance(t, "PostgreSQL")
}

// TestUnifiedRecordConformanceSnowflake uses an isolated Snowflake warehouse.
func TestUnifiedRecordConformanceSnowflake(t *testing.T) {
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
	testUnifiedRecordConformance(t, "Snowflake")
}

func testUnifiedRecordConformance(t *testing.T, platform string) {

	t.Helper()
	cases := append(warehouseexpr.ValueCases(), warehouseexpr.SQLFloatCases()...)
	columns := make([]warehouses.Column, len(cases))
	for i, c := range cases {
		columns[i] = warehouses.Column{Name: fmt.Sprintf("v%d", i), Type: c.Type, Nullable: true}
	}
	columns = append(columns, warehouses.Column{Name: "document", Type: types.JSON(), Nullable: true},
		warehouses.Column{Name: "scalar", Type: types.JSON(), Nullable: true})
	fixture := newUnifiedRecordFixture(t, platform, columns)
	table := warehouses.Table{Name: "krenalis_profiles_0", Keys: []string{"_kpid"}, Columns: append(
		[]warehouses.Column{{Name: "_kpid", Type: types.UUID()}, {Name: "_anonymous_count", Type: types.Int(32)},
			{Name: "_recognized_count", Type: types.Int(32)}, {Name: "_updated_at", Type: types.DateTime()}},
		columns...)}
	present := []any{unifiedTestID(1), 0, 1, fixture.at}
	absent := []any{unifiedTestID(2), 0, 1, fixture.at}
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
		var children []unified.Rule
		var matches []bool
		for i, c := range cases {
			if c.Operator == operator {
				children = append(children, unified.Rule{Record: &state.WhereCondition{
					Property: []string{columns[i].Name}, Operator: state.WhereOperator(operator), Values: c.Values}})
				matches = append(matches, c.Matches)
			}
		}
		if len(children) == 0 {
			continue
		}
		covered[operator] = true
		t.Run(fmt.Sprintf("native/%d", operator), func(t *testing.T) {

			root := &unified.Rule{Operator: state.OpAnd, Children: children}
			truth := unified.Match
			for _, match := range matches {
				if !match {
					truth = unified.NoMatch
				}
			}
			result := fixture.query(t, root, unified.Page{Limit: 100})
			assertUnifiedTruths(t, result, []unified.Truth{truth, unified.Unknown})
			for profile := 1; profile <= 2; profile++ {
				explained := fixture.explain(t, root, profile)
				if len(explained.Nodes) != len(children)+1 {
					t.Fatalf("expected %d explanation nodes, got %d", len(children)+1, len(explained.Nodes))
				}
				for i, match := range matches {
					want := unified.Unknown
					if profile == 1 {
						want = unified.NoMatch
						if match {
							want = unified.Match
						}
					}
					if explained.Nodes[i+1].Truth != want {
						t.Fatalf("expected native leaf %d truth %v for profile %d, got %+v",
							i, want, profile, explained.Nodes[i+1])
					}
				}
			}
			root.Operator = state.OpOr
			truth = unified.NoMatch
			for _, match := range matches {
				if match {
					truth = unified.Match
				}
			}
			result = fixture.query(t, root, unified.Page{Limit: 100})
			assertUnifiedTruths(t, result, []unified.Truth{truth, unified.Unknown})
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
			if len(legacy) == 1 && legacy[0]["_kpid"] != unifiedTestID(1) {
				t.Fatalf("expected present profile in legacy membership, got %v", legacy)
			}

		})
	}

	// Reuse every JSON truth table in both scalar and literal nested form.
	execUnifiedFixture(t, fixture.warehouse, "TRUNCATE TABLE krenalis_profiles_0")
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
		rows = append(rows, []any{unifiedTestID(i + 1), 0, 1, fixture.at, document, scalar})
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
			root := &unified.Rule{Record: condition}
			nulls := c.Nulls(len(values))
			want := make([]unified.Truth, len(values))
			for i := range want {
				want[i] = unified.NoMatch
				if slices.Contains(c.Matches, i) {
					want[i] = unified.Match
				} else if slices.Contains(nulls, i) {
					want[i] = unified.Unknown
				}
			}
			result := fixture.query(t, root, unified.Page{Limit: 100})
			assertUnifiedTruths(t, result, want)
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
	root := &unified.Rule{Record: &state.WhereCondition{Property: []string{"document"}, Operator: state.OpExists}}
	full := fixture.query(t, root, unified.Page{Limit: 100, MatchingOnly: true})
	empty := fixture.query(t, root, unified.Page{First: 100, Limit: 1, MatchingOnly: true})
	if len(empty.Profiles) != 0 || full.Match != empty.Match || full.NoMatch != empty.NoMatch ||
		full.Unknown != empty.Unknown {
		t.Fatalf("expected empty page preserving totals %+v, got %+v", full, empty)
	}
	// Pure Record must still execute when no event table exists.
	execUnifiedFixture(t, fixture.warehouse, "DROP VIEW events; DROP TABLE krenalis_events")
	afterDrop := fixture.query(t, root, unified.Page{Limit: 100, MatchingOnly: true})
	if !reflect.DeepEqual(afterDrop, full) {
		t.Fatalf("expected pure Record independent of events, got %+v", afterDrop)
	}
	t.Logf("Record budget: native_cases=%d JSON_cases=%d query=%d explanation=%d legacy=%d", len(cases),
		len(jsonCases), fixture.queries, fixture.explains, fixture.legacy)

}
