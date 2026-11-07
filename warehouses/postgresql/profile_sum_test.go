// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
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
	request := warehouses.ProfileOutcomeRequest{Population: population, Limit: 10}
	query, err := profilefilter.Build(filter, request, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected Sum listing SQL, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected exact arguments, got %v", err)
	}
	if len(values) != 1 || values[0] != "2026-02-01 00:00:00.123456" || strings.Contains(query.SQL, "lower_bound") ||
		!strings.Contains(query.SQL, "known_min<>known_max") || !strings.Contains(query.SQL, "amount_min<>amount_max") ||
		!strings.Contains(query.SQL, "occurrences>9223372036854775807") ||
		!strings.Contains(query.SQL, "CAST('30.0001' AS DECIMAL(38,4))") {
		t.Fatalf("expected exact Sum, source faults and upper-only period, got %s and %v", query.SQL, values)
	}
	profile := "00000000-0000-0000-0000-000000000001"
	explained, err := profilefilter.BuildExplanation(filter, population, profile, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected Sum explanation SQL, got %v", err)
	}
	if len(explained.Columns) != 8 || len(explained.SumNodes) != 1 ||
		!explained.SumNodes[0] || explained.CountNodes[0] ||
		!strings.Contains(explained.SQL, "sum_lo") || !strings.Contains(explained.SQL, "<=128") {
		t.Fatalf("expected bounded Sum explanation transport, got %+v", explained)
	}

	for _, minimum := range []int64{1, 9223372036854775807, -1} {
		sum.Minimum = minimum
		query, err := profilefilter.Build(filter, request, profileFilterDialect())
		if err != nil {
			if minimum >= 0 {
				t.Fatalf("expected supported sample minimum %d, got %v", minimum, err)
			}
			continue
		}
		if minimum < 0 {
			t.Fatal("expected invalid prepared sample minimum, got success")
		}
		guard := "COALESCE(v0.lo,0)>=" + strconv.FormatInt(minimum, 10)
		if !strings.Contains(query.SQL, guard) {
			t.Fatalf("expected sample guard %s, got %s", guard, query.SQL)
		}
		explained, err := profilefilter.BuildExplanation(filter, population, profile, profileFilterDialect())
		if err != nil {
			t.Fatalf("expected guarded Sum explanation, got %v", err)
		}
		if len(explained.Columns) != 8 || explained.CountNodes[0] || !explained.SumNodes[0] ||
			!strings.Contains(explained.SQL, guard) || !strings.Contains(explained.SQL, "COALESCE(v0.lo,0) AS") {
			t.Fatalf("expected Sum count and monetary bounds with the same guard, got %+v", explained)
		}
	}
	sum.Minimum = 0

	for _, literal := range []string{"1000.0000", "5000.0000", "-2000.0000",
		"1000000000000000000000000000000000.0000", "-1000000000000000000000000000000000.0000"} {
		t.Run(literal, func(t *testing.T) {
			sum.Threshold = decimal.MustParse(literal)
			query, err := profilefilter.Build(filter, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected exact Sum threshold SQL, got %v", err)
			}
			if !strings.Contains(query.SQL, "CAST('"+literal+"' AS DECIMAL(38,4))") {
				t.Fatalf("expected unchanged threshold %s in SQL, got %s", literal, query.SQL)
			}
		})
	}

	bad := reading
	bad.Type = types.Float(64)
	_, err = profileExactDecimal(bad)
	if err == nil {
		t.Fatal("expected invalid reading rejection, got success")
	}
	sum.Threshold = decimal.MustParse("1e38")
	_, err = profilefilter.Build(filter, request, profileFilterDialect())
	if err == nil {
		t.Fatal("expected invalid prepared threshold rejection, got success")
	}

}

// TestProfileSumWhereSQL composes the real Record JSON renderer into Sum.
func TestProfileSumWhereSQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	properties := warehouses.Column{Name: "properties", Type: types.JSON()}
	where := &warehouses.BaseExpr{Column: properties, Keys: []string{"coupon"}, Operator: warehouses.OpIs,
		Values: []any{json.Value(`"PROMO"`)}, JSONComparison: warehouses.JSONTypedComparison}
	sum := &warehouses.IndependentOccurrenceSum{Source: warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}},
		UnboundedStart: true, End: end, AtLeast: true, Threshold: decimal.MustParse("30"),
		Value: warehouses.DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
		Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd,
			Children: []warehouses.RowQualification{
				&warehouses.KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
				&warehouses.RecordQualification{Expr: where},
			}}}
	filter := warehouses.Filter{Root: sum}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	query, err := profilefilter.Build(filter, request, profileFilterDialect())
	if err != nil {
		t.Fatalf("expected qualified Sum SQL, got %v", err)
	}
	record, err := profileRecord(where)
	if err != nil {
		t.Fatalf("expected Record JSON rendering, got %v", err)
	}
	values, err := query.Values()
	if err != nil {
		t.Fatalf("expected qualified Sum arguments, got %v", err)
	}
	if !strings.Contains(query.SQL, " AND ("+record+")") || !strings.Contains(record, "coupon") ||
		!strings.Contains(record, "PROMO") || len(values) != 2 || values[1] != "EUR" {
		t.Fatalf("expected conjunctive typed Record qualification and native currency, got %s, %v", query.SQL, values)
	}
}
