// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"database/sql"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestJSONDoubleBounds checks canonical decimal enclosure and adjacency
// without using floating-point decimal comparisons as its oracle.
func TestJSONDoubleBounds(t *testing.T) {
	for _, text := range []string{"0", "-0", "100", "1.25", "0.1", "-0.1", "1e-37", "-1e-37", "1e75",
		"100.00000000000001", "100.00000000000000001", "-100.00000000000000001",
		"1234567.1", "12345678.9", "0.00001", "0.000015", "0.30000000000000004",
		"182771880194942800", "-182771880194942800",
		"99999999999999.9999", "-99999999999999.9999",
		"123456789012345678901234567890123456789", strings.Repeat("9", 76),
		"-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)} {
		t.Run(text, func(t *testing.T) {
			value, ok := new(big.Rat).SetString(text)
			if !ok {
				t.Fatalf("expected exact fixture, got %s", text)
			}
			lower, upper := jsonDoubleBounds(decimal.MustParse(text))
			lo, _ := new(big.Rat).SetString(strconv.FormatFloat(lower, 'g', -1, 64))
			hi, _ := new(big.Rat).SetString(strconv.FormatFloat(upper, 'g', -1, 64))
			if lo == nil || hi == nil || lo.Cmp(value) > 0 || hi.Cmp(value) < 0 {
				t.Fatalf("expected enclosing finite bounds for %s, got %.17g and %.17g", text, lower, upper)
			}
			if lower == upper {
				if lo.Cmp(value) != 0 {
					t.Fatalf("expected exact common bound for %s, got %s", text, lo)
				}
			} else if math.Nextafter(lower, math.Inf(1)) != upper || lo.Cmp(value) == 0 || hi.Cmp(value) == 0 {
				t.Fatalf("expected strictly enclosing adjacent bounds for %s, got %.17g and %.17g", text, lower, upper)
			}
		})
	}
}

// TestJSONNumbers compares the real renderer with exact rational arithmetic on
// the stored value. DOUBLEs use their shortest round-trip decimal, derived
// from binary transport rather than the rounded JSON display text.
func TestJSONNumbers(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	db.SetMaxOpenConns(1)
	mustExecSQL(t, db, "ALTER SESSION SET STATEMENT_TIMEOUT_IN_SECONDS=60")
	mustExecSQL(t, db, `CREATE TABLE "JSON_NUMBERS" ("ID" INTEGER,"VALUE" VARIANT,"ELEMENTS" ARRAY)`)
	column := warehouses.Column{Name: "value", Type: types.JSON(), Nullable: true}
	elements := warehouses.Column{Name: "elements", Type: types.Array(types.JSON()), Nullable: true}
	table := warehouses.Table{Name: "json_numbers", Keys: []string{"id"}, Columns: []warehouses.Column{
		{Name: "id", Type: types.Int(32)}, column, elements,
	}}
	inputs := []string{"", "null", `"100"`, "false", "{}", "[]", "0", "-0.0000", "100", "100.0", "1E2",
		"1.1", "1.1E0", "1.25E0", "1.0000000000000001E2", "1.00000000000000001E2", "-1.1E0",
		"1.2345671e+06", "1.23456789e+07", "1e-05", "1.5e-05",
		"0.30000000000000004", "3.0000000000000004E-1",
		"1.827718801949428E17", "-1.827718801949428E17", "182771880194942800", "-182771880194942800",
		"1E-310", "1E75", "9.99999999999999375E13", "-9.99999999999999375E13", "100.00001",
		"1.7976931348623157E308", "-1.7976931348623157E308",
		"99999999999999.9999", "-99999999999999.9999", "0.0000000000000000000000000000000000001",
		"0.1234567890123456789012345678901234567", "-0.1234567890123456789012345678901234567",
		"1.2345678901234567890123456789012345678", "12345678901234567890123456789012345678.9",
		"-12345678901234567890123456789012345678.9",
		"123456789012345678901234567890123456789", "123456789012345678901234567890123456788",
		"-123456789012345678901234567890123456789", strings.Repeat("9", 76)}
	data := make([][]any, len(inputs))
	for i, raw := range inputs {
		data[i] = []any{i, nil, nil}
		if raw != "" {
			data[i][1], data[i][2] = json.Value(raw), []any{json.Value(raw)}
		}
	}
	err := warehouse.Merge(t.Context(), table, data, nil)
	if err != nil {
		t.Fatalf("expected numeric fixture merge, got %v", err)
	}
	mergedFixtureCount := len(inputs)
	// Keep scientific DOUBLE fixtures as well as the new fixed-point writes.
	for _, raw := range inputs {
		if raw == "" || json.Value(raw).Kind() != json.Number || !strings.ContainsAny(raw, "eE") {
			continue
		}
		id := len(inputs)
		inputs = append(inputs, raw)
		mustExecSQL(t, db, fmt.Sprintf(`INSERT INTO "JSON_NUMBERS" SELECT %d,
PARSE_JSON('%s'),ARRAY_CONSTRUCT(PARSE_JSON('%s'))`, id, raw, raw))
	}
	jsonFixtureCount := len(inputs)
	// These values cannot arrive from JSON; exercise direct warehouse writes.
	for _, special := range []string{"NaN", "inf", "-inf"} {
		id := len(inputs)
		inputs = append(inputs, special)
		mustExecSQL(t, db, fmt.Sprintf(`INSERT INTO "JSON_NUMBERS" SELECT %d,
TO_VARIANT(TO_DOUBLE('%s')),ARRAY_CONSTRUCT(TO_VARIANT(TO_DOUBLE('%s')))`, id, special, special))
	}

	type test struct {
		operator warehouses.Operator
		literal  string
	}
	var tests []test
	var query strings.Builder
	query.WriteString(`SELECT "ID",TYPEOF("VALUE"),TO_JSON("VALUE"),AS_DOUBLE("VALUE")`)
	for _, literal := range []string{"0", "-1", "0.1", "-0.1", "1.25", "100", "100.00000000000001",
		"100.00000000000000001", "100.00000000000000002", "1234567.1", "12345678.9",
		"0.00001", "0.000015", "0.30000000000000004",
		"182771880194942800", "-182771880194942800",
		"99999999999999.9999", "-99999999999999.9999", "1e-37", "1e75", strings.Repeat("9", 76),
		"123456789012345678901234567890123456789", "123456789012345678901234567890123456788",
		"-123456789012345678901234567890123456789", "12345678901234567890123456789012345678.9",
		"-" + strings.Repeat("9", 39) + "." + strings.Repeat("8", 37)} {
		for _, op := range []warehouses.Operator{warehouses.OpIs, warehouses.OpIsNot, warehouses.OpIsLessThan,
			warehouses.OpIsLessThanOrEqualTo, warehouses.OpIsGreaterThan, warehouses.OpIsGreaterThanOrEqualTo,
			warehouses.OpIsBetween, warehouses.OpIsNotBetween, warehouses.OpIsOneOf, warehouses.OpIsNotOneOf,
			warehouses.OpContains, warehouses.OpDoesNotContain} {
			values := []any{json.Value(literal)}
			if op == warehouses.OpIsBetween || op == warehouses.OpIsNotBetween {
				values = append(values, values[0])
			}
			target := column
			if op == warehouses.OpContains || op == warehouses.OpDoesNotContain {
				target = elements
			}
			expr := newJSONFilterExpr(target, op, values...)
			query.WriteString(",(")
			err := renderExpr(&query, expr)
			if err != nil {
				t.Fatalf("expected numeric fixture rendering, got %v", err)
			}
			query.WriteByte(')')
			tests = append(tests, test{op, literal})
		}
	}
	fmt.Fprintf(&query, ` FROM "JSON_NUMBERS" ORDER BY "ID" LIMIT %d`, len(inputs)+1)
	rows, err := db.QueryContext(t.Context(), query.String())
	if err != nil {
		t.Fatalf("expected executable numeric comparisons, got %v", err)
	}
	defer rows.Close()
	fixedSyntax := regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	expectedFixed := map[int]*big.Rat{}
	count := 0
	for rows.Next() {
		var id int
		var kind, stored sql.NullString
		var double sql.NullFloat64
		results := make([]sql.NullBool, len(tests))
		destinations := []any{&id, &kind, &stored, &double}
		for i := range results {
			destinations = append(destinations, &results[i])
		}
		if err := rows.Scan(destinations...); err != nil {
			t.Fatalf("expected numeric comparison row, got %v", err)
		}
		if id != count || count >= len(inputs) {
			t.Fatalf("expected numeric fixture %d within %d rows, got %d", count, len(inputs), id)
		}
		t.Logf("fixture %q: TYPEOF=%q TO_JSON=%q", inputs[id], kind.String, stored.String)
		if id >= jsonFixtureCount && kind.String != "DOUBLE" {
			t.Fatalf("expected non-finite DOUBLE fixture, got %s", kind.String)
		}
		if id >= mergedFixtureCount &&
			(inputs[id] == "1.827718801949428E17" || inputs[id] == "-1.827718801949428E17") &&
			kind.String != "DOUBLE" {
			t.Fatalf("expected midpoint DOUBLE fixture, got %s", kind.String)
		}
		if id < mergedFixtureCount && (inputs[id] == "1.1E0" || inputs[id] == "1.00000000000000001E2") &&
			kind.String != "DECIMAL" {
			t.Fatalf("expected exact decimal write of %s, got %s", inputs[id], kind.String)
		}
		if (inputs[id] == "0.1234567890123456789012345678901234567" ||
			inputs[id] == "-0.1234567890123456789012345678901234567") && kind.String != "DECIMAL" {
			t.Fatalf("expected fixed-point scale-37 fixture, got %s", kind.String)
		}
		var value *big.Rat
		switch kind.String {
		case "INTEGER", "DECIMAL":
			if !stored.Valid || !fixedSyntax.MatchString(stored.String) {
				t.Fatalf("expected plain fixed-point JSON syntax, got %q", stored.String)
			}
			var ok bool
			value, ok = new(big.Rat).SetString(stored.String)
			if !ok {
				t.Fatalf("expected exact fixed-point value, got %s", stored.String)
			}
			original, ok := new(big.Rat).SetString(inputs[id])
			if !ok || value.Cmp(original) != 0 {
				t.Fatalf("expected fixed-point preservation of %s, got %s", inputs[id], stored.String)
			}
			if id < mergedFixtureCount {
				expectedFixed[id] = original
			}
		case "DOUBLE":
			if !double.Valid {
				t.Fatal("expected binary floating-point transport, got NULL")
			}
			if !math.IsNaN(double.Float64) && !math.IsInf(double.Float64, 0) {
				value, _ = new(big.Rat).SetString(strconv.FormatFloat(double.Float64, 'g', -1, 64))
			}
			if value == nil && id < jsonFixtureCount {
				t.Fatalf("expected finite numeric fixture, got %g", double.Float64)
			}
			if value != nil && id >= jsonFixtureCount {
				t.Fatalf("expected non-finite fixture, got %.17g", double.Float64)
			}
			if inputs[id] == "1.0000000000000001E2" && double.Float64 != math.Nextafter(100, math.Inf(1)) {
				t.Fatalf("expected the DOUBLE after 100, got %.17g", double.Float64)
			}
			if inputs[id] == "1.827718801949428E17" || inputs[id] == "-1.827718801949428E17" {
				// The canonical decimal is exactly halfway: choose the even neighbor.
				want := float64(182771880194942784)
				if inputs[id][0] == '-' {
					want = -want
				}
				if double.Float64 != want {
					t.Fatalf("expected nearest-even DOUBLE %.17g, got %.17g", want, double.Float64)
				}
			}
		default:
			// Other fixture kinds are not numbers.
		}
		for i, test := range tests {
			literal, ok := new(big.Rat).SetString(test.literal)
			if !ok {
				t.Fatalf("expected exact operand, got %s", test.literal)
			}
			want := sql.NullBool{Valid: inputs[id] != ""}
			cmp := 0
			if value != nil {
				cmp = value.Cmp(literal)
			}
			switch test.operator {
			case warehouses.OpIs, warehouses.OpIsOneOf, warehouses.OpContains:
				want.Bool = value != nil && cmp == 0
			case warehouses.OpIsNot, warehouses.OpIsNotOneOf, warehouses.OpDoesNotContain:
				want.Bool = value == nil || cmp != 0
			case warehouses.OpIsLessThan:
				want.Bool, want.Valid = cmp < 0, value != nil
			case warehouses.OpIsLessThanOrEqualTo:
				want.Bool, want.Valid = cmp <= 0, value != nil
			case warehouses.OpIsGreaterThan:
				want.Bool, want.Valid = cmp > 0, value != nil
			case warehouses.OpIsGreaterThanOrEqualTo:
				want.Bool, want.Valid = cmp >= 0, value != nil
			case warehouses.OpIsBetween:
				want.Bool, want.Valid = cmp == 0, value != nil
			case warehouses.OpIsNotBetween:
				want.Bool, want.Valid = cmp != 0, value != nil
			default:
				t.Fatalf("expected numeric operator, got %d", test.operator)
			}
			if !want.Valid {
				want.Bool = false
			}
			if results[i] != want {
				t.Errorf("expected %s (%s), operator %d, operand %s: %v, got %v",
					inputs[id], kind.String, test.operator, test.literal, want, results[i])
			}
		}
		count++
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete numeric comparisons, got %v", err)
	}
	if count != len(inputs) {
		t.Fatalf("expected %d numeric fixture rows, got %d", len(inputs), count)
	}
	err = rows.Close()
	if err != nil {
		t.Fatalf("expected numeric result close, got %v", err)
	}
	t.Logf("checked %d exact stored-value comparisons", count*len(tests))

	// Read the new fixed-point writes through the warehouse's actual scanner.
	// FLOAT JSON output preserves the server's text under the accepted contract.
	var ids []any
	for i := range mergedFixtureCount {
		if expectedFixed[i] != nil {
			ids = append(ids, i)
		}
	}
	result, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
		Table:   table.Name,
		Columns: table.Columns,
		Where:   warehouses.NewBaseExpr(table.Columns[0], warehouses.OpIsOneOf, ids...),
		OrderBy: []warehouses.Column{table.Columns[0]},
	}, false)
	if err != nil {
		t.Fatalf("expected fixed-point warehouse query, got %v", err)
	}
	defer result.Close()
	count = 0
	for result.Next() {
		row := make([]any, len(table.Columns))
		if err := result.Scan(row...); err != nil {
			t.Fatalf("expected decoded fixed-point row, got %v", err)
		}
		id, ok := row[0].(int)
		if !ok || count >= len(ids) || id != ids[count] {
			t.Fatalf("expected ordered fixed-point identity at %d, got %v", count, row[0])
		}
		raw, ok := row[1].(json.Value)
		if !ok {
			t.Fatalf("expected JSON number, got %T", row[1])
		}
		value, ok := new(big.Rat).SetString(string(raw))
		if !ok || value.Cmp(expectedFixed[id]) != 0 {
			t.Fatalf("expected exact JSON round-trip of %s, got %s", inputs[id], raw)
		}
		count++
	}
	err = result.Err()
	if err != nil {
		t.Fatalf("expected complete fixed-point result, got %v", err)
	}
	if count != len(ids) {
		t.Fatalf("expected %d fixed-point rows, got %d", len(ids), count)
	}
	err = result.Close()
	if err != nil {
		t.Fatalf("expected fixed-point result close, got %v", err)
	}

}

// TestProfileExactDecimal exercises the physical VARIANT reading through the
// real writer, including binary values whose display text loses information.
func TestProfileExactDecimal(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	db.SetMaxOpenConns(1)
	mustExecSQL(t, db, "ALTER SESSION SET STATEMENT_TIMEOUT_IN_SECONDS=60")
	mustExecSQL(t, db, `CREATE TABLE "EXACT_DECIMALS" ("ID" INTEGER,"VALUE" VARIANT)`)
	column := warehouses.Column{Name: "value", Type: types.JSON(), Nullable: true}
	table := warehouses.Table{Name: "exact_decimals", Keys: []string{"id"}, Columns: []warehouses.Column{
		{Name: "id", Type: types.Int(32)}, column,
	}}
	fixtures := []struct{ raw, known string }{
		{"", ""}, {"null", ""}, {`"100"`, ""}, {"true", ""}, {"{}", ""}, {"[]", ""},
		{"100", "100"}, {"100.0", "100"}, {"1E2", "100"}, {"1.2E3", "1200"}, {"9E13", "90000000000000"},
		{"-3E1", "-30"}, {"1.1", "1.1"}, {"1.1E0", "1.1"}, {"1.25E0", "1.25"}, {"-0E0", "0"},
		{"0.0001", "0.0001"}, {"99999999999999.9999", "99999999999999.9999"},
		{"-99999999999999.9999", "-99999999999999.9999"}, {"9.99999999999999999E13", "99999999999999.9999"},
		{"9.99999999999999375E13", "99999999999999.9375"}, {"-9.99999999999999375E13", "-99999999999999.9375"},
		{"1.0000000000000001E2", ""}, {"1.00000000000000001E2", ""}, {"100.00001", ""},
		{"1E14", ""}, {"1E307", ""}, {"1E-310", ""}, {"123456789012345678901234567890123456789", ""},
	}
	data := make([][]any, len(fixtures))
	for i, fixture := range fixtures {
		data[i] = []any{i, nil}
		if fixture.raw != "" {
			data[i][1] = json.Value(fixture.raw)
		}
	}
	err := warehouse.Merge(t.Context(), table, data, nil)
	if err != nil {
		t.Fatalf("expected decimal fixture merge, got %v", err)
	}
	// Historical DOUBLEs keep the approved binary-exact monetary reading.
	for _, old := range []struct{ raw, known string }{
		{"1.1E0", ""}, {"9.99999999999999999E13", ""}, {"1.00000000000000001E2", "100"},
	} {
		id := len(fixtures)
		fixtures = append(fixtures, old)
		mustExecSQL(t, db, fmt.Sprintf(`INSERT INTO "EXACT_DECIMALS" SELECT %d,PARSE_JSON('%s')`, id, old.raw))
	}
	reading, err := profileExactDecimal(warehouses.DecimalReading{Column: column, Type: types.Decimal(18, 4)})
	if err != nil {
		t.Fatalf("expected decimal reading, got %v", err)
	}
	rows, err := db.QueryContext(t.Context(), `SELECT "ID",TO_VARCHAR((`+reading+
		`)) FROM "EXACT_DECIMALS" ORDER BY "ID"`)
	if err != nil {
		t.Fatalf("expected executable decimal reading, got %v", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var id int
		var got sql.NullString
		if err := rows.Scan(&id, &got); err != nil {
			t.Fatalf("expected decimal row, got %v", err)
		}
		if id != count || count >= len(fixtures) {
			t.Fatalf("expected decimal fixture %d within %d rows, got %d", count, len(fixtures), id)
		}
		fixture := fixtures[id]
		if got.Valid != (fixture.known != "") {
			t.Errorf("expected %s readability %t, got %v", fixture.raw, fixture.known != "", got)
		}
		if got.Valid && fixture.known != "" {
			want := decimal.MustParse(fixture.known)
			value, err := decimal.Parse(got.String, 0, 0)
			if err != nil {
				t.Fatalf("expected exact readback, got %v", err)
			}
			if !value.Equal(want) {
				t.Errorf("expected %s as %s, got %s", fixture.raw, want, value)
			}
		}
		count++
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete decimal reading, got %v", err)
	}
	if count != len(fixtures) {
		t.Fatalf("expected %d decimal rows, got %d", len(fixtures), count)
	}
	err = rows.Close()
	if err != nil {
		t.Fatalf("expected decimal result close, got %v", err)
	}
	for _, special := range []string{"NaN", "inf", "-inf"} {
		statement := `SELECT TO_VARCHAR((` + strings.ReplaceAll(reading, `"VALUE"`,
			"TO_VARIANT(TO_DOUBLE('"+special+"'))") + `))`
		var got sql.NullString
		err := db.QueryRowContext(t.Context(), statement).Scan(&got)
		if err != nil {
			t.Fatalf("expected safe special-value reading, got %v", err)
		}
		if got.Valid {
			t.Errorf("expected %s to be unreadable, got %s", special, got.String)
		}
	}

}
