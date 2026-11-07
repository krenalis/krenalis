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

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const currencyCountConditionJSON = `{"count":{"source":{"kind":"recordedEvents",` +
	`"name":"Order Completed","grain":"registered-message"},"period":{"kind":"elapsedLookback",` +
	`"duration":{"value":30,"unit":"day"}},"currencyField":{"definition":"commerce.order_completed.currency",` +
	`"meaningRevision":"1"},"nativeCurrency":"EUR"},"operator":"is","values":[0]}`

// TestFilterCountCurrencyDecoding checks the optional pair and strict types.
func TestFilterCountCurrencyDecoding(t *testing.T) {

	for _, fields := range []string{
		`"currencyField":null,"nativeCurrency":"EUR"`,
		`"currencyField":false,"nativeCurrency":"EUR"`,
		`"currencyField":[],"nativeCurrency":"EUR"`,
		`"currencyField":{"definition":null},"nativeCurrency":"EUR"`,
		`"currencyField":{"meaningRevision":1},"nativeCurrency":"EUR"`,
		`"currencyField":{"unknown":"x"},"nativeCurrency":"EUR"`,
		`"currencyField":{"definition":"` + strings.Repeat("a", 261) + `"},"nativeCurrency":"EUR"`,
		`"nativeCurrency":null`, `"nativeCurrency":42`, `"nativeCurrency":true`, `"nativeCurrency":{}`,
		`"nativeCurrency":[]`, `"nativeCurrency":"EUR","nativeCurrency":"USD"`,
	} {
		t.Run(fields, func(t *testing.T) {
			input := strings.Replace(countConditionJSON, `"source":`, fields+`,"source":`, 1)
			var condition FilterCountCondition
			err := json.Unmarshal([]byte(input), &condition)
			if err == nil {
				t.Fatal("expected malformed currency fields to be rejected, got success")
			}
		})
	}

	var field *FilterCurrencyField
	err := field.UnmarshalJSON([]byte(`{}`))
	if err == nil {
		t.Fatal("expected nil field receiver rejection, got success")
	}

}

// TestFilterCountCurrencyRoundTrip retains the reference without changing raw
// Record or existing Count documents.
func TestFilterCountCurrencyRoundTrip(t *testing.T) {
	for _, literal := range []string{"EUR", "USD", "ZZZ"} {
		t.Run(literal, func(t *testing.T) {

			input := `{"operator":"and","rules":[` +
				strings.Replace(currencyCountConditionJSON, `"nativeCurrency":"EUR"`,
					`"nativeCurrency":"`+literal+`"`, 1) + `]}`
			var filter Filter
			err := json.Unmarshal([]byte(input), &filter)
			if err != nil {
				t.Fatalf("expected decoded currency Count, got %v", err)
			}
			encoded, err := json.Marshal(&filter)
			if err != nil {
				t.Fatalf("expected encoded currency Count, got %v", err)
			}
			if string(encoded) != input || !filter.HasCount() || !filter.RequiresUnified() {
				t.Fatalf("expected unchanged Count document %s requiring Unified, got %s", input, encoded)
			}
			nodes := 0
			schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
			root, err := prepareUnifiedRule(&filter, schema, 0, &nodes)
			if err != nil {
				t.Fatalf("expected resolved currency Count, got %v", err)
			}
			err = unified.Validate(&root, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), unified.Page{Limit: 1})
			if err != nil {
				t.Fatalf("expected admitted currency Count, got %v", err)
			}
			currency := root.Children[0].Count.Currency
			if currency == nil || currency.Definition != "commerce.order_completed.currency" ||
				currency.Revision != "1" || currency.Value != literal {
				t.Fatalf("expected resolved currency reference and %s literal, got %+v", literal, currency)
			}
			after, err := json.Marshal(&filter)
			if err != nil {
				t.Fatalf("expected encoding after preparation, got %v", err)
			}
			if !bytes.Equal(encoded, after) {
				t.Fatalf("expected unchanged public document %s, got %s", encoded, after)
			}
			_, _, err = prepareFilter(&filter, schema)
			if err == nil {
				t.Fatal("expected Record-only context rejection, got success")
			}

		})
	}
}

// TestFilterCountCurrencyValidation distinguishes invalid references and
// dependent qualifications from supported nearby JSON paths.
func TestFilterCountCurrencyValidation(t *testing.T) {
	for _, test := range []struct {
		name, old, replacement string
		unsupported, supported bool
	}{
		{"missing field", `"currencyField":{"definition":"commerce.order_completed.currency","meaningRevision":"1"},`, ``, false, false},
		{"missing literal", `,"nativeCurrency":"EUR"`, ``, false, false},
		{"empty literal", `"nativeCurrency":"EUR"`, `"nativeCurrency":""`, false, false},
		{"lowercase literal", `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, false, false},
		{"short literal", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EU"`, false, false},
		{"long literal", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EURO"`, false, false},
		{"non ASCII literal", `"nativeCurrency":"EUR"`, `"nativeCurrency":"€"`, false, false},
		{"unknown field", `commerce.order_completed.currency`, `NativeCurrency`, false, false},
		{"unknown revision", `"meaningRevision":"1"`, `"meaningRevision":"2"`, false, false},
		{"incompatible event", `"name":"Order Completed"`, `"name":"Refund"`, false, false},
		{"empty reference", `{"definition":"commerce.order_completed.currency","meaningRevision":"1"}`, `{}`, false, false},
		{"parent path", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties","operator":"exists"}]}`, true, false},
		{"currency path", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]}`, true, false},
		{"descendant path", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.currency.code","operator":"exists"}]}`, true, false},
		{"nested OR", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"operator":"or","rules":[{"property":"properties.currency","operator":"is","values":["EUR"]}]}]}`, true, false},
		{"prefix allowed", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.currency_code","operator":"is","values":["EUR"]}]}`, false, true},
		{"case allowed", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.Currency","operator":"is","values":["EUR"]}]}`, false, true},
		{"amount allowed", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.total","operator":"is greater than","values":[30]}]}`, false, true},
		{"coupon allowed", `"nativeCurrency":"EUR"`, `"nativeCurrency":"EUR","where":{"operator":"and","rules":[{"property":"properties.coupon","operator":"is","values":["PROMO"]}]}`, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var condition FilterCountCondition
			err := json.Unmarshal([]byte(strings.Replace(currencyCountConditionJSON, test.old, test.replacement, 1)),
				&condition)
			if err != nil {
				t.Fatalf("expected syntactically valid currency fields, got %v", err)
			}
			_, err = prepareFilterCount(&condition, nil)
			if err != nil {
				_, unsupported := errors.AsType[*unified.UnsupportedError](err)
				if unsupported != test.unsupported || test.supported {
					t.Fatalf("expected unsupported=%v/supported=%v, got %v", test.unsupported, test.supported, err)
				}
				return
			}
			if !test.supported {
				t.Fatal("expected currency qualification rejection, got success")
			}
		})
	}
}

// TestFilterCountCurrencyInvalidBeforeUnsupported protects both sibling orders
// and validity errors within a dependent currency qualification.
func TestFilterCountCurrencyInvalidBeforeUnsupported(t *testing.T) {
	for _, path := range []string{"properties", "properties.currency", "properties.currency.x"} {
		for _, nested := range []bool{false, true} {
			for _, reversed := range []bool{false, true} {
				for _, invalid := range []struct{ name, condition string }{
					{"revision", strings.Replace(currencyCountConditionJSON, `"meaningRevision":"1"`, `"meaningRevision":"2"`, 1)},
					{"source", strings.Replace(currencyCountConditionJSON, `"name":"Order Completed"`, `"name":"Refund"`, 1)},
					{"literal", strings.Replace(currencyCountConditionJSON, `"nativeCurrency":"EUR"`, `"nativeCurrency":"eur"`, 1)},
					{"operand", strings.Replace(currencyCountConditionJSON, `"values":[0]`, `"values":[-1]`, 1)},
				} {
					t.Run(fmt.Sprintf("%s/nested=%v/reversed=%v/%s", path, nested, reversed, invalid.name), func(t *testing.T) {
						where := `{"property":"` + path + `","operator":"exists"}`
						if nested {
							where = `{"operator":"or","rules":[` + where + `]}`
						}
						dependent := strings.Replace(currencyCountConditionJSON, `"nativeCurrency":"EUR"`,
							`"nativeCurrency":"EUR","where":{"operator":"and","rules":[`+where+`]}`, 1)
						var leaf FilterCountCondition
						err := json.Unmarshal([]byte(dependent), &leaf)
						if err != nil {
							t.Fatalf("expected dependent fixture decoded, got %v", err)
						}
						_, err = prepareFilterCount(&leaf, nil)
						if err != nil {
							if _, unsupported := errors.AsType[*unified.UnsupportedError](err); !unsupported {
								t.Fatalf("expected unsupported fixture, got %v", err)
							}
						}
						if err == nil {
							t.Fatal("expected unsupported fixture, got success")
						}
						first, second := dependent, invalid.condition
						if reversed {
							first, second = second, first
						}
						var filter Filter
						err = json.Unmarshal([]byte(`{"operator":"and","rules":[`+first+`,`+second+`]}`), &filter)
						if err != nil {
							t.Fatalf("expected structurally valid combined filter, got %v", err)
						}
						nodes := 0
						schema := types.Object([]types.Property{{Name: "country", Type: types.String()}})
						_, err = prepareUnifiedRule(&filter, schema, 0, &nodes)
						if err != nil {
							if _, unsupported := errors.AsType[*unified.UnsupportedError](err); unsupported {
								t.Fatalf("expected invalidity before unavailable currency qualification, got %v", err)
							}
							return
						}
						t.Fatal("expected invalid configuration, got success")
					})
				}
			}
		}
	}
}
