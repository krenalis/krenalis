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

// TestProfileExactText keeps source names literal and parameterized.
func TestProfileExactText(t *testing.T) {

	for _, column := range []warehouses.Column{{Name: "bad.name", Type: types.String()},
		{Name: "event", Type: types.JSON()}} {
		_, err := profileExactText(column)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid exact text column, got %+v", column)
	}

	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	node := &warehouses.IndependentOccurrenceCount{Source: warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		Start: end.Add(-time.Hour), End: end, Qualification: &warehouses.ExactColumnText{
			Column: warehouses.Column{Name: "event", Type: types.String(), Nullable: true}, Value: "Purchase'"}}
	statement, err := profilefilter.Build(warehouses.Filter{Root: node}, warehouses.ProfileOutcomeRequest{
		Population: warehouses.ProfileSource{Table: "profiles",
			Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}},
		Limit: 1}, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected exact source SQL, got %v", err)
	}
	if !strings.Contains(statement.SQL, `(COLLATE("EVENT", 'utf8'))=?`) ||
		strings.Contains(statement.SQL, "Purchase") ||
		len(statement.Arguments) != 3 || statement.Arguments[2].Value != "Purchase'" {
		t.Fatalf("expected literal parameter and fixed collation, got %+v", statement)
	}

}
