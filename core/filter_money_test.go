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

	"github.com/krenalis/krenalis/core/internal/behavioral"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const moneyConditionJSON = `{"money":{"field":{"definition":"commerce.order_completed.total",` +
	`"meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is greater than","values":[100]}`

// TestFilterMoneyAdmission distinguishes validity from availability.
func TestFilterMoneyAdmission(t *testing.T) {
	base := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, moneyConditionJSON, 1)
	otherCurrency := strings.Replace(moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"USD"`, 1)
	nestedMoney := `{"operator":"or","rules":[{"property":"event","operator":"is","values":["Other"]},` +
		otherCurrency + `]}`
	for _, test := range []struct {
		name, input string
		invalid     bool
		unavailable bool
	}{
		{"greater", base, false, false},
		{"at least", strings.Replace(base, `is greater than`, `is greater than or equal to`, 1), false, false},
		{"less", strings.Replace(base, `is greater than`, `is less than`, 1), false, false},
		{"at most", strings.Replace(base, `is greater than`, `is less than or equal to`, 1), false, false},
		{"negative", strings.Replace(base, `[100]`, `[-100]`, 1), false, false},
		{"fine threshold", strings.Replace(base, `[100]`, `[100.00001]`, 1), false, false},
		{"global maximum", strings.Replace(base, `[100]`, `[1e75]`, 1), false, false},
		{"global minimum scale", strings.Replace(base, `[100]`, `[1e-37]`, 1), false, false},
		{"precision", strings.Replace(base, `[100]`, `[1e76]`, 1), true, false},
		{"scale", strings.Replace(base, `[100]`, `[1e-38]`, 1), true, false},
		{"long literal", strings.Replace(base, `[100]`, `[1.`+strings.Repeat("0", 4096)+`]`, 1), true, false},
		{"string", strings.Replace(base, `[100]`, `["100"]`, 1), true, false},
		{"null", strings.Replace(base, `[100]`, `[null]`, 1), true, false},
		{"arity", strings.Replace(base, `[100]`, `[]`, 1), true, false},
		{"state operator", strings.Replace(base, `is greater than`, `exists`, 1), true, false},
		{"equality", strings.Replace(base, `is greater than`, `is`, 1), false, false},
		{"different", strings.Replace(base, `is greater than`, `is not`, 1), false, false},
		{"second money under OR", strings.Replace(base, moneyConditionJSON, moneyConditionJSON+`,`+nestedMoney, 1), false, true},
		{"interval", strings.Replace(strings.Replace(base, `is greater than`, `is between`, 1), `[100]`, `[100,200]`, 1), false, false},
		{"inverted interval", strings.Replace(strings.Replace(base, `is greater than`, `is between`, 1), `[100]`, `[200,100]`, 1), true, false},
		{"field", strings.Replace(base, `commerce.order_completed.total`, `unknown`, 1), true, false},
		{"revision", strings.Replace(base, `"definition":"commerce.order_completed.total","meaningRevision":"1"`, `"definition":"commerce.order_completed.total","meaningRevision":"2"`, 1), true, false},
		{"currency", strings.Replace(base, `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, 1), true, false},
		{"source", strings.Replace(base, `"name":"Order Completed"`, `"name":"Other"`, 1), true, false},
		{"page", strings.Replace(base, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1), true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+test.input+`]}`), &filter)
			if err != nil {
				t.Fatalf("expected decodable admission case, got %v", err)
			}
			nodes := 0
			rule, err := prepareBehavioralRule(&filter,
				types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid parameters, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid parameters, got success")
			}
			err = behavioral.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), behavioral.Page{Limit: 1})
			if err != nil {
				if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !test.unavailable || !unsupported {
					t.Fatalf("expected unavailable=%v, got %v", test.unavailable, err)
				}
				return
			}
			if test.unavailable {
				t.Fatal("expected unavailable predicate, got success")
			}
		})
	}
}

// TestFilterMoneyCodec preserves operands and requested comparison metadata.
func TestFilterMoneyCodec(t *testing.T) {
	input := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, moneyConditionJSON, 1)
	var condition FilterExperienceCondition
	err := json.Unmarshal([]byte(input), &condition)
	if err != nil {
		t.Fatalf("expected money decoded, got %v", err)
	}
	before, err := json.Marshal(&condition)
	if err != nil {
		t.Fatalf("expected money encoded, got %v", err)
	}
	rule, err := prepareFilterExperience(&condition, nil)
	if err != nil {
		t.Fatalf("expected money prepared, got %v", err)
	}
	after, err := json.Marshal(&condition)
	if err != nil {
		t.Fatalf("expected money reencoded, got %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("expected unchanged public operands %s, got %s", before, after)
	}
	leaves, err := explainMoneyPredicates(rule.Experience.Verify, nil)
	if err != nil {
		t.Fatalf("expected monetary explanation, got %v", err)
	}
	if len(leaves) != 1 || len(leaves[0].Path) != 1 || leaves[0].Path[0] != 0 ||
		leaves[0].Requested.Operator != OpIsGreaterThan || !leaves[0].Requested.Values[0].Equal(decimal.MustParse("100")) ||
		leaves[0].Normalized.Operator != OpIsGreaterThanOrEqualTo ||
		!leaves[0].Normalized.Values[0].Equal(decimal.MustParse("100.0001")) {
		t.Fatalf("expected requested >100 and normalized >=100.0001 at [0], got %+v", leaves)
	}
}

// TestFilterMoneyDependencies checks all body branches and source where paths.
func TestFilterMoneyDependencies(t *testing.T) {
	base := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, moneyConditionJSON, 1)
	for _, path := range []string{"properties", "properties.total", "properties.total.x", "properties.currency",
		"properties.currency.x", "properties.total_discount", "properties.Total", "properties.order_id"} {
		for _, where := range []bool{false, true} {
			for _, nested := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/where=%v/nested=%v", path, where, nested), func(t *testing.T) {
					leaf := `{"property":"` + path + `","operator":"is","values":[100]}`
					if nested {
						leaf = `{"operator":"or","rules":[` + leaf + `]}`
					}
					input := strings.Replace(base, moneyConditionJSON, leaf+`,`+moneyConditionJSON, 1)
					if where {
						input = strings.Replace(base, `"base":{`,
							`"base":{"where":{"operator":"and","rules":[`+leaf+`]},`, 1)
					}
					var filter Filter
					err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+input+`]}`), &filter)
					if err != nil {
						t.Fatalf("expected dependency fixture decoded, got %v", err)
					}
					nodes := 0
					rule, err := prepareBehavioralRule(&filter,
						types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
					if err != nil {
						t.Fatalf("expected valid dependency shape, got %v", err)
					}
					err = behavioral.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), behavioral.Page{Limit: 1})
					blocked := path == "properties" || path == "properties.total" || path == "properties.total.x" ||
						path == "properties.currency" || path == "properties.currency.x"
					if err != nil {
						if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !blocked || !unsupported {
							t.Fatalf("expected blocked=%v, got %v", blocked, err)
						}
						return
					}
					if blocked {
						t.Fatal("expected unsupported overlap, got success")
					}
				})
			}
		}
	}
}

// TestFilterMoneyInvalidBeforeUnsupported checks local and main-tree ordering.
func TestFilterMoneyInvalidBeforeUnsupported(t *testing.T) {
	base := strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, moneyConditionJSON, 1)
	bad := `{"property":"unknown","operator":"is","values":[1]}`
	for _, unavailable := range []string{
		strings.Replace(base, moneyConditionJSON, moneyConditionJSON+`,`+moneyConditionJSON, 1),
		strings.Replace(base, moneyConditionJSON,
			moneyConditionJSON+`,{"operator":"or","rules":[`+moneyConditionJSON+`]}`, 1),
		strings.Replace(base, moneyConditionJSON,
			moneyConditionJSON+`,{"property":"properties","operator":"is null"}`, 1),
		strings.Replace(base, `"base":{`,
			`"base":{"where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]},`, 1),
		strings.Replace(base, `"base":{`,
			`"base":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1),
	} {
		for _, inverse := range []bool{false, true} {
			for _, local := range []bool{false, true} {
				input := unavailable + `,` + bad
				if inverse {
					input = bad + `,` + unavailable
				}
				if local {
					input = unavailable
				}
				var filter Filter
				err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+input+`]}`), &filter)
				if err != nil {
					t.Fatalf("expected decoded precedence case, got %v", err)
				}
				if local {
					experience := filter.Rules[0].(*FilterExperienceCondition)
					invalid := &FilterCondition{Property: "unknown", Operator: OpIs, Values: []json.Value{json.Value(`1`)}}
					if inverse {
						experience.Experience.Verify.Rules = append([]FilterRule{invalid}, experience.Experience.Verify.Rules...)
					} else {
						experience.Experience.Verify.Rules = append(experience.Experience.Verify.Rules, invalid)
					}
				}
				nodes := 0
				_, err = prepareBehavioralRule(&filter,
					types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
				if err != nil {
					missing, ok := errors.AsType[types.PathNotExistError](err)
					if !ok || missing.Path != "unknown" {
						t.Fatalf("expected invalid property before unavailable capability, got %v", err)
					}
					continue
				}
				t.Fatal("expected invalid property, got success")
			}
		}
	}
}

// TestFilterMoneyReductionOperators preserves unsupported Sum/Average tests.
func TestFilterMoneyReductionOperators(t *testing.T) {
	for _, input := range []string{sumConditionJSON, averageConditionJSON} {
		for _, comparison := range []struct{ operator, values string }{
			{"is", "[100]"}, {"is not", "[100]"}, {"is between", "[100,200]"},
		} {
			var filter Filter
			data := strings.Replace(input, `"operator":"is greater than or equal to"`,
				`"operator":"`+comparison.operator+`"`, 1)
			data = strings.Replace(data, `"values":[30]`, `"values":`+comparison.values, 1)
			data = strings.Replace(data, `"values":[100]`, `"values":`+comparison.values, 1)
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+data+`]}`), &filter)
			if err != nil {
				t.Fatalf("expected valid reduction comparison, got %v", err)
			}
			nodes := 0
			_, err = prepareBehavioralRule(&filter,
				types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
			if err != nil {
				unavailable, unsupported := errors.AsType[*behavioral.UnsupportedError](err)
				if !unsupported || unavailable.Capability != "monetary equality or interval comparisons" {
					t.Fatalf("expected unsupported reduction comparison, got %v", err)
				}
				continue
			}
			t.Fatal("expected unsupported reduction comparison, got success")
		}
	}
}

// TestFilterMoneyStrict rejects mixed subjects and wrong owners.
func TestFilterMoneyStrict(t *testing.T) {
	for _, input := range []string{
		`{"operator":"and","rules":[` + moneyConditionJSON + `]}`,
		strings.Replace(moneyConditionJSON, `"money":{`, `"money":null,"unused":{`, 1),
		strings.Replace(moneyConditionJSON, `"money":{`, `"money":{"where":{},`, 1),
		strings.Replace(moneyConditionJSON, `"money":{`, `"money":{"field":{},`, 1),
		strings.Replace(moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":null`, 1),
		strings.Replace(moneyConditionJSON, `"money":{`, `"property":"properties.total","money":{`, 1),
		strings.Replace(moneyConditionJSON, `"values":[100]`, `"values":[100],"occurredIn":{}`, 1),
	} {
		if !strings.HasPrefix(input, `{"operator":"and","rules":`) {
			input = `{"operator":"and","rules":[` +
				strings.Replace(experienceConditionJSON, experienceRecordConditionJSON, input, 1) + `]}`
		}
		var filter Filter
		err := json.Unmarshal([]byte(input), &filter)
		if err == nil {
			t.Fatalf("expected strict money decoding error, got %s", input)
		}
	}
}

// TestFilterMoneyWhere keeps Sum, Average and E base qualification unavailable.
func TestFilterMoneyWhere(t *testing.T) {

	for _, input := range []string{
		strings.Replace(experienceConditionJSON, `"base":{`,
			`"base":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1),
		strings.Replace(sumConditionJSON, `"sum":{`,
			`"sum":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1),
		strings.Replace(averageConditionJSON, `"average":{`,
			`"average":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1),
	} {
		bad := `{"property":"unknown","operator":"is","values":[1]}`
		nestedMoney := `{"operator":"or","rules":[` + moneyConditionJSON + `]}`
		for _, pair := range []string{
			moneyConditionJSON + `,` + bad, bad + `,` + moneyConditionJSON,
			nestedMoney + `,` + bad, bad + `,` + nestedMoney,
		} {
			invalid := strings.Replace(input, moneyConditionJSON, pair, 1)
			var filter Filter
			err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+invalid+`]}`), &filter)
			if err != nil {
				t.Fatalf("expected unavailable where with invalid sibling decoded, got %v", err)
			}
			nodes := 0
			_, err = prepareBehavioralRule(&filter,
				types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
			if err != nil {
				missing, ok := errors.AsType[types.PathNotExistError](err)
				if !ok || missing.Path != "unknown" {
					t.Fatalf("expected invalid where property before missing capability, got %v", err)
				}
				continue
			}
			t.Fatal("expected invalid where property, got success")
		}
		var filter Filter
		err := json.Unmarshal([]byte(`{"operator":"and","rules":[`+input+`]}`), &filter)
		if err != nil {
			t.Fatalf("expected decodable unavailable where, got %v", err)
		}
		nodes := 0
		_, err = prepareBehavioralRule(&filter,
			types.Object([]types.Property{{Name: "country", Type: types.String()}}), 0, &nodes)
		if err != nil {
			if _, unsupported := errors.AsType[*behavioral.UnsupportedError](err); !unsupported {
				t.Fatalf("expected valid but unavailable monetary qualification, got %v", err)
			}
			continue
		}
		t.Fatal("expected unavailable qualification, got success")
	}

	var condition FilterMoneyCondition
	err := json.Unmarshal([]byte(moneyConditionJSON), &condition)
	if err != nil {
		t.Fatalf("expected standalone money decoded, got %v", err)
	}
	_, _, err = prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{&condition}},
		types.Object([]types.Property{{Name: "country", Type: types.String()}}))
	if err == nil {
		t.Fatal("expected money rejected in Record pipelines, got success")
	}

}

// TestFilterMoneyWhereParameters defers availability until Count is valid.
func TestFilterMoneyWhereParameters(t *testing.T) {
	base := strings.Replace(countConditionJSON, `"name":"Purchase"`, `"name":"Order Completed"`, 1)
	base = strings.Replace(base, `"count":{`,
		`"count":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1)
	largePeriod := strings.Replace(base, `"value":30`, `"value":1000000000`, 1)
	for _, test := range []struct {
		input       string
		unsupported bool
		valid       bool
	}{
		{strings.Replace(base, `"count":{`, `"count":{"nativeCurrency":"EUR",`, 1), false, false},
		{strings.Replace(base, `"count":{`, `"count":{"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},"nativeCurrency":"EUR",`, 1), false, true},
		{largePeriod, true, false},
		{strings.Replace(largePeriod, `commerce.order_completed.total`, `unresolved`, 1), false, false},
	} {
		var condition FilterCountCondition
		err := json.Unmarshal([]byte(test.input), &condition)
		if err != nil {
			t.Fatalf("expected where parameter fixture decoded, got %v", err)
		}
		_, err = prepareFilterCount(&condition, nil)
		if err != nil {
			_, unsupported := errors.AsType[*behavioral.UnsupportedError](err)
			if test.valid || unsupported != test.unsupported {
				t.Fatalf("expected unsupported=%v after parameter validation, got %v", test.unsupported, err)
			}
			continue
		}
		if !test.valid {
			t.Fatal("expected unavailable or invalid Count, got success")
		}
	}
}
