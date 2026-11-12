// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestJSONNumericTransport exercises real writes and the warehouse scanner.
// Arrow is an acceptance check; the explicitly requested JSON run is a probe.
func TestJSONNumericTransport(t *testing.T) {
	if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if snowflaketester.RequestedResultFormat() == "" {
		t.Skip("requires an isolated result-format overlay")
	}
	warehouse, db := newTestSnowflakeWarehouse(t)
	mustExecSQL(t, db, snowflaketester.NumericTransportDDL)
	cases := snowflaketester.NumericTransportCases()
	columns := snowflaketester.NumericTransportColumns()
	table := warehouses.Table{Name: snowflaketester.NumericTransportTable, Keys: []string{"ID"}, Columns: columns}
	var data [][]any
	for id, fixture := range cases {
		if !fixture.Historical {
			data = append(data, fixture.Row(id))
		}
	}
	err := warehouse.Merge(t.Context(), table, data, nil)
	if err != nil {
		t.Fatalf("expected numeric transport writes, got %v", err)
	}
	snowflaketester.WriteHistoricalNumericTransport(t, db, cases)
	references := snowflaketester.ReadNumericTransportReference(t, db, snowflaketester.WarehouseReferenceTag, cases)
	ctx := snowflaketester.NumericTransportContext(t.Context(), snowflaketester.WarehouseValuesTag)
	rows, _, err := warehouse.Query(ctx, warehouses.RowQuery{Table: table.Name, Columns: columns,
		OrderBy: columns[:1], Limit: 65}, false)
	if err != nil {
		t.Fatalf("expected application transport read, got %v", err)
	}
	t.Cleanup(func() { rows.Close() })
	count := 0
	for rows.Next() {
		values := make([]any, 4)
		if err := rows.Scan(values...); err != nil {
			t.Fatalf("expected decoded transport row, got %v", err)
		}
		id, ok := values[0].(int)
		if !ok {
			t.Fatalf("expected integer transport identity, got %T", values[0])
		}
		if id != count || count >= len(cases) {
			t.Fatalf("expected ordered transport identity %d, got %d", count, id)
		}
		for i, value := range values[1:] {
			if references[id].Null {
				if value != nil {
					t.Errorf("expected SQL NULL for %s, got %T", cases[id].Name, value)
				}
				continue
			}
			actual, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("expected application JSON, got %v", err)
			}
			err = snowflaketester.CheckNumericTransportJSON(string(actual), references[id].Cells[i], true)
			if err != nil {
				t.Errorf("expected unchanged server tokens for %s column %d, got %v", cases[id].Name, i, err)
			}
		}
		count++
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete application read, got %v", err)
	}
	err = rows.Close()
	if err != nil {
		t.Fatalf("expected transport cursor close, got %v", err)
	}
	if count != len(cases) {
		t.Fatalf("expected %d transport rows, got %d", len(cases), count)
	}
}

// TestNativeTransportDriverValues characterizes native Arrow and JSON values.
func TestNativeTransportDriverValues(t *testing.T) {
	for _, test := range []struct {
		typ   types.Type
		value any
		valid bool
	}{
		{types.Float(64), 1.1, true},
		{types.Float(64), "1.1", false},
		{types.Boolean(), true, true},
		{types.Boolean(), "true", false},
	} {
		scanner := &scanner{}
		_, err := scanner.normalize("value", test.typ, test.value)
		if err != nil {
			if test.valid {
				t.Fatalf("expected native driver value acceptance, got %v", err)
			}
			continue
		}
		if !test.valid {
			t.Fatal("expected current scanner to reject a textual native value, got success")
		}
	}
}

// TestNumericTransportEncoding checks the CSV cell encoder without a server.
func TestNumericTransportEncoding(t *testing.T) {
	columns := snowflaketester.NumericTransportColumns()
	for id, fixture := range snowflaketester.NumericTransportCases() {
		if fixture.Historical {
			continue
		}
		for i, value := range fixture.Row(id)[1:] {
			var encoded bytes.Buffer
			err := serializeValueToCSV(&encoded, columns[i+1].Type, value)
			if err != nil {
				t.Fatalf("expected transport fixture encoding, got %v", err)
			}
			if value == nil {
				if encoded.Len() != 0 {
					t.Fatal("expected empty SQL NULL field, got encoded value")
				}
				continue
			}
			text := encoded.String()
			if len(text) < 2 || text[0] != '\'' || text[len(text)-1] != '\'' {
				t.Fatal("expected quoted JSON cell, got invalid CSV framing")
			}
			text = strings.ReplaceAll(text[1:len(text)-1], "''", "'")
			want, err := json.Marshal(value)
			if err != nil {
				t.Fatalf("expected fixture JSON, got %v", err)
			}
			err = snowflaketester.CheckNumericTransportJSON(text, string(want), false)
			if err != nil {
				t.Fatalf("expected exact numeric CSV payload for %s column %d, got %v", fixture.Name, i, err)
			}
			if fixture.Kind == "fixed" {
				decoder := json.NewDecoder(strings.NewReader(text))
				for {
					token, err := decoder.ReadToken()
					if err != nil {
						if err == io.EOF {
							break
						}
						t.Fatalf("expected complete CSV JSON, got %v", err)
					}
					if token.Kind() == json.Number && strings.ContainsAny(token.String(), "eE") {
						t.Fatalf("expected plain fixed-point token for %s column %d, got exponent", fixture.Name, i)
					}
				}
			}
		}
	}
}
