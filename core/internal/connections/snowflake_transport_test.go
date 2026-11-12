// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/connectors/snowflake"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	_ "github.com/krenalis/krenalis/warehouses/snowflake"
)

type numericTransportSettings struct {
	data json.Value
}

// Load decodes the isolated environment's settings.
func (s numericTransportSettings) Load(_ context.Context, dest any) error {
	return s.data.Unmarshal(dest)
}

// Store rejects unexpected settings changes during the read-only probe.
func (s numericTransportSettings) Store(context.Context, any) error {
	return errors.New("core/internal/connections: unexpected transport settings write")
}

// TestNumericTransportJSONNormalization checks the connector's real scanner on
// exact, scientific, nested and null driver payloads, without a remote server.
func TestNumericTransportJSONNormalization(t *testing.T) {
	columns := snowflaketester.NumericTransportColumns()
	for id, fixture := range snowflaketester.NumericTransportCases() {
		for i, value := range fixture.Row(id)[1:] {
			column := columns[i+1]
			var source any
			var raw json.Value
			if value != nil {
				var err error
				raw, err = json.Marshal(value)
				if err != nil {
					t.Fatalf("expected driver fixture JSON, got %v", err)
				}
				source = string(raw)
			}
			row := map[string]any{}
			err := (queryScanValue{column: connectors.Column{Name: column.Name, Type: column.Type,
				Nullable: true}, row: row}).Scan(source)
			if err != nil {
				t.Fatalf("expected connector JSON normalization for %s, got %v", fixture.Name, err)
			}
			if source == nil {
				if row[column.Name] != nil {
					t.Fatal("expected normalized SQL NULL, got a value")
				}
				continue
			}
			actual, err := json.Marshal(row[column.Name])
			if err != nil {
				t.Fatalf("expected normalized JSON representation, got %v", err)
			}
			err = snowflaketester.CheckNumericTransportJSON(string(actual), string(raw), true)
			if err != nil {
				t.Fatalf("expected unchanged connector tokens for %s, got %v", fixture.Name, err)
			}
		}
	}
}

// TestNumericTransportNativeNormalization checks the actual connector scanner.
func TestNumericTransportNativeNormalization(t *testing.T) {
	for _, test := range []struct {
		typ   types.Type
		raw   any
		want  any
		valid bool
	}{
		{types.Float(64), "100.00000000000001", 100.00000000000001, true},
		{types.Float(64), 100.00000000000001, 100.00000000000001, true},
		{types.Boolean(), "true", true, true},
		{types.Boolean(), "false", false, true},
		{types.Boolean(), true, true, true},
		{types.Boolean(), "TRUE", nil, false},
		{types.Boolean(), "1", nil, false},
	} {
		row := map[string]any{}
		scanner := queryScanValue{column: connectors.Column{Name: "value", Type: test.typ}, row: row}
		err := scanner.Scan(test.raw)
		if err != nil {
			if test.valid {
				t.Fatalf("expected normalized native value, got %v", err)
			}
			continue
		}
		if !test.valid || !equalNativeTransport(row["value"], test.want) {
			t.Fatalf("expected native normalization %v with valid=%t, got %v", test.want, test.valid, row["value"])
		}
	}
}

// TestSnowflakeNativeTransport is a separate compatibility probe. It collects
// both readers' FLOAT and BOOLEAN failures before reporting one test failure.
func TestSnowflakeNativeTransport(t *testing.T) {

	if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if snowflaketester.RequestedResultFormat() == "" {
		t.Skip("requires an isolated result-format overlay")
	}
	testEnv, err := snowflaketester.CreateTestEnvironment()
	if err != nil {
		t.Fatalf("expected isolated native transport environment, got %v", err)
	}
	t.Cleanup(func() {
		err := testEnv.Teardown()
		if err != nil {
			t.Errorf("expected native environment cleanup, got %v", err)
		}
	})
	settings := numericTransportSettings{data: testEnv.Settings().JSON()}
	warehouse := warehouses.Registered("Snowflake").New(settings, nil)
	t.Cleanup(func() {
		err := warehouse.Close()
		if err != nil {
			t.Errorf("expected native warehouse cleanup, got %v", err)
		}
	})
	ddl := `CREATE TABLE "NUMERIC_NATIVE" ("ID" INTEGER,"F" FLOAT,"B" BOOLEAN,"I" BIGINT,"D" DECIMAL(38,37))`
	raw := warehouse.(interface {
		RawQuery(context.Context, string) (warehouses.Rows, int, error)
	})
	created, _, err := raw.RawQuery(t.Context(), ddl)
	if err != nil {
		t.Fatalf("expected native fixture table, got %v", err)
	}
	for created.Next() {
	}
	err = created.Err()
	closeErr := created.Close()
	if err != nil {
		t.Fatalf("expected complete native DDL, got %v", err)
	}
	if closeErr != nil {
		t.Fatalf("expected DDL cursor close, got %v", closeErr)
	}
	columns := []warehouses.Column{{Name: "ID", Type: types.Int(32)}, {Name: "F", Type: types.Float(64)},
		{Name: "B", Type: types.Boolean()}, {Name: "I", Type: types.Int(64)},
		{Name: "D", Type: types.Decimal(38, 37)}}
	fraction := decimal.MustParse("0.1234567890123456789012345678901234567")
	data := [][]any{
		{0, math.Nextafter(100, math.Inf(1)), true, int(9007199254740993), fraction},
		{1, 1.1, false, int(-9007199254740993), fraction.Neg()},
		{2, math.Copysign(0, -1), true, int(0), decimal.MustParse("0")},
	}
	err = warehouse.Merge(t.Context(), warehouses.Table{Name: "NUMERIC_NATIVE", Keys: []string{"ID"},
		Columns: columns}, data, nil)
	if err != nil {
		t.Fatalf("expected native fixture writes, got %v", err)
	}
	var issues []string
	for i, column := range columns[1:] {
		ctx := snowflaketester.NumericTransportContext(t.Context(), snowflaketester.WarehouseNativeTag)
		rows, _, err := warehouse.Query(ctx, warehouses.RowQuery{Table: "NUMERIC_NATIVE",
			Columns: []warehouses.Column{columns[0], column}, OrderBy: columns[:1], Limit: 4}, false)
		if err != nil {
			issues = append(issues, "warehouse "+column.Name+": query rejected")
			continue
		}
		count := 0
		for rows.Next() {
			values := make([]any, 2)
			err := rows.Scan(values...)
			if err != nil {
				issues = append(issues, "warehouse "+column.Name+": scanner rejected driver value")
				break
			}
			if count >= len(data) {
				issues = append(issues, "warehouse "+column.Name+": oversized population")
				break
			}
			if values[0] != count || !equalNativeTransport(values[1], data[count][i+1]) {
				issues = append(issues, "warehouse "+column.Name+": native value or order differs")
			} else {
				t.Logf("native observation path=warehouse column=%s fixture=%d accepted=true", column.Name, count)
			}
			count++
		}
		err = rows.Err()
		if err != nil {
			issues = append(issues, "warehouse "+column.Name+": iteration rejected")
		}
		err = rows.Close()
		if err != nil {
			issues = append(issues, "warehouse "+column.Name+": close rejected")
		}
		if count != len(data) {
			issues = append(issues, "warehouse "+column.Name+": incomplete fixture")
		}
	}

	connector, err := snowflake.New(&connectors.DatabaseEnv{Settings: settings})
	if err != nil {
		t.Fatalf("expected native transport connector, got %v", err)
	}
	t.Cleanup(func() {
		err := connector.Close()
		if err != nil {
			t.Errorf("expected native connector cleanup, got %v", err)
		}
	})
	ctx := snowflaketester.NumericTransportContext(t.Context(), snowflaketester.ConnectorNativeTag)
	rows, actualColumns, err := connector.Query(ctx,
		`SELECT "ID","F","B","I","D" FROM "NUMERIC_NATIVE" ORDER BY "ID" LIMIT 4`)
	if err != nil {
		issues = append(issues, "connector: query rejected")
	}
	if err == nil {
		validLayout := len(actualColumns) == len(columns)
		if validLayout {
			for i, column := range actualColumns {
				if column.Name != columns[i].Name || !column.Type.Valid() || column.Issue != "" {
					validLayout = false
					break
				}
			}
		}
		if !validLayout {
			issues = append(issues, "connector: column layout differs")
		} else {
			count := 0
			for rows.Next() {
				var id int
				var raw [4]any
				err := rows.Scan(&id, &raw[0], &raw[1], &raw[2], &raw[3])
				if err != nil {
					issues = append(issues, "connector: raw row rejected")
					break
				}
				if id != count || count >= len(data) {
					issues = append(issues, "connector: identity or population differs")
					break
				}
				for i, value := range raw {
					column := actualColumns[i+1]
					if text, ok := value.(string); ok && len(text) > 128 {
						issues = append(issues, "connector "+columns[i+1].Name+": oversized native value")
						continue
					}
					if i < 2 {
						_, text := value.(string)
						_, real := value.(float64)
						_, boolean := value.(bool)
						validRaw := snowflaketester.RequestedResultFormat() == "JSON" && text ||
							snowflaketester.RequestedResultFormat() == "ARROW" && (i == 0 && real || i == 1 && boolean)
						if !validRaw {
							issues = append(issues, "connector "+columns[i+1].Name+": raw transport type differs")
						}
						t.Logf("native observation path=connector column=%s fixture=%d driver_type=%T",
							columns[i+1].Name, count, value)
					}
					if i == 1 {
						text := "non-text"
						if raw, ok := value.(string); ok {
							switch raw {
							case "true", "false", "TRUE", "FALSE", "1", "0":
								text = raw
							default:
								text = "other"
							}
						}
						t.Logf("native BOOLEAN observation fixture=%d text=%s", count, text)
					}
					row := map[string]any{}
					err := (queryScanValue{column: column, row: row}).Scan(value)
					if err != nil {
						issues = append(issues, "connector "+columns[i+1].Name+": normalization rejected")
						continue
					}
					if !equalNativeTransport(row[column.Name], data[count][i+1]) {
						issues = append(issues, "connector "+columns[i+1].Name+": native value differs")
					}
					if v, ok := row[column.Name].(float64); ok {
						t.Logf("native FLOAT observation fixture=%d bits=%016x", count, math.Float64bits(v))
					}
				}
				count++
			}
			err = rows.Err()
			if err != nil {
				issues = append(issues, "connector: iteration rejected")
			}
			if count != len(data) {
				issues = append(issues, "connector: incomplete fixture")
			}
		}
		err = rows.Close()
		if err != nil {
			issues = append(issues, "connector: close rejected")
		}
	}
	if len(issues) != 0 {
		t.Errorf("expected unchanged native values through both application readers, got %s",
			strings.Join(issues, "; "))
	}

}

func equalNativeTransport(got, want any) bool {
	switch want := want.(type) {
	case float64:
		value, ok := got.(float64)
		return ok && (value == 0 && want == 0 || math.Float64bits(value) == math.Float64bits(want))
	case decimal.Decimal:
		value, ok := got.(decimal.Decimal)
		return ok && value.Equal(want)
	case int:
		if value, ok := got.(decimal.Decimal); ok {
			return value.Equal(decimal.MustParse(fmt.Sprintf("%d", want)))
		}
		return reflect.DeepEqual(got, want)
	default:
		return reflect.DeepEqual(got, want)
	}
}
