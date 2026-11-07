// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestProfileFilterDialect preserves all 26 original Record operators.
func TestProfileFilterDialect(t *testing.T) {
	dialect := profileFilterDialect()
	for _, malformed := range []warehouses.Expr{
		warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: types.String()}, warehouses.OpIs),
		(*warehouses.BaseExpr)(nil),
	} {
		_, err := dialect.Record(malformed)
		if err == nil {
			t.Fatal("expected malformed prepared expression error, got success")
		}
	}
	covered := map[warehouses.Operator]bool{}
	check := func(t *testing.T, expr warehouses.Expr) {
		t.Helper()
		var original strings.Builder
		err := renderExpr(&original, expr)
		if err != nil {
			t.Fatalf("expected original renderer success, got %v", err)
		}
		actual, err := dialect.Record(expr)
		if err != nil {
			t.Fatalf("expected compiled Record renderer success, got %v", err)
		}
		if original.String() != actual {
			t.Fatalf("expected original Record expression %s, got %s", original.String(), actual)
		}
		population := warehouses.ProfileSource{Table: "profiles",
			Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}
		statement, err := profilefilter.BuildExplanation(warehouses.Filter{Root: &warehouses.RecordFilter{Expr: expr}},
			population, "00000000-0000-0000-0000-000000000001", dialect)
		if err != nil {
			t.Fatalf("expected original Record renderer in explanation, got %v", err)
		}
		if !strings.Contains(statement.SQL, actual) {
			t.Fatalf("expected original Record expression %s in explanation, got %s", actual, statement.SQL)
		}
	}
	for _, c := range warehouseexpr.ValueCases() {
		t.Run(c.Name, func(t *testing.T) {
			check(t, warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: c.Type, Nullable: true}, c.Operator,
				c.Values...))
		})
		covered[c.Operator] = true
	}
	_, cases := warehouseexpr.JSONCases()
	for _, c := range cases {
		t.Run("json/"+c.Name, func(t *testing.T) {
			expr := warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: types.JSON(), Nullable: true},
				c.Operator, c.Values...)
			expr.Keys = []string{"literal.key", "quote'key", "back\\slash"}
			expr.JSONComparison = warehouses.JSONTypedComparison
			check(t, expr)
		})
		covered[c.Operator] = true
	}
	if len(covered) != 26 {
		t.Fatalf("expected all 26 existing operators, got %d", len(covered))
	}
	for _, typ := range []types.Type{types.String(), types.DateTime()} {
		parameter, err := dialect.Parameter(2, typ)
		if err != nil || parameter == "" {
			t.Fatalf("expected physical typed parameter, got %q, %v", parameter, err)
		}
	}
	_, err := dialect.Parameter(0, types.String())
	if err == nil {
		t.Fatal("expected invalid parameter index error, got success")
	}
	_, err = dialect.Parameter(1, types.Int(64))
	if err == nil {
		t.Fatal("expected unsupported argument type error, got success")
	}
	_, err = dialect.KnownText(warehouses.Column{Name: "slot", Type: types.String()}, nil)
	if err == nil {
		t.Fatal("expected invalid JSON binding error, got success")
	}
	_, err = (&PostgreSQL{}).ProfileOutcomes(t.Context(), warehouses.Filter{}, warehouses.ProfileOutcomeRequest{})
	if err == nil {
		t.Fatal("expected envelope error without database access, got success")
	}
}

// TestProfileFilterSQL checks real adapters and the shared physical compiler.
func TestProfileFilterSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.FixedZone("offset", 3600))
	source := warehouses.OccurrenceSource{Table: "krenalis_events", Profile: warehouses.Column{Name: "kpid",
		Type: types.UUID(), Nullable: true}, Identity: warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10, First: 20, MatchingOnly: true}
	selector := warehouses.NewBaseExpr(warehouses.Column{Name: "event", Type: types.String(), Nullable: true},
		warehouses.OpIs, "Purchase'")
	count := &warehouses.IndependentOccurrenceCount{Source: source, Start: end.AddDate(0, -1, 0), End: end,
		Accept: []warehouses.CountRange{{Lower: 1}}, Minimum: 1,
		Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: []warehouses.RowQualification{
			&warehouses.RecordQualification{Expr: selector},
			&warehouses.KnownTextEqual{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys: []string{"currency"}, Value: "EUR"},
		}}}
	filter := warehouses.Filter{Root: &warehouses.FilterGroup{Operator: warehouses.OpOr,
		Children: []warehouses.FilterCondition{count, count,
			&warehouses.RecordFilter{Expr: warehouses.NewBaseExpr(
				warehouses.Column{Name: "event", Type: types.String(), Nullable: true}, warehouses.OpIs, "Refund")}}}}
	dialect := profileFilterDialect()
	query, err := profilefilter.Build(filter, request, dialect)
	if err != nil {
		t.Fatalf("expected prepared SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected exact transport arguments, got %v", err)
	}
	if len(values) != 6 || values[1] != "2026-01-31 23:00:00.123456" || values[2] != "EUR" || values[5] != "EUR" {
		t.Fatalf("expected ordered exact UTC bounds and literals, got %v", values)
	}
	if len(query.Columns) != 8 || !strings.Contains(query.SQL, "GROUP BY occurrence_id") ||
		!strings.Contains(query.SQL, "period_min<>period_max") || !strings.Contains(query.SQL,
		"ORDER BY profile_id LIMIT 10 OFFSET 20") || !strings.Contains(query.SQL, "WHERE truth=TRUE") {
		t.Fatalf("expected identity checks and complete page protocol, got %s", query.SQL)
	}
	if !profileFilterVerified && strings.Count(query.SQL, "?") != len(values) {
		t.Fatalf("expected positional binds used exactly once, got %s", query.SQL)
	}
	_, err = profilefilter.Build(filter, request, profilefilter.Dialect{})
	if err == nil {
		t.Fatal("expected missing dialect error, got success")
	}
}

// TestProfileHistorySQL binds only the exclusive upper edge for all history.
func TestProfileHistorySQL(t *testing.T) {

	end := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.FixedZone("offset", 3600))
	count := &warehouses.IndependentOccurrenceCount{
		Source: warehouses.OccurrenceSource{Table: "krenalis_events",
			Profile:   warehouses.Column{Name: "kpid", Type: types.UUID(), Nullable: true},
			Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
			Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		UnboundedStart: true, End: end, Accept: []warehouses.CountRange{{Lower: 1}},
	}
	filter := warehouses.Filter{Root: count}
	population := warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}
	query, err := profilefilter.Build(filter, warehouses.ProfileOutcomeRequest{Population: population, Limit: 10},
		profileFilterDialect())
	if err != nil {
		t.Fatalf("expected history SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected history arguments, got %v", err)
	}
	if len(values) != 1 || values[0] != "2026-01-31 23:00:00.123456" ||
		strings.Contains(query.SQL, "lower_bound") || !strings.Contains(query.SQL, "<(SELECT upper_bound FROM bounds0)") {
		t.Fatalf("expected only exclusive T without a fabricated lower bound, got %s and %v", query.SQL, values)
	}
	explained, err := profilefilter.BuildExplanation(filter, population,
		"00000000-0000-0000-0000-000000000001", profileFilterDialect())
	if err != nil {
		t.Fatalf("expected history explanation SQL, got %v", err)
	}
	values, err = explained.Values()
	if err != nil {
		t.Fatalf("expected history explanation arguments, got %v", err)
	}
	if len(values) != 2 || values[1] != "2026-01-31 23:00:00.123456" ||
		strings.Contains(explained.SQL, "lower_bound") {
		t.Fatalf("expected target and exclusive T without a lower bound, got %s and %v", explained.SQL, values)
	}

}

// TestProfileRecordSQL projects leaves only within the population table.
func TestProfileRecordSQL(t *testing.T) {

	dialect := profileFilterDialect()
	leaf := &warehouses.RecordFilter{Expr: warehouses.NewBaseExpr(
		warehouses.Column{Name: "profile_id", Type: types.String(), Nullable: true}, warehouses.OpIs, "customer")}
	filter := warehouses.Filter{Root: &warehouses.FilterGroup{Operator: warehouses.OpOr,
		Children: []warehouses.FilterCondition{leaf, leaf}}}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}

	query, err := profilefilter.Build(filter, request, dialect)
	if err != nil {
		t.Fatalf("expected profile Record SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected no Record parameter conversion, got %v", err)
	}

	begin := strings.Index(query.SQL, "population AS (")
	end := strings.Index(query.SQL, "evaluated AS (")
	if begin < 0 || end <= begin {
		t.Fatalf("expected isolated population projection, got %s", query.SQL)
	}
	population := query.SQL[begin:end]
	rest := query.SQL[end:]
	if len(values) != 0 || !strings.Contains(population, "AS "+quoteIdent("$profile_id")) ||
		!strings.Contains(population, quoteIdent("profile_id")+" = 'customer'") ||
		!strings.Contains(population, "AS "+quoteIdent("$record0")) ||
		!strings.Contains(rest, "p."+quoteIdent("$record1")) ||
		strings.Contains(population, "WHERE") || strings.Contains(population, "JOIN") ||
		strings.Contains(rest, "'customer'") || strings.Contains(query.SQL, "SELECT *") {
		t.Fatalf("expected isolated nullable leaves and collision-free aliases, got %s and %v", query.SQL, values)
	}

}

// TestProfileProjectionSQL checks the common projection and stable page shape.
func TestProfileProjectionSQL(t *testing.T) {
	column := warehouses.Column{Name: "profile_id", Type: types.String(), Nullable: true}
	for _, desc := range []bool{false, true} {
		request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
			Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 2, First: 1, MatchingOnly: true,
			Columns: []warehouses.Column{column}, OrderBy: []warehouses.Column{column}, OrderDesc: desc}
		dialect := profileFilterDialect()
		query, err := profilefilter.Build(warehouses.Filter{}, request, dialect)
		if err != nil {
			t.Fatalf("expected projected SQL, got %v", err)
		}
		_, err = query.Values()
		if err != nil {
			t.Fatalf("expected projected transport, got %v", err)
		}
		q := dialect.QuoteIdentifier
		order := q("$sort")
		if desc {
			order += " DESC"
		}
		for _, fragment := range []string{
			q("profile_id") + " AS " + q("$value0"),
			"ORDER BY " + order + " NULLS LAST,profile_id LIMIT 2 OFFSET 1",
			"ORDER BY b." + order + " NULLS LAST,b.profile_id",
			"e.failures,(z.bytes>1048576)", "page_size AS", dialect.ValueSize("b." + q("$value0")), "WHERE truth=TRUE",
		} {
			if !strings.Contains(query.SQL, fragment) {
				t.Fatalf("expected projection fragment %s, got %s", fragment, query.SQL)
			}
		}
		if len(query.Columns) != 9 || !query.Columns[8].Nullable || !types.Equal(query.Columns[8].Type, column.Type) {
			t.Fatalf("expected nullable projected metadata-row column, got %+v", query.Columns)
		}
		if query.Columns[7].Name != "payload_exceeded" || query.Columns[7].Nullable || !query.Columns[7].Type.IsBoolean() {
			t.Fatalf("expected a separate non-null payload diagnostic, got %+v", query.Columns[7])
		}
		dialect.ValueSize = nil
		_, err = profilefilter.Build(warehouses.Filter{}, request, dialect)
		if err == nil {
			t.Fatal("expected missing projection adapter failure, got success")
		}
	}
}
