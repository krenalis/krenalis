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

const experienceCurrencyJSON = `"currencyField":{"definition":"commerce.order_completed.currency",` +
	`"meaningRevision":"1"},"nativeCurrency":"EUR",`

// TestFilterExperienceCurrencyAdmission checks references and compatibility.
func TestFilterExperienceCurrencyAdmission(t *testing.T) {
	base := strings.Replace(experienceConditionJSON, `"base":{`, `"base":{`+experienceCurrencyJSON, 1)
	unavailableWhere := strings.Replace(base, `"base":{`,
		`"base":{"where":{"operator":"and","rules":[`+moneyConditionJSON+`]},`, 1)
	for _, test := range []struct {
		name, input string
		invalid     bool
		unsupported bool
	}{
		{"Record verify", base, false, false},
		{"money same currency", strings.Replace(base, experienceRecordConditionJSON, moneyConditionJSON, 1), false, false},
		{"money different currency", strings.Replace(base, experienceRecordConditionJSON, strings.Replace(moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"USD"`, 1), 1), false, false},
		{"multiple false money", strings.Replace(base, experienceRecordConditionJSON, strings.Replace(moneyConditionJSON+`,`+moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"USD"`, -1), 1), false, true},
		{"false money amount overlap", strings.Replace(base, experienceRecordConditionJSON, strings.Replace(moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"USD"`, 1)+`,{"property":"properties.total","operator":"is null"}`, 1), false, true},
		{"currency only", strings.Replace(base, `"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},`, ``, 1), true, false},
		{"field only", strings.Replace(base, `"nativeCurrency":"EUR",`, ``, 1), true, false},
		{"currency case", strings.Replace(base, `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, 1), true, false},
		{"unavailable where invalid currency", strings.Replace(unavailableWhere, experienceCurrencyJSON, strings.Replace(experienceCurrencyJSON, `"EUR"`, `"eur"`, 1), 1), true, false},
		{"unavailable where incomplete pair", strings.Replace(unavailableWhere, experienceCurrencyJSON, strings.Replace(experienceCurrencyJSON, `"nativeCurrency":"EUR",`, ``, 1), 1), true, false},
		{"revision", strings.Replace(base, `"definition":"commerce.order_completed.currency","meaningRevision":"1"`, `"definition":"commerce.order_completed.currency","meaningRevision":"2"`, 1), true, false},
		{"source", strings.Replace(base, `"name":"Order Completed"`, `"name":"Other"`, 1), true, false},
		{"page", strings.Replace(base, `"kind":"recordedEvents"`, `"kind":"recordedEvents","type":"page"`, 1), true, false},
		{"different currency equality", strings.Replace(strings.Replace(base, experienceRecordConditionJSON, strings.Replace(moneyConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"USD"`, 1), 1), `"operator":"is greater than"`, `"operator":"is"`, 1), false, true},
		{"invalid before equality", strings.Replace(strings.Replace(strings.Replace(base, experienceRecordConditionJSON, moneyConditionJSON, 1), `"operator":"is greater than"`, `"operator":"is"`, 1), experienceCurrencyJSON, strings.Replace(experienceCurrencyJSON, `"EUR"`, `"eur"`, 1), 1), true, false},
		{"raw total without money", strings.Replace(base, `properties.order_id`, `properties.total`, 1), false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterExperienceCondition
			err := json.Unmarshal([]byte(test.input), &condition)
			if err != nil {
				t.Fatalf("expected decodable reference fixture, got %v", err)
			}
			rule, err := prepareFilterExperience(&condition, nil)
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !test.invalid &&
					(!test.unsupported || !unsupported) || test.invalid && unsupported {
					t.Fatalf("expected invalid=%v unsupported=%v, got %v", test.invalid, test.unsupported, err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid base currency, got success")
			}
			err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			if err != nil {
				if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !test.unsupported || !unsupported {
					t.Fatalf("expected unsupported=%v, got %v", test.unsupported, err)
				}
				return
			}
			if test.unsupported {
				t.Fatal("expected unsupported verification, got success")
			}
		})
	}
}

// TestFilterExperienceCurrencyCodec preserves omission and exact round trips.
func TestFilterExperienceCurrencyCodec(t *testing.T) {

	for _, fields := range []string{"", experienceCurrencyJSON} {
		input := strings.Replace(experienceConditionJSON, `"base":{`, `"base":{`+fields, 1)
		var condition FilterExperienceCondition
		err := json.Unmarshal([]byte(input), &condition)
		if err != nil {
			t.Fatalf("expected experience decoded, got %v", err)
		}
		encoded, err := json.Marshal(&condition)
		if err != nil {
			t.Fatalf("expected experience encoded, got %v", err)
		}
		if strings.Contains(string(encoded), `"currencyField"`) != (fields != "") ||
			strings.Contains(string(encoded), `"nativeCurrency"`) != (fields != "") {
			t.Fatalf("expected currency omission=%v, got %s", fields == "", encoded)
		}
		var restored FilterExperienceCondition
		err = json.Unmarshal(encoded, &restored)
		if err != nil {
			t.Fatalf("expected experience restored, got %v", err)
		}
		again, err := json.Marshal(&restored)
		if err != nil {
			t.Fatalf("expected restored encoding, got %v", err)
		}
		if !bytes.Equal(encoded, again) {
			t.Fatalf("expected unchanged round trip %s, got %s", encoded, again)
		}
		_, _, err = prepareFilter(&Filter{Operator: OpAnd, Rules: []FilterRule{&condition}},
			types.Object([]types.Property{{Name: "country", Type: types.String()}}))
		if err == nil {
			t.Fatal("expected Record pipeline rejection, got success")
		}
	}

	for _, field := range []string{`"currencyField":null,`, `"nativeCurrency":null,`, `"nativeCurrency":3,`} {
		var condition FilterExperienceCondition
		err := json.Unmarshal([]byte(strings.Replace(experienceConditionJSON, `"base":{`, `"base":{`+field, 1)), &condition)
		if err == nil {
			t.Fatalf("expected invalid currency field %s, got success", field)
		}
	}

}

// TestFilterExperienceCurrencyDependencies excludes raw overlapping readings.
func TestFilterExperienceCurrencyDependencies(t *testing.T) {
	base := strings.Replace(experienceConditionJSON, `"base":{`, `"base":{`+experienceCurrencyJSON, 1)
	for _, path := range []string{"properties", "properties.currency", "properties.currency.x",
		"properties.Currency", "properties.currency_code", "properties.total"} {
		for _, inBase := range []bool{false, true} {
			for _, nested := range []bool{false, true} {
				leaf := `{"property":"` + path + `","operator":"is null"}`
				if nested {
					leaf = `{"operator":"or","rules":[` + leaf + `]}`
				}
				input := strings.Replace(base, experienceRecordConditionJSON, leaf, 1)
				if inBase {
					input = strings.Replace(base, `"base":{`, `"base":{"where":{"operator":"and","rules":[`+leaf+`]},`, 1)
				}
				var condition FilterExperienceCondition
				err := json.Unmarshal([]byte(input), &condition)
				if err != nil {
					t.Fatalf("expected dependency decoded, got %v", err)
				}
				rule, err := prepareFilterExperience(&condition, nil)
				if err == nil {
					err = unified.Validate(&rule, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
				}
				blocked := path == "properties" || path == "properties.currency" || path == "properties.currency.x"
				if err != nil {
					if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !blocked || !unsupported {
						t.Fatalf("expected overlap=%v for %s, got %v", blocked, path, err)
					}
				}
				if err == nil && blocked {
					t.Fatalf("expected unsupported path %s, got success", path)
				}
				for _, inverse := range []bool{false, true} {
					bad := `{"property":"unknown","operator":"is","values":[1]}`
					pair := leaf + `,` + bad
					if inverse {
						pair = bad + `,` + leaf
					}
					invalid := strings.Replace(input, leaf, pair, 1)
					var condition FilterExperienceCondition
					err := json.Unmarshal([]byte(invalid), &condition)
					if err != nil {
						t.Fatalf("expected invalid sibling decoded, got %v", err)
					}
					_, err = prepareFilterExperience(&condition, nil)
					if err != nil {
						missing, ok := errors.AsType[types.PathNotExistError](err)
						if !ok || missing.Path != "unknown" {
							t.Fatalf("expected invalid property before missing capability, got %v", err)
						}
						continue
					}
					t.Fatal("expected invalid sibling, got success")
				}
			}
		}
	}
}
