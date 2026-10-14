// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

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
		{"unsupported JSON operator", warehouses.NewBaseExpr(column, warehouses.OpIsBefore, warehouseexpr.Operand("2026-01-01")),
			`operator 14 is not supported for type "json"`},
		{"missing operand", warehouses.NewBaseExpr(column, warehouses.OpIs),
			"operator 0 requires 1 values, got 0"},
		{"invalid operand type", warehouses.NewBaseExpr(column, warehouses.OpIs, "foo"),
			"expected a JSONConditionValue, got string"},
		{"between with one operand", warehouses.NewBaseExpr(column, warehouses.OpIsBetween, warehouseexpr.Operand("1")),
			"operator 6 requires 2 values, got 1"},
		{"one of without operands", warehouses.NewBaseExpr(column, warehouses.OpIsOneOf),
			"JSON membership requires at least one value"},
		{"empty with operand", warehouses.NewBaseExpr(column, warehouses.OpIsEmpty, warehouseexpr.Operand("foo")),
			"operator 20 requires 0 values, got 1"},
		{"array with scalar comparison", warehouses.NewBaseExpr(array, warehouses.OpIs, warehouseexpr.Operand("foo")),
			`operator 0 is not supported for type "array(json)"`},
		{"map with scalar contains", warehouses.NewBaseExpr(mapping, warehouses.OpContains, warehouseexpr.Operand("foo")),
			`operator 8 is not supported for type "map(json)"`},
		{"array with JSON keys", &warehouses.BaseExpr{Column: array, Keys: []string{"key"}, Operator: warehouses.OpExists},
			`JSON keys cannot be used with type "array(json)"`},
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
				expr := warehouses.NewBaseExpr(column, c.Operator, c.Values...)
				if len(path.keys) > 0 {
					expr.Column = documentColumn
					expr.Keys = path.keys
				}
				checkJSONExpression(t, warehouse, table.Name, idColumn, expr, c.Matches)
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
				warehouses.NewBaseExpr(arrayColumn, op, c.Values...), c.Matches)
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
					warehouses.NewBaseExpr(container, c.op), c.matches)
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
		{"number", "5", true}, {"string", "foo", true}, {"nested number", "6", false},
		{"boolean text", "true", false}, {"null text", "null", false},
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
					warehouses.NewBaseExpr(idColumn, warehouses.OpIs, mixedID),
					warehouses.NewBaseExpr(arrayColumn, op, warehouseexpr.Operand(c.text)),
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
			idExpr := warehouses.NewBaseExpr(idColumn, warehouses.OpIs, id)
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
			original := warehouseexpr.Operand(fixture.raw)
			if original.Number == nil {
				t.Fatal("expected numeric original operand, got nil")
			}
			changed := !number.Equal(*original.Number)
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
				operand warehouses.JSONConditionValue
				matches bool
			}{
				{"stored equality", warehouses.OpIs, warehouseexpr.Operand(string(stored)), true},
				{"original equality", warehouses.OpIs, original, !changed},
				{"original less than", warehouses.OpIsLessThan, original, number.Less(*original.Number)},
				{"original greater than", warehouses.OpIsGreaterThan, original, number.Greater(*original.Number)},
				{"less than upper bound", warehouses.OpIsLessThan, warehouseexpr.Operand(fixture.upper), true},
				{"greater than lower bound", warehouses.OpIsGreaterThan, warehouseexpr.Operand(fixture.lower), true},
			} {
				t.Run(c.name, func(t *testing.T) {
					var want []int
					if c.matches {
						want = []int{id}
					}
					expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
						idExpr, warehouses.NewBaseExpr(column, c.op, c.operand),
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
