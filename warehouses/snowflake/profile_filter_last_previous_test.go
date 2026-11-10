// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// TestProfileFilterLastPreviousSQL checks suffixes and ordered-pair metadata.
func TestProfileFilterLastPreviousSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, unbounded := range []bool{false, true} {
		for _, mode := range []string{"", "none", "all"} {
			threshold := decimal.MustParse("-30")
			node := &warehouses.LastPreviousDifference{Source: warehouses.OccurrenceSource{Table: "events",
				Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
				Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
				Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
				End: end, UnboundedStart: unbounded, ReadableDifferences: mode,
				Qualification: &warehouses.KnownTextEqual{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
					Keys: []string{"currency"}, Value: "EUR"},
				Value: warehouses.DecimalReading{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
					Keys: []string{"total"}, Type: types.Decimal(18, 4)}}
			if mode == "" {
				node.Threshold = &threshold
			}
			if !unbounded {
				node.Start = end.AddDate(0, -1, 0)
			}
			statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected generated ordered-pair SQL, got %v", err)
			}
			for _, fragment := range []string{
				"GROUP BY profile_id,time_min", "ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
				"LAG(cum_c) OVER (PARTITION BY profile_id ORDER BY time_min DESC)",
				"later_c=0 AND k_t>=2", "later_c=1 AND later_cu=0 AND k_t>0",
				"b.profile_id=m.profile_id", "time_min<>time_max AND state_max<>0",
				"known_min<>known_max", "amount_min<>amount_max",
				"lo IS NULL OR hi IS NULL OR lo>hi", "qualification IS DISTINCT FROM FALSE",
				"unknowns>0 THEN CAST(NULL AS BOOLEAN)", "DECIMAL(19,4)",
			} {
				if !strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected pair invariant %q, got %s", fragment, statement.SQL)
				}
			}
			for _, fragment := range []string{"ROW_NUMBER", " IS NOT FALSE"} {
				if strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected closed compatible pairs without %q, got %s", fragment, statement.SQL)
				}
			}
			values, err := statement.Values()
			if err != nil {
				t.Fatalf("expected typed period and currency arguments, got %v", err)
			}
			arity := 4
			if unbounded {
				arity = 3
			}
			if len(values) != arity {
				t.Fatalf("expected %d arguments, got %v", arity, values)
			}
			statement, err = profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected bounded pair explanation, got %v", err)
			}
			if len(statement.Columns) != 7 || len(statement.PairNodes) != 1 || !statement.PairNodes[0] ||
				statement.CountNodes[0] || statement.Columns[6].Name != "pair_unique" {
				t.Fatalf("expected dedicated pair metadata, got %+v", statement)
			}
		}
	}
}
