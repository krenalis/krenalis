// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

func TestProfileExplanationSQL(t *testing.T) {
	population := warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}
	source := warehouses.OccurrenceSource{Table: "krenalis_events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID(), Nullable: true},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	record := warehouses.NewBaseExpr(warehouses.Column{Name: "country", Type: types.String(), Nullable: true},
		warehouses.OpIs, "IT")
	filter := warehouses.Filter{Root: &warehouses.FilterGroup{Operator: warehouses.OpOr,
		Children: []warehouses.FilterCondition{
			&warehouses.IndependentOccurrenceCount{Source: source, Start: end.AddDate(0, -1, 0), End: end,
				Accept: []warehouses.CountRange{{Lower: 2}}},
			&warehouses.RecordFilter{Expr: record},
		}}}
	const profile = "00000000-0000-0000-0000-000000000001"
	query, err := profilefilter.BuildExplanation(filter, population, profile, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected explanation SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected typed explanation arguments, got %v", err)
	}
	if len(values) != 3 || len(query.Columns) != 6 || len(query.CountNodes) != 3 ||
		query.CountNodes[0] || !query.CountNodes[1] || query.CountNodes[2] {
		t.Fatalf("expected three arguments, six columns and one Count node, got %+v", query)
	}
	if !strings.Contains(query.SQL, "UNION ALL") || !strings.Contains(query.SQL, "LIMIT 4") ||
		strings.Contains(query.SQL, "00000000-0000-0000-0000-000000000001") {
		t.Fatalf("expected bounded rows and a parameterized identity, got %s", query.SQL)
	}
	// Positional placeholders must occur exactly once for each supplied argument.
	if strings.Contains(query.SQL, "?") && strings.Count(query.SQL, "?") != len(query.Arguments) {
		t.Fatalf("expected one positional placeholder per argument, got %s", query.SQL)
	}
	_, err = (&PostgreSQL{}).ExplainProfileFilter(t.Context(), warehouses.Filter{}, warehouses.ProfileSource{}, "bad")
	if err != nil {
		return
	}
	t.Fatal("expected invalid envelope before accessing a connection, got success")
}
