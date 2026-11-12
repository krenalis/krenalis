// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestJSONNumericTransport exercises real writes and an unchanged query.
// Arrow is an acceptance check; the explicitly requested JSON run is a probe.
func TestJSONNumericTransport(t *testing.T) {
	if os.Getenv("KRENALIS_SKIP_SNOWFLAKE_TESTS") == "true" {
		t.Skip("remote Snowflake tests are disabled")
	}
	if snowflaketester.RequestedResultFormat() == "" {
		t.Skip("requires an isolated result-format overlay")
	}
	testEnv, err := snowflaketester.CreateTestEnvironment()
	if err != nil {
		t.Fatalf("expected isolated transport environment, got %v", err)
	}
	t.Cleanup(func() {
		err := testEnv.Teardown()
		if err != nil {
			t.Errorf("expected transport environment cleanup, got %v", err)
		}
	})
	connector, err := New(&connectors.DatabaseEnv{
		Settings: newTestSettingsStore(testEnv.Settings().JSON()), DialWith: testDialWith})
	if err != nil {
		t.Fatalf("expected transport connector, got %v", err)
	}
	t.Cleanup(func() {
		err := connector.Close()
		if err != nil {
			t.Errorf("expected transport connector cleanup, got %v", err)
		}
	})
	err = connector.openDB(t.Context())
	if err != nil {
		t.Fatalf("expected transport database, got %v", err)
	}
	_, err = connector.db.ExecContext(t.Context(), snowflaketester.NumericTransportDDL)
	if err != nil {
		t.Fatalf("expected transport table, got %v", err)
	}
	cases := snowflaketester.NumericTransportCases()
	columns := snowflaketester.NumericTransportColumns()
	table := connectors.Table{Name: snowflaketester.NumericTransportTable, Keys: []string{"ID"}}
	for _, column := range columns {
		table.Columns = append(table.Columns, connectors.Column{Name: column.Name, Type: column.Type,
			Nullable: column.Nullable})
	}
	var data [][]any
	for id, fixture := range cases {
		if !fixture.Historical {
			data = append(data, fixture.Row(id))
		}
	}
	err = connector.Merge(t.Context(), table, data)
	if err != nil {
		t.Fatalf("expected numeric transport writes, got %v", err)
	}
	snowflaketester.WriteHistoricalNumericTransport(t, connector.db, cases)
	references := snowflaketester.ReadNumericTransportReference(t, connector.db,
		snowflaketester.ConnectorReferenceTag, cases)
	ctx := snowflaketester.NumericTransportContext(t.Context(), snowflaketester.ConnectorValuesTag)
	rows, actualColumns, err := connector.Query(ctx, snowflaketester.NumericTransportSQL)
	if err != nil {
		t.Fatalf("expected unchanged application query, got %v", err)
	}
	t.Cleanup(func() { rows.Close() })
	if len(actualColumns) != len(columns) {
		t.Fatalf("expected %d transport columns, got %d", len(columns), len(actualColumns))
	}
	for i, column := range actualColumns {
		typ := columns[i].Type
		if i == 0 {
			typ = types.Decimal(38, 0) // Snowflake INTEGER metadata.
		}
		if column.Name != columns[i].Name || !types.Equal(column.Type, typ) || column.Issue != "" {
			t.Fatalf("expected supported column %s of type %s, got metadata mismatch", columns[i].Name, typ)
		}
	}
	count := 0
	for rows.Next() {
		var id int
		var values [3]any
		if err := rows.Scan(&id, &values[0], &values[1], &values[2]); err != nil {
			t.Fatalf("expected raw application row, got %v", err)
		}
		if id != count || count >= len(cases) {
			t.Fatalf("expected ordered transport identity %d, got %d", count, id)
		}
		for i, value := range values {
			if references[id].Null {
				if value != nil {
					t.Errorf("expected SQL NULL for %s, got %T", cases[id].Name, value)
				}
				continue
			}
			text, ok := value.(string)
			if !ok {
				t.Fatalf("expected server text for fixture %d, got %T", id, value)
			}
			err = snowflaketester.CheckNumericTransportJSON(text, references[id].Cells[i], true)
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
