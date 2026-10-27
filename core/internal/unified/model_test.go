// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
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
		{"unknown grain", func(c *Count) { c.Scope.Grain = "business-order" }, true},
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
