// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const sumConditionJSON = `{"sum":{"source":{"kind":"recordedEvents","name":"Order Completed",` +
	`"grain":"registered-message"},"period":{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}},` +
	`"field":{"definition":"commerce.order_completed.total","meaningRevision":"1"},"nativeCurrency":"EUR"},` +
	`"operator":"is greater than or equal to","values":[30]}`

// TestFilterSumBudget retains the common main-tree limits for the new leaf.
func TestFilterSumBudget(t *testing.T) {
	var condition FilterSumCondition
	err := json.Unmarshal([]byte(sumConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected Sum fixture, got %v", err)
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
				t.Fatalf("expected 100 Sum descendants, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected rejection at 101 descendants, got success")
		}
	}
	var root FilterRule = &condition
	for depth := 1; depth <= 5; depth++ {
		root = &Filter{Operator: OpAnd, Rules: []FilterRule{root}}
		nodes := 0
		_, err := prepareUnifiedRule(root, schema, 0, &nodes)
		if err != nil {
			if depth < 5 {
				t.Fatalf("expected four levels including root, got %v", err)
			}
			continue
		}
		if depth == 5 {
			t.Fatal("expected fifth group rejection, got success")
		}
	}
}

// TestFilterSumInvalidBeforeUnsupported checks the full input in either order.
func TestFilterSumInvalidBeforeUnsupported(t *testing.T) {
	for _, period := range []string{`{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}`,
		`{"kind":"elapsedLookback","duration":{"value":999999999999999999999999,"unit":"day"}}`} {
		bad := strings.Replace(sumConditionJSON, `"EUR"`, `"eur"`, 1)
		bad = strings.Replace(bad, `{"kind":"elapsedLookback","duration":{"value":30,"unit":"day"}}`, period, 1)
		unavailable := strings.Replace(sumConditionJSON, `is greater than or equal to`, `is`, 1)
		for _, rules := range []string{bad + "," + unavailable, unavailable + "," + bad} {
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+rules+`]}`), &filter)
			if err != nil {
				t.Fatalf("expected decodable input, got %v", err)
			}
			nodes := 0
			schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
			_, err = prepareUnifiedRule(&filter, schema, 0, &nodes)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
					t.Fatalf("expected invalidity before unavailable Sum, got %v", err)
				}
				continue
			}
			t.Fatal("expected invalidity, got success")
		}
	}
}

// TestFilterSumRoundTrip preserves exact operands and references.
func TestFilterSumRoundTrip(t *testing.T) {
	for _, literal := range []string{"30", "-30.00001", "1e75", "1e-37"} {
		input := `{"operator":"and","rules":[` + strings.Replace(sumConditionJSON, "[30]", "["+literal+"]", 1) + `]}`
		var filter Filter
		err := json.Unmarshal([]byte(input), &filter)
		if err != nil {
			t.Fatalf("expected Sum decoding, got %v", err)
		}
		before, err := json.Marshal(&filter)
		if err != nil {
			t.Fatalf("expected Sum encoding, got %v", err)
		}
		if string(before) != input || !filter.RequiresUnified() || filter.HasCount() {
			t.Fatalf("expected unchanged Sum document requiring Unified, got %s", before)
		}
		nodes := 0
		schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
		root, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
		if err != nil {
			t.Fatalf("expected prepared Sum, got %v", err)
		}
		err = unified.Validate(&root, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
		if err != nil {
			t.Fatalf("expected admitted Sum, got %v", err)
		}
		after, err := json.Marshal(&filter)
		if err != nil {
			t.Fatalf("expected encoding after preparation, got %v", err)
		}
		if !bytes.Equal(before, after) {
			t.Fatalf("expected unmodified public filter %s, got %s", before, after)
		}
		_, _, err = prepareFilter(&filter, schema)
		if err == nil {
			t.Fatal("expected Sum rejected in Record-only contexts, got success")
		}
	}
}

// TestFilterSumValidation distinguishes invalid input from unavailable forms.
func TestFilterSumValidation(t *testing.T) {
	for _, test := range []struct {
		name, from, to string
		unsupported    bool
	}{
		{"missing currency", `,"nativeCurrency":"EUR"`, ``, false},
		{"null currency", `"EUR"`, `null`, false},
		{"lowercase currency", `"EUR"`, `"eur"`, false},
		{"currency width", `"EUR"`, `"EURO"`, false},
		{"currency unicode", `"EUR"`, `"€UR"`, false},
		{"wrong source", `"Order Completed"`, `"Refund"`, false},
		{"wrong grain", `"registered-message"`, `"order"`, false},
		{"unresolved field", `"meaningRevision":"1"`, `"meaningRevision":"2"`, false},
		{"null field", `{"definition":"commerce.order_completed.total","meaningRevision":"1"}`, `null`, false},
		{"unknown field member", `"definition":`, `"path":"properties.total","definition":`, false},
		{"duplicate currency", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","nativeCurrency":"EUR"`, false},
		{"null Sum", `"sum":{`, `"sum":null,"unused":{`, false},
		{"null literal", `[30]`, `[null]`, false},
		{"string literal", `[30]`, `["30"]`, false},
		{"large precision", `[30]`, `[1e76]`, false},
		{"large scale", `[30]`, `[1e-38]`, false},
		{"literal arity", `[30]`, `[30,40]`, false},
		{"unary operator", `is greater than or equal to`, `exists`, false},
		{"empty minimum", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":0`, false},
		{"sample minimum", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":1`, true},
		{"minimum with where", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","minimum":1,"where":{"operator":"and","rules":[{"property":"properties.coupon","operator":"exists"}]}`, true},
		{"equality", `is greater than or equal to`, `is`, true},
		{"different", `is greater than or equal to`, `is not`, true},
		{"interval", `"operator":"is greater than or equal to","values":[30]`, `"operator":"is between","values":[20,30]`, true},
		{"reversed interval", `"operator":"is greater than or equal to","values":[30]`, `"operator":"is between","values":[40,30]`, false},
		{"dependent where", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.total","operator":"exists"}]}`, true},
		{"invalid where", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"absent","operator":"exists"}]}`, false},
		{"nested Count", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[` + countConditionJSON + `]}`, false},
		{"nested Sum", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[` + sumConditionJSON + `]}`, false},
		{"dual subject", `"sum":`, `"property":"country","sum":`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := strings.Replace(sumConditionJSON, test.from, test.to, 1)
			var condition FilterSumCondition
			err := json.Unmarshal([]byte(input), &condition)
			if err == nil {
				_, err = prepareFilterSum(&condition, nil)
			}
			if err != nil {
				_, unavailable := errors.AsType[*unified.UnsupportedError](err)
				if unavailable != test.unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			t.Fatal("expected rejected input, got success")
		})
	}
}

// TestFilterSumWhere checks independent paths and invalidity precedence.
func TestFilterSumWhere(t *testing.T) {

	for _, test := range []struct {
		name, where string
		unsupported bool
	}{
		{"parent", `{"operator":"and","rules":[{"property":"properties","operator":"exists"}]}`, true},
		{"amount", `{"operator":"and","rules":[{"property":"properties.total","operator":"exists"}]}`, true},
		{"amount descendant", `{"operator":"and","rules":[{"property":"properties.total.x","operator":"exists"}]}`, true},
		{"currency", `{"operator":"and","rules":[{"property":"properties.currency","operator":"exists"}]}`, true},
		{"currency descendant", `{"operator":"and","rules":[{"property":"properties.currency.x","operator":"exists"}]}`, true},
		{"nested OR", `{"operator":"and","rules":[{"operator":"or","rules":[{"property":"properties.coupon","operator":"is","values":["PROMO"]},{"property":"properties.currency","operator":"exists"}]}]}`, true},
		{"near amount", `{"operator":"and","rules":[{"property":"properties.total_discount","operator":"exists"}]}`, false},
		{"near currency", `{"operator":"and","rules":[{"property":"properties.currency_code","operator":"exists"}]}`, false},
		{"amount case", `{"operator":"and","rules":[{"property":"properties.Total","operator":"exists"}]}`, false},
		{"currency case", `{"operator":"and","rules":[{"property":"properties.Currency","operator":"exists"}]}`, false},
		{"traits", `{"operator":"and","rules":[{"property":"traits.total","operator":"exists"}]}`, false},
		{"coupon", `{"operator":"and","rules":[{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}`, false},
		{"nested independent", `{"operator":"or","rules":[{"operator":"and","rules":[{"property":"properties.Total","operator":"exists"},{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}]}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {

			input := strings.Replace(sumConditionJSON, `"nativeCurrency":"EUR"`,
				`"nativeCurrency":"EUR","where":`+test.where, 1)
			var condition FilterSumCondition
			err := json.Unmarshal([]byte(input), &condition)
			if err != nil {
				t.Fatalf("expected decodable Sum where, got %v", err)
			}
			rule, err := prepareFilterSum(&condition, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !test.unsupported || !unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
			}
			if err == nil {
				if test.unsupported || rule.Sum == nil || rule.Sum.Where == nil {
					t.Fatalf("expected independent prepared where, got %+v", rule)
				}
				encoded, err := json.Marshal(&condition)
				if err != nil {
					t.Fatalf("expected where round-trip, got %v", err)
				}
				if string(encoded) != input {
					t.Fatalf("expected unchanged filter %s, got %s", input, encoded)
				}
			}

			bad := `{"property":"absent","operator":"exists"}`
			for _, rules := range []string{input + "," + bad, bad + "," + input} {
				var filter Filter
				err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+rules+`]}`), &filter)
				if err != nil {
					t.Fatalf("expected decodable invalid sibling, got %v", err)
				}
				nodes := 0
				schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
				_, err = prepareUnifiedRule(&filter, schema, 0, &nodes)
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
						t.Fatalf("expected invalidity before availability in either order, got %v", err)
					}
					continue
				}
				t.Fatal("expected invalid sibling rejection, got success")
			}

		})
	}

	// A dependent leaf must not mask an invalid sibling inside its own where.
	dependent := `{"property":"properties.total","operator":"exists"}`
	invalid := `{"property":"properties.coupon","operator":"is","values":[]}`
	for _, rules := range []string{dependent + "," + invalid, invalid + "," + dependent} {
		input := strings.Replace(sumConditionJSON, `"nativeCurrency":"EUR"`,
			`"nativeCurrency":"EUR","where":{"operator":"and","rules":[`+rules+`]}`, 1)
		var condition FilterSumCondition
		err := json.Unmarshal([]byte(input), &condition)
		if err != nil {
			t.Fatalf("expected decodable where, got %v", err)
		}
		_, err = prepareFilterSum(&condition, nil)
		if err != nil {
			if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
				t.Fatalf("expected invalid where before availability, got %v", err)
			}
			continue
		}
		t.Fatal("expected invalid where rejection, got success")
	}

}

// TestFilterSumWhereLimits preserves the Count occurrence-body limits.
func TestFilterSumWhereLimits(t *testing.T) {
	var condition FilterSumCondition
	err := json.Unmarshal([]byte(sumConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected Sum fixture, got %v", err)
	}
	for _, size := range []int{100, 101} {
		where := &Filter{Operator: OpAnd, Rules: make([]FilterRule, size)}
		for i := range where.Rules {
			where.Rules[i] = &FilterCondition{Property: "properties.coupon", Operator: OpExists}
		}
		condition.Sum.Where = where
		_, err := prepareFilterSum(&condition, nil)
		if err != nil {
			if size == 100 {
				t.Fatalf("expected 100 occurrence nodes, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected rejection at 101 occurrence nodes, got success")
		}
	}
	var rule FilterRule = &FilterCondition{Property: "properties.coupon", Operator: OpExists}
	for depth := 1; depth <= 3; depth++ {
		rule = &Filter{Operator: OpOr, Rules: []FilterRule{rule}}
		condition.Sum.Where = rule.(*Filter)
		_, err := prepareFilterSum(&condition, nil)
		if err != nil {
			if depth <= 2 {
				t.Fatalf("expected two occurrence group levels, got %v", err)
			}
			continue
		}
		if depth == 3 {
			t.Fatal("expected third occurrence group rejection, got success")
		}
	}
}
