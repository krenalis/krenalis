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
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestArrayContains checks membership and SQL truth values after a warehouse
// merge, including null and empty arrays.
func TestArrayContains(t *testing.T) {
	warehouse, db := newTestSnowflakeWarehouse(t)
	instant := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	offsetInstant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	for i, test := range []struct {
		name  string
		typ   types.Type
		value any
		other any
	}{
		{"string", types.String(), "foo", "bar"},
		{"int8", types.Int(8), -128, 127},
		{"int16", types.Int(16), -32768, 32767},
		{"int24", types.Int(24), -8388608, 8388607},
		{"int32", types.Int(32), -2147483648, 2147483647},
		{"uint32", types.Int(32).Unsigned(), uint(4294967295), uint(1)},
		{"int64", types.Int(64), 9223372036854775807, -9223372036854775808},
		{"uint64", types.Int(64).Unsigned(), uint(18446744073709551615), uint(1)},
		{"decimal", types.Decimal(18, 4), decimal.MustParse("1.25"), decimal.MustParse("2.50")},
		{"float32", types.Float(32), float64(float32(1.2345678)), 1.5},
		{"float64", types.Float(64), 0.1, 1.5},
		{"datetime", types.DateTime(), instant, instant.Add(time.Hour)},
		{"datetime-positive", types.DateTime(), offsetInstant.In(time.FixedZone("", 7200)), offsetInstant.Add(time.Hour)},
		{"datetime-negative", types.DateTime(), offsetInstant.In(time.FixedZone("", -18000)), offsetInstant.Add(time.Hour)},
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
				t.Run(fmt.Sprint(op), func(t *testing.T) {

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
						if err := rows.Scan(row...); err != nil {
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

// TestArrayContainsRendering checks element encoding for both array operators.
func TestArrayContainsRendering(t *testing.T) {
	instant := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	offsetInstant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	offsetClock := time.Date(2026, 10, 20, 0, 30, 0, 123456000, time.FixedZone("", 7200))
	for _, test := range []struct {
		name  string
		typ   types.Type
		value any
		want  string
	}{
		{"string", types.String(), "foo", `ARRAY_CONTAINS(PARSE_JSON('"foo"'), "VALUE")`},
		{"int8", types.Int(8), -128, `ARRAY_CONTAINS(PARSE_JSON('-128'), "VALUE")`},
		{"int16", types.Int(16), -32768, `ARRAY_CONTAINS(PARSE_JSON('-32768'), "VALUE")`},
		{"int24", types.Int(24), -8388608, `ARRAY_CONTAINS(PARSE_JSON('-8388608'), "VALUE")`},
		{"int32", types.Int(32), -2147483648, `ARRAY_CONTAINS(PARSE_JSON('-2147483648'), "VALUE")`},
		{"uint32", types.Int(32).Unsigned(), uint(4294967295), `ARRAY_CONTAINS(PARSE_JSON('4294967295'), "VALUE")`},
		{"int64", types.Int(64), 9223372036854775807, `ARRAY_CONTAINS(PARSE_JSON('"9223372036854775807"'), "VALUE")`},
		{"uint64", types.Int(64).Unsigned(), uint(18446744073709551615), `ARRAY_CONTAINS(PARSE_JSON('"18446744073709551615"'), "VALUE")`},
		{"decimal", types.Decimal(18, 4), decimal.MustParse("1.25"), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1.25'))) > 0)`},
		{"float32", types.Float(32), 0.5, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_DOUBLE(krenalis_element) = 0.5)) > 0)`},
		{"float64", types.Float(64), 0.1, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_DOUBLE(krenalis_element) = 0.1)) > 0)`},
		{"datetime", types.DateTime(), instant, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> CONVERT_TIMEZONE('UTC', TO_TIMESTAMP_TZ(AS_VARCHAR(krenalis_element)))::TIMESTAMP_NTZ = '2026-10-20 10:00:00')) > 0)`},
		{"datetime-positive", types.DateTime(), offsetInstant.In(time.FixedZone("", 7200)), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> CONVERT_TIMEZONE('UTC', TO_TIMESTAMP_TZ(AS_VARCHAR(krenalis_element)))::TIMESTAMP_NTZ = '2026-10-20 23:30:45.123456')) > 0)`},
		{"datetime-negative", types.DateTime(), offsetInstant.In(time.FixedZone("", -18000)), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> CONVERT_TIMEZONE('UTC', TO_TIMESTAMP_TZ(AS_VARCHAR(krenalis_element)))::TIMESTAMP_NTZ = '2026-10-20 23:30:45.123456')) > 0)`},
		{"date", types.Date(), instant, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_DATE(AS_VARCHAR(krenalis_element)) = '2026-10-20')) > 0)`},
		{"date-offset", types.Date(), offsetClock, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_DATE(AS_VARCHAR(krenalis_element)) = '2026-10-20')) > 0)`},
		{"time", types.Time(), clock, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_TIME(AS_VARCHAR(krenalis_element)) = '10:30:00.123456')) > 0)`},
		{"time-offset", types.Time(), offsetClock, `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_TIME(AS_VARCHAR(krenalis_element)) = '00:30:00.123456')) > 0)`},
		{"year", types.Year(), 2026, `ARRAY_CONTAINS(PARSE_JSON('2026'), "VALUE")`},
		{"uuid", types.UUID(), "550e8400-e29b-41d4-a716-446655440000", `ARRAY_CONTAINS(PARSE_JSON('"550e8400-e29b-41d4-a716-446655440000"'), "VALUE")`},
		{"ip", types.IP(), "192.0.2.1", `ARRAY_CONTAINS(PARSE_JSON('"192.0.2.1"'), "VALUE")`},
		{"decimal-small", types.Decimal(18, 8), decimal.MustParse("0.00000001"), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('0.00000001'))) > 0)`},
		{"decimal38", types.Decimal(38, 0), decimal.MustParse("12345678901234567890123456789012345678"), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('12345678901234567890123456789012345678'))) > 0)`},
		{"decimal39", types.Decimal(39, 0), decimal.MustParse("123456789012345678901234567890123456789"), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> FALSE)) > 0)`},
		{"decimal76", types.Decimal(76, 0), decimal.MustParse("1234567890123456789012345678901234567800000000000000000000000000000000000000"), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT('1234567890123456789012345678901234567800000000000000000000000000000000000000'))) > 0)`},
		{"float32-rounded", types.Float(32), float64(float32(1.2345678)), `(ARRAY_SIZE(FILTER("VALUE", krenalis_element -> TO_DOUBLE(krenalis_element) = 1.2345678)) > 0)`},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
			t.Run(fmt.Sprintf("%s/%d", test.name, op), func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.Array(test.typ), Nullable: true}
				var b strings.Builder
				err := renderExpr(&b, warehouses.NewBaseExpr(column, op, test.value))
				if err != nil {
					t.Fatalf("expected array expression, got %v", err)
				}
				want := test.want
				if op == warehouses.OpDoesNotContain {
					want = "NOT " + want
				}
				want = `CASE WHEN "VALUE" IS NOT NULL THEN ` + want + " ELSE NULL END"
				if b.String() != want {
					t.Errorf("expected SQL %s, got %s", want, b.String())
				}
			})
		}
	}
}
