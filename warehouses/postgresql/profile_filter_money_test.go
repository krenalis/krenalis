// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
	"github.com/krenalis/krenalis/warehouses/internal/profilefilter"
)

// TestProfileFilterMoneySQL checks transport and the unchanged E protocol.
func TestProfileFilterMoneySQL(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	properties := warehouses.Column{Name: "properties", Type: types.JSON()}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	yes, no := true, false
	for _, test := range []struct {
		threshold string
		atLeast   bool
		known     *bool
	}{
		{"100.0001", true, nil}, {"-100.0001", false, nil},
		{"1e14", true, &no}, {"-1e14", true, &yes},
	} {
		money := &warehouses.MoneyQualification{
			Currency:  warehouses.KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
			Amount:    warehouses.DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
			Threshold: decimal.MustParse(test.threshold), AtLeast: test.atLeast, KnownResult: test.known,
		}
		// Two user groups must still leave room for the closed monetary leaf.
		verify := &warehouses.QualificationGroup{Operator: warehouses.OpAnd,
			Children: []warehouses.RowQualification{&warehouses.QualificationGroup{Operator: warehouses.OpOr,
				Children: []warehouses.RowQualification{money}}}}
		node := &warehouses.SelectedOccurrence{Source: source, Start: end.AddDate(0, -1, 0), End: end, Verify: verify}
		statement, err := profilefilter.Build(warehouses.Filter{Root: node}, request, profileFilterDialect())
		if err != nil {
			t.Fatalf("expected monetary SQL, got %v", err)
		}
		values, err := statement.Values()
		if err != nil {
			t.Fatalf("expected exact typed monetary arguments, got %v", err)
		}
		expected := 3
		if test.known != nil {
			truth := "FALSE"
			if *test.known {
				truth = "TRUE"
			}
			if !strings.Contains(statement.SQL, "IS NULL THEN CAST(NULL AS BOOLEAN) ELSE "+truth+" END") {
				t.Fatalf("expected readable-only precomputed result, got %s", statement.SQL)
			}
		}
		if len(values) != expected || values[2] != "EUR" {
			t.Fatalf("expected %d arguments with native currency, got %v", expected, values)
		}
		if test.known == nil {
			literal := fmt.Sprintf("CAST('%.4f' AS DECIMAL(38,4))", money.Threshold)
			if !strings.Contains(statement.SQL, literal) {
				t.Fatalf("expected exact normalized decimal %s, got %s", literal, statement.SQL)
			}
		}
		explained, err := profilefilter.BuildExplanation(warehouses.Filter{Root: node}, request.Population,
			"00000000-0000-0000-0000-000000000001", profileFilterDialect())
		if err != nil {
			t.Fatalf("expected selected monetary explanation, got %v", err)
		}
		if len(explained.Columns) != 6 || !explained.ExperienceNodes[0] || explained.CountNodes[0] {
			t.Fatalf("expected unchanged candidate/absence protocol, got %+v", explained)
		}
	}
}
