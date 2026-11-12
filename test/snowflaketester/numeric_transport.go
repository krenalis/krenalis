// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflaketester

import (
	"context"
	"database/sql"
	stdjson "encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/google/go-cmp/cmp"
	"github.com/snowflakedb/gosnowflake/v2"
)

// Numeric transport fixtures use one bounded, identically named table.
const (
	NumericTransportTable = "NUMERIC_TRANSPORT"
	NumericTransportDDL   = `CREATE TABLE "NUMERIC_TRANSPORT"
 ("ID" INTEGER,"VALUE" VARIANT,"ELEMENTS" ARRAY,"ATTRIBUTES" OBJECT)`
	NumericTransportSQL = `SELECT "ID","VALUE","ELEMENTS","ATTRIBUTES"
 FROM "NUMERIC_TRANSPORT" ORDER BY "ID" LIMIT 65`
)

// NumericTransportCase describes a stored numeric value or a null fixture.
type NumericTransportCase struct {
	Name       string
	Number     string
	Kind       string
	Historical bool
	Compact    bool
}

// Row builds a writer input; an empty Number means SQL NULL in all three cells.
func (c NumericTransportCase) Row(id int) []any {
	if id < 0 || id >= 64 || len(c.Number) > 400 || c.Number != "" && !json.Valid([]byte(c.Number)) {
		panic("test/snowflaketester: invalid numeric transport row")
	}
	if c.Number == "" {
		return []any{id, nil, nil, nil}
	}
	return []any{id,
		json.Value(`{"n":` + c.Number + `,"nested":{"a.b[0]":[` + c.Number + `,` +
			c.Number + `]},"null":null,"text":"1E2"}`),
		[]any{json.Value(c.Number), json.Value(`{"n":` + c.Number + `,"text":"1E2"}`), json.Value(`null`)},
		map[string]any{"n": json.Value(c.Number), "a.b[0]": json.Value(`{"n":[` + c.Number + `]}`),
			"null": json.Value(`null`), "text": json.Value(`"1E2"`)}}
}

// NumericTransportReference holds bounded server text from a diagnostic read.
type NumericTransportReference struct {
	Cells [3]string
	Null  bool
}

type numericTransportDifference struct {
	path    cmp.Path
	message string
}

// PopStep leaves the current JSON node.
func (d *numericTransportDifference) PopStep() {
	d.path = d.path[:len(d.path)-1]
}

// PushStep enters the current JSON node.
func (d *numericTransportDifference) PushStep(step cmp.PathStep) {
	d.path = append(d.path, step)
}

// Report records the first unequal pair of numeric tokens, bounded and quoted.
func (d *numericTransportDifference) Report(result cmp.Result) {
	if result.Equal() || d.message != "" {
		return
	}
	got, want := d.path.Last().Values()
	if !got.IsValid() || !want.IsValid() {
		return
	}
	a, gotNumber := got.Interface().(stdjson.Number)
	b, wantNumber := want.Interface().(stdjson.Number)
	if !gotNumber || !wantNumber {
		return
	}
	tokens := [2]string{a.String(), b.String()}
	for i, token := range tokens {
		if len(token) > 400 {
			tokens[i] = token[:397] + "..."
		}
	}
	d.message = fmt.Sprintf("numeric JSON differs: expected token %q, got %q", tokens[1], tokens[0])
}

// CheckNumericTransportJSON compares JSON structure while preserving number
// tokens when exactTokens is true. Numeric equivalence uses bounded rationals.
func CheckNumericTransportJSON(got, want string, exactTokens bool) error {
	values := [2]any{}
	for i, text := range []string{got, want} {
		if len(text) > 16*1024 || !stdjson.Valid([]byte(text)) {
			return fmt.Errorf("test/snowflaketester: invalid or oversized numeric JSON")
		}
		decoder := stdjson.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		err := decoder.Decode(&values[i])
		if err != nil {
			return fmt.Errorf("test/snowflaketester: invalid numeric JSON")
		}
	}
	equalNumber := cmp.Comparer(func(a, b stdjson.Number) bool {
		if exactTokens || a == b {
			return a == b
		}
		for _, text := range []string{a.String(), b.String()} {
			if len(text) > 400 {
				return false
			}
			if index := strings.IndexAny(text, "eE"); index >= 0 {
				exponent, err := strconv.ParseInt(text[index+1:], 10, 32)
				if err != nil {
					return false
				}
				if exponent < -400 || exponent > 400 {
					return false
				}
			}
		}
		x, ok := new(big.Rat).SetString(a.String())
		if !ok {
			return false
		}
		y, ok := new(big.Rat).SetString(b.String())
		return ok && x.Cmp(y) == 0
	})
	difference := &numericTransportDifference{}
	if !cmp.Equal(values[0], values[1], equalNumber, cmp.Reporter(difference)) {
		if difference.message != "" {
			return fmt.Errorf("test/snowflaketester: %s", difference.message)
		}
		return fmt.Errorf("test/snowflaketester: numeric JSON differs")
	}
	return nil
}

// NumericTransportCases returns the Arrow matrix or the compact JSON probe.
func NumericTransportCases() []NumericTransportCase {
	cases := []NumericTransportCase{
		{"SQL null", "", "SQL_NULL", false, true},
		{"JSON null", "null", "NULL_VALUE", false, true},
		{"boolean inside JSON", "true", "BOOLEAN", false, true},
		{"numeric text", `"1E2"`, "VARCHAR", false, true},
		{"zero", "0", "fixed", false, false},
		{"negative zero", "-0E0", "fixed", false, false},
		{"integer beyond float", "9007199254740993", "fixed", false, true},
		{"integer precision 38", "12345678901234567890123456789012345678", "fixed", false, true},
		{"integer negative", "-12345678901234567890123456789012345678", "fixed", false, false},
		{"decimal scale 37", "1E-37", "fixed", false, true},
		{"decimal negative", "-1E-37", "fixed", false, false},
		{"expanded large integer", "1.234567890123456789E19", "fixed", false, true},
		{"expanded fine fraction", "1.00000000000000001E2", "fixed", false, true},
		{"expanded 1.1", "1.1E0", "fixed", false, true},
		{"money positive border", "99999999999999.9999", "fixed", false, false},
		{"money negative border", "-99999999999999.9999", "fixed", false, false},
		{"beyond fixed precision", "1E38", "DOUBLE", false, true},
		{"beyond fixed scale", "1E-38", "DOUBLE", false, true},
		{"historical exact double", "1E2", "DOUBLE", true, true},
		{"historical fractional double", "1.1E0", "DOUBLE", true, true},
		{"historical successor", "100.00000000000001E0", "DOUBLE", true, true},
		{"historical adjacent", "100.00000000000003E0", "DOUBLE", true, false},
		{"historical negative", "-100.00000000000001E0", "DOUBLE", true, false},
		{"historical tiny", "5E-324", "DOUBLE", true, true},
		{"historical subnormal", "1E-310", "DOUBLE", true, false},
		{"historical maximum", "1.7976931348623157E308", "DOUBLE", true, true},
		{"historical negative zero", "-0E0", "numeric", true, false},
	}
	if RequestedResultFormat() != "JSON" {
		return cases
	}
	var compact []NumericTransportCase
	for _, fixture := range cases {
		if fixture.Compact {
			compact = append(compact, fixture)
		}
	}
	return compact
}

// NumericTransportColumns describes the same supported columns in both readers.
func NumericTransportColumns() []warehouses.Column {
	return []warehouses.Column{
		{Name: "ID", Type: types.Int(32)},
		{Name: "VALUE", Type: types.JSON(), Nullable: true},
		{Name: "ELEMENTS", Type: types.Array(types.JSON()), Nullable: true},
		{Name: "ATTRIBUTES", Type: types.Map(types.JSON()), Nullable: true},
	}
}

// NumericTransportContext tags a primary application read.
func NumericTransportContext(ctx context.Context, tag string) context.Context {
	if ctx == nil || tag != WarehouseValuesTag && tag != ConnectorValuesTag &&
		tag != WarehouseNativeTag && tag != ConnectorNativeTag {
		panic("test/snowflaketester: invalid transport application tag")
	}
	return gosnowflake.WithQueryTag(ctx, tag)
}

// ReadNumericTransportReference checks physical kinds and captures server text.
// Native DOUBLE bits and tokens are observations, not output guarantees.
func ReadNumericTransportReference(t *testing.T, db *sql.DB, tag string, cases []NumericTransportCase) []NumericTransportReference {
	t.Helper()
	if db == nil || tag != WarehouseReferenceTag && tag != ConnectorReferenceTag || len(cases) == 0 || len(cases) > 64 {
		t.Fatal("expected bounded transport reference request, got invalid tag or population")
	}
	paths := []string{
		`GET("VALUE",'n')`,
		`GET(GET(GET("VALUE",'nested'),'a.b[0]'),0)`,
		`GET(GET(GET("VALUE",'nested'),'a.b[0]'),1)`,
		`GET("ELEMENTS",0)`,
		`GET(GET("ELEMENTS",1),'n')`,
		`GET("ATTRIBUTES",'n')`,
		`GET(GET(GET("ATTRIBUTES",'a.b[0]'),'n'),0)`,
	}
	var query strings.Builder
	query.WriteString(`SELECT "ID","VALUE","ELEMENTS","ATTRIBUTES"`)
	for _, path := range paths {
		query.WriteString(",COALESCE(TYPEOF(" + path + "),'SQL_NULL')")
	}
	query.WriteString(`,AS_DOUBLE(GET("VALUE",'n')) FROM "NUMERIC_TRANSPORT" ORDER BY "ID" LIMIT 65`)
	rows, err := db.QueryContext(gosnowflake.WithQueryTag(t.Context(), tag), query.String())
	if err != nil {
		t.Fatalf("expected transport reference read, got %v", err)
	}
	t.Cleanup(func() { rows.Close() })
	var references []NumericTransportReference
	for rows.Next() {
		var id int
		var cells [3]sql.NullString
		var kinds [7]string
		var native any
		dest := []any{&id, &cells[0], &cells[1], &cells[2]}
		for i := range kinds {
			dest = append(dest, &kinds[i])
		}
		dest = append(dest, &native)
		if err := rows.Scan(dest...); err != nil {
			t.Fatalf("expected diagnostic row, got %v", err)
		}
		if id != len(references) || id >= len(cases) {
			t.Fatalf("expected ordered fixture identity %d, got %d", len(references), id)
		}
		fixture := cases[id]
		reference := NumericTransportReference{Null: fixture.Number == ""}
		for path, kind := range kinds {
			switch kind {
			case "INTEGER", "DECIMAL", "DOUBLE", "BOOLEAN", "VARCHAR", "NULL_VALUE", "SQL_NULL":
				// Recognized kinds are safe to include in diagnostics.
			default:
				kind = "other"
			}
			kinds[path] = kind
			fixed := kind == "INTEGER" || kind == "DECIMAL"
			if kind != fixture.Kind && !(fixture.Kind == "fixed" && fixed) &&
				!(fixture.Kind == "numeric" && (fixed || kind == "DOUBLE")) {
				t.Errorf("expected %s path %d physical kind %s, got %s", fixture.Name, path, fixture.Kind, kind)
			}
		}
		if fixture.Kind == "numeric" {
			t.Logf("zero observation fixture=%q kind=%s", fixture.Name, kinds[0])
		}
		for i, cell := range cells {
			if len(cell.String) > 16*1024 {
				t.Fatalf("expected bounded payload for %s column %d, got oversized data", fixture.Name, i)
			}
			if cell.Valid == reference.Null {
				t.Errorf("expected %s column %d SQL null=%t, got SQL null=%t",
					fixture.Name, i, reference.Null, !cell.Valid)
				continue
			}
			reference.Cells[i] = cell.String
			if fixture.Kind != "DOUBLE" && !reference.Null {
				want, err := json.Marshal(fixture.Row(id)[i+1])
				if err != nil {
					t.Fatalf("expected fixture JSON, got %v", err)
				}
				err = CheckNumericTransportJSON(cell.String, string(want), false)
				if err != nil {
					t.Errorf("expected exact stored fixed value for %s column %d, got %v", fixture.Name, i, err)
				}
			}
		}
		references = append(references, reference)
		if kinds[0] == "DOUBLE" {
			var token struct {
				N stdjson.Number `json:"n"`
			}
			err := stdjson.Unmarshal([]byte(reference.Cells[0]), &token)
			if err != nil {
				t.Fatalf("expected bounded DOUBLE token, got invalid JSON")
			}
			if len(token.N) > 400 {
				t.Fatal("expected bounded DOUBLE token, got oversized number")
			}
			if value, ok := native.(float64); ok {
				parsed, err := strconv.ParseFloat(token.N.String(), 64)
				if err != nil {
					t.Logf("DOUBLE observation fixture=%q bits=%016x token=%s roundtrip=false parse=unrepresentable",
						fixture.Name, math.Float64bits(value), token.N)
					continue
				}
				roundtrip := math.Float64bits(value) == math.Float64bits(parsed)
				t.Logf("DOUBLE observation fixture=%q bits=%016x token=%s roundtrip=%t",
					fixture.Name, math.Float64bits(value), token.N, roundtrip)
			} else if _, ok := native.(string); !ok {
				t.Fatalf("expected native DOUBLE diagnostic value, got %T", native)
			} else {
				t.Logf("DOUBLE observation fixture=%q token=%s native_transport=text", fixture.Name, token.N)
			}
		}
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete reference read, got %v", err)
	}
	err = rows.Close()
	if err != nil {
		t.Fatalf("expected reference cursor close, got %v", err)
	}
	if len(references) != len(cases) {
		t.Fatalf("expected %d reference rows, got %d", len(cases), len(references))
	}
	return references
}

// WriteHistoricalNumericTransport inserts explicit PARSE_JSON DOUBLE fixtures.
// It does not use the writer that intentionally preserves fixed-point numbers.
func WriteHistoricalNumericTransport(t *testing.T, db *sql.DB, cases []NumericTransportCase) {
	t.Helper()
	if db == nil || len(cases) == 0 || len(cases) > 64 {
		t.Fatal("expected bounded historical transport fixtures, got invalid database or population")
	}
	for id, fixture := range cases {
		if !fixture.Historical {
			continue
		}
		row := fixture.Row(id)
		arguments := make([]any, 4)
		arguments[0] = id
		for i, value := range row[1:] {
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("expected historical fixture JSON, got %v", err)
			}
			arguments[i+1] = string(raw)
		}
		_, err := db.ExecContext(t.Context(), `INSERT INTO "NUMERIC_TRANSPORT"
 SELECT ?,PARSE_JSON(?),PARSE_JSON(?)::ARRAY,PARSE_JSON(?)::OBJECT`, arguments...)
		if err != nil {
			t.Fatalf("expected historical fixture write, got %v", err)
		}
	}
}
