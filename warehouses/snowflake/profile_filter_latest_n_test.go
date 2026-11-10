// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// TestProfileFilterLatestNSQL checks blocks with explicit tie frames.
func TestProfileFilterLatestNSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, size := range []int{1, 2, 10} {
		for _, unbounded := range []bool{false, true} {
			node := &warehouses.LatestOccurrences{Size: size, Source: source, End: end, UnboundedStart: unbounded,
				Predicate: &warehouses.RecordQualification{Expr: warehouses.NewBaseExpr(
					warehouses.Column{Name: "event", Type: types.String()}, warehouses.OpStartsWith, "WEB-")}}
			if !unbounded {
				node.Start = end.AddDate(0, -1, 0)
			}
			statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected generated block SQL, got %v", err)
			}
			for _, fragment := range []string{
				"GROUP BY profile_id,time_min",
				"ORDER BY time_min DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
				"r.profile_id=b.profile_id AND r.time_min=b.time_min",
				"r.later_bad_c=0", "r.later_c+r.later_optional_good+r.good_t>=",
				"COALESCE(v0.absent,1)=0", "COALESCE(v0.reachable,0)=0",
				"qualification IS DISTINCT FROM FALSE", "time_min<>time_max",
			} {
				if !strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected block invariant %q, got %s", fragment, statement.SQL)
				}
			}
			if strings.Contains(statement.SQL, "ROW_NUMBER") || strings.Contains(statement.SQL, " IS NOT FALSE") {
				t.Fatalf("expected compatible sets and portable NULL handling, got %s", statement.SQL)
			}
			values, err := statement.Values()
			if err != nil {
				t.Fatalf("expected typed arguments, got %v", err)
			}
			arity := 2
			if unbounded {
				arity = 1
			}
			if len(values) != arity {
				t.Fatalf("expected %d arguments, got %v", arity, values)
			}
			statement, err = profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected bounded block metadata, got %v", err)
			}
			if len(statement.Columns) != 6 || len(statement.ExperienceNodes) != 1 || !statement.ExperienceNodes[0] ||
				statement.CountNodes[0] {
				t.Fatalf("expected candidate/absence protocol, got %+v", statement)
			}
		}
	}
}

// TestProfileFilterLatestNCardinalitySQL checks the shared extrema path.
func TestProfileFilterLatestNCardinalitySQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, size := range []int{1, 2, 10} {
		for _, bounds := range [][2]int64{{0, 0}, {0, int64(size)}, {1, int64(size)}, {int64(size), int64(size)}} {
			node := &warehouses.LatestOccurrences{Size: size, Source: source, Start: end.AddDate(0, -1, 0), End: end,
				Accepted: &warehouses.CountRange{Lower: bounds[0], Upper: &bounds[1]},
				Predicate: &warehouses.RecordQualification{Expr: warehouses.NewBaseExpr(
					warehouses.Column{Name: "event", Type: types.String()}, warehouses.OpStartsWith, "WEB-")}}
			statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected generated cardinality SQL, got %v", err)
			}
			for _, fragment := range []string{
				"GROUP BY profile_id,time_min",
				"ORDER BY time_min DESC ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
				"r.profile_id=b.profile_id AND r.time_min=b.time_min",
				"r.later_true+r.later_unknown", "r.later_nontrue+r.nontrue_t", "r.later_nonfalse+r.nonfalse_t",
				"AS min_count", "AS max_count", "COALESCE(v0.candidates,0)=0 OR v0.max_count<",
				"COALESCE(v0.absent,1)=0 AND v0.min_count>=",
				"qualification IS DISTINCT FROM FALSE", "time_min<>time_max",
			} {
				if !strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected cardinality invariant %q, got %s", fragment, statement.SQL)
				}
			}
			for _, fragment := range []string{"ROW_NUMBER", "GREATEST", "LEAST", " IS NOT FALSE"} {
				if strings.Contains(statement.SQL, fragment) {
					t.Fatalf("expected portable compatible-block calculation without %q, got %s", fragment, statement.SQL)
				}
			}
			values, err := statement.Values()
			if err != nil {
				t.Fatalf("expected typed arguments, got %v", err)
			}
			legacy := *node
			legacy.Accepted = nil
			original, err := profilefilter.Build(warehouses.Filter{Root: &legacy}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected legacy All SQL, got %v", err)
			}
			previous, err := original.Values()
			if err != nil {
				t.Fatalf("expected legacy All arguments, got %v", err)
			}
			if len(values) != 2 || !reflect.DeepEqual(values, previous) {
				t.Fatalf("expected unchanged period arguments %v, got %v", previous, values)
			}
			statement, err = profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected cardinality explanation, got %v", err)
			}
			if len(statement.Columns) != 6 || len(statement.ExperienceNodes) != 1 || !statement.ExperienceNodes[0] {
				t.Fatalf("expected existing selection protocol, got %+v", statement)
			}
		}
	}
}
