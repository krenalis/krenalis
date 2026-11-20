// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestJSONDecimalRendering checks that decimal values are rendered as exact,
// normalized SQL numeric literals.
func TestJSONDecimalRendering(t *testing.T) {

	operators := []struct {
		name string
		op   warehouses.Operator
	}{
		{"is", warehouses.OpIs},
		{"is one of", warehouses.OpIsOneOf},
		{"contains", warehouses.OpContains},
	}

	for _, test := range []struct {
		name    string
		number  decimal.Decimal
		literal string
	}{
		{"zero", decimal.Decimal{}, "0"},
		{"zero with maximum scale", decimal.New(0, decimal.MaxScale), "0"},
		{"zero with negative scale", decimal.New(0, -6), "0"},
		{"integer", decimal.MustParse("1000"), "1000"},
		{"trailing fractional zeros", decimal.New(12300, 4), "1.23"},
		{"negative fraction", decimal.MustParse("-0.0012300"), "-0.00123"},
		{"normalized scale 37", decimal.New(10, 38), "0." + strings.Repeat("0", 36) + "1"},
		{"large power of ten", decimal.MustParse("1" + strings.Repeat("0", 38)), "1" + strings.Repeat("0", 38)},
		{"integer with fractional zeros", decimal.MustParse("1." + strings.Repeat("0", 38)), "1"},
		{"positive exponent", decimal.MustParse("1e75"), "1" + strings.Repeat("0", 75)},
		{"positive exponent with fractional mantissa", decimal.MustParse("1.23e5"), "123000"},
		{"38 digits", decimal.MustParse(strings.Repeat("9", 38)), strings.Repeat("9", 38)},
		{"39 digits", decimal.MustParse(strings.Repeat("9", 39)), strings.Repeat("9", 39)},
		{"negative 39 digits", decimal.MustParse("-" + strings.Repeat("9", 39)), "-" + strings.Repeat("9", 39)},
		{"76 digits", decimal.MustParse(strings.Repeat("9", 76)), strings.Repeat("9", 76)},
		{"large negative fraction", decimal.MustParse("-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)),
			"-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)},
	} {
		for _, operator := range operators {
			t.Run(test.name+"/"+operator.name, func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.JSON()}
				if operator.op == warehouses.OpContains {
					column.Type = types.Array(types.JSON())
				}
				// Use distinct text for the string branch of the same operand.
				operand := warehouses.JSONConditionValue{String: "threshold", Number: &test.number}
				var b strings.Builder
				err := renderExpr(&b, warehouses.NewBaseExpr(column, operator.op, operand))
				if err != nil {
					t.Fatalf("expected valid numeric expression, got %v", err)
				}
				want := `("value")::numeric IN (` + test.literal + `)`
				if operator.op == warehouses.OpIs {
					want = `("value")::numeric = ` + test.literal + ` ELSE`
				}
				if operator.op == warehouses.OpContains {
					want = `(krenalis_json_element.value)::numeric = ` + test.literal + ` ELSE`
				}
				if !strings.Contains(b.String(), want) {
					t.Fatalf("expected numeric SQL containing %s, got %s", want, b.String())
				}
			})
		}
	}

}

// TestJSONExpressionErrors verifies that invalid JSON expressions are rejected.
func TestJSONExpressionErrors(t *testing.T) {

	// jsonNumber sets Number even for decimals outside the renderer's limits.
	jsonNumber := func(text string) warehouses.JSONConditionValue {
		return warehouses.JSONConditionValue{String: text, Number: new(decimal.MustParse(text))}
	}

	column := warehouses.Column{Name: "value", Type: types.JSON()}
	array := warehouses.Column{Name: "value", Type: types.Array(types.JSON())}
	mapping := warehouses.Column{Name: "value", Type: types.Map(types.JSON())}
	for _, test := range []struct {
		name    string
		expr    *warehouses.BaseExpr
		wantErr string
	}{
		{"unsupported JSON operator", warehouses.NewBaseExpr(column, warehouses.OpIsBefore, warehouseexpr.Operand("2026-01-01")),
			`warehouses/postgresql: operator 14 is not supported for type "json"`},
		{"missing operand", warehouses.NewBaseExpr(column, warehouses.OpIs),
			"warehouses/postgresql: operator 0 requires 1 values, got 0"},
		{"invalid operand type", warehouses.NewBaseExpr(column, warehouses.OpIs, "foo"),
			"warehouses/postgresql: expected a JSONConditionValue, got string"},
		{"between with one operand", warehouses.NewBaseExpr(column, warehouses.OpIsBetween, warehouseexpr.Operand("1")),
			"warehouses/postgresql: operator 6 requires 2 values, got 1"},
		{"one of without operands", warehouses.NewBaseExpr(column, warehouses.OpIsOneOf),
			"warehouses/postgresql: JSON membership requires at least one value"},
		{"empty with operand", warehouses.NewBaseExpr(column, warehouses.OpIsEmpty, warehouseexpr.Operand("foo")),
			"warehouses/postgresql: operator 20 requires 0 values, got 1"},
		{"array with scalar comparison", warehouses.NewBaseExpr(array, warehouses.OpIs, warehouseexpr.Operand("foo")),
			`warehouses/postgresql: operator 0 is not supported for type "array(json)"`},
		{"array contains nil", warehouses.NewBaseExpr(array, warehouses.OpContains, nil),
			"warehouses/postgresql: expected a JSONConditionValue, got <nil>"},
		{"array does not contain nil", warehouses.NewBaseExpr(array, warehouses.OpDoesNotContain, nil),
			"warehouses/postgresql: expected a JSONConditionValue, got <nil>"},
		{"map with scalar contains", warehouses.NewBaseExpr(mapping, warehouses.OpContains, warehouseexpr.Operand("foo")),
			`warehouses/postgresql: operator 8 is not supported for type "map(json)"`},
		{"array with JSON keys", &warehouses.BaseExpr{Column: array, Keys: []string{"key"}, Operator: warehouses.OpExists},
			`warehouses/postgresql: JSON keys cannot be used with type "array(json)"`},
		{"number exceeds precision", warehouses.NewBaseExpr(column, warehouses.OpIs, jsonNumber("1e76")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"number exceeds scale", warehouses.NewBaseExpr(column, warehouses.OpIs, jsonNumber("1e-38")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"number has large positive exponent", warehouses.NewBaseExpr(column, warehouses.OpIs, jsonNumber("1e999999")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"number has large negative exponent", warehouses.NewBaseExpr(column, warehouses.OpIs, jsonNumber("1e-999999")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"between with invalid second number", warehouses.NewBaseExpr(column, warehouses.OpIsBetween,
			warehouseexpr.Operand("0"), jsonNumber("1e-38")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"one of with invalid second number", warehouses.NewBaseExpr(column, warehouses.OpIsOneOf,
			warehouseexpr.Operand("0"), jsonNumber("1e76")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"array number exceeds precision", warehouses.NewBaseExpr(array, warehouses.OpContains, jsonNumber("1e76")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
		{"array number exceeds scale", warehouses.NewBaseExpr(array, warehouses.OpDoesNotContain, jsonNumber("1e-38")),
			"warehouses/postgresql: decimal operand exceeds precision 76 or scale 37"},
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

// TestJSONExpressions verifies JSON filters on data written by Merge.
func TestJSONExpressions(t *testing.T) {

	warehouse, db := newTestPostgreSQLWarehouse(t)
	mustExecSQL(t, db, `CREATE TABLE "json_expressions" ("id" integer PRIMARY KEY, "value" jsonb, `+
		`"document" jsonb, "array" jsonb[], "map" jsonb)`)
	idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
	column := warehouses.Column{Name: "value", Type: types.JSON(), Nullable: true}
	documentColumn := warehouses.Column{Name: "document", Type: types.JSON(), Nullable: true}
	arrayColumn := warehouses.Column{Name: "array", Type: types.Array(types.JSON()), Nullable: true}
	mapColumn := warehouses.Column{Name: "map", Type: types.Map(types.JSON()), Nullable: true}
	table := warehouses.Table{
		Name:    "json_expressions",
		Columns: []warehouses.Column{idColumn, column, documentColumn, arrayColumn, mapColumn},
		Keys:    []string{"id"},
	}
	values, cases := warehouseexpr.JSONCases()
	if len(values) == 0 || len(cases) == 0 {
		t.Fatalf("expected nonempty JSON fixtures and cases, got %d and %d", len(values), len(cases))
	}
	emptyID := len(values)
	mixedID := emptyID + 1
	numericFixtures := []struct {
		raw, lower, upper       string
		compareOriginalAsNumber bool
	}{
		{"123456789012345678901234567890123456789", "123456789012345678901234567890123456788", "123456789012345678901234567890123456790", true},
		{"-123456789012345678901234567890123456789", "-123456789012345678901234567890123456790", "-123456789012345678901234567890123456788", true},
		{"9007199254740993e0", "9007199254740991", "9007199254740994", true},
		{"1.234567890123456789e0", "1.23456789012345", "1.23456789012347", true},
		{"0.1234567890123456789012345678901234567890123456789", "0.12345678901234", "0.12345678901235", false},
	}
	data := make([][]any, len(values), mixedID+1+len(numericFixtures))
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
	data = append(data,
		[]any{emptyID, nil, nil, []any{}, map[string]any{}},
		[]any{mixedID, nil, nil, []any{
			json.Value("null"), json.Value("true"), json.Value("5"), json.Value("5"),
			json.Value("\"foo\""), json.Value("[6]"), json.Value("{\"key\":6}"),
		}, nil},
	)
	for i, fixture := range numericFixtures {
		data = append(data, []any{mixedID + 1 + i, json.Value(fixture.raw), nil, nil, nil})
	}
	err := warehouse.Merge(t.Context(), table, data, nil)
	if err != nil {
		t.Fatalf("expected fixture merge, got %v", err)
	}

	fixtureRange := warehouses.NewBaseExpr(idColumn, warehouses.OpIsLessThan, emptyID)
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
				where := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{fixtureRange, expr})
				checkJSONExpression(t, warehouse, table.Name, idColumn, where, c.Matches)
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
			expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
				fixtureRange, warehouses.NewBaseExpr(arrayColumn, op, c.Values...),
			})
			checkJSONExpression(t, warehouse, table.Name, idColumn, expr, c.Matches)
		})
	}

	// Container filters treat SQL NULL as empty, but still distinguish it
	// from a present empty container for null and existence checks.
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
	emptyIDs := append(slices.Clone(absentIDs), emptyID)
	presentIDs := append(slices.Clone(nonEmptyIDs), emptyID)
	containerRange := warehouses.NewBaseExpr(idColumn, warehouses.OpIsLessThanOrEqualTo, emptyID)
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
				expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
					containerRange, warehouses.NewBaseExpr(container, c.op),
				})
				checkJSONExpression(t, warehouse, table.Name, idColumn, expr, c.matches)
			})
		}
	}

	// A mixed SQL array searches all immediate scalar elements, including
	// duplicates, without matching scalars nested inside JSON arrays or objects.
	for _, c := range []struct {
		name    string
		operand any
		matches bool
	}{
		{"number", warehouseexpr.Operand("5"), true}, {"string", warehouseexpr.Operand("foo"), true},
		{"nested number", warehouseexpr.Operand("6"), false}, {"boolean text", warehouseexpr.Operand("true"), false},
		{"null text", warehouseexpr.Operand("null"), false},
		{"encoded object", json.Value(`{"key":6}`), true}, {"different object", json.Value(`{"key":7}`), false},
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
					warehouses.NewBaseExpr(arrayColumn, op, c.operand),
				})
				checkJSONExpression(t, warehouse, table.Name, idColumn, expr, want)
			})
		}
	}

	// Check storage precision separately from comparisons with stored values.
	// Explicit lower and upper bounds also exercise positive comparisons.
	for i, fixture := range numericFixtures {
		t.Run("numeric representation/"+fixture.raw, func(t *testing.T) {

			id := mixedID + 1 + i
			idExpr := warehouses.NewBaseExpr(idColumn, warehouses.OpIs, id)
			stored := readJSONValue(t, warehouse, table.Name, column, idExpr)
			number, err := stored.Decimal(0, 0)
			if err != nil {
				t.Fatalf("expected stored JSON number, got %v", err)
			}
			originalNumber, err := json.Value(fixture.raw).Decimal(0, 0)
			if err != nil {
				t.Fatalf("expected numeric original value, got %v", err)
			}
			if !number.Equal(originalNumber) {
				t.Fatalf("expected the same numeric value as %s, got %s", fixture.raw, stored)
			}

			original := warehouses.JSONConditionValue{String: fixture.raw}
			storedOperand := warehouses.JSONConditionValue{String: string(stored)}
			if fixture.compareOriginalAsNumber {
				original.Number = &originalNumber
				storedOperand.Number = &number
			}
			lower := warehouses.JSONConditionValue{String: fixture.lower, Number: new(decimal.MustParse(fixture.lower))}
			upper := warehouses.JSONConditionValue{String: fixture.upper, Number: new(decimal.MustParse(fixture.upper))}

			for _, c := range []struct {
				name    string
				op      warehouses.Operator
				operand warehouses.JSONConditionValue
				matches bool
			}{
				{"stored equality", warehouses.OpIs, storedOperand, fixture.compareOriginalAsNumber},
				{"original equality", warehouses.OpIs, original, fixture.compareOriginalAsNumber},
				{"original less than", warehouses.OpIsLessThan, original, false},
				{"original greater than", warehouses.OpIsGreaterThan, original, false},
				{"less than upper bound", warehouses.OpIsLessThan, upper, true},
				{"greater than lower bound", warehouses.OpIsGreaterThan, lower, true},
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

// checkJSONExpression checks the matching row IDs and total count.
// want must be sorted in ascending order.
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

// readJSONValue reads a JSON value, expecting exactly one matching row.
func readJSONValue(t *testing.T, warehouse warehouses.Warehouse, table string, column warehouses.Column, where warehouses.Expr) json.Value {
	t.Helper()
	rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
		Table: table, Columns: []warehouses.Column{column}, Where: where,
	}, false)
	if err != nil {
		t.Fatalf("expected JSON value query, got %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			t.Fatalf("expected JSON value iteration, got %v", err)
		}
		t.Fatal("expected one JSON value, got no rows")
	}
	scanned := make([]any, 1)
	err = rows.Scan(scanned...)
	if err != nil {
		t.Fatalf("expected JSON value scan, got %v", err)
	}
	stored, ok := scanned[0].(json.Value)
	if !ok {
		t.Fatalf("expected JSON value, got %T", scanned[0])
	}
	if rows.Next() {
		t.Fatal("expected one JSON value, got additional rows")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("expected complete JSON value iteration, got %v", err)
	}
	return stored
}
