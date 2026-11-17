// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/filters"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	_ "github.com/krenalis/krenalis/warehouses/postgresql"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestJSONFilters verifies that SQL filters return the same records as the
// in-memory evaluator on both PostgreSQL and Snowflake.
func TestJSONFilters(t *testing.T) {

	// Check the differential oracle against independently specified matches.
	fixtures, cases := warehouseexpr.JSONCases()
	for _, c := range cases {
		t.Run("oracle/"+c.Name, func(t *testing.T) {
			values := slices.Clone(c.Values)
			where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"value"}, Operator: state.WhereOperator(c.Operator), Values: values},
			}}
			var matches []int
			for i, value := range fixtures {
				attributes := map[string]any{}
				if value != nil {
					attributes["value"] = value
				}
				if filters.Applies(where, attributes) {
					matches = append(matches, i)
				}
			}
			if !slices.Equal(c.Matches, matches) {
				t.Fatalf("expected matching fixture indices %v, got %v", c.Matches, matches)
			}
		})
	}

	columns := map[string]warehouses.Column{
		"value":        {Name: "value", Type: types.JSON(), Nullable: true},
		"document":     {Name: "document", Type: types.JSON(), Nullable: true},
		"array":        {Name: "array", Type: types.Array(types.JSON()), Nullable: true},
		"map":          {Name: "map", Type: types.Map(types.JSON()), Nullable: true},
		"object.value": {Name: "object_value", Type: types.JSON(), Nullable: true},
	}
	profileColumns := []warehouses.Column{
		columns["value"], columns["document"], columns["array"], columns["map"], columns["object.value"],
	}
	// Object existence must inspect SQL presence even for JSON descendants.
	columns["containers.json"] = columns["value"]
	columns["containers.array"] = columns["array"]
	columns["containers.map"] = columns["map"]
	idColumn := warehouses.Column{Name: "_kpid", Type: types.UUID()}
	table := warehouses.Table{
		Name: "krenalis_profiles_0",
		Columns: append([]warehouses.Column{
			idColumn,
			{Name: "_anonymous_count", Type: types.Int(32)},
			{Name: "_recognized_count", Type: types.Int(32)},
			{Name: "_updated_at", Type: types.DateTime()},
		}, profileColumns...),
		Keys: []string{"_kpid"},
	}

	rawValues := []string{
		"", "null", "true", "false", "0", "-0.0", "5", "5.000", "-5", "50", "499.9", "670", "670.00",
		"1e100", "-1e100", "1e-100", "-1e-100",
		"\"1e76\"", "\"1e-38\"", "\"1e999999\"",
		"0.005", "-0.005", "1e3", "1e-3", "9007199254740993", "99999999999999999999999999999999999999",
		"0.1234567890123456789012345678901234567",
		"\"\"", "\"foo\"", "\"foobar\"", "\"barfoo\"", "\"Foo\"", "\"foo \"", "\" foo\"",
		"\"670\"", "\"670.0\"", "\"+670\"", "\"6.7e2\"", "\"null\"", "\"true\"",
		"\"a'b\\\\c\"", "\"a\\nb\\tc\"", "\"é\"", "\"e\\u0301\"", "\"😀\"", "\"ÿ\"", "\"Ā\"",
		"{}", "{\"x\":null}", "{\"x\":\"foo\"}", "[]", "[null]", "[true,false]",
		"[\"foo\",670,null,true,{},[]]", "[670.0]", "[\"670\"]", "[[670]]", "[{\"x\":670}]",
		"9007199254740992", "9007199254740993e0", "1.234567890123456789e0",
		strings.Repeat("9", 37), strings.Repeat("9", 39), strings.Repeat("9", 76),
		"-9007199254740993", "-" + strings.Repeat("9", 38),
		"12.345678901234567890123456789012345678", "-12.345678901234567890123456789012345678",
		"1e308", "-1e308", "1e-308", "-1e-308", "1e-324",
	}
	var data [][]any
	// Nil represents a missing property (SQL NULL); json.Value("null") is present.
	add := func(value, document any, array []any, mapping map[string]any) {
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", len(data)+1)
		var arrayValue, mapValue any
		if array != nil {
			arrayValue = array
		}
		if mapping != nil {
			mapValue = mapping
		}
		data = append(data, []any{id, 0, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			value, document, arrayValue, mapValue, value})
	}
	for _, raw := range rawValues {
		if raw == "" {
			add(nil, nil, nil, nil)
			continue
		}
		value := json.Value(raw)
		document := json.Value("{\"a\":{\"b\":" + raw + "},\"a.b\":" + raw +
			",\"quoted'key\":{\"back\\\\slash\":" + raw + "},\"0\":" + raw + "}")
		add(value, document, []any{value}, map[string]any{"key": value})
	}
	add(nil, json.Value("{}"), []any{}, map[string]any{})
	for _, document := range []string{
		"null", "true", "5", "\"foo\"", "[]", "[{\"a\":{\"b\":670}}]",
		"{\"a\":null}", "{\"a\":5}", "{\"a\":[]}", "{\"a\":[{\"b\":670}]}",
		"{\"a\":{}}", "{\"a.b\":\"foo\"}", "{\"0\":670}", "[670]",
	} {
		add(nil, json.Value(document), nil, nil)
	}
	add(nil, nil, []any{json.Value("null"), json.Value("true"), json.Value("\"foo\""), json.Value("670")},
		map[string]any{"key": json.Value("null")})

	type condition struct {
		operator state.WhereOperator
		values   []any
	}
	var conditions []condition
	for _, c := range cases {
		conditions = append(conditions, condition{state.WhereOperator(c.Operator), slices.Clone(c.Values)})
	}
	operands := []json.Value{
		json.Value("670"), json.Value("670.0"), json.Value("6.7e2"), json.Value("0.005"), json.Value("-0"),
		json.Value("-5"), json.Value("0.001"), json.Value("9007199254740993"),
		json.Value(strings.Repeat("9", 38)), json.Value(strings.Repeat("9", 76)),
		json.Value("1e75"), json.Value("1e-37"), json.Value("0e999999"),
		json.Value("0.1234567890123456789012345678901234567"),
		json.Value("9007199254740993." + strings.Repeat("0", 36) + "1"),
		json.Value("-9007199254740993." + strings.Repeat("0", 36) + "1"),
		json.Value("12.3456789012345678901234567890123456781"),
		json.Value("-12.3456789012345678901234567890123456781"),
	}
	for _, text := range []string{
		"", "foo", "670", "670.0", "+670", "6.7e2", ".005", "0.", "-0", "-5", "0.001",
		"9007199254740993", strings.Repeat("9", 38), strings.Repeat("9", 76),
		"1e75", "1e76", "1e-37", "1e-38", "0e999999",
		"1e999999", "-1e999999", "1e-999999", "null", "true", "false",
		"a'b\\c", "a\nb\tc", "é", "e\u0301", "😀", "ÿ", "Ā",
	} {
		value, err := json.Marshal(text)
		if err != nil {
			t.Fatalf("expected String operand encoding, got %v", err)
		}
		operands = append(operands, value)
	}
	for _, op := range []state.WhereOperator{
		state.OpIs, state.OpIsNot, state.OpIsLessThan, state.OpIsLessThanOrEqualTo,
		state.OpIsGreaterThan, state.OpIsGreaterThanOrEqualTo, state.OpContains, state.OpDoesNotContain,
		state.OpStartsWith, state.OpEndsWith,
	} {
		for _, operand := range operands {
			if (op == state.OpStartsWith || op == state.OpEndsWith) && !operand.IsString() {
				continue
			}
			conditions = append(conditions, condition{op, []any{operand}})
		}
	}
	for _, op := range []state.WhereOperator{state.OpIsBetween, state.OpIsNotBetween} {
		for _, bounds := range [][2]json.Value{
			{json.Value("5"), json.Value("670")}, {json.Value("670"), json.Value("5")},
			{json.Value("670"), json.Value("670")}, {json.Value("-5"), json.Value("0.005")},
			{json.Value("1e-37"), json.Value("1e75")},
			{json.Value(`"foo"`), json.Value(`"foobar"`)}, {json.Value(`"foo"`), json.Value(`"670"`)},
			{json.Value(`"ÿ"`), json.Value(`"😀"`)},
			{json.Value("12.345678901234567890123456789012345678"), json.Value("12.3456789012345678901234567890123456781")},
			{json.Value("-12.3456789012345678901234567890123456781"), json.Value("-12.345678901234567890123456789012345678")},
		} {
			conditions = append(conditions, condition{op, []any{bounds[0], bounds[1]}})
		}
	}
	for _, op := range []state.WhereOperator{state.OpIsOneOf, state.OpIsNotOneOf} {
		for _, list := range [][]json.Value{
			{json.Value(`"foo"`)},
			{json.Value("670"), json.Value(`"foo"`), json.Value(`"null"`), json.Value("null"), json.Value("false")},
			{json.Value("670"), json.Value("670.0"), json.Value(`"+670"`)},
			{json.Value(`""`), json.Value(`"true"`), json.Value(`"é"`), json.Value("true")},
			{json.Value(`"1e76"`), json.Value(`"1e-38"`), json.Value("1e75"), json.Value("1e-37"), json.Value(strings.Repeat("9", 76))},
			{json.Value("9007199254740993"), json.Value("9007199254740993." + strings.Repeat("0", 36) + "1"), json.Value(`"foo"`)},
			{json.Value("12.3456789012345678901234567890123456781"), json.Value("-12.3456789012345678901234567890123456781")},
		} {
			values := make([]any, len(list))
			for i, value := range list {
				values[i] = value
			}
			conditions = append(conditions, condition{op, values})
		}
	}

	for _, name := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(name, func(t *testing.T) {

			warehouse := newJSONFilterTestWarehouse(t, name)
			err := warehouse.Initialize(t.Context(), profileColumns)
			if err != nil {
				t.Fatalf("expected warehouse initialization, got %v", err)
			}
			err = warehouse.Merge(t.Context(), table, data, nil)
			if err != nil {
				t.Fatalf("expected fixture merge, got %v", err)
			}

			// Evaluate the stored values, including warehouse numeric rounding.
			rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
				Table: table.Name, Columns: append([]warehouses.Column{idColumn}, profileColumns...),
				OrderBy: []warehouses.Column{idColumn},
			}, false)
			if err != nil {
				t.Fatalf("expected stored fixture query, got %v", err)
			}
			defer rows.Close()
			type record struct {
				id         string
				attributes map[string]any
			}
			var storedRecords []record
			values := make([]any, 1+len(profileColumns))
			for rows.Next() {
				err = rows.Scan(values...)
				if err != nil {
					t.Fatalf("expected stored fixture scan, got %v", err)
				}
				id, ok := values[0].(string)
				if !ok {
					t.Fatalf("expected stored string ID, got %T", values[0])
				}
				attributes := map[string]any{}
				for i, column := range profileColumns {
					if values[i+1] == nil {
						continue
					}
					if column.Name == "object_value" {
						attributes["object"] = map[string]any{"value": values[i+1]}
						continue
					}
					attributes[column.Name] = values[i+1]
				}
				containers := map[string]any{}
				for _, name := range []string{"value", "array", "map"} {
					if value, exists := attributes[name]; exists {
						key := name
						if name == "value" {
							key = "json"
						}
						containers[key] = value
					}
				}
				if len(containers) > 0 {
					attributes["containers"] = containers
				}
				storedRecords = append(storedRecords, record{id: id, attributes: attributes})
			}
			err = rows.Err()
			if err != nil {
				t.Fatalf("expected complete stored fixture iteration, got %v", err)
			}
			err = rows.Close()
			if err != nil {
				t.Fatalf("expected stored fixture query close, got %v", err)
			}
			if len(storedRecords) != len(data) {
				t.Fatalf("expected %d stored fixtures, got %d", len(data), len(storedRecords))
			}
			for i, record := range storedRecords {
				if record.id != data[i][0] {
					t.Fatalf("expected stored fixture ID %s, got %s", data[i][0], record.id)
				}
			}
			if name == "Snowflake" {

				// DOUBLE comparisons use canonical text, not the scanner's display.
				// Every numeric fixture repeats one scalar in these known containers.
				raw := warehouse.(interface {
					RawQuery(context.Context, string) (warehouses.Rows, int, error)
				})
				rows, _, err := raw.RawQuery(t.Context(), `SELECT "_KPID",AS_DOUBLE("VALUE") FROM `+
					`"KRENALIS_PROFILES_0" WHERE TYPEOF("VALUE")='DOUBLE'`)
				if err != nil {
					t.Fatalf("expected native numeric fixture query, got %v", err)
				}
				defer rows.Close()

				for rows.Next() {
					var id string
					var number float64
					err = rows.Scan(&id, &number)
					if err != nil {
						t.Fatalf("expected native numeric fixture scan, got %v", err)
					}
					index := slices.IndexFunc(storedRecords, func(r record) bool { return r.id == id })
					if index < 0 || index >= len(rawValues) {
						t.Fatalf("expected scalar numeric fixture ID, got %s", id)
					}
					value := json.Value(strconv.FormatFloat(number, 'g', -1, 64))
					document, err := json.Marshal(map[string]any{
						"a": map[string]any{"b": value}, "a.b": value,
						"quoted'key": map[string]any{"back\\slash": value}, "0": value,
					})
					if err != nil {
						t.Fatalf("expected canonical numeric document, got %v", err)
					}
					attributes := storedRecords[index].attributes
					attributes["value"], attributes["document"] = value, document
					attributes["array"], attributes["map"] = []any{value}, map[string]any{"key": value}
					attributes["object"] = map[string]any{"value": value}
					attributes["containers"] = map[string]any{
						"json": value, "array": attributes["array"], "map": attributes["map"],
					}
				}

				err = rows.Err()
				closeErr := rows.Close()
				if err != nil {
					t.Fatalf("expected complete native numeric fixtures, got %v", err)
				}
				if closeErr != nil {
					t.Fatalf("expected native numeric fixture close, got %v", closeErr)
				}

			}

			type queryCase struct {
				name  string
				where *state.Where
			}
			var queries []queryCase

			for _, path := range [][]string{
				{"value"}, {"document", "a", "b"}, {"document", "a.b"},
				{"document", "quoted'key", "back\\slash"}, {"document", "0"}, {"object", "value"},
			} {
				for _, cond := range conditions {
					texts := make([]string, len(cond.values))
					for i, value := range cond.values {
						texts[i] = string(value.(json.Value))
					}
					queries = append(queries, queryCase{
						name: fmt.Sprintf("%s/%s/%q", strings.Join(path, "/"), cond.operator, texts),
						where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
							&state.WhereCondition{Property: path, Operator: cond.operator, Values: cond.values},
						}},
					})
				}
			}
			for _, path := range [][]string{{"containers", "json"}, {"containers", "array"}, {"containers", "map"}} {
				for _, op := range []state.WhereOperator{state.OpExists, state.OpDoesNotExist} {
					queries = append(queries, queryCase{
						name: strings.Join(path, "/") + "/" + op.String(),
						where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
							&state.WhereCondition{Property: path, Operator: op},
						}},
					})
				}
			}
			for _, path := range []string{"array", "map", "object", "containers"} {
				for _, cond := range conditions {
					allowed := cond.operator == state.OpExists || cond.operator == state.OpDoesNotExist
					if path == "array" || path == "map" {
						allowed = allowed || cond.operator == state.OpIsEmpty || cond.operator == state.OpIsNotEmpty
					}
					if path == "array" {
						allowed = allowed || cond.operator == state.OpContains || cond.operator == state.OpDoesNotContain
					}
					if !allowed {
						continue
					}
					texts := make([]string, len(cond.values))
					for i, value := range cond.values {
						texts[i] = string(value.(json.Value))
					}
					queries = append(queries, queryCase{
						name: fmt.Sprintf("%s/%s/%q", path, cond.operator, texts),
						where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
							&state.WhereCondition{Property: []string{path}, Operator: cond.operator, Values: cond.values},
						}},
					})
				}
			}
			queries = append(queries, queryCase{
				name: "nested groups",
				where: &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"document", "a", "b"}, Operator: state.OpIsNotNull},
					&state.Where{Operator: state.OpOr, Rules: []state.WhereRule{
						&state.WhereCondition{Property: []string{"value"}, Operator: state.OpIsNot, Values: []any{json.Value("670")}},
						&state.WhereCondition{Property: []string{"array"}, Operator: state.OpContains, Values: []any{json.Value("\"foo\"")}},
					}},
				}},
			})

			// Exact membership follows from the expected count and no unexpected IDs.
			// Batch predicates to avoid thousands of Snowflake round trips.
			// Each condition contributes two count expressions.
			const batchSize = 16
			for start := 0; start < len(queries); start += batchSize {
				batch := queries[start:min(start+batchSize, len(queries))]
				expressions := make([]warehouses.Expr, 0, 2*len(batch))
				wants := make([]int, len(batch))
				for i, query := range batch {
					var ids []any
					for _, record := range storedRecords {
						if filters.Applies(query.where, record.attributes) {
							ids = append(ids, record.id)
						}
					}
					wants[i] = len(ids)
					expr, err := convertWhere(query.where, columns)
					if err != nil {
						t.Fatalf("expected conversion for %s, got %v", query.name, err)
					}
					unexpected := expr
					if len(ids) > 0 {
						unexpected = warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
							expr, warehouses.NewBaseExpr(idColumn, warehouses.OpIsNotOneOf, ids...),
						})
					}
					expressions = append(expressions, expr, unexpected)
				}
				counts, err := warehouse.Counts(t.Context(), table.Name, expressions)
				if err != nil {
					t.Fatalf("expected query execution for batch starting at %s, got %v", batch[0].name, err)
				}
				for i, query := range batch {
					t.Run(query.name, func(t *testing.T) {
						if counts[2*i] != wants[i] || counts[2*i+1] != 0 {
							t.Fatalf("expected %d matching rows and 0 unexpected IDs, got %d and %d", wants[i], counts[2*i], counts[2*i+1])
						}
					})
				}
			}

		})
	}

}

// TestJSONFloatFilters compares canonical Float JSON and exact operands on both
// warehouses without relying on the warehouse's potentially rounded JSON text.
func TestJSONFloatFilters(t *testing.T) {
	for _, name := range []string{"PostgreSQL", "Snowflake"} {
		t.Run(name, func(t *testing.T) {

			warehouse := newJSONFilterTestWarehouse(t, name)
			idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
			valueColumn := warehouses.Column{Name: "value", Type: types.JSON()}
			documentColumn := warehouses.Column{Name: "document", Type: types.JSON()}
			table := warehouses.Table{Name: "json_floats", Keys: []string{"id"},
				Columns: []warehouses.Column{idColumn, valueColumn, documentColumn}}
			switch name {
			case "PostgreSQL":
				execBehavioralFixture(t, warehouse, `CREATE TABLE json_floats (id integer PRIMARY KEY,value jsonb,document jsonb)`)
			case "Snowflake":
				execBehavioralFixture(t, warehouse, `CREATE TABLE "JSON_FLOATS" ("ID" INTEGER,"VALUE" VARIANT,"DOCUMENT" VARIANT)`)
			default:
				t.Fatalf("expected supported warehouse, got %s", name)
			}
			var data [][]any
			for i, fixture := range []struct {
				value any
				typ   types.Type
			}{
				{float64(0), types.Float(64)}, {math.Copysign(0, -1), types.Float(64)},
				{float64(100), types.Float(64)}, {decimal.MustParse("100"), types.Decimal(18, 4)},
				{float64(1234567.1), types.Float(64)}, {float64(-1234567.1), types.Float(64)},
				{float64(12345678.9), types.Float(64)}, {float64(0.00001), types.Float(64)},
				{float64(0.000015), types.Float(64)}, {float64(float32(1234567.1)), types.Float(32)},
				{math.Nextafter(1234567.1, math.Inf(1)), types.Float(64)},
				{float64(0.30000000000000004), types.Float(64)},
			} {
				value, err := types.Marshal(fixture.value, fixture.typ)
				if err != nil {
					t.Fatalf("expected numeric fixture serialization, got %v", err)
				}
				data = append(data, []any{i, value, json.Value(`{"score":` + string(value) + `}`)})
			}
			err := warehouse.Merge(t.Context(), table, data, nil)
			if err != nil {
				t.Fatalf("expected numeric fixture merge, got %v", err)
			}

			var expressions []warehouses.Expr
			var expected []int
			var descriptions []string
			for _, record := range data {
				id := record[0].(int)
				value := record[1].(json.Value)
				number, ok := new(big.Rat).SetString(string(value))
				if !ok {
					t.Fatalf("expected decimal fixture, got %s", value)
				}
				// A finer operand must not be rounded back to this stored Float.
				step, _ := new(big.Rat).SetString("1e-20")
				after := json.Value(new(big.Rat).Add(number, step).FloatString(20))
				attributes := map[string]any{"value": value, "document": record[2]}
				for _, literal := range []json.Value{value, after} {
					operand, err := state.DecodeFilterOperand(literal, types.JSON(), state.OpIs)
					if err != nil {
						t.Fatalf("expected exact numeric operand, got %v", err)
					}
					for _, op := range []state.WhereOperator{state.OpIs, state.OpIsNot, state.OpIsLessThan,
						state.OpIsLessThanOrEqualTo, state.OpIsGreaterThan, state.OpIsGreaterThanOrEqualTo,
						state.OpIsBetween, state.OpIsNotBetween, state.OpIsOneOf, state.OpIsNotOneOf} {
						for _, path := range [][]string{{"value"}, {"document", "score"}} {
							values := []any{operand}
							if op == state.OpIsBetween || op == state.OpIsNotBetween {
								values = append(values, operand)
							}
							where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
								&state.WhereCondition{Property: path, Operator: op, Values: values},
							}}
							expr, err := convertWhere(where, map[string]warehouses.Column{
								"value": valueColumn, "document": documentColumn,
							})
							if err != nil {
								t.Fatalf("expected numeric binding, got %v", err)
							}
							expressions = append(expressions, warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
								warehouses.NewBaseExpr(idColumn, warehouses.OpIs, id), expr,
							}))
							want := 0
							if filters.Applies(where, attributes) {
								want = 1
							}
							expected = append(expected, want)
							descriptions = append(descriptions, fmt.Sprintf("%d/%s/%s/%s", id, strings.Join(path, "/"), op, literal))
						}
					}
				}
			}
			const batchSize = 32
			for start := 0; start < len(expressions); start += batchSize {
				end := min(start+batchSize, len(expressions))
				counts, err := warehouse.Counts(t.Context(), table.Name, expressions[start:end])
				if err != nil {
					t.Fatalf("expected numeric filter batch, got %v", err)
				}
				if len(counts) != end-start {
					t.Fatalf("expected %d numeric counts, got %d", end-start, len(counts))
				}
				for i, count := range counts {
					index := start + i
					t.Run(descriptions[index], func(t *testing.T) {
						if count != expected[index] {
							t.Fatalf("expected %d matching records, got %d", expected[index], count)
						}
					})
				}
			}

		})
	}
}

// newJSONFilterTestWarehouse creates one environment for all JSON filter tests.
func newJSONFilterTestWarehouse(t *testing.T, name string) warehouses.Warehouse {

	t.Helper()
	var settings json.Value
	switch name {
	case "PostgreSQL":
		container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
			postgres.WithDatabase(testDatabase), postgres.WithUsername(testUser), postgres.WithPassword(testPassword),
			postgres.BasicWaitStrategies())
		if err != nil {
			t.Fatalf("expected PostgreSQL container, got %v", err)
		}
		t.Cleanup(func() {
			err := testcontainers.TerminateContainer(container)
			if err != nil {
				t.Errorf("expected container cleanup, got %v", err)
			}
		})
		host, err := container.Host(t.Context())
		if err != nil {
			t.Fatalf("expected container host, got %v", err)
		}
		port, err := container.MappedPort(t.Context(), "5432/tcp")
		if err != nil {
			t.Fatalf("expected PostgreSQL port, got %v", err)
		}
		settings, err = json.Marshal(map[string]any{
			"host": host, "port": port.Num(), "username": testUser, "password": testPassword,
			"database": testDatabase, "schema": "public",
		})
		if err != nil {
			t.Fatalf("expected warehouse settings, got %v", err)
		}
	case "Snowflake":
		if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
			t.Skip("Snowflake tests disabled")
		}
		environment, err := snowflaketester.CreateTestEnvironment()
		if err != nil {
			t.Fatalf("expected Snowflake environment, got %v", err)
		}
		t.Cleanup(func() {
			err := environment.Teardown()
			if err != nil {
				t.Errorf("expected Snowflake cleanup, got %v", err)
			}
		})
		settings = environment.Settings().JSON()
	default:
		t.Fatalf("expected supported JSON filter test warehouse, got %q", name)
	}
	warehouse := warehouses.Registered(name).New(newTestSettingsLoader(settings), nil)
	t.Cleanup(func() {
		err := warehouse.Close()
		if err != nil {
			t.Errorf("expected warehouse close, got %v", err)
		}
	})

	return warehouse
}
