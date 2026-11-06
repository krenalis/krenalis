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

// TestProfileSumSQL checks shared compilation and the local decimal adapter.
func TestProfileSumSQL(t *testing.T) {

	reading := warehouses.DecimalReading{Column: warehouses.Column{Name: "properties", Type: types.JSON()},
		Keys: []string{"literal.key", "quote'key"}, Type: types.Decimal(18, 4)}
	value, err := profileExactDecimal(reading)
	if err != nil {
		t.Fatalf("expected exact decimal reading, got %v", err)
	}
	if strings.Contains(value, "DECFLOAT") || !strings.Contains(value, "literal.key") || !strings.Contains(value, "18,4") {
		t.Fatalf("expected exact typed reading without floating point conversion, got %s", value)
	}
	end := time.Date(2026, 2, 1, 0, 0, 0, 123456000, time.UTC)
	sum := &warehouses.IndependentOccurrenceSum{Source: warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID(), Nullable: true},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		UnboundedStart: true, End: end, Value: reading, AtLeast: true, Threshold: decimal.MustParse("30.0001")}
	filter := warehouses.Filter{Root: sum}
	population := warehouses.ProfileSource{Table: "profiles", Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}
	query, err := profilefilter.Build(filter, warehouses.ProfileOutcomeRequest{Population: population, Limit: 10}, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected Sum listing SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected exact arguments, got %v", err)
	}
	if len(values) != 1 || values[0] != "2026-02-01 00:00:00.123456" || strings.Contains(query.SQL, "lower_bound") ||
		!strings.Contains(query.SQL, "known_min<>known_max") || !strings.Contains(query.SQL, "amount_min<>amount_max") ||
		!strings.Contains(query.SQL, "occurrences>9223372036854775807") || !strings.Contains(query.SQL, "CAST('30.0001' AS DECIMAL(38,4))") {
		t.Fatalf("expected exact Sum, source faults and upper-only period, got %s and %v", query.SQL, values)
	}
	explained, err := profilefilter.BuildExplanation(filter, population, "00000000-0000-0000-0000-000000000001", profileFilterDialect())
	if err != nil {
		t.Fatalf("expected Sum explanation SQL, got %v", err)
	}
	if len(explained.Columns) != 8 || len(explained.SumNodes) != 1 || !explained.SumNodes[0] || explained.CountNodes[0] ||
		!strings.Contains(explained.SQL, "sum_lo") || !strings.Contains(explained.SQL, "<=128") {
		t.Fatalf("expected bounded Sum explanation transport, got %+v", explained)
	}
	bad := reading
	bad.Type = types.Float(64)
	_, err = profileExactDecimal(bad)
	if err == nil {
		t.Fatal("expected invalid reading rejection, got success")
	}
	sum.Threshold = decimal.MustParse("1e38")
	_, err = profilefilter.Build(filter, warehouses.ProfileOutcomeRequest{Population: population, Limit: 10}, profileFilterDialect())
	if err == nil {
		t.Fatal("expected invalid prepared threshold rejection, got success")
	}

}
