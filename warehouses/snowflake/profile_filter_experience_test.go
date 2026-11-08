// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// TestProfileFilterExperienceSQL checks First/Last without a technical tie-break.
func TestProfileFilterExperienceSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	verify := &warehouses.RecordQualification{Expr: &warehouses.BaseExpr{
		Column:   warehouses.Column{Name: "event", Type: types.String()},
		Operator: warehouses.OpStartsWith, Values: []any{"WEB-"}}}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, test := range []struct{ first, unbounded, temporal bool }{
		{false, false, false}, {false, true, false}, {true, false, false}, {true, true, false},
		{false, false, true}, {true, true, true},
	} {
		first, unbounded := test.first, test.unbounded
		node := &warehouses.SelectedOccurrence{First: first, Source: source, End: end,
			UnboundedStart: unbounded, Verify: verify}
		if !unbounded {
			node.Start = end.AddDate(0, -1, 0)
		}
		if test.temporal {
			start := time.Date(2026, 1, 10, 0, 0, 0, 123456000, time.UTC)
			node.Verify = &warehouses.RecordQualification{Expr: warehouses.NewMultiExpr(warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(source.Timestamp, warehouses.OpIsOnOrAfter, start),
					warehouses.NewBaseExpr(source.Timestamp, warehouses.OpIsBefore, start.Add(3*time.Microsecond)),
				})}
		}
		statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
		if err != nil {
			t.Fatalf("expected generated selection SQL, got %v", err)
		}
		extreme, direction := "MAX", ">="
		if first {
			extreme, direction = "MIN", "<="
		}
		for _, fragment := range []string{
			"OVER (PARTITION BY profile_id)", "certain_time IS NULL OR time_min" + direction + "certain_time",
			extreme + "(CASE WHEN state_min=1 THEN time_min END)",
			"time_min<>time_max", "COALESCE(v0.absent,1)=0", "COALESCE(v0.failed,0)=COALESCE(v0.candidates,0)",
		} {
			if !strings.Contains(statement.SQL, fragment) {
				t.Fatalf("expected selection invariant %q, got %s", fragment, statement.SQL)
			}
		}
		if test.temporal {
			for _, fragment := range []string{
				quoteIdent(source.Timestamp.Name) + ` >= '2026-01-10 00:00:00.123456'`,
				quoteIdent(source.Timestamp.Name) + ` < '2026-01-10 00:00:00.123459'`,
			} {
				if !strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected exact half-open microsecond comparison %q, got %s", fragment, statement.SQL)
				}
			}
		}
		if strings.Contains(statement.SQL, "ROW_NUMBER") {
			t.Fatalf("expected all compatible winners, got %s", statement.SQL)
		}
		if strings.Contains(statement.SQL, " IS NOT FALSE") || !strings.Contains(statement.SQL, " IS DISTINCT FROM FALSE") {
			t.Fatalf("expected portable ternary qualification, got %s", statement.SQL)
		}
		values, err := statement.Values()
		if err != nil {
			t.Fatalf("expected typed time arguments, got %v", err)
		}
		expected := 2
		if unbounded {
			expected = 1
		}
		if len(values) != expected {
			t.Fatalf("expected %d time arguments, got %v", expected, values)
		}
		statement, err = profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
			"00000000-0000-0000-0000-000000000001", profileFilterDialect())
		if err != nil {
			t.Fatalf("expected bounded selection metadata, got %v", err)
		}
		if len(statement.Columns) != 6 || len(statement.ExperienceNodes) != 1 || !statement.ExperienceNodes[0] ||
			statement.CountNodes[0] {
			t.Fatalf("expected dedicated candidate/absence metadata, got %+v", statement)
		}
	}
}
