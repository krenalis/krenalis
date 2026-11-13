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

// TestProfileIdentitySQL keeps identity comparisons literal across CTEs.
func TestProfileIdentitySQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	reading := warehouses.DecimalReading{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
		Keys: []string{"total"}, Type: types.Decimal(18, 4)}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	threshold := decimal.MustParse("0")
	for _, test := range []struct {
		name string
		node warehouses.FilterCondition
	}{
		{"count", &warehouses.IndependentOccurrenceCount{Source: source, Start: end.Add(-time.Hour), End: end, Accept: []warehouses.CountRange{{Lower: 1}}}},
		{"sum", &warehouses.IndependentOccurrenceSum{Source: source, Start: end.Add(-time.Hour), End: end, Value: reading}},
		{"selection", &warehouses.SelectedOccurrence{Source: source, Start: end.Add(-time.Hour), End: end, Verify: &warehouses.RecordQualification{Expr: warehouses.NewBaseExpr(source.Identity, warehouses.OpExists)}}},
		{"pair", &warehouses.LastPreviousDifference{Source: source, Start: end.Add(-time.Hour), End: end, Value: reading, Threshold: &threshold}},
	} {
		t.Run(test.name, func(t *testing.T) {
			filter := warehouses.Filter{Root: test.node}
			query, err := profilefilter.Build(filter, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected literal identity query, got %v", err)
			}
			explanation, err := profilefilter.BuildExplanation(filter, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected literal identity explanation, got %v", err)
			}
			id := `COLLATE("MESSAGE_ID", 'utf8')`
			occurrenceID := `COLLATE("OCCURRENCE_ID", 'utf8')`
			for _, statement := range []profilefilter.Statement{query, explanation} {
				for _, fragment := range []string{
					"SELECT DISTINCT " + id + " AS occurrence_id", id + " AS occurrence_id,",
					" WHERE " + id + " IN (SELECT " + occurrenceID + " FROM ids0)",
					quoteIdent(source.Identity.Name) + " IS NULL",
				} {
					if !strings.Contains(statement.SQL, fragment) {
						t.Fatalf("expected identity fragment %q, got %s", fragment, statement.SQL)
					}
				}
			}
			fragment := " AND " + id + " IN (SELECT " + id + " FROM " + quoteIdent(source.Table) + " WHERE CAST("
			if !strings.Contains(explanation.SQL, fragment) {
				t.Fatalf("expected both sides of profile identity seed %q, got %s", fragment, explanation.SQL)
			}
			for i, statement := range []profilefilter.Statement{query, explanation} {
				values, err := statement.Values()
				if err != nil {
					t.Fatalf("expected unchanged timestamp arguments, got %v", err)
				}
				if len(values) != 2+i || values[i] != "2026-01-31 23:00:00.123456" || values[i+1] != "2026-02-01 00:00:00.123456" {
					t.Fatalf("expected two exact timestamps after %d target arguments, got %v", i, values)
				}
			}
		})
	}
}
