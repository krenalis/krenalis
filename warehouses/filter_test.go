// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"math"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateProfileFilter checks the prepared contract and its budgets.
func TestValidateProfileFilter(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	base := IndependentOccurrenceCount{
		Source: OccurrenceSource{Table: "krenalis_events", Profile: Column{Name: "kpid", Type: types.UUID(),
			Nullable: true}, Identity: Column{Name: "message_id", Type: types.String()},
			Timestamp: Column{Name: "timestamp", Type: types.DateTime()}},
		Start: end.AddDate(0, -1, 0), End: end,
	}
	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles", Key: Column{Name: "_kpid",
		Type: types.UUID()}}, Limit: 1}
	upper := func(n int64) *int64 { return &n }
	for _, test := range []struct {
		name   string
		change func(*IndependentOccurrenceCount)
	}{
		{"source", func(c *IndependentOccurrenceCount) { c.Source.Table = "bad.name" }},
		{"identity", func(c *IndependentOccurrenceCount) { c.Source.Identity.Nullable = true }},
		{"unknown time", func(c *IndependentOccurrenceCount) { c.Source.Timestamp.Nullable = true }},
		{"empty period", func(c *IndependentOccurrenceCount) { c.Start = c.End }},
		{"precision", func(c *IndependentOccurrenceCount) { c.Start = c.Start.Add(time.Nanosecond) }},
		{"negative range", func(c *IndependentOccurrenceCount) { c.Accept = []CountRange{{Lower: -1}} }},
		{"reverse range", func(c *IndependentOccurrenceCount) { c.Accept = []CountRange{{Lower: 2, Upper: upper(1)}} }},
		{"adjacent", func(c *IndependentOccurrenceCount) {
			c.Accept = []CountRange{{Upper: upper(0)}, {Lower: 1, Upper: upper(1)}}
		}},
		{"overlap", func(c *IndependentOccurrenceCount) { c.Accept = []CountRange{{Upper: upper(2)}, {Lower: 1}} }},
		{"unbounded first", func(c *IndependentOccurrenceCount) { c.Accept = []CountRange{{}, {Lower: 3}} }},
		{"max first", func(c *IndependentOccurrenceCount) {
			c.Accept = []CountRange{{Upper: upper(math.MaxInt64)}, {Lower: math.MaxInt64}}
		}},
		{"three ranges", func(c *IndependentOccurrenceCount) {
			c.Accept = []CountRange{{Upper: upper(0)}, {Lower: 2, Upper: upper(2)}, {Lower: 4}}
		}},
		{"known native String", func(c *IndependentOccurrenceCount) {
			c.Qualification = &KnownTextEqual{Column: Column{Name: "event", Type: types.String()}, Value: "Purchase"}
		}},
		{"bad known key", func(c *IndependentOccurrenceCount) {
			c.Qualification = &KnownTextEqual{Column: Column{Name: "properties", Type: types.JSON()},
				Keys: []string{"bad\x00key"}}
		}},
		{"nil qualifier", func(c *IndependentOccurrenceCount) { c.Qualification = (*KnownTextEqual)(nil) }},
		{"empty group", func(c *IndependentOccurrenceCount) { c.Qualification = &QualificationGroup{Operator: OpAnd} }},
		{"column operand", func(c *IndependentOccurrenceCount) {
			c.Qualification = &RecordQualification{Expr: NewBaseExpr(Column{Name: "event", Type: types.String()},
				OpIs, Column{Name: "other", Type: types.String()})}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := base
			test.change(&c)
			err := ValidateProfileFilter(Filter{Root: &c}, request)
			if err == nil {
				t.Fatal("expected invalid prepared input, got success")
			}
		})
	}
	for _, ranges := range [][]CountRange{nil, {{}}, {{Upper: upper(0)}}, {{Lower: 1, Upper: upper(1)},
		{Lower: 3}}, {{Lower: math.MaxInt64, Upper: upper(math.MaxInt64)}}} {
		c := base
		c.Accept = ranges
		err := ValidateProfileFilter(Filter{Root: &c}, request)
		if err != nil {
			t.Fatalf("expected valid normalized ranges, got %v", err)
		}
	}
	for _, limit := range []int{0, 1001} {
		r := request
		r.Limit = limit
		err := ValidateProfileFilter(Filter{}, r)
		if err == nil {
			t.Fatal("expected invalid page, got success")
		}
	}
	var node FilterCondition = &base
	for range 4 {
		node = &FilterGroup{Operator: OpAnd, Children: []FilterCondition{node}}
	}
	err := ValidateProfileFilter(Filter{Root: node}, request)
	if err != nil {
		t.Fatalf("expected four groups including root, got %v", err)
	}
	err = ValidateProfileFilter(Filter{Root: &FilterGroup{Operator: OpAnd, Children: []FilterCondition{node}}}, request)
	if err == nil {
		t.Fatal("expected fifth group rejection, got success")
	}
	cyclic := &QualificationGroup{Operator: OpAnd}
	cyclic.Children = []RowQualification{cyclic}
	c := base
	c.Qualification = cyclic
	err = ValidateProfileFilter(Filter{Root: &c}, request)
	if err == nil {
		t.Fatal("expected cyclic qualification rejection, got success")
	}
	record := NewMultiExpr(OpAnd, nil)
	record.Operands = []Expr{record}
	c.Qualification = &RecordQualification{Expr: record}
	err = ValidateProfileFilter(Filter{Root: &c}, request)
	if err == nil {
		t.Fatal("expected cyclic Record rejection, got success")
	}
}

// TestValidateProfileRecord distinguishes semantic and physical budgets.
func TestValidateProfileRecord(t *testing.T) {

	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles", Key: Column{Name: "_kpid",
		Type: types.UUID()}}, Limit: 1}
	base := NewBaseExpr(Column{Name: "settings_value", Type: types.String(), Nullable: true}, OpExists)
	record := &RecordFilter{Expr: NewMultiExpr(OpAnd, []Expr{NewMultiExpr(OpOr, []Expr{base})})}
	var node FilterCondition = record
	for range 4 {
		node = &FilterGroup{Operator: OpAnd, Children: []FilterCondition{node}}
	}
	err := ValidateProfileFilter(Filter{Root: node}, request)
	if err != nil {
		t.Fatalf("expected four semantic groups with object expansion, got %v", err)
	}
	node = &FilterGroup{Operator: OpOr, Children: []FilterCondition{node}}
	err = ValidateProfileFilter(Filter{Root: node}, request)
	if err == nil {
		t.Fatal("expected fifth semantic group rejection, got success")
	}

	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := &IndependentOccurrenceCount{Source: OccurrenceSource{Table: "events",
		Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "message_id", Type: types.String()},
		Timestamp: Column{Name: "timestamp", Type: types.DateTime()}}, Start: end.AddDate(0, -1, 0), End: end}
	group := &FilterGroup{Operator: OpAnd, Children: make([]FilterCondition, 100)}
	for i := range group.Children {
		group.Children[i] = record
		if i%2 == 0 {
			group.Children[i] = count
		}
	}
	err = ValidateProfileFilter(Filter{Root: group}, request)
	if err != nil {
		t.Fatalf("expected 100 mixed descendant nodes, got %v", err)
	}
	group.Children = append(group.Children, record)
	err = ValidateProfileFilter(Filter{Root: group}, request)
	if err == nil {
		t.Fatal("expected 101 mixed descendant rejection, got success")
	}

	// Two individually bounded expansions consume the entire shared budget.
	expanded := NewMultiExpr(OpOr, make([]Expr, 8191))
	for i := range expanded.Operands {
		expanded.Operands[i] = base
	}
	wide := &RecordFilter{Expr: expanded}
	group.Children = []FilterCondition{wide, wide}
	err = ValidateProfileFilter(Filter{Root: group}, request)
	if err != nil {
		t.Fatalf("expected exact physical budget boundary, got %v", err)
	}
	for _, extra := range []FilterCondition{record, count} {
		count.Qualification = &RecordQualification{Expr: base}
		group.Children = []FilterCondition{wide, wide, extra}
		err = ValidateProfileFilter(Filter{Root: group}, request)
		if err == nil {
			t.Fatal("expected shared physical budget rejection, got success")
		}
	}

	for _, invalid := range []*RecordFilter{nil, {},
		{Expr: NewBaseExpr(Column{Name: "event", Type: types.String()}, OpIs, Column{Name: "other", Type: types.String()})},
	} {
		err = ValidateProfileFilter(Filter{Root: invalid}, request)
		if err == nil {
			t.Fatal("expected invalid Record rejection, got success")
		}
	}

}
