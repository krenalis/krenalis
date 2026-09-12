// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_renderCountQuery verifies counts with optional joins and filters.
func Test_renderCountQuery(t *testing.T) {

	id := warehouses.Column{Name: "id", Type: types.Int(32)}
	joins := []warehouses.Join{{
		Type:      warehouses.InnerJoin,
		Table:     "identities",
		Condition: warehouses.NewBaseExpr(id, warehouses.OpIs, warehouses.Column{Name: "fk", Type: types.Int(32)}),
	}}
	where := warehouses.NewBaseExpr(id, warehouses.OpIs, 1)

	for _, tc := range []struct {
		name  string
		joins []warehouses.Join
		where warehouses.Expr
		want  string
	}{
		{name: "all rows", want: `SELECT COUNT(*) FROM "PROFILES"`},
		{name: "filtered rows", where: where, want: `SELECT COUNT(*) FROM "PROFILES" WHERE "ID" = 1`},
		{
			name:  "joined rows",
			joins: joins,
			want:  `SELECT COUNT(*) FROM "PROFILES" JOIN "IDENTITIES" ON "ID" = "FK"`,
		},
		{
			name:  "filtered joined rows",
			joins: joins,
			where: where,
			want:  `SELECT COUNT(*) FROM "PROFILES" JOIN "IDENTITIES" ON "ID" = "FK" WHERE "ID" = 1`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statement, err := renderCountQuery("profiles", tc.joins, tc.where)
			if err != nil {
				t.Fatal(err)
			}
			if statement != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, statement)
			}
		})
	}

}

// TestQueryOrderDirection verifies that the common direction applies to every sort column.
func TestQueryOrderDirection(t *testing.T) {

	db, _ := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{{
		match: `ORDER BY "_UPDATED_AT" DESC, "_KPID" DESC`,
		cols:  []string{"_KPID"},
	}})
	defer db.Close()
	warehouse := &Snowflake{db: db}
	rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
		Table:   "profiles",
		Columns: []warehouses.Column{{Name: "_kpid", Type: types.UUID()}},
		OrderBy: []warehouses.Column{
			{Name: "_updated_at", Type: types.DateTime()},
			{Name: "_kpid", Type: types.UUID()},
		},
		OrderDesc: true,
		Limit:     2,
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	err = rows.Close()
	if err != nil {
		t.Fatal(err)
	}

}

func Test_renderExpr(t *testing.T) {
	tests := []struct {
		expr    warehouses.Expr
		query   string
		invalid bool
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
			query: `"VALUES" = PARSE_JSON('{"foo":2,"boo":true}')`,
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNull),
			query: `"ID" IS NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNotNull),
			query: `"ID" IS NOT NULL`,
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
					// warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsEmpty), // See issue https://github.com/krenalis/krenalis/issues/1804.
					warehouses.NewBaseExpr(warehouses.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.Map(types.String())}, warehouses.OpIsNotEmpty),
				}),
			query: `("TYPE" IS NOT NULL AND "TYPE" <> '') OR ("SCORES" IS NULL OR "SCORES" = ARRAY_CONSTRUCT()) OR "PROPERTIES" <> OBJECT_CONSTRUCT()`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var b strings.Builder
			err := renderExpr(&b, test.expr)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected query %q, got error: %s", test.query, err)
				}
				return
			}
			got := b.String()
			if test.invalid {
				t.Fatalf("expected invalid, got query %q", got)
			}
			if test.query != got {
				t.Fatalf("\nexpected query:  %s\ngot:             %s", test.query, got)
			}
		})
	}
}
