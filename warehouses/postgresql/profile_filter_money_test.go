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

// TestProfileFilterMoneyConditioning requires an identical direct base reading.
func TestProfileFilterMoneyConditioning(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	properties := warehouses.Column{Name: "properties", Type: types.JSON()}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, test := range []struct {
		name, column, key, value string
		or, nested, outside      bool
		arguments                int
		amount                   bool
	}{
		{"same", "properties", "currency", "EUR", false, false, false, 4, true},
		{"different currency", "properties", "currency", "USD", false, false, false, 4, false},
		{"different keys", "properties", "billing_currency", "EUR", false, false, false, 5, true},
		{"different column", "traits", "currency", "EUR", false, false, false, 5, true},
		{"OR is not required", "properties", "currency", "EUR", true, false, false, 5, true},
		{"nested is not direct", "properties", "currency", "EUR", false, true, false, 5, true},
		{"Count same", "properties", "currency", "EUR", false, false, true, 3, true},
		{"Count different currency", "properties", "currency", "USD", false, false, true, 3, false},
		{"Count different keys", "properties", "billing_currency", "EUR", false, false, true, 4, true},
		{"Count OR is not required", "properties", "currency", "EUR", true, false, true, 4, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			known := &warehouses.KnownTextEqual{Column: warehouses.Column{Name: test.column, Type: types.JSON()},
				Keys: []string{test.key}, Value: test.value}
			other := &warehouses.RecordQualification{Expr: warehouses.NewBaseExpr(
				warehouses.Column{Name: "type", Type: types.String()}, warehouses.OpIs, "track")}
			var fact warehouses.RowQualification = known
			if test.nested {
				fact = &warehouses.QualificationGroup{Operator: warehouses.OpOr,
					Children: []warehouses.RowQualification{known, other}}
			}
			base := &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: []warehouses.RowQualification{fact}}
			if test.or {
				base.Operator = warehouses.OpOr
				base.Children = append(base.Children, other)
			}
			money := &warehouses.MoneyQualification{Currency: warehouses.KnownTextEqual{Column: properties,
				Keys: []string{"currency"}, Value: "EUR"}, Amount: warehouses.DecimalReading{Column: properties,
				Keys: []string{"total"}, Type: types.Decimal(18, 4)}, Threshold: decimal.MustParse("100"), AtLeast: true}
			verify := &warehouses.QualificationGroup{Operator: warehouses.OpOr, Children: []warehouses.RowQualification{money}}
			var root warehouses.FilterCondition = &warehouses.SelectedOccurrence{Source: source, Start: end.Add(-time.Hour),
				End: end, Qualification: base, Verify: verify}
			if test.outside {
				base.Children = append(base.Children, money)
				root = &warehouses.IndependentOccurrenceCount{Source: source, Start: end.Add(-time.Hour), End: end,
					Qualification: base, Accept: []warehouses.CountRange{{Lower: 1}}}
			}
			statement, err := profilefilter.Build(warehouses.Filter{Root: root}, request, profileFilterDialect())
			if err != nil {
				t.Fatalf("expected contextual monetary SQL, got %v", err)
			}
			values, err := statement.Values()
			if err != nil {
				t.Fatalf("expected contextual arguments, got %v", err)
			}
			if len(values) != test.arguments || values[2] != test.value {
				t.Fatalf("expected %d arguments with base currency %s, got %v", test.arguments, test.value, values)
			}
			literal := "CAST('100.0000' AS DECIMAL(38,4))"
			if strings.Contains(statement.SQL, literal) != test.amount {
				t.Fatalf("expected amount comparison=%v, got %s", test.amount, statement.SQL)
			}
			if !test.amount && !test.outside &&
				!strings.Contains(statement.SQL, "THEN ((FALSE)) ELSE CAST(NULL AS BOOLEAN) END AS verification") {
				t.Fatalf("expected conditional FALSE verification, got %s", statement.SQL)
			}
			explained, err := profilefilter.BuildExplanation(warehouses.Filter{Root: root}, request.Population,
				"00000000-0000-0000-0000-000000000001", profileFilterDialect())
			if err != nil {
				t.Fatalf("expected contextual explanation SQL, got %v", err)
			}
			for _, query := range []profilefilter.Statement{statement, explained} {
				values, err := query.Values()
				if err != nil {
					t.Fatalf("expected exact query arguments, got %v", err)
				}
				marker, err := profileParameter(len(values), types.String())
				if err != nil {
					t.Fatalf("expected native parameter marker, got %v", err)
				}
				uses := 1
				if marker == "?" {
					uses = len(values)
				}
				if strings.Count(query.SQL, marker) != uses {
					t.Fatalf("expected %d uses of %s, got %s", uses, marker, query.SQL)
				}
			}
		})
	}
}

// TestProfileFilterMoneyCountGroups accounts for the source wrapper depth.
func TestProfileFilterMoneyCountGroups(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	source := warehouses.OccurrenceSource{Table: "events",
		Profile:   warehouses.Column{Name: "kpid", Type: types.UUID()},
		Identity:  warehouses.Column{Name: "message_id", Type: types.String()},
		Timestamp: warehouses.Column{Name: "timestamp", Type: types.DateTime()}}
	properties := warehouses.Column{Name: "properties", Type: types.JSON()}
	money := &warehouses.MoneyQualification{Currency: warehouses.KnownTextEqual{Column: properties,
		Keys: []string{"currency"}, Value: "EUR"}, Amount: warehouses.DecimalReading{Column: properties,
		Keys: []string{"total"}, Type: types.Decimal(18, 4)}, Threshold: decimal.MustParse("100"), AtLeast: true}
	request := warehouses.ProfileOutcomeRequest{Population: warehouses.ProfileSource{Table: "profiles",
		Key: warehouses.Column{Name: "_kpid", Type: types.UUID()}}, Limit: 10}
	for _, groups := range []int{1, 2, 3} {
		var body warehouses.RowQualification = money
		for range groups {
			body = &warehouses.QualificationGroup{Operator: warehouses.OpOr, Children: []warehouses.RowQualification{body}}
		}
		root := &warehouses.IndependentOccurrenceCount{Source: source, Start: end.Add(-time.Hour), End: end,
			Qualification: &warehouses.QualificationGroup{Operator: warehouses.OpAnd, Children: []warehouses.RowQualification{
				&warehouses.RecordQualification{Expr: warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String()},
					warehouses.OpIs, "track")}, body}}, Accept: []warehouses.CountRange{{Lower: 1}}}
		_, err := profilefilter.Build(warehouses.Filter{Root: root}, request, profileFilterDialect())
		if err != nil {
			if groups != 3 {
				t.Fatalf("expected %d user groups accepted, got %v", groups, err)
			}
			continue
		}
		if groups == 3 {
			t.Fatal("expected third user group rejected, got success")
		}
	}
}

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
	for _, test := range []struct {
		threshold string
		atLeast   bool
	}{
		{"100.0001", true}, {"-100.0001", false},
		{"1e14", true}, {"-1e14", true}, {"1e14", false}, {"-1e14", false},
	} {
		money := &warehouses.MoneyQualification{
			Currency:  warehouses.KnownTextEqual{Column: properties, Keys: []string{"currency"}, Value: "EUR"},
			Amount:    warehouses.DecimalReading{Column: properties, Keys: []string{"total"}, Type: types.Decimal(18, 4)},
			Threshold: decimal.MustParse(test.threshold), AtLeast: test.atLeast,
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
		if len(values) != expected || values[2] != "EUR" {
			t.Fatalf("expected %d arguments with native currency, got %v", expected, values)
		}
		literal := fmt.Sprintf("CAST('%.4f' AS DECIMAL(38,4))", money.Threshold)
		if !strings.Contains(statement.SQL, literal) {
			t.Fatalf("expected exact normalized decimal %s, got %s", literal, statement.SQL)
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
