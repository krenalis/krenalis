// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestJSONDecimalRendering checks that every numeric operand remains available
// to the exact fixed-point comparison, including membership and arrays.
func TestJSONDecimalRendering(t *testing.T) {
	for _, test := range []struct {
		number  decimal.Decimal
		literal string
	}{
		{decimal.Decimal{}, "0."},
		{decimal.New(0, decimal.MaxScale), "0."},
		{decimal.MustParse("1000"), "1000."},
		{decimal.New(12300, 4), "1.23"},
		{decimal.MustParse("-0.0012300"), "0.00123"},
		{decimal.New(10, 38), "0." + strings.Repeat("0", 36) + "1"},
		{decimal.MustParse("1" + strings.Repeat("0", 38)), "1" + strings.Repeat("0", 38) + "."},
		{decimal.MustParse("1." + strings.Repeat("0", 38)), "1."},
		{decimal.MustParse("1e75"), "1" + strings.Repeat("0", 75) + "."},
		{decimal.MustParse(strings.Repeat("9", 38)), strings.Repeat("9", 38) + "."},
		{decimal.MustParse(strings.Repeat("9", 39)), strings.Repeat("9", 39) + "."},
		{decimal.MustParse("-" + strings.Repeat("9", 39)), strings.Repeat("9", 39) + "."},
		{decimal.MustParse(strings.Repeat("9", 76)), strings.Repeat("9", 76) + "."},
		{decimal.MustParse("-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)), strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpIs, warehouses.OpIsOneOf, warehouses.OpContains,
			warehouses.OpIsLessThan, warehouses.OpIsGreaterThanOrEqualTo} {
			t.Run(fmt.Sprintf("%d/%s", op, test.number), func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.JSON()}
				if op == warehouses.OpContains {
					column.Type = types.Array(types.JSON())
				}
				var b strings.Builder
				err := renderExpr(&b, newJSONFilterExpr(column, op, warehouseexpr.Number(test.number)))
				if err != nil {
					t.Fatalf("expected valid numeric expression, got %v", err)
				}
				query := b.String()
				if strings.Contains(query, "DECFLOAT") || !strings.Contains(query, "TYPEOF(") ||
					!strings.Contains(query, "COLLATE(") || !strings.Contains(query, "'"+test.literal+"'") {
					t.Fatalf("expected exact numeric SQL retaining %s, got %s", test.literal, query)
				}
			})
		}
	}
}

// TestJSONExpressionErrors verifies that invalid JSON expressions are rejected.
func TestJSONExpressionErrors(t *testing.T) {

	column := warehouses.Column{Name: "value", Type: types.JSON()}
	array := warehouses.Column{Name: "value", Type: types.Array(types.JSON())}
	mapping := warehouses.Column{Name: "value", Type: types.Map(types.JSON())}
	for _, test := range []struct {
		name    string
		expr    *warehouses.BaseExpr
		wantErr string
	}{
		{"unsupported JSON operator", newJSONFilterExpr(column, warehouses.OpIsBefore, json.Value("\"2026-01-01\"")),
			`operator 14 is not supported for type "json"`},
		{"missing operand", newJSONFilterExpr(column, warehouses.OpIs),
			"operator 0 requires 1 values, got 0"},
		{"invalid operand type", newJSONFilterExpr(column, warehouses.OpIs, "foo"),
			"JSON filter operand must contain valid encoded JSON"},
		{"between with one operand", newJSONFilterExpr(column, warehouses.OpIsBetween, json.Value("1")),
			"operator 6 requires 2 values, got 1"},
		{"one of without operands", newJSONFilterExpr(column, warehouses.OpIsOneOf),
			"JSON membership requires at least one value"},
		{"empty with operand", newJSONFilterExpr(column, warehouses.OpIsEmpty, json.Value("\"foo\"")),
			"operator 20 requires 0 values, got 1"},
		{"array with scalar comparison", newJSONFilterExpr(array, warehouses.OpIs, json.Value("\"foo\"")),
			`operator 0 is not supported for type "array(json)"`},
		{"map with scalar contains", newJSONFilterExpr(mapping, warehouses.OpContains, json.Value("\"foo\"")),
			`operator 8 is not supported for type "map(json)"`},
		{"array with JSON keys", &warehouses.BaseExpr{Column: array, Keys: []string{"key"}, Operator: warehouses.OpExists},
			`JSON keys cannot be used with type "array(json)"`},
		{"number exceeds precision", newJSONFilterExpr(column, warehouses.OpIs,
			warehouseexpr.Number(decimal.MustParse("1e76"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"number exceeds scale", newJSONFilterExpr(column, warehouses.OpIs,
			warehouseexpr.Number(decimal.MustParse("1e-38"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"number has large positive exponent", newJSONFilterExpr(column, warehouses.OpIs,
			warehouseexpr.Number(decimal.MustParse("1e999999"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"number has large negative exponent", newJSONFilterExpr(column, warehouses.OpIs,
			warehouseexpr.Number(decimal.MustParse("1e-999999"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"between with invalid second number", newJSONFilterExpr(column, warehouses.OpIsBetween,
			json.Value("0"), warehouseexpr.Number(decimal.MustParse("1e-38"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"one of with invalid second number", newJSONFilterExpr(column, warehouses.OpIsOneOf,
			json.Value("0"), warehouseexpr.Number(decimal.MustParse("1e76"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"array number exceeds precision", newJSONFilterExpr(array, warehouses.OpContains,
			warehouseexpr.Number(decimal.MustParse("1e76"))),
			"decimal operand exceeds precision 76 or scale 37"},
		{"array number exceeds scale", newJSONFilterExpr(array, warehouses.OpDoesNotContain,
			warehouseexpr.Number(decimal.MustParse("1e-38"))),
			"decimal operand exceeds precision 76 or scale 37"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var b strings.Builder
			err := renderExpr(&b, test.expr)
			if err != nil {
				if err.Error() != test.wantErr {
					t.Fatalf("expected error %q, got %q", test.wantErr, err.Error())
				}
				return
			}
			t.Fatalf("expected error %q, got nil and SQL %q", test.wantErr, b.String())
		})
	}

}

// TestJSONExpressions verifies JSON filtering on data written by Merge.
func TestJSONExpressions(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	mustExecSQL(t, db, "CREATE TABLE \"JSON_EXPRESSIONS\" (\"ID\" INTEGER PRIMARY KEY, \"VALUE\" VARIANT, \"DOCUMENT\" VARIANT, \"ARRAY\" ARRAY, \"MAP\" VARIANT)")
	idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
	column := warehouses.Column{Name: "value", Type: types.JSON(), Nullable: true}
	documentColumn := warehouses.Column{Name: "document", Type: types.JSON(), Nullable: true}
	arrayColumn := warehouses.Column{Name: "array", Type: types.Array(types.JSON()), Nullable: true}
	mapColumn := warehouses.Column{Name: "map", Type: types.Map(types.JSON()), Nullable: true}
	table := warehouses.Table{
		Name: "json_expressions", Columns: []warehouses.Column{idColumn, column, documentColumn, arrayColumn, mapColumn},
		Keys: []string{"id"},
	}
	values, cases := warehouseexpr.JSONCases()
	if len(values) == 0 || len(cases) == 0 {
		t.Fatalf("expected nonempty JSON fixtures and cases, got %d and %d", len(values), len(cases))
	}
	data := make([][]any, len(values))
	for i, value := range values {
		var scalar, document, array, mapping any
		if value != nil {
			scalar = value
			document = json.Value("{\"a.b\":{\"key'\\\\\":" + string(value) + "}}")
			array = []any{value}
			mapping = map[string]any{"key": value}
		}
		data[i] = []any{i, scalar, document, array, mapping}
	}
	err := warehouse.Merge(t.Context(), table, data, nil)
	if err != nil {
		t.Fatalf("expected fixture merge, got %v", err)
	}

	for _, path := range []struct {
		name string
		keys []string
	}{{"scalar", nil}, {"document", []string{"a.b", "key'\\"}}} {
		for _, c := range cases {
			t.Run(path.name+"/"+c.Name, func(t *testing.T) {

				expr := newJSONFilterExpr(column, c.Operator, c.Values...)
				if len(path.keys) > 0 {
					expr.Column = documentColumn
					expr.Keys = path.keys
				}
				checkJSONExpression(t, warehouse, table.Name, idColumn, expr, c.Matches)

				var b strings.Builder
				err := renderExpr(&b, expr)
				if err != nil {
					t.Fatalf("expected JSON expression, got %v", err)
				}
				statement := "SELECT " + quoteIdent(idColumn.Name) + ", (" + b.String() + ") FROM " +
					quoteIdent(table.Name) + " ORDER BY " + quoteIdent(idColumn.Name)
				rows, err := db.QueryContext(t.Context(), statement)
				if err != nil {
					t.Fatalf("expected truth-table query, got %v", err)
				}
				defer rows.Close()
				count := 0
				nulls := c.Nulls(len(values))
				for rows.Next() {
					var id int
					var got sql.NullBool
					if err := rows.Scan(&id, &got); err != nil {
						t.Fatalf("expected truth-table row, got %v", err)
					}
					want := sql.NullBool{Bool: slices.Contains(c.Matches, id), Valid: !slices.Contains(nulls, id)}
					if got != want {
						t.Errorf("expected fixture %d result %v, got %v", id, want, got)
					}
					count++
				}
				err = rows.Err()
				if err != nil {
					t.Fatalf("expected complete truth table, got %v", err)
				}
				if count != len(values) {
					t.Fatalf("expected %d truth-table rows, got %d", len(values), count)
				}

			})
		}
	}

	// Composition preserves SQL NULL until the query uses the filter.
	for _, op := range []warehouses.LogicalOperator{warehouses.OpAnd, warehouses.OpOr} {
		for _, definite := range []bool{false, true} {
			t.Run(fmt.Sprintf("composition/%d/%t", op, definite), func(t *testing.T) {
				knownID := -1
				if definite {
					knownID = 0
				}
				expr := warehouses.NewMultiExpr(op, []warehouses.Expr{
					newJSONFilterExpr(column, warehouses.OpIsNot, json.Value("5")),
					warehouses.NewBaseExpr(idColumn, warehouses.OpIs, knownID),
				})
				var b strings.Builder
				err := renderExpr(&b, expr)
				if err != nil {
					t.Fatalf("expected composed expression, got %v", err)
				}
				statement := "SELECT (" + b.String() + ") FROM " + quoteIdent(table.Name) +
					" WHERE " + quoteIdent(idColumn.Name) + " = 0"
				var got sql.NullBool
				err = db.QueryRowContext(t.Context(), statement).Scan(&got)
				if err != nil {
					t.Fatalf("expected composed truth value, got %v", err)
				}
				want := sql.NullBool{}
				if op == warehouses.OpAnd && !definite {
					want.Valid = true
				}
				if op == warehouses.OpOr && definite {
					want = sql.NullBool{Bool: true, Valid: true}
				}
				if got != want {
					t.Fatalf("expected composition %v, got %v", want, got)
				}
			})
		}
	}

	// A SQL array contains a JSON scalar only when the scalar matches "is".
	for _, c := range cases {
		if c.Operator != warehouses.OpIs && c.Operator != warehouses.OpIsNot {
			continue
		}
		t.Run("array/"+c.Name, func(t *testing.T) {
			op := warehouses.OpContains
			if c.Operator == warehouses.OpIsNot {
				op = warehouses.OpDoesNotContain
			}
			checkJSONExpression(t, warehouse, table.Name, idColumn,
				newJSONFilterExpr(arrayColumn, op, c.Values...), c.Matches)
		})
	}

	// Container filters treat SQL NULL as empty, but still distinguish it
	// from a present empty container for null and existence checks.
	mustExecSQL(t, db, "INSERT INTO \"JSON_EXPRESSIONS\" SELECT "+strconv.Itoa(len(values))+", NULL, NULL, ARRAY_CONSTRUCT(), OBJECT_CONSTRUCT()")
	var absentIDs, nonEmptyIDs []int
	for i, value := range values {
		if value == nil {
			absentIDs = append(absentIDs, i)
			continue
		}
		nonEmptyIDs = append(nonEmptyIDs, i)
	}
	if len(absentIDs) == 0 || len(nonEmptyIDs) == 0 {
		t.Fatalf("expected absent and present JSON fixtures, got %d and %d", len(absentIDs), len(nonEmptyIDs))
	}
	emptyIDs := append(slices.Clone(absentIDs), len(values))
	presentIDs := append(slices.Clone(nonEmptyIDs), len(values))
	for _, container := range []warehouses.Column{arrayColumn, mapColumn} {
		for _, c := range []struct {
			name    string
			op      warehouses.Operator
			matches []int
		}{
			{"empty", warehouses.OpIsEmpty, emptyIDs},
			{"not empty", warehouses.OpIsNotEmpty, nonEmptyIDs},
			{"null", warehouses.OpIsNull, absentIDs},
			{"not null", warehouses.OpIsNotNull, presentIDs},
			{"exists", warehouses.OpExists, presentIDs},
			{"does not exist", warehouses.OpDoesNotExist, absentIDs},
		} {
			t.Run(container.Name+"/"+c.name, func(t *testing.T) {
				checkJSONExpression(t, warehouse, table.Name, idColumn,
					newJSONFilterExpr(container, c.op), c.matches)
			})
		}
	}

	// A mixed SQL array searches all immediate scalar elements, including
	// duplicates, without matching scalars nested inside JSON arrays or objects.
	mixedID := len(values) + 1
	err = warehouse.Merge(t.Context(), table, [][]any{{mixedID, nil, nil, []any{
		json.Value("null"), json.Value("true"), json.Value("5"), json.Value("5"),
		json.Value("\"foo\""), json.Value("[6]"), json.Value("{\"key\":6}"),
	}, nil}}, nil)
	if err != nil {
		t.Fatalf("expected mixed array fixture merge, got %v", err)
	}
	for _, c := range []struct {
		name, text string
		matches    bool
	}{
		{"number", "5", true}, {"string", "\"foo\"", true}, {"nested number", "6", false},
		{"boolean text", "\"true\"", false}, {"null text", "\"null\"", false},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
			name := "contains"
			matches := c.matches
			if op == warehouses.OpDoesNotContain {
				name, matches = "does not contain", !matches
			}
			t.Run("array/mixed/"+name+"/"+c.name, func(t *testing.T) {
				var want []int
				if matches {
					want = []int{mixedID}
				}
				expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
					newJSONFilterExpr(idColumn, warehouses.OpIs, mixedID),
					newJSONFilterExpr(arrayColumn, op, json.Value(c.text)),
				})
				checkJSONExpression(t, warehouse, table.Name, idColumn, expr, want)
			})
		}
	}

	// Check storage precision separately from comparisons with stored values.
	// Explicit lower and upper bounds also exercise positive comparisons.
	for i, fixture := range []struct {
		raw, lower, upper string
		changed           bool
	}{
		{"9007199254740993e0", "9007199254740991", "9007199254740994", true},
		{"1.234567890123456789e0", "1.23456789012345", "1.23456789012347", true},
		{"0.1234567890123456789012345678901234567890123456789", "0.12345678901234", "0.12345678901235", true},
	} {
		t.Run("numeric representation/"+fixture.raw, func(t *testing.T) {

			id := mixedID + 1 + i
			err := warehouse.Merge(t.Context(), table, [][]any{{id, json.Value(fixture.raw), nil, nil, nil}}, nil)
			if err != nil {
				t.Fatalf("expected numeric fixture merge, got %v", err)
			}
			idExpr := newJSONFilterExpr(idColumn, warehouses.OpIs, id)
			rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
				Table: table.Name, Columns: []warehouses.Column{column}, Where: idExpr,
			}, false)
			if err != nil {
				t.Fatalf("expected numeric fixture query, got %v", err)
			}
			defer rows.Close()
			if !rows.Next() {
				err = rows.Err()
				if err != nil {
					t.Fatalf("expected numeric fixture iteration, got %v", err)
				}
				t.Fatal("expected stored numeric fixture, got no rows")
			}
			scanned := make([]any, 1)
			err = rows.Scan(scanned...)
			if err != nil {
				t.Fatalf("expected numeric fixture scan, got %v", err)
			}
			stored, ok := scanned[0].(json.Value)
			if !ok {
				t.Fatalf("expected stored JSON value, got %T", scanned[0])
			}
			if rows.Next() {
				t.Fatal("expected one numeric fixture, got additional rows")
			}
			err = rows.Err()
			if err != nil {
				t.Fatalf("expected complete numeric fixture iteration, got %v", err)
			}
			number, err := stored.Decimal(0, 0)
			if err != nil {
				t.Fatalf("expected stored JSON number, got %v", err)
			}
			originalNumber, err := json.Value(fixture.raw).Decimal(0, 0)
			if err != nil {
				t.Fatalf("expected numeric original value, got %v", err)
			}
			original := json.Value(fixture.raw)
			storedOperand := json.Value(string(stored))
			// The scanner's JSON can lose digits from the canonical DOUBLE text.
			var binary float64
			row := db.QueryRowContext(t.Context(), `SELECT AS_DOUBLE("VALUE") FROM `+
				quoteIdent(table.Name)+` WHERE "ID"=`+strconv.Itoa(id))
			err = row.Scan(&binary)
			if err != nil {
				t.Fatalf("expected exact binary fixture transport, got %v", err)
			}
			exact, _ := new(big.Rat).SetString(strconv.FormatFloat(binary, 'g', -1, 64))
			if exact == nil {
				t.Fatalf("expected finite stored fixture, got %g", binary)
			}
			originalValue, _ := new(big.Rat).SetString(originalNumber.String())
			displayValue, _ := new(big.Rat).SetString(number.String())
			changed := exact.Cmp(originalValue) != 0
			if changed != fixture.changed {
				t.Fatalf("expected numeric storage change %t, got %t: %s became %s", fixture.changed, changed, fixture.raw, stored)
			}
			err = rows.Close()
			if err != nil {
				t.Fatalf("expected numeric query close, got %v", err)
			}

			for _, c := range []struct {
				name    string
				op      warehouses.Operator
				operand json.Value
				matches bool
			}{
				{"display equality", warehouses.OpIs, storedOperand, types.IsDecimal(number) && exact.Cmp(displayValue) == 0},
				{"original equality", warehouses.OpIs, original, types.IsDecimal(originalNumber) && !changed},
				{"original less than", warehouses.OpIsLessThan, original, types.IsDecimal(originalNumber) && exact.Cmp(originalValue) < 0},
				{"original greater than", warehouses.OpIsGreaterThan, original, types.IsDecimal(originalNumber) && exact.Cmp(originalValue) > 0},
				{"less than upper bound", warehouses.OpIsLessThan, json.Value(fixture.upper), true},
				{"greater than lower bound", warehouses.OpIsGreaterThan, json.Value(fixture.lower), true},
			} {
				t.Run(c.name, func(t *testing.T) {
					operandNumber, err := c.operand.Decimal(0, 0)
					if err != nil {
						t.Fatalf("expected numeric fixture operand, got %v", err)
					}
					if !types.IsDecimal(operandNumber) {
						var b strings.Builder
						err := renderExpr(&b, newJSONFilterExpr(column, c.op, c.operand))
						if err != nil {
							return
						}
						t.Fatal("expected out-of-domain operand error, got nil")
					}
					var want []int
					if c.matches {
						want = []int{id}
					}
					expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
						idExpr, newJSONFilterExpr(column, c.op, c.operand),
					})
					checkJSONExpression(t, warehouse, table.Name, idColumn, expr, want)
				})
			}

		})
	}

}

// checkJSONExpression checks the matching IDs and the total row count.
// want must contain IDs in ascending order.
func checkJSONExpression(t *testing.T, warehouse warehouses.Warehouse, table string, idColumn warehouses.Column, expr warehouses.Expr, want []int) {

	t.Helper()
	rows, total, err := warehouse.Query(t.Context(), warehouses.RowQuery{
		Table: table, Columns: []warehouses.Column{idColumn}, Where: expr, OrderBy: []warehouses.Column{idColumn},
	}, true)
	if err != nil {
		t.Fatalf("expected query execution, got %v", err)
	}
	defer rows.Close()
	var got []int
	values := make([]any, 1)
	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			t.Fatalf("expected row scan, got %v", err)
		}
		id, ok := values[0].(int)
		if !ok {
			t.Fatalf("expected integer row ID, got %T", values[0])
		}
		got = append(got, id)
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete query results, got %v", err)
	}
	if !slices.Equal(want, got) {
		t.Errorf("expected matching IDs %v, got %v", want, got)
	}
	if total != len(want) {
		t.Errorf("expected total %d, got %d", len(want), total)
	}

}

// newJSONFilterExpr marks raw scalar operands as typed filter operands.
func newJSONFilterExpr(column warehouses.Column, operator warehouses.Operator, values ...any) *warehouses.BaseExpr {
	expr := warehouses.NewBaseExpr(column, operator, values...)
	if column.Type.IsJSON() || (column.Type.IsArray() || column.Type.IsMap()) && column.Type.Elem().IsJSON() {
		expr.JSONComparison = warehouses.JSONTypedComparison
	}
	return expr
}
