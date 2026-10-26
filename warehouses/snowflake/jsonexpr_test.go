// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
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

// TestJSONDecimalRendering checks numeric literals and comparison bounds.
func TestJSONDecimalRendering(t *testing.T) {

	for _, test := range []struct {
		number  decimal.Decimal
		literal string // empty when the operand needs more than 38 significant digits
	}{
		{decimal.Decimal{}, "0"},
		{decimal.New(0, decimal.MaxScale), "0"},
		{decimal.MustParse("1000"), "1e3"},
		{decimal.New(12300, 4), "123e-2"},
		{decimal.MustParse("-0.0012300"), "-123e-5"},
		{decimal.New(10, 38), "1e-37"},
		{decimal.MustParse("1" + strings.Repeat("0", 38)), "1e38"},
		{decimal.MustParse("1." + strings.Repeat("0", 38)), "1e0"},
		{decimal.MustParse("1e75"), "1e75"},
		{decimal.MustParse(strings.Repeat("9", 38)), strings.Repeat("9", 38) + "e0"},
		{decimal.MustParse(strings.Repeat("9", 39)), ""},
		{decimal.MustParse("-" + strings.Repeat("9", 39)), ""},
		{decimal.MustParse(strings.Repeat("9", 76)), ""},
		{decimal.MustParse("-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)), ""},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpIs, warehouses.OpIsOneOf, warehouses.OpContains} {
			t.Run(fmt.Sprintf("%d/%s", op, test.number), func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.JSON()}
				target := `"VALUE"`
				if op == warehouses.OpContains {
					column.Type = types.Array(types.JSON())
					target = "krenalis_json_element"
				}
				operand := warehouseexpr.Number(test.number)
				var b strings.Builder
				err := renderExpr(&b, newJSONFilterExpr(column, op, operand))
				if err != nil {
					t.Fatalf("expected valid numeric expression, got %v", err)
				}
				query := b.String()
				want := "TRY_TO_DECFLOAT(TO_JSON(" + target + ")) = TO_DECFLOAT('" + test.literal + "')"
				if op == warehouses.OpIsOneOf {
					want = "TRY_TO_DECFLOAT(TO_JSON(" + target + ")) IN (TO_DECFLOAT('" + test.literal + "'))"
				}
				if test.literal == "" {
					if strings.Contains(query, "TO_DECFLOAT(") {
						t.Fatalf("expected no numeric conversion for an inexact equality operand, got %s", query)
					}
					want = "WHEN (IS_DECIMAL(" + target + ") OR IS_DOUBLE(" + target + ")) THEN FALSE"
					if op == warehouses.OpIsOneOf {
						want = `COALESCE(FALSE, FALSE)`
						if query != want {
							t.Fatalf("expected SQL %s, got %s", want, query)
						}
						return
					}
				}
				if !strings.Contains(query, want) {
					t.Fatalf("expected numeric SQL containing %s, got %s", want, query)
				}
			})
		}
	}

	for _, test := range []struct {
		number     string
		operator   warehouses.Operator
		comparison string
	}{
		{"12345678901234567890123456789012345678.9", warehouses.OpIsLessThan, "<= TO_DECFLOAT('12345678901234567890123456789012345678e0')"},
		{"12345678901234567890123456789012345678.9", warehouses.OpIsLessThanOrEqualTo, "<= TO_DECFLOAT('12345678901234567890123456789012345678e0')"},
		{"12345678901234567890123456789012345678.9", warehouses.OpIsGreaterThan, "> TO_DECFLOAT('12345678901234567890123456789012345678e0')"},
		{"12345678901234567890123456789012345678.9", warehouses.OpIsGreaterThanOrEqualTo, "> TO_DECFLOAT('12345678901234567890123456789012345678e0')"},
		{"-12345678901234567890123456789012345678.9", warehouses.OpIsLessThan, "< TO_DECFLOAT('-12345678901234567890123456789012345678e0')"},
		{"-12345678901234567890123456789012345678.9", warehouses.OpIsLessThanOrEqualTo, "< TO_DECFLOAT('-12345678901234567890123456789012345678e0')"},
		{"-12345678901234567890123456789012345678.9", warehouses.OpIsGreaterThan, ">= TO_DECFLOAT('-12345678901234567890123456789012345678e0')"},
		{"-12345678901234567890123456789012345678.9", warehouses.OpIsGreaterThanOrEqualTo, ">= TO_DECFLOAT('-12345678901234567890123456789012345678e0')"},
		{"99999999999999999999999999999999999999.9", warehouses.OpIsLessThan, "<= TO_DECFLOAT('99999999999999999999999999999999999999e0')"},
		{"-99999999999999999999999999999999999999.9", warehouses.OpIsGreaterThan, ">= TO_DECFLOAT('-99999999999999999999999999999999999999e0')"},
	} {
		t.Run(fmt.Sprintf("%d/%s", test.operator, test.number), func(t *testing.T) {
			column := warehouses.Column{Name: "value", Type: types.JSON()}
			var b strings.Builder
			err := renderExpr(&b, newJSONFilterExpr(column, test.operator, json.Value(test.number)))
			if err != nil {
				t.Fatalf("expected valid numeric expression, got %v", err)
			}
			want := `WHEN (IS_DECIMAL("VALUE") OR IS_DOUBLE("VALUE")) THEN TRY_TO_DECFLOAT(TO_JSON("VALUE")) ` +
				test.comparison
			if !strings.Contains(b.String(), want) {
				t.Fatalf("expected numeric SQL containing %s, got %s", want, b.String())
			}
		})
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
			changed := !number.Equal(originalNumber)
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
				{"stored equality", warehouses.OpIs, storedOperand, types.IsDecimal(number)},
				{"original equality", warehouses.OpIs, original, types.IsDecimal(originalNumber) && !changed},
				{"original less than", warehouses.OpIsLessThan, original, types.IsDecimal(originalNumber) && number.Less(originalNumber)},
				{"original greater than", warehouses.OpIsGreaterThan, original, types.IsDecimal(originalNumber) && number.Greater(originalNumber)},
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
