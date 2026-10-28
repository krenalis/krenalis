// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestPrepareUnifiedProfileRecord preserves the public Record input and tree.
func TestPrepareUnifiedProfileRecord(t *testing.T) {

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
	rule, err := prepareUnifiedRule(unifiedRule{Profile: &filter}, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected prepared Unified tree, got %v", err)
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

	for _, invalid := range []unifiedRule{
		{Profile: &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "absent", Operator: OpExists}}}},
		{Profile: &Filter{Operator: OpAnd, Rules: []FilterRule{
			&FilterCondition{Property: "country", Operator: OpIsNull}}}},
		{Profile: (*Filter)(nil)},
		{Profile: (*FilterCondition)(nil)},
		{Profile: &FilterCondition{Property: "absent", Operator: OpExists}},
		{Profile: &FilterCondition{Property: "country", Operator: OpIsNull}},
		{Profile: &filter, Count: &unifiedCount{}},
		{Profile: &filter, Operator: OpAnd},
		{Operator: "invalid", Rules: []unifiedRule{{Profile: &filter}}},
	} {
		nodes := 0
		_, err := prepareUnifiedRule(invalid, schema, 0, &nodes)
		if err == nil {
			t.Fatal("expected invalid internal request rejection, got success")
		}
	}

}

// TestPrepareUnifiedProfileLimits counts explicit Record groups and Count.
func TestPrepareUnifiedProfileLimits(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{
		&FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}}}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	count := &unifiedCount{Scope: unified.Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Test: unified.CountTest{Operator: unified.Equal}}
	check := func(input unifiedRule, valid bool) {
		t.Helper()
		nodes := 0
		rule, err := prepareUnifiedRule(input, schema, 0, &nodes)
		if err != nil {
			t.Fatalf("expected bounded input preparation, got %v", err)
		}
		err = unified.Validate(&rule, at, unified.Page{Limit: 1})
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
		check(unifiedRule{Profile: filter}, true)
		check(unifiedRule{Operator: OpAnd, Rules: []unifiedRule{{Profile: filter}, {Count: count}}}, levels < 4)
		filter = &Filter{Operator: OpOr, Rules: []FilterRule{filter}}
	}

	for _, n := range []int{98, 99, 100} {
		filter := &Filter{Operator: OpAnd, Rules: make([]FilterRule, n)}
		for i := range filter.Rules {
			filter.Rules[i] = &FilterCondition{Property: "country", Operator: OpIs, Values: filterValues("IT")}
		}
		check(unifiedRule{Profile: filter}, true)
		check(unifiedRule{Operator: OpAnd, Rules: []unifiedRule{{Profile: filter}, {Count: count}}}, n == 98)
	}

}

// TestPrepareUnifiedProfileCondition admits Record and Count as sibling leaves.
func TestPrepareUnifiedProfileCondition(t *testing.T) {

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
	rule, err := prepareUnifiedRule(unifiedRule{Profile: condition}, schema, 0, &nodes)
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
	count := unifiedRule{Count: &unifiedCount{Scope: unified.Scope{Event: "Purchase", Grain: "registered-message",
		Start: at.AddDate(0, -1, 0), End: at}, Test: unified.CountTest{Operator: unified.Equal}}}
	root := unifiedRule{Operator: OpAnd, Rules: []unifiedRule{{Profile: condition}, count}}
	for levels := 1; levels <= 5; levels++ {
		nodes := 0
		prepared, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if levels <= 4 {
				t.Fatalf("expected %d mixed group levels, got %v", levels, err)
			}
			break
		}
		err = unified.Validate(&prepared, at, unified.Page{Limit: 1})
		if err != nil {
			if levels <= 4 {
				t.Fatalf("expected %d mixed group levels, got %v", levels, err)
			}
		}
		if levels == 5 && err == nil {
			t.Fatal("expected fifth mixed group rejection, got success")
		}
		root = unifiedRule{Operator: OpOr, Rules: []unifiedRule{root}}
	}
	for _, size := range []int{99, 100} {
		root := unifiedRule{Operator: OpAnd, Rules: make([]unifiedRule, size+1)}
		for i := range size {
			root.Rules[i] = unifiedRule{Profile: condition}
		}
		root.Rules[size] = count
		nodes := 0
		prepared, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if size == 99 {
				t.Fatalf("expected 99 Record leaves plus Count, got %v", err)
			}
			continue
		}
		err = unified.Validate(&prepared, at, unified.Page{Limit: 1})
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
