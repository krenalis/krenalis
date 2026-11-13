// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"math"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestUnifiedRecordCollationPostgreSQL checks declared ICU and binary rules.
func TestUnifiedRecordCollationPostgreSQL(t *testing.T) {
	if testing.Short() {
		t.Skip("requires disposable PostgreSQL")
	}
	testUnifiedRecordCollation(t, "PostgreSQL")
}

// TestUnifiedRecordCollationSnowflake checks declared locale and binary rules.
func TestUnifiedRecordCollationSnowflake(t *testing.T) {
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
	testUnifiedRecordCollation(t, "Snowflake")
}

func testUnifiedRecordCollation(t *testing.T, platform string) {

	t.Helper()
	dw := newJSONFilterTestWarehouse(t, platform)
	switch platform {
	case "PostgreSQL":
		execUnifiedFixture(t, dw, `CREATE COLLATION krenalis_en (provider=icu,locale='en-US',deterministic=true);
CREATE COLLATION krenalis_ci (provider=icu,locale='und-u-ks-level2',deterministic=false);
CREATE TABLE profiles (_kpid uuid PRIMARY KEY,"binary" text COLLATE "C",locale text COLLATE krenalis_en,
 insensitive text COLLATE krenalis_ci,real_number double precision);
CREATE TABLE krenalis_events (kpid uuid,message_id text,type text,event text,timestamp timestamp(6),properties jsonb)`)
	case "Snowflake":
		execUnifiedFixture(t, dw, `CREATE TABLE "PROFILES" ("_KPID" VARCHAR(36),"BINARY" VARCHAR COLLATE 'utf8',
 "LOCALE" VARCHAR COLLATE 'en',"INSENSITIVE" VARCHAR COLLATE 'en-ci',"REAL_NUMBER" DOUBLE);
CREATE TABLE "KRENALIS_EVENTS" ("KPID" VARCHAR(36),"MESSAGE_ID" VARCHAR,"TYPE" VARCHAR,"EVENT" VARCHAR,
 "TIMESTAMP" TIMESTAMP_NTZ(6),"PROPERTIES" VARIANT)`)
	default:
		t.Fatalf("expected supported warehouse, got %s", platform)
	}
	store := &Store{mc: newModeCoordinator(state.Normal)}
	store.wh.Store(dw)
	store.columnByProperty.user = map[string]warehouses.Column{
		"_kpid":       {Name: "_kpid", Type: types.UUID()},
		"binary":      {Name: "binary", Type: types.String(), Nullable: true},
		"locale":      {Name: "locale", Type: types.String(), Nullable: true},
		"insensitive": {Name: "insensitive", Type: types.String(), Nullable: true},
		"real_number": {Name: "real_number", Type: types.Float(64).Real(), Nullable: true},
	}
	fixture := &unifiedRecordFixture{store: store, warehouse: dw,
		at: time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)}
	table := warehouses.Table{Name: "profiles", Keys: []string{"_kpid"}, Columns: []warehouses.Column{
		store.columnByProperty.user["_kpid"], store.columnByProperty.user["binary"],
		store.columnByProperty.user["locale"], store.columnByProperty.user["insensitive"],
	}}
	var rows [][]any
	for i, value := range []any{"a", "B", "é", "e\u0301", nil, "Abc", "abc"} {
		rows = append(rows, []any{unifiedTestID(i + 1), value, value, value})
	}
	err := dw.Merge(t.Context(), table, rows, nil)
	if err != nil {
		t.Fatalf("expected collation fixture writes, got %v", err)
	}
	f, m, u := unified.NoMatch, unified.Match, unified.Unknown
	for _, test := range []struct {
		name, property string
		op             state.WhereOperator
		value          string
		want           []unified.Truth
	}{
		{"binary order", "binary", state.OpIsLessThan, "B", []unified.Truth{f, f, f, f, u, m, f}},
		{"locale order", "locale", state.OpIsLessThan, "B", []unified.Truth{m, f, f, f, u, m, m}},
		{"reverse binary order", "binary", state.OpIsGreaterThan, "B", []unified.Truth{m, f, m, m, u, f, m}},
		{"reverse locale order", "locale", state.OpIsGreaterThan, "B", []unified.Truth{f, f, m, m, u, f, f}},
		{"binary equality", "binary", state.OpIs, "abc", []unified.Truth{f, f, f, f, u, f, m}},
		{"case insensitive equality", "insensitive", state.OpIs, "abc", []unified.Truth{f, f, f, f, u, m, m}},
		{"binary membership", "binary", state.OpIsOneOf, "abc", []unified.Truth{f, f, f, f, u, f, m}},
		{"case insensitive membership", "insensitive", state.OpIsOneOf, "abc", []unified.Truth{f, f, f, f, u, m, m}},
		{"binary contains", "binary", state.OpContains, "abc", []unified.Truth{f, f, f, f, u, f, m}},
		{"binary prefix", "binary", state.OpStartsWith, "ab", []unified.Truth{f, f, f, f, u, f, m}},
		{"case insensitive prefix", "insensitive", state.OpStartsWith, "ab", []unified.Truth{f, f, f, f, u, m, m}},
		{"binary suffix", "binary", state.OpEndsWith, "ABC", []unified.Truth{f, f, f, f, u, f, f}},
		{"case insensitive suffix", "insensitive", state.OpEndsWith, "ABC", []unified.Truth{f, f, f, f, u, m, m}},
		{"binary Unicode", "binary", state.OpIs, "é", []unified.Truth{f, f, m, f, u, f, f}},
		{"case insensitive Unicode", "insensitive", state.OpIs, "é", []unified.Truth{f, f, m, m, u, f, f}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := &unified.Rule{Record: &state.WhereCondition{Property: []string{test.property},
				Operator: test.op, Values: []any{test.value}}}
			assertUnifiedTruths(t, fixture.query(t, root, unified.Page{Limit: 10}), test.want)
		})
	}
	for _, test := range []struct {
		name string
		op   state.WhereOperator
	}{
		{"locale Unicode", state.OpIs},
		{"case insensitive contains", state.OpContains},
	} {
		t.Run(test.name, func(t *testing.T) {
			property, value := "locale", "é"
			want := []unified.Truth{f, f, m, m, u, f, f}
			if test.op == state.OpContains {
				property, value = "insensitive", "abc"
				want = []unified.Truth{f, f, f, f, u, m, m}
			}
			root := &unified.Rule{Record: &state.WhereCondition{Property: []string{property},
				Operator: test.op, Values: []any{value}}}
			if platform == "PostgreSQL" && test.op == state.OpIs {
				want[3] = f // Deterministic ICU equality breaks locale ties by bytes.
			}
			assertUnifiedTruths(t, fixture.query(t, root, unified.Page{Limit: 10}), want)
		})
	}
	if platform == "Snowflake" {
		t.Run("unsupported SQL collation", func(t *testing.T) {
			execUnifiedFixture(t, dw, `ALTER TABLE "PROFILES" ADD COLUMN "UNSUPPORTED" VARCHAR COLLATE 'en-pi';
UPDATE "PROFILES" SET "UNSUPPORTED"='abc'`)
			store.columnByProperty.user["unsupported"] = warehouses.Column{Name: "unsupported", Type: types.String()}
			root := &unified.Rule{Record: &state.WhereCondition{Property: []string{"unsupported"},
				Operator: state.OpContains, Values: []any{"a"}}}
			result, err := store.UnifiedProfiles(t.Context(), root, unified.Bindings{}, fixture.at,
				unified.Page{Limit: 10})
			if err != nil {
				if _, ok := errors.AsType[*UnavailableError](err); !ok || result != nil {
					t.Fatalf("expected complete execution failure, got %+v, %v", result, err)
				}
				return
			}
			t.Fatal("expected unsupported SQL collation error, got success")
		})
	}

	// Exercise the four decisive Kleene combinations in the real path.
	events := warehouses.Table{Name: "krenalis_events", Keys: []string{"message_id"}, Columns: []warehouses.Column{
		{Name: "kpid", Type: types.UUID()}, {Name: "message_id", Type: types.String()},
		{Name: "type", Type: types.String()}, {Name: "event", Type: types.String()},
		{Name: "timestamp", Type: types.DateTime()}, {Name: "properties", Type: types.JSON()},
	}}
	err = dw.Merge(t.Context(), events, [][]any{{unifiedTestID(1), "uncertain", "track", "Purchase",
		fixture.at.Add(-time.Hour), json.Value(`{}`)}}, nil)
	if err != nil {
		t.Fatalf("expected uncertain qualification fixture, got %v", err)
	}
	count := unified.Rule{Count: &unified.Count{Scope: unified.Scope{Event: "Purchase", Grain: "registered-message",
		Start: fixture.at.Add(-24 * 30 * time.Hour), End: fixture.at},
		Where: &unified.OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"properties", "flag"},
			Operator: state.OpIsTrue}}, Test: unified.CountTest{Operator: unified.GreaterOrEqual, Value: 1}}}
	for _, test := range []struct {
		name     string
		operator state.WhereLogical
		record   state.WhereCondition
		want     []unified.Truth
	}{
		{"false AND unknown", state.OpAnd, state.WhereCondition{Property: []string{"binary"}, Operator: state.OpIs, Values: []any{"zz"}}, []unified.Truth{f, f, f, f, f, f, f}},
		{"true OR unknown", state.OpOr, state.WhereCondition{Property: []string{"binary"}, Operator: state.OpExists}, []unified.Truth{m, m, m, m, f, m, m}},
		{"unknown AND true", state.OpAnd, state.WhereCondition{Property: []string{"real_number"}, Operator: state.OpIs, Values: []any{0.0}}, []unified.Truth{u, f, f, f, f, f, f}},
		{"unknown OR false", state.OpOr, state.WhereCondition{Property: []string{"real_number"}, Operator: state.OpIs, Values: []any{0.0}}, []unified.Truth{u, u, u, u, u, u, u}},
	} {
		t.Run(test.name, func(t *testing.T) {
			occurrences := count
			countTruth := u
			if test.name == "unknown AND true" {
				known := *count.Count
				known.Where = nil
				occurrences.Count, countTruth = &known, m
			}
			root := &unified.Rule{Operator: test.operator,
				Children: []unified.Rule{{Record: &test.record}, occurrences}}
			assertUnifiedTruths(t, fixture.query(t, root, unified.Page{Limit: 10}), test.want)
			explained := fixture.explain(t, root, 1)
			if len(explained.Nodes) != 3 || explained.Nodes[0].Truth != test.want[0] ||
				explained.Nodes[2].Count == nil || explained.Nodes[2].Truth != countTruth {
				t.Fatalf("expected Kleene explanation, got %+v", explained)
			}
		})
	}

	// A non-finite value in a Real column is a projection failure, not UNKNOWN.
	_, err = warehouses.ValidateFloat("real_number", types.Float(64).Real(), math.NaN())
	if err == nil {
		t.Fatal("expected Real value rejected at the numeric boundary, got success")
	}
	execUnifiedFixture(t, dw, "UPDATE "+quoteUnifiedRecordIdentifier(platform, "profiles")+" SET "+
		quoteUnifiedRecordIdentifier(platform, "real_number")+"=CAST('NaN' AS DOUBLE PRECISION)")
	result, err := store.UnifiedProfiles(t.Context(), nil, unified.Bindings{}, fixture.at,
		unified.Page{Limit: 10, Properties: []string{"real_number"}})
	if err != nil {
		if _, ok := errors.AsType[*UnavailableError](err); !ok || result != nil {
			t.Fatalf("expected no partial Real projection, got %+v, %v", result, err)
		}
	}
	if err == nil {
		t.Fatal("expected Real projection failure, got success")
	}
	t.Logf("Collation budget: query=%d explanation=%d", fixture.queries, fixture.explains)

}
