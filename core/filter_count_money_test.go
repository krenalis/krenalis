// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const countMoneyConditionJSON = `{"count":{"source":{"kind":"recordedEvents","name":"Order Completed",` +
	`"grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},` +
	`"where":{"operator":"and","rules":[` + moneyConditionJSON + `]}},"operator":"is","values":[0]}`

// TestFilterCountMoneyAdmission covers operators, sources and metadata.
func TestFilterCountMoneyAdmission(t *testing.T) {
	for _, test := range []struct {
		name, input string
		invalid     bool
		unsupported bool
	}{
		{"money", countMoneyConditionJSON, false, false},
		{"less", strings.Replace(countMoneyConditionJSON, "is greater than", "is less than", 1), false, false},
		{"at most", strings.Replace(countMoneyConditionJSON, "is greater than", "is less than or equal to", 1), false, false},
		{"at least", strings.Replace(countMoneyConditionJSON, "is greater than", "is greater than or equal to", 1), false, false},
		{"equality", strings.Replace(countMoneyConditionJSON, "is greater than", "is", 1), false, true},
		{"difference", strings.Replace(countMoneyConditionJSON, "is greater than", "is not", 1), false, true},
		{"interval", strings.Replace(strings.Replace(countMoneyConditionJSON, "is greater than", "is between", 1), "[100]", "[50,100]", 1), false, true},
		{"wrong source", strings.Replace(countMoneyConditionJSON, "Order Completed", "Other", 1), true, false},
		{"page", strings.Replace(countMoneyConditionJSON, `"name":"Order Completed"`, `"type":"page","name":"Order Completed"`, 1), true, false},
		{"currency", strings.Replace(countMoneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, 1), true, false},
		{"revision", strings.Replace(countMoneyConditionJSON, `"meaningRevision":"1"`, `"meaningRevision":"2"`, 1), true, false},
		{"time", strings.Replace(countMoneyConditionJSON, moneyConditionJSON, `{"occurredIn":{"kind":"elapsedLookback","duration":{"value":1,"unit":"day"}}}`, 1), true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.input+`]}`), &filter)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected decoded input, got %v", err)
				}
				return
			}
			before, err := json.Marshal(&filter)
			if err != nil {
				t.Fatalf("expected filter encoding, got %v", err)
			}
			nodes := 0
			root, err := prepareUnifiedRule(&filter, types.Object([]types.Property{{Name: "country", Type: types.String()}}),
				0, &nodes)
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if test.invalid && unsupported || !test.invalid && (!test.unsupported || !unsupported) {
					t.Fatalf("expected invalid=%v unsupported=%v, got %v", test.invalid, test.unsupported, err)
				}
				return
			}
			if test.invalid || test.unsupported {
				t.Fatal("expected input rejected, got success")
			}
			err = unified.Validate(&root, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			if err != nil {
				t.Fatalf("expected admitted Count, got %v", err)
			}
			metadata, err := explainMoneyPredicates(root.Children[0].Count.Where, nil)
			if err != nil {
				t.Fatalf("expected monetary metadata, got %v", err)
			}
			if len(metadata) != 1 || !reflect.DeepEqual(metadata[0].Path, []int{0}) ||
				metadata[0].NativeCurrency != "EUR" || metadata[0].Requested.Operator == "" {
				t.Fatalf("expected where money metadata, got %+v", metadata)
			}
			after, err := json.Marshal(&filter)
			if err != nil {
				t.Fatalf("expected encoded filter after preparation, got %v", err)
			}
			if string(before) != string(after) {
				t.Fatalf("expected unchanged public filter %s, got %s", before, after)
			}
		})
	}
}

// TestFilterCountMoneyInvalidFirst checks invalid siblings in the same where.
func TestFilterCountMoneyInvalidFirst(t *testing.T) {
	unavailable := strings.Replace(moneyConditionJSON, `is greater than`, `is`, 1)
	invalid := `{"property":"unknown","operator":"exists"}`
	for _, body := range []string{unavailable + `,` + invalid, invalid + `,` + unavailable} {
		var condition FilterCountCondition
		err := json.Unmarshal([]byte(strings.Replace(countMoneyConditionJSON, moneyConditionJSON, body, 1)), &condition)
		if err != nil {
			t.Fatalf("expected decoded where, got %v", err)
		}
		_, err = prepareFilterCount(&condition, nil)
		if err != nil {
			missing, ok := errors.AsType[types.PathNotExistError](err)
			if !ok || missing.Path != "unknown" {
				t.Fatalf("expected invalid property before unavailable comparison, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid property rejected, got success")
	}
}

// TestFilterCountMoneyWhereBudget counts groups, Record and money leaves.
func TestFilterCountMoneyWhereBudget(t *testing.T) {
	for _, test := range []struct{ depth, size int }{{1, 100}, {1, 101}, {2, 1}, {3, 1}} {
		var condition FilterCountCondition
		err := json.Unmarshal([]byte(countMoneyConditionJSON), &condition)
		if err != nil {
			t.Fatalf("expected Count money fixture, got %v", err)
		}
		body := &Filter{Operator: OpAnd, Rules: make([]FilterRule, test.size)}
		body.Rules[0] = condition.Count.Where.Rules[0]
		for i := 1; i < test.size; i++ {
			body.Rules[i] = &FilterCondition{Property: "properties.flag", Operator: OpExists}
		}
		for i := 1; i < test.depth; i++ {
			body = &Filter{Operator: OpAnd, Rules: []FilterRule{body}}
		}
		condition.Count.Where = body
		_, err = prepareFilterCount(&condition, nil)
		invalid := test.depth == 3 || test.size == 101
		if err != nil {
			if !invalid {
				t.Fatalf("expected depth=%d size=%d accepted, got %v", test.depth, test.size, err)
			}
			continue
		}
		if invalid {
			t.Fatalf("expected depth=%d size=%d rejected, got success", test.depth, test.size)
		}
	}
}

// TestFilterCountMoneyDependencies traverses every nested qualification branch.
func TestFilterCountMoneyDependencies(t *testing.T) {
	for _, path := range []string{
		"properties", "properties.total", "properties.total.x", "properties.currency", "properties.currency.x",
	} {
		for _, nested := range []bool{false, true} {
			body := `{"property":"` + path + `","operator":"is null"}`
			if nested {
				body = `{"operator":"or","rules":[` + body + `]}`
			}
			unavailable := strings.Replace(countMoneyConditionJSON, moneyConditionJSON, moneyConditionJSON+`,`+body, 1)
			var sibling FilterCountCondition
			err := json.Unmarshal([]byte(unavailable), &sibling)
			if err != nil {
				t.Fatalf("expected dependent predicate decoded, got %v", err)
			}
			_, err = prepareFilterCount(&sibling, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
					t.Fatalf("expected unavailable path %s, got %v", path, err)
				}
			}
			if err == nil {
				t.Fatalf("expected unavailable path %s, got success", path)
			}
			invalid := strings.Replace(countMoneyConditionJSON, `"name":"Order Completed"`, `"name":"Other"`, 1)
			for _, rules := range []string{unavailable + `,` + invalid, invalid + `,` + unavailable} {
				var filter Filter
				err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+rules+`]}`), &filter)
				if err != nil {
					t.Fatalf("expected decoded sibling order, got %v", err)
				}
				nodes := 0
				_, err = prepareUnifiedRule(&filter, types.Object([]types.Property{{Name: "country", Type: types.String()}}),
					0, &nodes)
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalidity before dependent path %s, got %v", path, err)
					}
					continue
				}
				t.Fatal("expected invalid filter rejected, got success")
			}
		}
	}
	for _, body := range []string{
		moneyConditionJSON + `,` + moneyConditionJSON,
		moneyConditionJSON + `,{"operator":"or","rules":[` + moneyConditionJSON + `]}`,
	} {
		var condition FilterCountCondition
		err := json.Unmarshal([]byte(strings.Replace(countMoneyConditionJSON, moneyConditionJSON, body, 1)), &condition)
		if err != nil {
			t.Fatalf("expected repeated money decoded, got %v", err)
		}
		_, err = prepareFilterCount(&condition, nil)
		if err != nil {
			if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
				t.Fatalf("expected repeated money unavailable, got %v", err)
			}
			continue
		}
		t.Fatal("expected repeated money unavailable, got success")
	}
}
