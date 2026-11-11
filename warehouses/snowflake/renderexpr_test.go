// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestArrayContains checks queries and counts for arrays containing or missing
// a given value, including null and empty arrays.
func TestArrayContains(t *testing.T) {
	warehouse, db := newTestSnowflakeWarehouse(t)
	instant := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	offsetInstant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	// Use fixed-point JSON so Snowflake stores this value as an exact INTEGER.
	integerBytes, err := decimal.MustParse("123456789012345678901234567890123456790").Binary(0)
	if err != nil {
		t.Fatalf("expected integer fixture encoding, got %v", err)
	}
	integer39, err := decimal.Binary(integerBytes, 76, 0)
	if err != nil {
		t.Fatalf("expected fixed-point integer fixture, got %v", err)
	}
	for i, test := range []struct {
		name  string
		typ   types.Type
		value any
		other any
	}{
		{"string", types.String(), "f'o\\o", "bar"},
		{"int32", types.Int(32), -2147483648, 2147483647},
		{"int64", types.Int(64), 9223372036854775807, -9223372036854775808},
		{"uint64", types.Int(64).Unsigned(), uint(18446744073709551615), uint(1)},
		{"decimal", types.Decimal(18, 4), decimal.MustParse("1.25"), decimal.MustParse("1")},
		{"decimal-small", types.Decimal(18, 8), decimal.MustParse("0.00000001"), decimal.MustParse("0.00000002")},
		{"decimal38", types.Decimal(38, 0), decimal.MustParse("12345678901234567890123456789012345678"), decimal.MustParse("1")},
		{"decimal39", types.Decimal(76, 0), decimal.MustParse("123456789012345678901234567890123456789"), integer39},
		{"decimal39-swapped", types.Decimal(76, 0), integer39, decimal.MustParse("123456789012345678901234567890123456789")},
		{"decimal-negative-zero", types.Decimal(76, 0), decimal.MustParse("-0"), decimal.MustParse("1")},
		{"float32", types.Float(32), float64(float32(1.2345678)), 1.5},
		{"float64", types.Float(64), 0.1, 1.5},
		{"datetime", types.DateTime(), instant, instant.Add(time.Hour)},
		// The same local date and time with different offsets must not match.
		{"datetime-positive", types.DateTime(), offsetInstant.In(time.FixedZone("", 7200)), offsetInstant.Add(2 * time.Hour)},
		{"datetime-negative", types.DateTime(), offsetInstant.In(time.FixedZone("", -18000)), offsetInstant.Add(-5 * time.Hour)},
		{"date", types.Date(), instant, instant.AddDate(0, 0, 1)},
		{"time", types.Time(), clock, clock.Add(time.Hour)},
		{"year", types.Year(), 2026, 2025},
		{"uuid", types.UUID(), "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"},
		{"ip", types.IP(), "192.0.2.1", "192.0.2.2"},
	} {
		t.Run(test.name, func(t *testing.T) {

			idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
			column := warehouses.Column{Name: "value", Type: types.Array(test.typ), Nullable: true}
			table := warehouses.Table{
				Name:    fmt.Sprintf("array_contains_%d", i),
				Columns: []warehouses.Column{idColumn, column},
				Keys:    []string{"id"},
			}
			mustExecSQL(t, db, "CREATE TABLE "+quoteIdent(table.Name)+" ("+quoteIdent(idColumn.Name)+
				" INTEGER PRIMARY KEY, "+quoteIdent(column.Name)+" "+typeToSnowflakeType(column.Type)+")")
			err := warehouse.Merge(t.Context(), table, [][]any{
				{0, nil},
				{1, []any{}},
				{2, []any{test.value}},
				{3, []any{test.other}},
				{4, []any{test.other, test.value, test.value}},
			}, nil)
			if err != nil {
				t.Fatalf("expected array fixture merge, got %v", err)
			}

			for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
				name := "contains"
				if op == warehouses.OpDoesNotContain {
					name = "does_not_contain"
				}
				t.Run(name, func(t *testing.T) {

					expr := warehouses.NewBaseExpr(column, op, test.value)
					want := []int{2, 4}
					if op == warehouses.OpDoesNotContain {
						want = []int{1, 3}
					}
					query := warehouses.RowQuery{
						Table: table.Name, Columns: []warehouses.Column{idColumn},
						Where: expr, OrderBy: []warehouses.Column{idColumn},
					}
					rows, total, err := warehouse.Query(t.Context(), query, true)
					if err != nil {
						t.Fatalf("expected array query, got %v", err)
					}
					defer rows.Close()
					var got []int
					row := make([]any, 1)
					for rows.Next() {
						err = rows.Scan(row...)
						if err != nil {
							t.Fatalf("expected array query row, got %v", err)
						}
						got = append(got, row[0].(int))
					}
					err = rows.Err()
					if err != nil {
						t.Fatalf("expected complete array query, got %v", err)
					}
					if !slices.Equal(got, want) || total != len(want) {
						t.Errorf("expected ids %v and total %d, got %v and %d", want, len(want), got, total)
					}

					counts, err := warehouse.Counts(t.Context(), table.Name, []warehouses.Expr{expr})
					if err != nil {
						t.Fatalf("expected array counts, got %v", err)
					}
					if !slices.Equal(counts, []int{len(want)}) {
						t.Errorf("expected counts [%d], got %v", len(want), counts)
					}

					var b strings.Builder
					err = renderExpr(&b, expr)
					if err != nil {
						t.Fatalf("expected array expression, got %v", err)
					}
					statement := "SELECT " + quoteIdent(idColumn.Name) + ", (" + b.String() + ") FROM " +
						quoteIdent(table.Name) + " ORDER BY " + quoteIdent(idColumn.Name)
					truthRows, err := db.QueryContext(t.Context(), statement)
					if err != nil {
						t.Fatalf("expected array truth query, got %v", err)
					}
					defer truthRows.Close()
					count := 0
					for truthRows.Next() {
						var id int
						var truth sql.NullBool
						if err := truthRows.Scan(&id, &truth); err != nil {
							t.Fatalf("expected array truth row, got %v", err)
						}
						wantTruth := sql.NullBool{Bool: slices.Contains(want, id), Valid: id != 0}
						if truth != wantTruth {
							t.Errorf("expected id %d truth %v, got %v", id, wantTruth, truth)
						}
						count++
					}
					err = truthRows.Err()
					if err != nil {
						t.Fatalf("expected complete array truth query, got %v", err)
					}
					if count != 5 {
						t.Errorf("expected 5 array truth rows, got %d", count)
					}

				})
			}

		})
	}
}

func Test_renderExpr(t *testing.T) {
	column := warehouses.Column{Name: "value", Type: types.Decimal(3, 2)}
	array := warehouses.Column{Name: "value", Type: types.Array(types.Decimal(3, 2))}
	tests := []struct {
		name    string
		expr    warehouses.Expr
		query   string
		wantErr string
	}{
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "qwerty"),
			query: `"ID" = 'qwerty'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "age", Type: types.Float(32)}, warehouses.OpIsNot, 18.0),
			query: `"AGE" <> 18`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.String()}, warehouses.OpIs, warehouses.Column{Name: "b", Type: types.String()}),
			query: `"A" = "B"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.JSON()}, warehouses.OpIs, json.Value(`{"foo":2,"boo":true}`)),
			query: `COALESCE("VALUES" = PARSE_JSON('{"foo":2,"boo":true}'), FALSE)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OpIsLessThan, 6.5),
			query: `"WEIGHT" < 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OpIsLessThanOrEqualTo, 6.5),
			query: `"WEIGHT" <= 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OpIsGreaterThan, 6.5),
			query: `"WEIGHT" > 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OpIsGreaterThanOrEqualTo, decimal.MustInt(3289)),
			query: `"COUNT" >= 3289`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253621, time.UTC)),
			query: `"TIMESTAMP" < '1900-01-02 23:32:11.940253621'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsOnOrBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253621, time.UTC)),
			query: `"TIMESTAMP" <= '1900-01-02 23:32:11.940253621'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.Date()}, warehouses.OpIsAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"TIMESTAMP" > '1900-01-02'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.Date()}, warehouses.OpIsOnOrAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"TIMESTAMP" >= '1900-01-02'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Float(64)}, warehouses.OpIsBetween, 5.0, 10.0),
			query: `"ID" BETWEEN 5 AND 10`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpContains, "foo"),
			query: `CONTAINS("NAME", 'foo')`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpDoesNotContain, "boo"),
			query: `NOT CONTAINS("NAME", 'boo')`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Int(32))}, warehouses.OpContains, 5),
			query: `ARRAY_CONTAINS(PARSE_JSON('5'), "VALUES")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Int(32))}, warehouses.OpDoesNotContain, 7),
			query: `NOT ARRAY_CONTAINS(PARSE_JSON('7'), "VALUES")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.String())}, warehouses.OpContains, "foo"),
			query: `ARRAY_CONTAINS(PARSE_JSON('"foo"'), "VALUES")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.String())}, warehouses.OpDoesNotContain, "boo"),
			query: `NOT ARRAY_CONTAINS(PARSE_JSON('"boo"'), "VALUES")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(76, 0))}, warehouses.OpContains, decimal.MustParse("123456789012345678901234567890123456789")),
			query: `(ARRAY_SIZE(FILTER("VALUES", krenalis_element -> IFF(TYPEOF(krenalis_element) = 'INTEGER', TO_JSON(krenalis_element) = '123456789012345678901234567890123456789', FALSE))) > 0)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(18, 4))}, warehouses.OpContains, decimal.MustParse("1.25")),
			query: `(ARRAY_SIZE(FILTER("VALUES", krenalis_element -> IFF(TYPEOF(krenalis_element) = 'INTEGER', FALSE, TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1.25')))) > 0)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(76, 0))}, warehouses.OpContains, decimal.MustParse("1e2")),
			query: `(ARRAY_SIZE(FILTER("VALUES", krenalis_element -> IFF(TYPEOF(krenalis_element) = 'INTEGER', TO_JSON(krenalis_element) = '100', TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1e2')))) > 0)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Float(32)}, warehouses.OpIsOneOf, 3.5, 9.2, 5.0),
			query: `"ID" IN (3.5,9.2,5)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.Float(64)}, warehouses.OpIsOneOf, 5.3, 12.6, 9.0),
			query: `"A" IN (5.3,12.6,9)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpStartsWith, "foo"),
			query: `STARTSWITH("NAME", 'foo')`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpEndsWith, "foo"),
			query: `ENDSWITH("NAME", 'foo')`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "active", Type: types.Boolean()}, warehouses.OpIsTrue),
			query: `"ACTIVE"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "active", Type: types.Boolean()}, warehouses.OpIsFalse),
			query: `NOT "ACTIVE"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsTrue),
			query: `AS_BOOLEAN("PROPERTIES") IS NOT DISTINCT FROM TRUE`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsFalse),
			query: `AS_BOOLEAN("PROPERTIES") IS NOT DISTINCT FROM FALSE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"enabled"},
				Operator: warehouses.OpIsTrue,
			},
			query: `AS_BOOLEAN(GET("PROPERTIES", 'enabled')) IS NOT DISTINCT FROM TRUE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"archived"},
				Operator: warehouses.OpIsFalse,
			},
			query: `AS_BOOLEAN(GET("PROPERTIES", 'archived')) IS NOT DISTINCT FROM FALSE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a", "b"},
				Operator: warehouses.OpIsTrue,
			},
			query: `AS_BOOLEAN(GET(GET("PROPERTIES", 'a'), 'b')) IS NOT DISTINCT FROM TRUE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a", "b"},
				Operator: warehouses.OpIsFalse,
			},
			query: `AS_BOOLEAN(GET(GET("PROPERTIES", 'a'), 'b')) IS NOT DISTINCT FROM FALSE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a.b"},
				Operator: warehouses.OpIsTrue,
			},
			query: `AS_BOOLEAN(GET("PROPERTIES", 'a.b')) IS NOT DISTINCT FROM TRUE`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a'b", "c'd"},
				Operator: warehouses.OpIsFalse,
			},
			query: `AS_BOOLEAN(GET(GET("PROPERTIES", 'a\'b'), 'c\'d')) IS NOT DISTINCT FROM FALSE`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNull),
			query: `"ID" IS NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNotNull),
			query: `"ID" IS NOT NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpExists),
			query: `"ID" IS NOT NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpDoesNotExist),
			query: `"ID" IS NULL`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"TIMESTAMP" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"TIMESTAMP" > '1700-01-02 23:32:11' AND "TIMESTAMP" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpOr,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"TIMESTAMP" > '1700-01-02 23:32:11' OR "TIMESTAMP" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_42"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_50"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_60"),
					}),
					warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OpIs, decimal.MustInt(100)),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OpIs, decimal.MustInt(200)),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OpIs, decimal.MustInt(300)),
					}),
				}),
			query: `("ID" = 'abc_42' OR "ID" = 'abc_50' OR "ID" = 'abc_60') AND ("COUNT" = 100 OR "COUNT" = 200 OR "COUNT" = 300)`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpOr,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String(), Nullable: true}, warehouses.OpIsNotEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.Map(types.String())}, warehouses.OpIsNotEmpty),
				}),
			query: `("TYPE" IS NOT NULL AND "TYPE" <> '') OR COALESCE(("PROPERTIES" IS NULL OR IS_NULL_VALUE("PROPERTIES") OR AS_VARCHAR("PROPERTIES") = '' OR ARRAY_SIZE(AS_ARRAY("PROPERTIES")) = 0 OR ARRAY_SIZE(OBJECT_KEYS(AS_OBJECT("PROPERTIES"))) = 0), FALSE) OR ("SCORES" IS NULL OR "SCORES" = ARRAY_CONSTRUCT()) OR "PROPERTIES" <> OBJECT_CONSTRUCT()`,
		},
		{
			name:  "decimal zero",
			expr:  warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.Decimal{}),
			query: `"VALUE" = 0`,
		},
		{
			name:  "decimal trailing zeros",
			expr:  warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1.2300")),
			query: `"VALUE" = 1.23`,
		},
		{
			name:  "decimal maximum precision exceeds column precision",
			expr:  warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1e75")),
			query: `"VALUE" = 1e75`,
		},
		{
			name:  "decimal maximum scale exceeds column scale",
			expr:  warehouses.NewBaseExpr(column, warehouses.OpIsLessThan, decimal.MustParse("1e-37")),
			query: `"VALUE" < 1e-37`,
		},
		{
			name:  "decimal array maximum precision",
			expr:  warehouses.NewBaseExpr(array, warehouses.OpContains, decimal.MustParse("1e75")),
			query: `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> IFF(TYPEOF(krenalis_element) = 'INTEGER', TO_JSON(krenalis_element) = '1000000000000000000000000000000000000000000000000000000000000000000000000000', TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1e75')))) > 0)`,
		},
		{
			name:  "decimal array maximum scale",
			expr:  warehouses.NewBaseExpr(array, warehouses.OpDoesNotContain, decimal.MustParse("1e-37")),
			query: `NOT (ARRAY_SIZE(FILTER("VALUE", krenalis_element -> IFF(TYPEOF(krenalis_element) = 'INTEGER', FALSE, TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1e-37')))) > 0)`,
		},
		{
			name:    "decimal excessive precision",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1e76")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal excessive scale",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIsLessThan, decimal.MustParse("1e-38")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal large positive exponent",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1e999999")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal large negative exponent",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1e-999999")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal between invalid second operand",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIsBetween, decimal.Decimal{}, decimal.MustParse("1e-38")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal membership invalid second operand",
			expr:    warehouses.NewBaseExpr(column, warehouses.OpIsOneOf, decimal.Decimal{}, decimal.MustParse("1e76")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal array excessive precision",
			expr:    warehouses.NewBaseExpr(array, warehouses.OpContains, decimal.MustParse("1e76")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name:    "decimal array excessive scale",
			expr:    warehouses.NewBaseExpr(array, warehouses.OpDoesNotContain, decimal.MustParse("1e-38")),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name: "decimal invalid nested expression after valid expression",
			expr: warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
				warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.Decimal{}),
				warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
					warehouses.NewBaseExpr(column, warehouses.OpIs, decimal.MustParse("1e76")),
				}),
			}),
			wantErr: "decimal operand exceeds precision 76 or scale 37",
		},
		{
			name: "JSON array contains",
			expr: warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.JSON())},
				warehouses.OpContains, warehouses.JSONConditionValue{String: "foo"}),
			query: `COALESCE((ARRAY_SIZE(FILTER("VALUES", krenalis_json_element -> CASE WHEN IS_VARCHAR(krenalis_json_element) THEN COLLATE(AS_VARCHAR(krenalis_json_element), 'utf8') = 'foo' ELSE FALSE END)) > 0), FALSE)`,
		},
		{
			name: "JSON array does not contain",
			expr: warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.JSON())},
				warehouses.OpDoesNotContain, warehouses.JSONConditionValue{String: "foo"}),
			query: `NOT COALESCE((ARRAY_SIZE(FILTER("VALUES", krenalis_json_element -> CASE WHEN IS_VARCHAR(krenalis_json_element) THEN COLLATE(AS_VARCHAR(krenalis_json_element), 'utf8') = 'foo' ELSE FALSE END)) > 0), FALSE)`,
		},
		{
			name: "JSON numeric text without number",
			expr: warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.JSON()}, warehouses.OpIs,
				warehouses.JSONConditionValue{String: "1e999999"}),
			query: `COALESCE(CASE WHEN IS_VARCHAR("VALUES") THEN COLLATE(AS_VARCHAR("VALUES"), 'utf8') = '1e999999' ELSE FALSE END, FALSE)`,
		},
		{
			name: "structural JSON number outside decimal limits",
			expr: warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.JSON()}, warehouses.OpIs,
				json.Value("1e76")),
			query: `COALESCE("VALUES" = PARSE_JSON('1e76'), FALSE)`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b strings.Builder
			err := renderExpr(&b, test.expr)
			if err != nil {
				if test.wantErr == "" {
					t.Fatalf("expected query %q, got error %q", test.query, err)
				}
				if err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got %q", test.wantErr, err.Error())
				}
				return
			}
			if test.wantErr != "" {
				t.Fatalf("expected error %q, got nil and SQL %q", test.wantErr, b.String())
			}
			if got := b.String(); test.query != got {
				t.Fatalf("expected query %q, got %q", test.query, got)
			}
		})
	}
}
