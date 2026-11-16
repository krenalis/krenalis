// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const experienceConditionJSON = `{"experience":{"base":{"source":{"kind":"recordedEvents",` +
	`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"fullRelevantHistory",` +
	`"history":{"definition":"registered-queryable-events","meaningRevision":"1"}}},` +
	`"select":{"kind":"last"},"verify":{"operator":"and","rules":[` +
	`{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}]}}}`

// TestFilterExperienceBodies checks both local budgets independently.
func TestFilterExperienceBodies(t *testing.T) {
	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(experienceConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected experience fixture, got %v", err)
	}
	leaf := condition.Experience.Verify.Rules[0]
	for _, verify := range []bool{false, true} {
		for _, test := range []struct{ size, depth int }{{1, 1}, {1, 2}, {1, 3}, {100, 1}, {101, 1}} {
			body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.size)}
			for i := range body.Rules {
				body.Rules[i] = leaf
			}
			for depth := 1; depth < test.depth; depth++ {
				body = &Filter{Operator: OpAnd, Rules: []FilterRule{body}}
			}
			modified := condition
			if verify {
				modified.Experience.Verify = body
			} else {
				modified.Experience.Base.Where = body
			}
			_, err := prepareFilterExperience(&modified, nil)
			invalid := test.depth == 3 || test.size == 101
			if err != nil {
				if !invalid {
					t.Fatalf("expected local size=%d depth=%d allowed, got %v", test.size, test.depth, err)
				}
				continue
			}
			if invalid {
				t.Fatalf("expected local size=%d depth=%d rejected, got success", test.size, test.depth)
			}
		}
	}
}

// TestFilterExperienceBudget preserves the independent main-tree budget.
func TestFilterExperienceBudget(t *testing.T) {

	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(experienceConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected experience fixture, got %v", err)
	}

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for _, size := range []int{100, 101} {
		root := &Filter{Operator: OpAnd, Rules: make([]FilterRule, size)}
		for i := range root.Rules {
			root.Rules[i] = &condition
		}
		nodes := 0
		_, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if size == 100 {
				t.Fatalf("expected 100 descendants, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected descendant budget error, got success")
		}
	}

	var rule FilterRule = &condition
	for depth := 1; depth <= 5; depth++ {
		rule = &Filter{Operator: OpAnd, Rules: []FilterRule{rule}}
		nodes := 0
		_, err := prepareUnifiedRule(rule, schema, 0, &nodes)
		if err != nil {
			if depth <= 4 {
				t.Fatalf("expected four groups allowed, got %v", err)
			}
			continue
		}
		if depth == 5 {
			t.Fatal("expected depth error, got success")
		}
	}

}

// TestFilterExperienceCodec checks round-tripping and the Record boundary.
func TestFilterExperienceCodec(t *testing.T) {

	var filter Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+experienceConditionJSON+`]}`), &filter)
	if err != nil {
		t.Fatalf("expected experience decoded, got %v", err)
	}
	if !filter.RequiresUnified() {
		t.Fatal("expected Unified requirement, got false")
	}
	before, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected encoded filter, got %v", err)
	}
	nodes := 0
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	prepared, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected experience prepared, got %v", err)
	}
	if prepared.Children[0].Experience == nil {
		t.Fatal("expected canonical experience, got no experience")
	}
	after, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("expected encoding after preparation, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged encoding %s, got %s", before, after)
	}

	_, _, err = prepareFilter(&filter, schema)
	if err == nil {
		t.Fatal("expected experience rejected in Record pipelines, got success")
	}

}

// TestFilterExperienceFirst preserves the selector through public preparation.
func TestFilterExperienceFirst(t *testing.T) {
	var condition FilterExperienceCondition
	input := strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"first"`, 1)
	err := json.Unmarshal([]byte(input), &condition)
	if err != nil {
		t.Fatalf("expected First decoded, got %v", err)
	}
	rule, err := prepareFilterExperience(&condition, nil)
	if err != nil {
		t.Fatalf("expected First prepared, got %v", err)
	}
	if rule.Experience == nil || !rule.Experience.First {
		t.Fatalf("expected canonical First selection, got %+v", rule)
	}
	encoded, err := json.Marshal(&condition)
	if err != nil {
		t.Fatalf("expected First encoded, got %v", err)
	}
	if !strings.Contains(string(encoded), `"select":{"kind":"first"}`) {
		t.Fatalf("expected First round-trip, got %s", encoded)
	}
}

// TestFilterExperienceInvalidBeforeUnsupported checks both sibling orders.
func TestFilterExperienceInvalidBeforeUnsupported(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	unavailable := strings.Replace(experienceConditionJSON, `"base":{`, `"base":{`+
		`"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},`+
		`"nativeCurrency":"EUR","where":{"operator":"and","rules":[`+
		`{"property":"properties.currency","operator":"is null"}]},`, 1)
	var sibling FilterExperienceCondition
	err := json.Unmarshal([]byte(unavailable), &sibling)
	if err != nil {
		t.Fatalf("expected valid unavailable sibling decoded, got %v", err)
	}
	_, err = prepareFilterExperience(&sibling, nil)
	if err != nil {
		if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
			t.Fatalf("expected unavailable sibling, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected unavailable sibling, got success")
	}

	for _, test := range []struct{ name, input string }{
		{"empty verify", strings.Replace(experienceConditionJSON, `"rules":[{"property":"properties.order_id","operator":"starts with","values":["WEB-"]}]`, `"rules":[]`, 1)},
		{"invalid selector", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"random"`, 1)},
		{"last position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"last","position":1`, 1)},
		{"zero position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"nth","position":0`, 1)},
		{"eleventh position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"nth","position":11`, 1)},
		{"missing position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"nth"`, 1)},
		{"negative position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"nth","position":-1`, 1)},
		{"fractional position", strings.Replace(experienceConditionJSON, `"kind":"last"`, `"kind":"nth","position":1.5`, 1)},
		{"bad property", strings.Replace(experienceConditionJSON, `properties.order_id`, `missing`, 1)},
		{"bad Record operator", strings.Replace(experienceConditionJSON, `starts with`, `not an operator`, 1)},
	} {
		for _, rules := range []string{unavailable + "," + test.input, test.input + "," + unavailable} {
			t.Run(test.name, func(t *testing.T) {
				var filter Filter
				err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+rules+`]}`), &filter)
				if err != nil {
					t.Fatalf("expected syntax decoded before validation, got %v", err)
				}
				nodes := 0
				_, err = prepareUnifiedRule(&filter, schema, 0, &nodes)
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalidity before unimplemented capability, got %v", err)
					}
					return
				}
				t.Fatal("expected invalid filter rejected, got success")
			})
		}
	}

}

// TestFilterExperienceNth preserves each admitted ordinal through preparation.
func TestFilterExperienceNth(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for position := 1; position <= 10; position++ {
		selector := fmt.Sprintf(`{"kind":"nth","position":%d}`, position)
		input := strings.Replace(experienceConditionJSON, `{"kind":"last"}`, selector, 1)
		var condition FilterExperienceCondition
		err := json.Unmarshal([]byte(input), &condition)
		if err != nil {
			t.Fatalf("expected valid selector decoded, got %v", err)
		}
		rule, err := prepareFilterExperience(&condition, nil)
		if err != nil {
			t.Fatalf("expected Nth prepared, got %v", err)
		}
		if rule.Experience == nil || rule.Experience.First || rule.Experience.Nth != position {
			t.Fatalf("expected Nth(%d), got %+v", position, rule.Experience)
		}
		encoded, err := json.Marshal(&condition)
		if err != nil {
			t.Fatalf("expected Nth encoded, got %v", err)
		}
		if string(encoded) != input {
			t.Fatalf("expected Nth round-trip %s, got %s", input, encoded)
		}
		filter := &Filter{Operator: OpAnd, Rules: []FilterRule{&condition}}
		_, _, err = prepareFilter(filter, schema)
		if err == nil {
			t.Fatal("expected Nth rejected in Record pipelines, got success")
		}
	}
}

// TestFilterExperienceStrict rejects mixed subjects and malformed local bodies.
func TestFilterExperienceStrict(t *testing.T) {
	for _, input := range []string{
		`{"experience":{}}`,
		`{"experience":{"base":{},"select":{"kind":"last"},"verify":null}}`,
		strings.Replace(experienceConditionJSON, `"experience":{`, `"operator":"is","experience":{`, 1),
		strings.Replace(experienceConditionJSON, `"experience":{`, `"values":[],"experience":{`, 1),
		strings.Replace(experienceConditionJSON, `"experience":{`, `"count":{},"experience":{`, 1),
		strings.Replace(experienceConditionJSON, `"experience":{`, `"experience":{},"experience":{`, 1),
		strings.Replace(experienceConditionJSON, `"base":{`, `"base":{"minimum":1,`, 1),
		strings.Replace(experienceConditionJSON, `"select":{`, `"select":{"unknown":true,`, 1),
		strings.Replace(experienceConditionJSON, `"verify":{`, `"verify":{"defaultCalendar":{},`, 1),
		strings.Replace(experienceConditionJSON, `"rules":[`, `"rules":[`+averageConditionJSON+`,`, 1),
	} {
		var condition FilterExperienceCondition
		err := json.Unmarshal([]byte(input), &condition)
		if err == nil {
			t.Fatalf("expected strict decoding error, got %s", input)
		}
	}
}
