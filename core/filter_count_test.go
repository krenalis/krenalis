// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/schemas"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const countConditionJSON = `{"count":{"source":{"kind":"recordedEvents","name":"Purchase",` +
	`"grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
	`"duration":{"value":30,"unit":"day"}}},"operator":"is","values":[0]}`

// TestFilterCountDecoding enforces an exclusive rule shape and bounded Record
// bodies independently of member order, without selecting a backend.
func TestFilterCountDecoding(t *testing.T) {

	for _, test := range []struct{ name, condition string }{
		{"null", `{"count":null,"operator":"is","values":[0]}`},
		{"root discriminator", `{"count":{},"property":"country","operator":"is","values":[0]}`},
		{"group discriminator", `{"count":{},"operator":"and","rules":[],"values":[0]}`},
		{"missing operator", strings.Replace(countConditionJSON, `,"operator":"is"`, ``, 1)},
		{"missing values", strings.Replace(countConditionJSON, `,"values":[0]`, ``, 1)},
		{"unknown subject member", strings.Replace(countConditionJSON, `"source":`, `"unexpected":0,"source":`, 1)},
		{"unknown source member", strings.Replace(countConditionJSON, `"kind":"recordedEvents"`, `"kind":"recordedEvents","x":0`, 1)},
		{"unknown period member", strings.Replace(countConditionJSON, `"kind":"elapsedLookback"`, `"kind":"elapsedLookback","x":0`, 1)},
		{"unknown duration member", strings.Replace(countConditionJSON, `"unit":"day"`, `"unit":"day","x":0`, 1)},
		{"duplicate count", strings.Replace(countConditionJSON, `"count":`, `"count":{},"count":`, 1)},
		{"duplicate source", strings.Replace(countConditionJSON, `"source":`, `"source":{},"source":`, 1)},
		{"duplicate duration", strings.Replace(countConditionJSON, `"unit":"day"`, `"unit":"day","unit":"hour"`, 1)},
		{"null source", strings.Replace(countConditionJSON, `{"kind":"recordedEvents","name":"Purchase","grain":"registered-message"}`, `null`, 1)},
		{"null duration", strings.Replace(countConditionJSON, `{"value":30,"unit":"day"}`, `null`, 1)},
		{"missing source", `{"count":{"period":{"kind":"elapsedLookback"}},"operator":"is","values":[0]}`},
		{"missing period", `{"count":{"source":{}},"operator":"is","values":[0]}`},
		{"Count inside where", `{"count":{"source":{},"period":{},"where":{"operator":"and","rules":[` + countConditionJSON + `]}},"operator":"is","values":[0]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.condition+`]}`), &filter)
			if err == nil {
				t.Fatal("expected malformed Count document rejection, got success")
			}
		})
	}

	var root Filter
	err := json.Unmarshal([]byte(countConditionJSON), &root)
	if err == nil {
		t.Fatal("expected a root group requirement, got success")
	}

	var leaf FilterCountCondition
	err = json.Unmarshal([]byte(`{"values":[0],"operator":"is","count":{"period":`+
		`{"duration":{"unit":"day","value":30},"kind":"elapsedLookback"},`+
		`"source":{"grain":"registered-message","name":"Purchase","kind":"recordedEvents"}}}`), &leaf)
	if err != nil {
		t.Fatalf("expected field-order-independent Count decoding, got %v", err)
	}

}

// TestFilterCountEncoding keeps syntax distinct from semantic admission.
func TestFilterCountEncoding(t *testing.T) {

	for _, rule := range []FilterRule{(*FilterCountCondition)(nil), nil} {
		_, err := json.Marshal(&Filter{Operator: OpAnd, Rules: []FilterRule{rule}})
		if err == nil {
			t.Fatal("expected nil rule encoding rejection, got success")
		}
	}

	condition := countFilterCondition()
	condition.Operator = "future operator"
	condition.Values = []json.Value{json.Value(`9223372036854775808`)}
	data, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("expected syntax encoding without capability validation, got %v", err)
	}
	var decoded FilterCountCondition
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("expected syntax round-trip, got %v", err)
	}
	if decoded.Operator != condition.Operator || string(decoded.Values[0]) != string(condition.Values[0]) {
		t.Fatalf("expected exact operator and literal round-trip, got %+v", decoded)
	}

}

// TestFilterCountLimits keeps main and occurrence-body budgets independent.
func TestFilterCountLimits(t *testing.T) {
	for _, test := range []struct {
		main, body, nodes int
		valid             bool
	}{
		{4, 2, 99, true}, {4, 1, 100, true}, {5, 2, 99, false}, {4, 3, 1, false}, {1, 1, 101, false},
	} {
		t.Run(fmt.Sprintf("%d/%d/%d", test.main, test.body, test.nodes), func(t *testing.T) {
			count := countFilterCondition()
			body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.nodes)}
			for i := range body.Rules {
				body.Rules[i] = &FilterCondition{Property: "properties.currency",
					Operator: OpIs, Values: filterValues("EUR")}
			}
			for level := 1; level < test.body; level++ {
				body = &Filter{Operator: OpAnd, Rules: []FilterRule{body}}
			}
			count.Count.Where = body
			root := &Filter{Operator: OpAnd, Rules: []FilterRule{count}}
			for level := 1; level < test.main; level++ {
				root = &Filter{Operator: OpAnd, Rules: []FilterRule{root}}
			}
			data, err := json.Marshal(root)
			if err != nil {
				t.Fatalf("expected syntax encoding, got %v", err)
			}
			var restored Filter
			err = json.Unmarshal(data, &restored)
			if err != nil {
				if test.valid {
					t.Fatalf("expected independent main/body budgets, got %v", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("expected exceeded budget rejection, got success")
			}
		})
	}
}

// TestFilterCountPeriods checks exact fixed units, conflicting forms and
// unavailable duration ranges without truncation or Float conversion.
func TestFilterCountPeriods(t *testing.T) {
	for _, test := range []struct {
		unit string
		want time.Duration
	}{
		{"microsecond", time.Microsecond}, {"millisecond", time.Millisecond}, {"second", time.Second},
		{"minute", time.Minute}, {"hour", time.Hour}, {"day", 24 * time.Hour}, {"week", 7 * 24 * time.Hour},
	} {
		condition := countFilterCondition()
		condition.Count.Period.Duration = &FilterDuration{Value: json.Value(`1`), Unit: test.unit}
		rule, err := prepareFilterCount(condition)
		if err != nil {
			t.Fatalf("expected fixed unit %s, got %v", test.unit, err)
		}
		if rule.Count.Scope.Elapsed.Older != test.want {
			t.Fatalf("expected elapsed duration %s, got %s", test.want, rule.Count.Scope.Elapsed.Older)
		}
	}
	for _, test := range []struct {
		name        string
		change      func(*FilterCountCondition)
		unsupported bool
	}{
		{"zero duration", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`0`) }, false},
		{"negative duration", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`-1`) }, false},
		{"months", func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "month" }, false},
		{"years", func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "year" }, false},
		{"duration overflow", func(c *FilterCountCondition) { c.Count.Period.Duration.Value = json.Value(`106752`) }, true},
		{"missing grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "" }, false},
		{"other grain", func(c *FilterCountCondition) { c.Count.Source.Grain = "business-order" }, true},
		{"unknown source", func(c *FilterCountCondition) { c.Count.Source.Kind = "SQL" }, false},
		{"missing instant", func(c *FilterCountCondition) { c.Count.Period = FilterPeriod{Kind: "absoluteHalfOpen"} }, false},
		{"conflicting fields", func(c *FilterCountCondition) { c.Count.Period.Older = &FilterDuration{} }, false},
		{"empty band", func(c *FilterCountCondition) {
			c.Count.Period = FilterPeriod{Kind: "elapsedAgoBand",
				Older: &FilterDuration{Value: json.Value(`1`), Unit: "day"},
				Newer: &FilterDuration{Value: json.Value(`24`), Unit: "hour"}}
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			condition := countFilterCondition()
			test.change(condition)
			rule, err := prepareFilterCount(condition)
			if err == nil {
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected period, got success")
		})
	}
}

// TestFilterCountRoundTrip preserves mixed groups, operands and relative time.
func TestFilterCountRoundTrip(t *testing.T) {

	data := `{"operator":"and","rules":[{"property":"country","operator":"is","values":["IT"]},` +
		countConditionJSON + `]}`
	input := []byte(data)
	var filter Filter
	err := json.Unmarshal(input, &filter)
	if err != nil {
		t.Fatalf("expected mixed filter decoding, got %v", err)
	}
	clear(input)
	encoded, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected mixed filter encoding, got %v", err)
	}
	if string(encoded) != data {
		t.Fatalf("expected exact public round-trip %s, got %s", data, encoded)
	}

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	nodes := 0
	rule, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected mixed preparation, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	err = unified.Validate(&rule, at, unified.Page{Limit: 1})
	if err != nil {
		t.Fatalf("expected admitted mixed filter, got %v", err)
	}
	if len(rule.Children) != 2 || rule.Children[1].Count.Scope.Elapsed.Older != 30*24*time.Hour ||
		!rule.Children[1].Count.Scope.Start.IsZero() || nodes != 3 {
		t.Fatalf("expected a Record and unresolved relative Count, got %+v", rule)
	}

	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected unchanged input encoding, got %v", err)
	}
	if !bytes.Equal(encoded, after) {
		t.Fatalf("expected unchanged input %s, got %s", encoded, after)
	}

	for _, recordSchema := range []types.Type{schema, schemas.Event} {
		_, _, err = prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{filter.Rules[1]}}, recordSchema)
		if err == nil {
			t.Fatal("expected Count rejection in Record filtering, got success")
		}
	}

}

// TestPrepareFilterCount checks the closed comparisons and exact integer input.
func TestPrepareFilterCount(t *testing.T) {

	for _, op := range operators {
		condition := countFilterCondition()
		condition.Operator = op
		if op == OpIsBetween {
			condition.Values = filterValues(0, 2)
		}
		_, err := prepareFilterCount(condition)
		allowed := op == OpIs || op == OpIsNot || op == OpIsLessThan || op == OpIsLessThanOrEqualTo ||
			op == OpIsGreaterThan || op == OpIsGreaterThanOrEqualTo || op == OpIsBetween
		if err != nil {
			if allowed {
				t.Fatalf("expected Count operator %q, got %v", op, err)
			}
			continue
		}
		if !allowed {
			t.Fatalf("expected Count operator %q rejected, got success", op)
		}
	}

	for _, test := range []struct {
		value       string
		unsupported bool
	}{
		{`-1`, false}, {`1.0`, false}, {`1e1`, false}, {`"1"`, false}, {`null`, false}, {`true`, false},
		{`9223372036854775808`, true}, {`18446744073709551616`, true},
	} {
		t.Run(test.value, func(t *testing.T) {
			condition := countFilterCondition()
			condition.Values = []json.Value{json.Value(test.value)}
			rule, err := prepareFilterCount(condition)
			if err == nil {
				err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			}
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected Count operand, got success")
		})
	}

	for _, literal := range []string{`-0`, `9007199254740993`, `9223372036854775807`} {
		condition := countFilterCondition()
		condition.Values = []json.Value{json.Value(literal)}
		rule, err := prepareFilterCount(condition)
		if err != nil {
			t.Fatalf("expected exact Count literal %s, got %v", literal, err)
		}
		if rule.Count.Test.Value > 0 && fmt.Sprint(rule.Count.Test.Value) != literal {
			t.Fatalf("expected exact integer %s, got %d", literal, rule.Count.Test.Value)
		}
	}

	for _, change := range []func(*FilterCountCondition){
		func(c *FilterCountCondition) { c.Values = nil },
		func(c *FilterCountCondition) { c.Values = filterValues(0, 1) },
		func(c *FilterCountCondition) { c.Operator = OpIsBetween },
		func(c *FilterCountCondition) { c.Count.Minimum = json.Value(`0`) },
		func(c *FilterCountCondition) { c.Count.Source.Grain = "" },
		func(c *FilterCountCondition) { c.Count.Period.Duration.Unit = "month" },
		func(c *FilterCountCondition) { c.Count.Period.Older = &FilterDuration{} },
	} {
		condition := countFilterCondition()
		change(condition)
		_, err := prepareFilterCount(condition)
		if err == nil {
			t.Fatal("expected invalid Count rejection, got success")
		}
	}

}

func countFilterCondition() *FilterCountCondition {
	return &FilterCountCondition{Count: FilterCount{
		Source: FilterCountSource{Kind: "recordedEvents", Name: "Purchase", Grain: "registered-message"},
		Period: FilterPeriod{Kind: "elapsedLookback", Duration: &FilterDuration{Value: json.Value(`31`), Unit: "day"}},
	}, Operator: OpIs, Values: filterValues(0)}
}
