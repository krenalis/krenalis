// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"fmt"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestValidate checks capability admission and canonical structural limits.
func TestValidate(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	base := Count{Scope: Scope{Event: "Purchase", Grain: "registered-message", Start: at.AddDate(0, -1, 0),
		End: at}, Test: CountTest{Operator: Equal}}
	for _, test := range []struct {
		name        string
		change      func(*Count)
		unsupported bool
	}{
		{"shared g", func(c *Count) { c.Shared = &SharedBoolean{Fact: "g", EqualTo: true} }, true},
		{"shared false comparison", func(c *Count) { c.Shared = &SharedBoolean{Fact: "g", EqualTo: false} }, true},
		{"unknown grain", func(c *Count) { c.Scope.Grain = "business-order" }, false},
		{"arbitrary grain", func(c *Count) { c.Scope.Grain = "banana" }, false},
		{"missing grain", func(c *Count) { c.Scope.Grain = "" }, false},
		{"too large", func(c *Count) { c.Test.Value = 1 << 63 }, true},
		{"invalid period", func(c *Count) { c.Scope.FullHistory = false; c.Scope.Start = at; c.Scope.End = at }, false},
		{"history bounds", func(c *Count) { c.Scope.FullHistory = true }, false},
		{"empty local group", func(c *Count) { c.Where = &state.Where{Operator: state.OpAnd} }, false},
		{"incompatible guard", func(c *Count) { c.Minimum = 2; c.Test.Value = 1 }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := base
			test.change(&c)
			err := Validate(&Rule{Count: &c}, at, Page{Limit: 1})
			if err != nil {
				_, unsupported := errors.AsType[*UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
			}
			if err == nil {
				t.Fatal("expected rejected input, got success")
			}
		})
	}
	rule := Rule{Count: &base}
	for range 4 {
		rule = Rule{Operator: state.OpAnd, Children: []Rule{rule}}
	}
	err := Validate(&rule, at, Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected four groups including root, got %v", err)
	}
	rule = Rule{Operator: state.OpAnd, Children: []Rule{rule}}
	err = Validate(&rule, at, Page{Limit: 1})
	if err == nil {
		t.Fatal("expected fifth group rejection, got success")
	}
	rule = Rule{Operator: state.OpOr, Children: make([]Rule, 100)}
	for i := range rule.Children {
		rule.Children[i] = Rule{Count: &base}
	}
	err = Validate(&rule, at, Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected 100 descendant nodes, got %v", err)
	}
	rule.Children = append(rule.Children, Rule{Count: &base})
	err = Validate(&rule, at, Page{Limit: 1})
	if err == nil {
		t.Fatal("expected 101 descendant rejection, got success")
	}
	err = Validate(nil, at.Add(time.Nanosecond), Page{Limit: 1})
	if err != nil {
		if _, ok := errors.AsType[*UnsupportedError](err); !ok {
			t.Fatalf("expected unsupported precision, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected precision rejection, got success")
	}
}

// TestValidateElapsedPeriod distinguishes invalid periods from unsupported
// precision and timestamp ranges before any binding or execution.
func TestValidateElapsedPeriod(t *testing.T) {
	for _, test := range []struct {
		name        string
		change      func(*Scope, *time.Time)
		unsupported bool
	}{
		{"zero width", func(s *Scope, at *time.Time) { s.Elapsed.Older = 0 }, false},
		{"negative width", func(s *Scope, at *time.Time) { s.Elapsed.Older = -time.Hour }, false},
		{"minimum duration", func(s *Scope, at *time.Time) { s.Elapsed.Older = -1 << 63 }, false},
		{"negative newer", func(s *Scope, at *time.Time) { s.Elapsed.Newer = -time.Microsecond }, false},
		{"equal offsets", func(s *Scope, at *time.Time) { s.Elapsed.Newer = s.Elapsed.Older }, false},
		{"reversed offsets", func(s *Scope, at *time.Time) { s.Elapsed.Newer = 48 * time.Hour }, false},
		{"explicit start", func(s *Scope, at *time.Time) { s.Start = at.Add(-time.Hour) }, false},
		{"explicit end", func(s *Scope, at *time.Time) { s.End = *at }, false},
		{"history", func(s *Scope, at *time.Time) { s.FullHistory = true }, false},
		{"older nanoseconds", func(s *Scope, at *time.Time) { s.Elapsed.Older += time.Nanosecond }, true},
		{"newer nanoseconds", func(s *Scope, at *time.Time) { s.Elapsed.Newer = time.Nanosecond }, true},
		{"T nanoseconds", func(s *Scope, at *time.Time) { *at = at.Add(time.Nanosecond) }, true},
		{"before supported dates", func(s *Scope, at *time.Time) {
			*at = time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)
			s.Elapsed.Older = 48 * time.Hour
		}, true},
		{"zero timestamp", func(s *Scope, at *time.Time) {
			*at = time.Date(1, 1, 2, 0, 0, 0, 0, time.UTC)
		}, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			scope := Scope{Event: "Purchase", Grain: "registered-message",
				Elapsed: &ElapsedPeriod{Older: 24 * time.Hour}}
			test.change(&scope, &at)
			err := Validate(&Rule{Count: &Count{Scope: scope}}, at, Page{Limit: 1})
			if err != nil {
				_, unsupported := errors.AsType[*UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected elapsed period, got success")
		})
	}
}

// TestValidateInvalidBeforeUnsupported gives shape errors precedence across
// the canonical tree, including unavailable earlier siblings and T precision.
func TestValidateInvalidBeforeUnsupported(t *testing.T) {

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	unavailable := Count{Scope: Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.Add(-time.Hour), End: at}, Shared: &SharedBoolean{Fact: "g"}}
	err := Validate(&Rule{Count: &unavailable}, at, Page{Limit: 1})
	if err != nil {
		if _, unsupported := errors.AsType[*UnsupportedError](err); !unsupported {
			t.Fatalf("expected an unsupported sibling fixture, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected an unsupported sibling fixture, got success")
	}

	for _, test := range []struct {
		name   string
		change func(*Count)
	}{
		{"event", func(c *Count) { c.Scope.Event = ""; c.Shared = &SharedBoolean{Fact: "g"} }},
		{"grain", func(c *Count) { c.Scope.Grain = "business-order"; c.Shared = &SharedBoolean{Fact: "g"} }},
		{"between", func(c *Count) { c.Test = CountTest{Operator: Between, Value: 1 << 63, Upper: 1} }},
		{"operator", func(c *Count) { c.Test = CountTest{Operator: 255, Value: 1 << 63} }},
		{"minimum", func(c *Count) { c.Minimum = 1 << 63 }},
		{"period", func(c *Count) { c.Scope.Start = c.Scope.End; c.Shared = &SharedBoolean{Fact: "g"} }},
		{"period precision", func(c *Count) { c.Scope.Start = c.Scope.End.Add(time.Nanosecond); c.Scope.End = c.Scope.Start }},
		{"qualification", func(c *Count) { c.Where = &state.Where{Operator: state.OpAnd}; c.Shared = &SharedBoolean{Fact: "g"} }},
		{"currency", func(c *Count) { c.Currency = &NativeCurrency{}; c.Where = &state.Where{Operator: state.OpAnd} }},
	} {
		for _, reversed := range []bool{false, true} {
			for _, nanos := range []time.Duration{0, time.Nanosecond} {
				t.Run(fmt.Sprintf("%s/reversed=%v/T+%s", test.name, reversed, nanos), func(t *testing.T) {
					c := Count{Scope: Scope{Event: "Purchase", Grain: "registered-message",
						Start: at.Add(-time.Hour), End: at}}
					test.change(&c)
					children := []Rule{{Count: &unavailable}, {Count: &c}}
					if reversed {
						children[0], children[1] = children[1], children[0]
					}
					err := Validate(&Rule{Operator: state.OpAnd, Children: children}, at.Add(nanos), Page{Limit: 1})
					if err != nil {
						if _, unsupported := errors.AsType[*UnsupportedError](err); unsupported {
							t.Fatalf("expected invalid canonical shape before unavailable capability, got %v", err)
						}
						return
					}
					t.Fatal("expected invalid canonical input, got success")
				})
			}
		}
	}

	for _, size := range []int{100, 101} {
		root := Rule{Operator: state.OpAnd, Children: make([]Rule, size)}
		for i := range root.Children {
			root.Children[i] = Rule{Count: &unavailable}
		}
		err := Validate(&root, at, Page{Limit: 1})
		if err != nil {
			_, unsupported := errors.AsType[*UnsupportedError](err)
			if unsupported != (size == 100) {
				t.Fatalf("expected unsupported=%v for %d descendants, got %v", size == 100, size, err)
			}
			continue
		}
		t.Fatal("expected canonical budget or capability error, got success")
	}

}

// TestValidateProfileRecord checks the leaf union and mixed structural limits.
func TestValidateProfileRecord(t *testing.T) {

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	record := Rule{Record: &state.WhereCondition{Property: []string{"country"}, Operator: state.OpIs,
		Values: []any{"IT"}}}
	count := Rule{Count: &Count{Scope: Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Test: CountTest{Operator: Equal}}}

	for _, rule := range []Rule{
		{Record: record.Record, Count: count.Count},
		{Record: record.Record, Children: []Rule{count}},
		{Record: &state.WhereCondition{Operator: state.OpExists}},
		{Record: &state.WhereCondition{Property: []string{"country"}, Operator: -1}},
	} {
		err := Validate(&rule, at, Page{Limit: 1})
		if err == nil {
			t.Fatal("expected invalid Record leaf rejection, got success")
		}
	}

	rule := record
	for range 4 {
		rule = Rule{Operator: state.OpOr, Children: []Rule{rule}}
	}
	err := Validate(&rule, at, Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected four Record group levels, got %v", err)
	}
	rule = Rule{Operator: state.OpAnd, Children: []Rule{rule, count}}
	err = Validate(&rule, at, Page{Limit: 1})
	if err == nil {
		t.Fatal("expected fifth mixed level rejection, got success")
	}

	rule = Rule{Operator: state.OpAnd, Children: make([]Rule, 100)}
	for i := range rule.Children {
		rule.Children[i] = record
		if i%2 == 0 {
			rule.Children[i] = count
		}
	}
	err = Validate(&rule, at, Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected 100 mixed descendant nodes, got %v", err)
	}
	rule.Children = append(rule.Children, record)
	err = Validate(&rule, at, Page{Limit: 1})
	if err == nil {
		t.Fatal("expected 101 mixed descendant rejection, got success")
	}

}
