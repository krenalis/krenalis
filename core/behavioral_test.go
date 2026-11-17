// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/json"
)

// TestPrepareBehavioralRecordCount preserves the existing Record round-trip.
func TestPrepareBehavioralRecordCount(t *testing.T) {
	filter := &Filter{Operator: OpAnd, Rules: []FilterRule{&FilterCondition{Property: "properties.currency",
		Operator: OpIs, Values: []json.Value{json.Value(`"EUR"`)}}}}
	before, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("expected public filter encoding, got %v", err)
	}
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := behavioral.Scope{Event: "Purchase", Grain: "registered-message", Start: at.AddDate(0, -1, 0), End: at}
	rule, err := prepareBehavioralCount(scope, filter, behavioral.CountTest{Operator: behavioral.Equal})
	if err != nil {
		t.Fatalf("expected prepared internal Count, got %v", err)
	}
	condition := rule.Count.Where.Children[0].Record
	if string(condition.Values[0].(json.Value)) != `"EUR"` || len(condition.Property) != 2 {
		t.Fatalf("expected preserved typed Record operands and path, got %+v", condition)
	}
	err = behavioral.Validate(&rule, at, behavioral.Page{Limit: 10})
	if err != nil {
		t.Fatalf("expected admitted Count, got %v", err)
	}
	after, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("expected public filter encoding, got %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("expected public round-trip %s, got %s", before, after)
	}
	var restored Filter
	err = json.Unmarshal(before, &restored)
	if err != nil {
		t.Fatalf("expected existing public filter decoding, got %v", err)
	}
	roundtrip, err := prepareBehavioralCount(scope, &restored, behavioral.CountTest{Operator: behavioral.Equal})
	if err != nil {
		t.Fatalf("expected prepared round-trip, got %v", err)
	}
	if !reflect.DeepEqual(rule.Count.Where, roundtrip.Count.Where) {
		t.Fatal("expected identical prepared Record predicates, got different conditions")
	}
	filter.Rules = []FilterRule{&FilterCondition{Property: "noSuchField", Operator: OpIsTrue}}
	_, err = prepareBehavioralCount(scope, filter, behavioral.CountTest{})
	if err == nil {
		t.Fatal("expected existing schema validation error, got success")
	}
}
