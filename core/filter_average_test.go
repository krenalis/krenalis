// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const averageConditionJSON = `{"average":{"source":{"kind":"recordedEvents","name":"Order Completed",` +
	`"grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},` +
	`"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},` +
	`"nativeCurrency":"EUR","minimum":2},"operator":"is greater than or equal to","values":[100]}`

// TestFilterAverageBudget preserves the main-tree budget and Record boundary.
func TestFilterAverageBudget(t *testing.T) {
	var condition FilterAverageCondition
	err := json.Unmarshal([]byte(averageConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected Average fixture, got %v", err)
	}
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for _, size := range []int{100, 101} {
		root := &Filter{Operator: OpAnd, Rules: make([]FilterRule, size)}
		for i := range root.Rules {
			root.Rules[i] = &condition
		}
		if !root.RequiresUnified() || root.HasCount() {
			t.Fatal("expected Unified requirement without a Count, got incorrect detection")
		}
		nodes := 0
		_, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if size == 100 {
				t.Fatalf("expected 100 Average descendants, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected 101 descendants rejected, got success")
		}
	}
	var root FilterRule = &condition
	for depth := 1; depth <= 5; depth++ {
		root = &Filter{Operator: OpAnd, Rules: []FilterRule{root}}
		nodes := 0
		_, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if depth <= 4 {
				t.Fatalf("expected up to four groups, got %v", err)
			}
			continue
		}
		if depth == 5 {
			t.Fatal("expected fifth group rejected, got success")
		}
	}
	_, _, err = prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{&condition}}, schema)
	if err == nil {
		t.Fatal("expected Average rejected in Record filtering, got success")
	}
}

// TestFilterAverageCodec keeps the public subject strict and unmodified.
func TestFilterAverageCodec(t *testing.T) {
	var root Filter
	err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+averageConditionJSON+`]}`), &root)
	if err != nil {
		t.Fatalf("expected Average filter decoded, got %v", err)
	}
	before, err := json.Marshal(&root)
	if err != nil {
		t.Fatalf("expected Average filter encoded, got %v", err)
	}
	nodes := 0
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	prepared, err := prepareUnifiedRule(&root, schema, 0, &nodes)
	if err != nil {
		t.Fatalf("expected Average prepared, got %v", err)
	}
	if prepared.Children[0].Average == nil || prepared.Children[0].Average.Minimum != 2 {
		t.Fatalf("expected Average with minimum 2, got %+v", prepared)
	}
	after, err := json.Marshal(&root)
	if err != nil {
		t.Fatalf("expected Average encoding after preparation, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged filter %s, got %s", before, after)
	}
	for _, input := range []string{
		strings.Replace(averageConditionJSON, `"average":{`, `"sum":{},"average":{`, 1),
		strings.Replace(averageConditionJSON, `"average":{`, `"count":{},"average":{`, 1),
		strings.Replace(averageConditionJSON, `"average":{`, `"property":"country","average":{`, 1),
		strings.Replace(averageConditionJSON, `"average":{`, `"average":{},"average":{`, 1),
		strings.Replace(averageConditionJSON, `"average":{`, `"average":{"unknown":true,`, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`, `"minimum":null`, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`, `"minimum":2,"where":`+averageConditionJSON, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`,
			`"minimum":2,"where":{"operator":"and","rules":[`+averageConditionJSON+`]}`, 1),
	} {
		var condition FilterAverageCondition
		err := json.Unmarshal([]byte(input), &condition)
		if err == nil {
			t.Fatalf("expected malformed Average rejected, got %s", input)
		}
	}
}

// TestFilterAverageInvalidBeforeUnsupported preserves full-filter validity.
func TestFilterAverageInvalidBeforeUnsupported(t *testing.T) {
	schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
	for _, invalid := range []string{
		strings.Replace(averageConditionJSON, `,"minimum":2`, ``, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`, `"minimum":0`, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`, `"minimum":-1`, 1),
		strings.Replace(averageConditionJSON, `"minimum":2`, `"minimum":"2"`, 1),
		strings.Replace(averageConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, 1),
		strings.Replace(averageConditionJSON, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1),
		strings.Replace(averageConditionJSON, `"values":[100]`, `"values":[1e76]`, 1),
	} {
		unavailable := strings.Replace(averageConditionJSON, `"values":[100]`, `"values":[0.00001]`, 1)
		for _, input := range []string{invalid + "," + unavailable, unavailable + "," + invalid} {
			var root Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+input+`]}`), &root)
			if err != nil {
				t.Fatalf("expected syntax decoded before validation, got %v", err)
			}
			nodes := 0
			_, err = prepareUnifiedRule(&root, schema, 0, &nodes)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
					t.Fatalf("expected invalidity before unavailable precision, got %v", err)
				}
				continue
			}
			t.Fatal("expected invalid Average rejected, got success")
		}
	}
}

// TestFilterAverageThresholds distinguishes exact supported values from limits.
func TestFilterAverageThresholds(t *testing.T) {
	for _, test := range []struct {
		value       string
		unsupported bool
	}{
		{"100", false}, {"-100", false}, {"0.0001", false}, {"0.0001000", false},
		{"90071992547409.1234", false}, {"1e14", false}, {"-1e14", false},
		{"1e75", false}, {"-1e75", false}, {"0.00001", true}, {"-0.00001", true},
	} {
		t.Run(test.value, func(t *testing.T) {
			var condition FilterAverageCondition
			err := json.Unmarshal([]byte(strings.Replace(averageConditionJSON,
				`"values":[100]`, `"values":[`+test.value+`]`, 1)), &condition)
			if err != nil {
				t.Fatalf("expected exact literal decoded, got %v", err)
			}
			_, err = prepareFilterAverage(&condition, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported || !test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			if test.unsupported {
				t.Fatal("expected finer literal rejected without rounding, got success")
			}
		})
	}
}
