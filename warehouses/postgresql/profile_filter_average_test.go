// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// TestProfileFilterAverageSQL covers the common exact terminal and adapter.
func TestProfileFilterAverageSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	node := &warehouses.DeterminedOccurrenceAverage{
		Source: warehouses.OccurrenceSource{Table: "events",
			Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
			Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
			Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		Start: end.AddDate(0, -1, 0), End: end,
		Value: warehouses.DecimalReading{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
			Keys: []string{"total"}, Type: types.Decimal(18, 4)},
		Threshold: decimal.MustParse("100.0001"), Minimum: 2,
	}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, test := range []struct {
		name, operator  string
		atLeast, strict bool
	}{
		{"greater", ">", true, true}, {"greater or equal", ">=", true, false},
		{"less", "<", false, true}, {"less or equal", "<=", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			node.AtLeast, node.Strict = test.atLeast, test.strict
			statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected exact Average SQL, got %v", err)
			}
			product := test.operator + "(CAST('100.0001' AS DECIMAL(18,4))*CAST(COALESCE(v0.lo,0) AS DECIMAL(19,0)))"
			if !strings.Contains(statement.SQL, product) || !strings.Contains(statement.SQL, "v0.unknowns,0)=0") ||
				strings.Contains(statement.SQL, "AVG(") || strings.Contains(statement.SQL, "/") {
				t.Fatalf("expected cross multiplication with determined evidence, got %s", statement.SQL)
			}
			statement, err = profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected exact Average input protocol, got %v", err)
			}
			if len(statement.Columns) != 8 || len(statement.AverageNodes) != 1 || !statement.AverageNodes[0] ||
				statement.CountNodes[0] || !statement.SumNodes[0] {
				t.Fatalf("expected Average total/count metadata, got %+v", statement)
			}
		})
	}
	node.Threshold = decimal.Decimal{}
	for _, value := range []bool{false, true} {
		node.DeterminedResult = &value
		statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
		if err != nil {
			t.Fatalf("expected predecided value test, got %v", err)
		}
		literal := "FALSE"
		if value {
			literal = "TRUE"
		}
		if !strings.Contains(statement.SQL, "THEN ("+literal+") ELSE NULL END") ||
			strings.Contains(statement.SQL, " AS DECIMAL(19,0))") {
			t.Fatalf("expected predecided test without multiplication, got %s", statement.SQL)
		}
	}
}
