// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestPrepareBehavioralProfileRecord preserves the public Record input and
// tree.
func TestPrepareBehavioralProfileRecord(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "country", Type: types.String(), ReadOptional: true},
		{Name: "account", Type: types.Int(64), ReadOptional: true},
		{Name: "details", Type: types.JSON(), ReadOptional: true},
	})
	var filter Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[
        {"property":"account","operator":"is","values":[9007199254740993]},
        {"operator":"or","rules":[
            {"property":"country","operator":"is not","values":["IT"]},
            {"property":"details.currency","operator":"is null"}]}]}`), &filter)
	if err != nil {
		t.Fatalf("expected existing Record decoding, got %v", err)
	}
	before, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected existing Record encoding, got %v", err)
	}
	expected, _, err := prepareFilter(&filter, schema)
	if err != nil {
		t.Fatalf("expected prepared Record filter, got %v", err)
	}

	nodes := 0
	rule, err := prepareBehavioralRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected prepared behavioral filter tree, got %v", err)
	}
	if rule.Operator != expected.Operator || len(rule.Children) != 2 ||
		!reflect.DeepEqual(rule.Children[0].Record, expected.Rules[0]) ||
		!reflect.DeepEqual(rule.Children[1].Children[1].Record, expected.Rules[1].(*state.Where).Rules[1]) {
		t.Fatalf("expected unchanged prepared conditions and groups, got %+v", rule)
	}

	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected unchanged Record encoding, got %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("expected unchanged input %s, got %s", before, after)
	}
	var restored Filter
	err = json.Unmarshal(after, &restored)
	if err != nil {
		t.Fatalf("expected existing Record round-trip, got %v", err)
	}
	prepared, _, err := prepareFilter(&restored, schema)
	if err != nil {
		t.Fatalf("expected prepared restored filter, got %v", err)
	}
	if !expected.Equal(prepared) {
		t.Fatal("expected existing Where equality, got different prepared inputs")
	}

	for _, invalid := range []FilterRule{
		&Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "absent", Operator: OpExists}}},
		&Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "country", Operator: OpIsNull}}},
		(*Filter)(nil),
		(*FilterCondition)(nil),
		(*FilterCountCondition)(nil),
		&FilterCondition{Property: "absent", Operator: OpExists},
		&FilterCondition{Property: "country", Operator: OpIsNull},
		&Filter{Operator: "invalid", Rules: []FilterRule{&filter}},
	} {
		nodes := 0
		_, err := prepareBehavioralRule(invalid, schema, 0, &nodes)
		if err == nil {
			t.Fatal("expected invalid internal request rejection, got success")
		}
	}

}

// TestPrepareBehavioralProfileLimits counts explicit Record groups and Count.
func TestPrepareBehavioralProfileLimits(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}}}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := countFilterCondition()
	check := func(input FilterRule, valid bool) {
		t.Helper()
		nodes := 0
		rule, err := prepareBehavioralRule(input, schema, 0, &nodes)
		if err != nil {
			if valid {
				t.Fatalf("expected bounded input preparation, got %v", err)
			}
			return
		}
		err = behavioral.Validate(&rule, at, behavioral.Page{Limit: 1})
		if err != nil {
			if valid {
				t.Fatalf("expected valid combined root limits, got %v", err)
			}
			return
		}
		if !valid {
			t.Fatal("expected combined root limit rejection, got success")
		}
	}

	for levels := 1; levels <= 4; levels++ {
		check(filter, true)
		check(&Filter{Operator: OpAnd, Rules: []FilterRule{filter, count}}, levels < 4)
		filter = &Filter{Operator: OpOr, Rules: []FilterRule{filter}}
	}

	for _, n := range []int{98, 99, 100} {
		filter := &Filter{Operator: OpAnd, Rules: make([]FilterRule, n)}
		for i := range filter.Rules {
			filter.Rules[i] = &FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}
		}
		check(filter, true)
		check(&Filter{Operator: OpAnd, Rules: []FilterRule{filter, count}}, n == 98)
	}

}

// TestPrepareBehavioralProfileCondition admits Record and Count as sibling
// leaves.
func TestPrepareBehavioralProfileCondition(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	condition := &FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}
	before, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("expected Record condition encoding, got %v", err)
	}
	expected, _, err := prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{condition}}, schema)
	if err != nil {
		t.Fatalf("expected existing Record preparation, got %v", err)
	}
	nodes := 0
	rule, err := prepareBehavioralRule(condition, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected bare Record leaf preparation, got %v", err)
	}
	if rule.Record == nil || len(rule.Children) != 0 || !reflect.DeepEqual(rule.Record, expected.Rules[0]) {
		t.Fatalf("expected one unchanged prepared Record leaf, got %+v", rule)
	}
	after, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("expected unchanged Record encoding, got %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("expected unchanged Record input %s, got %s", before, after)
	}

	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := countFilterCondition()
	root := &Filter{Operator: OpAnd, Rules: []FilterRule{condition, count}}
	for levels := 1; levels <= 5; levels++ {
		nodes := 0
		prepared, err := prepareBehavioralRule(root, schema, 0, &nodes)
		if err != nil {
			if levels <= 4 {
				t.Fatalf("expected %d mixed group levels, got %v", levels, err)
			}
			break
		}
		err = behavioral.Validate(&prepared, at, behavioral.Page{Limit: 1})
		if err != nil {
			if levels <= 4 {
				t.Fatalf("expected %d mixed group levels, got %v", levels, err)
			}
		}
		if levels == 5 && err == nil {
			t.Fatal("expected fifth mixed group rejection, got success")
		}
		root = &Filter{Operator: OpOr, Rules: []FilterRule{root}}
	}
	for _, size := range []int{99, 100} {
		root := &Filter{Operator: OpAnd, Rules: make([]FilterRule, size+1)}
		for i := range size {
			root.Rules[i] = condition
		}
		root.Rules[size] = count
		nodes := 0
		prepared, err := prepareBehavioralRule(root, schema, 0, &nodes)
		if err != nil {
			if size == 99 {
				t.Fatalf("expected 99 Record leaves plus Count, got %v", err)
			}
			continue
		}
		err = behavioral.Validate(&prepared, at, behavioral.Page{Limit: 1})
		if err != nil {
			if size == 99 {
				t.Fatalf("expected 99 Record leaves plus Count, got %v", err)
			}
			continue
		}
		if size == 100 {
			t.Fatal("expected 101 descendant rejection, got success")
		}
		if len(prepared.Children) != 100 || prepared.Children[0].Record == nil || prepared.Children[99].Count == nil {
			t.Fatalf("expected sibling Record/Count leaves without extra groups, got %+v", prepared)
		}
	}

}
