// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"context"
	"database/sql/driver"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/warehouses"
)

// TestEventDiscoveryRead exercises database/sql cursor failures without a
// Snowflake connection, including discarded prefixes and cumulative budgets.
func TestEventDiscoveryRead(t *testing.T) {

	window := warehouses.EventDiscoveryWindow{Start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		End: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)}
	for _, operation := range []struct {
		name    string
		columns []string
		good    []driver.Value
		control []driver.Value
		payload int
		ordinal int
		call    func(context.Context, *Snowflake) (int, error)
	}{
		{"sources", []string{"type", "size", "name", "rn", "charged"},
			[]driver.Value{"track", "1", "A", "1", "1"},
			[]driver.Value{"track", "1", nil, "2", "2"}, 2, 3,
			func(ctx context.Context, warehouse *Snowflake) (int, error) {
				page, err := warehouse.DiscoverEventSources(ctx, warehouses.EventSourceDiscoveryRequest{Window: window, Limit: 1})
				return len(page.Sources), err
			}},
		{"sample", []string{"rn", "charged", "traits_size", "properties_size", "traits", "properties"},
			[]driver.Value{"1", "7", nil, "7", nil, `{"x":1}`},
			[]driver.Value{"2", "7", nil, nil, nil, nil}, 5, 0,
			func(ctx context.Context, warehouse *Snowflake) (int, error) {
				sample, err := warehouse.SampleEventFields(ctx, warehouses.EventFieldSampleRequest{
					Window: window, Source: warehouses.ObservedEventSource{Type: "track"}, Limit: 1,
				})
				return len(sample.Rows), err
			}},
	} {
		for _, name := range []string{"complete", "schema", "query failure", "close failure", "ordinal",
			"oversized payload", "invalid type", "too many rows"} {
			t.Run(operation.name+"/"+name, func(t *testing.T) {

				response := checkReadOnlyQuery{match: "WITH parameters", cols: operation.columns,
					rows: [][]driver.Value{slices.Clone(operation.good), slices.Clone(operation.control)}}
				switch name {
				case "complete":
					// Keep a valid prefix plus its control row.
				case "schema":
					response.cols, response.rows = []string{"unexpected"}, nil
				case "query failure":
					response.err = errors.New("WAREHOUSE_DATA")
				case "close failure":
					response.closeErr = errors.New("WAREHOUSE_DATA")
				case "ordinal":
					response.rows[0][operation.ordinal] = "-1"
				case "oversized payload":
					response.rows[0][operation.payload] = strings.Repeat("x", 65537)
				case "invalid type":
					response.rows[0][operation.payload] = int64(1)
				case "too many rows":
					response.rows = append(response.rows, slices.Clone(operation.control))
				default:
					t.Fatalf("expected known case, got %q", name)
				}
				db, queries := newCheckReadOnlyTestDB(t, []checkReadOnlyQuery{response})
				t.Cleanup(func() {
					err := db.Close()
					if err != nil {
						t.Errorf("expected test database cleanup, got %v", err)
					}
				})
				count, err := operation.call(t.Context(), &Snowflake{db: db})
				if err != nil {
					if name == "complete" || count != 0 || strings.Contains(err.Error(), "WAREHOUSE_DATA") ||
						len(*queries) != 1 || db.Stats().InUse != 0 {
						t.Fatalf("expected one closed query without partial data, got %d rows and %v", count, err)
					}
					return
				}
				if name != "complete" || count != 1 || len(*queries) != 1 || db.Stats().InUse != 0 {
					t.Fatalf("expected one closed successful query, got %d rows, %d queries", count, len(*queries))
				}

			})
		}
	}

}
